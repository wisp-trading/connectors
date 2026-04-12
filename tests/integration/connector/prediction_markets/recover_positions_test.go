package prediction_markets_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

const (
	ctfContractAddress = "0x4D97DCd97eC945f40cF65F87097ACe5EA0476045"
	// PositionSplit(address,address,bytes32,bytes32,uint256[],uint256) topic0
	positionSplitTopic0 = "0x2e6bb91f8cbcda0c93623c54d0403a43514fabc40084ec96b6d5379a74786298"
)

// alchemyPost sends a JSON-RPC request to the Alchemy endpoint and decodes
// the result field into dest.
func alchemyPost(rpcURL string, method string, params interface{}, dest interface{}) error {
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
		return fmt.Errorf("alchemy: %s", wrapper.Error.Message)
	}
	return json.Unmarshal(wrapper.Result, dest)
}

// alchemyNFTGet calls the Alchemy NFT V3 REST endpoint and decodes into dest.
func alchemyNFTGet(nftURL string, dest interface{}) error {
	resp, err := http.Get(nftURL) //nolint:gosec
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(dest)
}

// heldPositions returns all ERC-1155 token IDs (as *big.Int) currently held
// by the EOA at the CTF contract, along with their on-chain balances.
func heldPositions(rpcBaseURL, owner string) (map[string]*big.Int, error) {
	// Parse the base RPC URL to build the NFT V3 URL, e.g.:
	// https://polygon-mainnet.g.alchemy.com/v2/<key>
	// → https://polygon-mainnet.g.alchemy.com/nft/v3/<key>/getNFTsForOwner
	rpcURL := strings.TrimSuffix(rpcBaseURL, "/")
	parts := strings.SplitN(rpcURL, "/v2/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("unexpected Alchemy RPC URL format: %s", rpcURL)
	}
	apiKey := parts[1]
	host := parts[0]
	nftBase := fmt.Sprintf("%s/nft/v3/%s", host, apiKey)

	url := fmt.Sprintf("%s/getNFTsForOwner?owner=%s&contractAddresses[]=%s&withMetadata=false",
		nftBase, owner, ctfContractAddress)

	var result struct {
		OwnedNFTs []struct {
			TokenID  string `json:"tokenId"`
			Balance  string `json:"balance"`
		} `json:"ownedNfts"`
	}
	if err := alchemyNFTGet(url, &result); err != nil {
		return nil, err
	}

	out := make(map[string]*big.Int, len(result.OwnedNFTs))
	for _, nft := range result.OwnedNFTs {
		// tokenId may be decimal or hex depending on API version — normalise to
		// lowercase hex with 0x prefix and 64 hex digits so it matches the
		// token IDs returned by alchemy_getAssetTransfers.
		var tidInt *big.Int
		raw := strings.TrimPrefix(strings.ToLower(nft.TokenID), "0x")
		if strings.HasPrefix(strings.ToLower(nft.TokenID), "0x") {
			tidInt, _ = new(big.Int).SetString(raw, 16)
		} else {
			tidInt, _ = new(big.Int).SetString(nft.TokenID, 10)
		}
		if tidInt == nil {
			continue
		}
		tid := fmt.Sprintf("0x%064x", tidInt)

		bal, ok := new(big.Int).SetString(nft.Balance, 10)
		if !ok {
			continue
		}
		if bal.Sign() > 0 {
			out[tid] = bal
		}
	}
	return out, nil
}

