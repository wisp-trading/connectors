package order_manager

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/ctf"
	"github.com/ethereum/go-ethereum/common"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

// isEmptyBodyErr returns true for errors that indicate an HTTP 200 response
// with an empty body — the CLOB's balance-allowance/update endpoint does this.
func isEmptyBodyErr(err error) bool {
	if err == nil {
		return false
	}
	if err == io.EOF {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unexpected end of json") || strings.Contains(msg, "eof")
}

func maxUint256() *big.Int {
	return new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
}

// usdcAddressHex is the native USDC ERC-20 on Polygon mainnet (chain 137).
// Polymarket migrated from the bridged USDC.e (0x2791…) to native USDC in 2024.
const usdcAddressHex = "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359"

// clobBalancePollInterval is how often we re-check the CLOB's cached balance.
const clobBalancePollInterval = 500 * time.Millisecond

// clobBalancePollTimeout is the maximum time to wait for the CLOB to reflect
// an on-chain balance after calling UpdateBalanceAllowance.
const clobBalancePollTimeout = 30 * time.Second

// SplitPosition ensures the CTF contract is approved to spend USDC, submits
// the split transaction, and returns immediately with the tx hash and a ready
// channel. The ready channel receives nil (then closes) once the transaction
// is mined AND the CLOB has confirmed the conditional token balance is >= amountUSDC.
// Callers MUST receive from ready before placing any SELL orders.
//
// amountUSDC is in raw USDC units (6 decimals): $1.00 = 1_000_000.
func (c *orderManager) SplitPosition(ctx context.Context, market prediction.Market, amountUSDC *big.Int) (string, <-chan error, error) {
	usdcAddr := common.HexToAddress(usdcAddressHex)

	// Ensure the CTF contract has sufficient ERC-20 allowance before submitting.
	// The CTF client automatically detects if this is a SafeSigner and routes
	// appropriately (uses owner address for signatures).
	if err := c.tokenManagement.EnsureCollateralApproved(ctx, usdcAddr, amountUSDC); err != nil {
		return "", nil, fmt.Errorf("approve USDC for CTF: %w", err)
	}

	txHash, mined, err := c.tokenManagement.SplitPositionAsync(ctx, &ctf.SplitPositionRequest{
		CollateralToken:    usdcAddr,
		ParentCollectionID: common.Hash{},
		ConditionID:        common.HexToHash(market.MarketID.String()),
		Partition:          ctf.BinaryPartition,
		Amount:             amountUSDC,
	})
	if err != nil {
		return "", nil, err
	}

	// ready closes once the tx is mined AND the CLOB has confirmed the balance.
	// Uses context.Background() so the goroutine outlives the caller's context.
	ready := make(chan error, 1)
	amount := new(big.Int).Set(amountUSDC)
	go func() {
		defer close(ready)
		if err := <-mined; err != nil {
			ready <- fmt.Errorf("split tx not mined: %w", err)
			return
		}
		// Tx confirmed — notify CLOB and poll until it confirms the balance.
		// The UpdateBalanceAllowance endpoint is async server-side; we must poll
		// BalanceAllowance until the CLOB reflects the on-chain state, otherwise
		// SELL orders land before the CLOB's cache is updated and are rejected.
		if err := confirmConditionalBalances(context.Background(), c, market, amount); err != nil {
			// Log but don't fail — the SELL order will surface its own error if needed.
			fmt.Printf("[polymarket:split] warn: CLOB balance confirmation timed out: %v\n", err)
		}
	}()

	return txHash.Hex(), ready, nil
}

// confirmConditionalBalances triggers a CLOB balance refresh for every outcome
// token in the market, then polls BalanceAllowance until each token's cached
// balance is >= minAmount. Returns an error only on timeout.
func confirmConditionalBalances(ctx context.Context, c *orderManager, market prediction.Market, minAmount *big.Int) error {
	deadline := time.Now().Add(clobBalancePollTimeout)
	pollCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	for _, outcome := range market.Outcomes {
		tokenID := outcome.OutcomeID.String()

		// Trigger the CLOB to re-read the on-chain ERC-1155 balance.
		_, err := c.client.UpdateBalanceAllowance(pollCtx, &clobtypes.BalanceAllowanceUpdateRequest{
			AssetType: clobtypes.AssetTypeConditional,
			TokenID:   tokenID,
		})
		if err != nil && !isEmptyBodyErr(err) {
			fmt.Printf("[polymarket:split] warn: balance notify failed for token %s: %v\n", tokenID, err)
		}

		// Poll until the CLOB's cached balance reflects what was minted.
		for {
			resp, err := c.client.BalanceAllowance(pollCtx, &clobtypes.BalanceAllowanceRequest{
				AssetType: clobtypes.AssetTypeConditional,
				TokenID:   tokenID,
			})
			if err == nil {
				bal, ok := new(big.Int).SetString(resp.Balance, 10)
				if ok && bal.Cmp(minAmount) >= 0 {
					fmt.Printf("[polymarket:split] CLOB confirmed balance for token %s: %s\n", tokenID, resp.Balance)
					break
				}
			}

			select {
			case <-pollCtx.Done():
				return fmt.Errorf("timed out waiting for CLOB to confirm balance for token %s (wanted >=%s)", tokenID, minAmount)
			case <-time.After(clobBalancePollInterval):
			}
		}
	}
	return nil
}

// confirmCollateralBalance triggers a CLOB refresh of the EOA's on-chain USDC
// balance and polls until the CLOB reports balance > 0. Required before BUY
// orders — the CLOB's collateral balance is not automatically kept in sync with
// on-chain state for EOA signers.
func confirmCollateralBalance(ctx context.Context, c *orderManager) error {
	deadline := time.Now().Add(clobBalancePollTimeout)
	pollCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	// Trigger CLOB to re-read on-chain USDC balance for this EOA.
	_, err := c.client.UpdateBalanceAllowance(pollCtx, &clobtypes.BalanceAllowanceUpdateRequest{
		AssetType: clobtypes.AssetTypeCollateral,
	})
	if err != nil && !isEmptyBodyErr(err) {
		fmt.Printf("[polymarket:order] warn: collateral balance notify failed: %v\n", err)
	}

	// Poll until the CLOB has a non-zero USDC balance cached for this EOA.
	for {
		resp, err := c.client.BalanceAllowance(pollCtx, &clobtypes.BalanceAllowanceRequest{
			AssetType: clobtypes.AssetTypeCollateral,
		})
		if err == nil {
			bal, ok := new(big.Int).SetString(resp.Balance, 10)
			if ok && bal.Sign() > 0 {
				fmt.Printf("[polymarket:order] CLOB confirmed collateral balance: %s\n", resp.Balance)
				return nil
			}
		}

		select {
		case <-pollCtx.Done():
			return fmt.Errorf("timed out waiting for CLOB to confirm collateral balance")
		case <-time.After(clobBalancePollInterval):
		}
	}
}

// ConfirmConditionalBalance triggers a CLOB balance refresh for every outcome
// in the market and polls until each reports a balance of at least minAmount.
// Use after CLOB buy fills to wait for on-chain settlement before MergePositions.
func (c *orderManager) ConfirmConditionalBalance(ctx context.Context, market prediction.Market, minAmount *big.Int) error {
	return confirmConditionalBalances(ctx, c, market, minAmount)
}

// MergePositions burns amountUSDC worth of YES + NO tokens and returns USDC.
// amountUSDC is in raw USDC units (6 decimals): $1.00 = 1_000_000.
func (c *orderManager) MergePositions(ctx context.Context, market prediction.Market, amountUSDC *big.Int) (string, error) {
	resp, err := c.tokenManagement.MergePositions(ctx, &ctf.MergePositionsRequest{
		CollateralToken:    common.HexToAddress(usdcAddressHex),
		ParentCollectionID: common.Hash{},
		ConditionID:        common.HexToHash(market.MarketID.String()),
		Partition:          ctf.BinaryPartition,
		Amount:             amountUSDC,
	})
	if err != nil {
		return "", err
	}
	return resp.TransactionHash.Hex(), nil
}
