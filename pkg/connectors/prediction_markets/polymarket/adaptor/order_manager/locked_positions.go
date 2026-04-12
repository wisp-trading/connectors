package order_manager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

const (
	ctfContractAddress = "0x4D97DCd97eC945f40cF65F87097ACe5EA0476045"
	// PositionSplit(address,address,bytes32,bytes32,uint256[],uint256)
	positionSplitTopic0 = "0x2e6bb91f8cbcda0c93623c54d0403a43514fabc40084ec96b6d5379a74786298"
)

// GetLockedPositions returns all CTF ERC-1155 positions currently held on-chain
// by the signing EOA. It uses:
//
//  1. Alchemy NFT V3 API to enumerate current holdings at the CTF contract.
//  2. alchemy_getAssetTransfers to find the minting transaction for each held token.
//  3. eth_getTransactionReceipt to extract the conditionID from the PositionSplit event.
//
// Returns an empty slice (not an error) when no Polygon RPC URL is configured.
func (c *orderManager) GetLockedPositions(ctx context.Context) ([]prediction.LockedPosition, error) {
	if c.rpcURL == "" {
		return nil, nil
	}

	eoa := c.signer.Address().Hex()

	// 1. Current ERC-1155 holdings at the CTF contract.
	held, err := heldCTFTokens(c.rpcURL, eoa)
	if err != nil {
		return nil, fmt.Errorf("fetch held positions: %w", err)
	}
	if len(held) == 0 {
		return nil, nil
	}

	// 2. All minting transactions (TransferBatch/Single from 0x0 → EOA).
	txTokens, err := mintingTransactions(ctx, c.rpcURL, eoa)
	if err != nil {
		return nil, fmt.Errorf("fetch minting transactions: %w", err)
	}

	// 3. For each minting tx that contains a currently-held token, resolve
	//    the conditionID from the PositionSplit event and group token IDs.
	conditionTokens := make(map[string][]string) // conditionID → []normalised hex tokenID
	for txHash, tokenIDs := range txTokens {
		anyHeld := false
		for _, tid := range tokenIDs {
			if _, ok := held[tid]; ok {
				anyHeld = true
				break
			}
		}
		if !anyHeld {
			continue
		}

		conditionID, err := conditionFromReceipt(ctx, c.rpcURL, txHash)
		if err != nil {
			continue // best-effort; skip txs whose receipt can't be parsed
		}

		for _, tid := range tokenIDs {
			if _, ok := held[tid]; !ok {
				continue
			}
			// Deduplicate: same condition may span multiple split transactions.
			duplicate := false
			for _, existing := range conditionTokens[conditionID] {
				if existing == tid {
					duplicate = true
					break
				}
			}
			if !duplicate {
				conditionTokens[conditionID] = append(conditionTokens[conditionID], tid)
			}
		}
	}

	// 4. Build a LockedPosition for each condition that has ≥2 held tokens.
	var positions []prediction.LockedPosition
	for conditionID, tokenHexIDs := range conditionTokens {
		if len(tokenHexIDs) < 2 {
			continue
		}

		var minBal *big.Int
		outcomeBalances := make(map[prediction.OutcomeID]*big.Int, len(tokenHexIDs))
		for _, hexID := range tokenHexIDs {
			bal := held[hexID]
			tidInt, _ := new(big.Int).SetString(strings.TrimPrefix(hexID, "0x"), 16)
			oid := prediction.OutcomeIDFromString(tidInt.String())
			outcomeBalances[oid] = bal
			if minBal == nil || bal.Cmp(minBal) < 0 {
				minBal = bal
			}
		}

		outcomes := make([]prediction.Outcome, len(tokenHexIDs))
		for i, hexID := range tokenHexIDs {
			tidInt, _ := new(big.Int).SetString(strings.TrimPrefix(hexID, "0x"), 16)
			outcomes[i] = prediction.Outcome{
				OutcomeID: prediction.OutcomeIDFromString(tidInt.String()),
			}
		}
		market := prediction.Market{
			MarketID: prediction.MarketIDFromString(common.HexToHash(conditionID).Hex()),
			Outcomes: outcomes,
		}

		positions = append(positions, prediction.LockedPosition{
			Market:          market,
			OutcomeBalances: outcomeBalances,
			MergeableAmount: minBal,
		})
	}

	return positions, nil
}

// ── Alchemy helpers ───────────────────────────────────────────────────────────