// mintingTxsForOwner returns a map of txHash → []tokenIDHex for all
// TransferBatch/TransferSingle mint events (from=0x0) at the CTF contract
// targeting the owner.
func mintingTxsForOwner(rpcURL, owner string) (map[string][]string, error) {
	type transfer struct {
		Hash          string `json:"hash"`
		ERC1155Meta   []struct {
			TokenID string `json:"tokenId"`
		} `json:"erc1155Metadata"`
	}
	var result struct {
		Transfers []transfer `json:"transfers"`
		PageKey   string     `json:"pageKey"`
	}

	txTokens := make(map[string][]string)
	params := map[string]interface{}{
		"category":         []string{"erc1155"},
		"fromAddress":      "0x0000000000000000000000000000000000000000",
		"toAddress":        owner,
		"contractAddresses": []string{ctfContractAddress},
		"order":            "desc",
		"maxCount":         "0x64",
		"withMetadata":     false,
		"excludeZeroValue": true,
	}

	for {
		if err := alchemyPost(rpcURL, "alchemy_getAssetTransfers", params, &result); err != nil {
			return nil, err
		}
		for _, t := range result.Transfers {
			for _, meta := range t.ERC1155Meta {
				// Normalise to 0x-prefixed 64-char hex to match heldPositions keys.
				raw := strings.TrimPrefix(strings.ToLower(meta.TokenID), "0x")
				tidInt, ok := new(big.Int).SetString(raw, 16)
				if !ok {
					continue
				}
				tid := fmt.Sprintf("0x%064x", tidInt)
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

// conditionForTx fetches the transaction receipt and returns the conditionID
// from the PositionSplit event (topic[3]).
func conditionForTx(rpcURL, txHash string) (string, error) {
	var receipt struct {
		Logs []struct {
			Topics []string `json:"topics"`
		} `json:"logs"`
	}
	if err := alchemyPost(rpcURL, "eth_getTransactionReceipt", txHash, &receipt); err != nil {
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

var _ = Describe("Recover locked positions", func() {
	var runner *connector_test.PredictionMarketTestRunner

	BeforeEach(func() {
		if os.Getenv("POLYGON_RPC_URL") == "" {
			Skip("POLYGON_RPC_URL not set")
		}
		var err error
		runner, err = connector_test.NewPredictionMarketTestRunner(
			connector_test.GetTestPredictionMarketConnectorName(),
			connector_test.GetPredictionMarketConnectorConfig(),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if runner != nil {
			runner.Cleanup()
		}
	})

	It("merges all locked YES+NO token pairs back to USDC", func() {
		rpcURL := os.Getenv("POLYGON_RPC_URL")

		privateKeyHex := strings.TrimPrefix(os.Getenv("POLYMARKET_PRIVATE_KEY"), "0x")
		key, err := crypto.HexToECDSA(privateKeyHex)
		Expect(err).ToNot(HaveOccurred())
		eoa := crypto.PubkeyToAddress(key.PublicKey)

		rpcClient, err := ethclient.Dial(rpcURL)
		Expect(err).ToNot(HaveOccurred())
		defer rpcClient.Close()

		ctx := context.Background()
		_ = ctx

		fmt.Fprintf(GinkgoWriter, "EOA: %s\n", eoa.Hex())

		// ── 1. Get current ERC-1155 holdings at the CTF contract ──────────────
		held, err := heldPositions(rpcURL, eoa.Hex())
		Expect(err).ToNot(HaveOccurred())
		fmt.Fprintf(GinkgoWriter, "Held positions: %d token IDs\n", len(held))
		if len(held) == 0 {
			fmt.Fprintf(GinkgoWriter, "No positions held — nothing to recover\n")
			return
		}

		// ── 2. Get all minting transactions (from 0x0) for this EOA ───────────
		txTokens, err := mintingTxsForOwner(rpcURL, eoa.Hex())
		Expect(err).ToNot(HaveOccurred())

		// ── 3. For each minting tx containing held tokens, get the conditionID ─
		// condition → [tokenIDHex, ...]
		conditionTokens := make(map[string][]string)
		for txHash, tokenIDs := range txTokens {
			// Check if this tx minted any currently-held tokens
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

			conditionID, err := conditionForTx(rpcURL, txHash)
			if err != nil {
				fmt.Fprintf(GinkgoWriter, "  warn: %v\n", err)
				continue
			}
			for _, tid := range tokenIDs {
				if _, ok := held[tid]; ok {
					// Deduplicate: same condition may appear across multiple split txs.
					already := false
					for _, existing := range conditionTokens[conditionID] {
						if existing == tid {
							already = true
							break
						}
					}
					if !already {
						conditionTokens[conditionID] = append(conditionTokens[conditionID], tid)
					}
				}
			}
		}
		fmt.Fprintf(GinkgoWriter, "Active conditions with held tokens: %d\n", len(conditionTokens))

		conn := runner.GetPredictionMarketConnector()
		totalRecovered := big.NewInt(0)

		// ── 4. Merge each condition ────────────────────────────────────────────
		for conditionID, tokenHexIDs := range conditionTokens {
			if len(tokenHexIDs) < 2 {
				fmt.Fprintf(GinkgoWriter, "  %s...: only %d token(s) — cannot merge\n",
					conditionID[:10], len(tokenHexIDs))
				continue
			}

			// Minimum balance across all outcome tokens
			var minBal *big.Int
			for _, hexID := range tokenHexIDs {
				bal := held[hexID]
				if minBal == nil || bal.Cmp(minBal) < 0 {
					minBal = bal
				}
			}
			if minBal == nil || minBal.Sign() == 0 {
				continue
			}

			human := new(big.Float).Quo(new(big.Float).SetInt(minBal), new(big.Float).SetFloat64(1_000_000))
			fmt.Fprintf(GinkgoWriter, "  %s...: merging $%s USDC (tokens: %v)\n",
				conditionID[:10], human.Text('f', 2), tokenHexIDs)

			// Build prediction.Market from conditionID and token hex IDs
			outcomes := make([]prediction.Outcome, len(tokenHexIDs))
			for i, hexID := range tokenHexIDs {
				// Convert hex token ID to decimal string for OutcomeID
				b, _ := new(big.Int).SetString(strings.TrimPrefix(hexID, "0x"), 16)
				outcomes[i] = prediction.Outcome{
					OutcomeID: prediction.OutcomeIDFromString(b.String()),
				}
			}
			market := prediction.Market{
				MarketID: prediction.MarketIDFromString(common.HexToHash(conditionID).Hex()),
				Outcomes: outcomes,
			}

			txHash, err := conn.MergePositions(market, minBal)
			if err != nil {
				fmt.Fprintf(GinkgoWriter, "  %s...: MergePositions failed: %v\n",
					conditionID[:10], err)
				continue
			}
			fmt.Fprintf(GinkgoWriter, "  %s...: tx %s\n", conditionID[:10], txHash)
			totalRecovered.Add(totalRecovered, minBal)
			time.Sleep(3 * time.Second)
		}

		recovered := new(big.Float).Quo(new(big.Float).SetInt(totalRecovered), new(big.Float).SetFloat64(1_000_000))
		fmt.Fprintf(GinkgoWriter, "\nTotal recovered: $%s USDC\n", recovered.Text('f', 2))
		Expect(totalRecovered.Sign()).To(BeNumerically(">", 0), "should recover at least some USDC")
	})
})

var _ prediction.Connector