// alchemyRPC sends a JSON-RPC call to the Alchemy endpoint and unmarshals
// the result field into dest.
func alchemyRPC(rpcURL, method string, params interface{}, dest interface{}) error {
	body, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  []interface{}{params},
	})
	resp, err := http.Post(rpcURL, "application/json", bytes.NewReader(body)) //nolint:gosec
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var wrapper struct {
		Result json.RawMessage            `json:"result"`
		Error  *struct{ Message string } `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return err
	}
	if wrapper.Error != nil {
		return fmt.Errorf("alchemy rpc %s: %s", method, wrapper.Error.Message)
	}
	return json.Unmarshal(wrapper.Result, dest)
}

// normaliseTokenID converts a hex or decimal token ID string to the canonical
// form used as map keys: lowercase "0x" + 64 hex digits.
func normaliseTokenID(raw string) string {
	raw = strings.ToLower(raw)
	if strings.HasPrefix(raw, "0x") {
		n, _ := new(big.Int).SetString(raw[2:], 16)
		if n != nil {
			return fmt.Sprintf("0x%064x", n)
		}
	} else {
		n, _ := new(big.Int).SetString(raw, 10)
		if n != nil {
			return fmt.Sprintf("0x%064x", n)
		}
	}
	return raw
}

// heldCTFTokens returns a map of normalised hex tokenID → balance for all
// ERC-1155 tokens currently held by eoa at the CTF contract.
func heldCTFTokens(rpcURL, eoa string) (map[string]*big.Int, error) {
	// Derive the NFT V3 base URL from the Alchemy RPC URL.
	// e.g. https://polygon-mainnet.g.alchemy.com/v2/<key>
	//    → https://polygon-mainnet.g.alchemy.com/nft/v3/<key>
	parts := strings.SplitN(rpcURL, "/v2/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("unsupported RPC URL format (expected Alchemy /v2/<key>): %s", rpcURL)
	}
	nftURL := fmt.Sprintf("%s/nft/v3/%s/getNFTsForOwner?owner=%s&contractAddresses[]=%s&withMetadata=false",
		strings.TrimSuffix(parts[0], "/"), parts[1], eoa, ctfContractAddress)

	var result struct {
		OwnedNFTs []struct {
			TokenID string `json:"tokenId"`
			Balance string `json:"balance"`
		} `json:"ownedNfts"`
	}
	resp, err := http.Get(nftURL) //nolint:gosec
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	out := make(map[string]*big.Int, len(result.OwnedNFTs))
	for _, nft := range result.OwnedNFTs {
		tid := normaliseTokenID(nft.TokenID)
		bal, ok := new(big.Int).SetString(nft.Balance, 10)
		if !ok || bal.Sign() == 0 {
			continue
		}
		out[tid] = bal
	}
	return out, nil
}

// mintingTransactions returns a map of txHash → []normalisedHexTokenID for all
// ERC-1155 mint events (from = 0x0) at the CTF contract targeting the EOA.
func mintingTransactions(_ context.Context, rpcURL, eoa string) (map[string][]string, error) {
	type erc1155Meta struct {
		TokenID string `json:"tokenId"`
	}
	type transfer struct {
		Hash        string        `json:"hash"`
		ERC1155Meta []erc1155Meta `json:"erc1155Metadata"`
	}
	var result struct {
		Transfers []transfer `json:"transfers"`
		PageKey   string     `json:"pageKey"`
	}

	params := map[string]interface{}{
		"category":          []string{"erc1155"},
		"fromAddress":       "0x0000000000000000000000000000000000000000",
		"toAddress":         eoa,
		"contractAddresses": []string{ctfContractAddress},
		"order":             "desc",
		"maxCount":          "0x64",
		"withMetadata":      false,
		"excludeZeroValue":  true,
	}

	txTokens := make(map[string][]string)
	for {
		if err := alchemyRPC(rpcURL, "alchemy_getAssetTransfers", params, &result); err != nil {
			return nil, err
		}
		for _, t := range result.Transfers {
			for _, meta := range t.ERC1155Meta {
				tid := normaliseTokenID(meta.TokenID)
				txTokens[t.Hash] = append(txTokens[t.Hash], tid)
			}
		}
		if result.PageKey == "" {
			break
		}
		params["pageKey"] = result.PageKey
		result = struct {
			Transfers []transfer `json:"transfers"`
			PageKey   string     `json:"pageKey"`
		}{}
	}
	return txTokens, nil
}

// conditionFromReceipt fetches the transaction receipt and returns the
// conditionID from the PositionSplit event (topic[3]).
func conditionFromReceipt(_ context.Context, rpcURL, txHash string) (string, error) {
	var receipt struct {
		Logs []struct {
			Topics []string `json:"topics"`
		} `json:"logs"`
	}
	if err := alchemyRPC(rpcURL, "eth_getTransactionReceipt", txHash, &receipt); err != nil {
		return "", err
	}
	for _, log := range receipt.Logs {
		if len(log.Topics) >= 4 &&
			strings.EqualFold(log.Topics[0], positionSplitTopic0) {
			return log.Topics[3], nil
		}
	}
	return "", fmt.Errorf("no PositionSplit event in tx %s", txHash)
}
