package order_manager

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/auth"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/ctf"
	"github.com/ethereum/go-ethereum/common"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

// GetBalance returns the spendable collateral balance.
//
// When a Polygon RPC backend is configured (polygon_rpc_url set), it reads the
// on-chain ERC-20 balance of the signing address directly from the chain — this
// is required when the signer is an EOA whose capital sits on-chain rather than
// in a CLOB-managed Safe wallet.
//
// When no backend is available (ErrMissingBackend / ErrMissingTransactor), it
// falls back to the Polymarket CLOB API balance. The caller sees the same
// BalanceAllowanceResponse either way.
func (c *orderManager) GetBalance(ctx context.Context) (clobtypes.BalanceAllowanceResponse, error) {
	// For Safe wallets, skip on-chain balance query and go directly to CLOB API
	// because the CLOB tracks Safe's balance under the Safe address, not the EOA's address.
	// For EOA wallets, query on-chain first for accuracy.
	if c.sigType != auth.SignatureEOA {
		// Safe or Proxy mode: use CLOB API balance (which tracks funder address)
		return c.client.BalanceAllowance(ctx, &clobtypes.BalanceAllowanceRequest{
			AssetType: clobtypes.AssetTypeCollateral,
		})
	}

	onChain, err := c.tokenManagement.CollateralBalance(ctx, common.HexToAddress(usdcAddressHex))
	if err == nil {
		return clobtypes.BalanceAllowanceResponse{Balance: onChain.String()}, nil
	}

	// Only fall back for "no backend" errors — any other error (RPC timeout,
	// bad chain state) should surface rather than silently reading stale CLOB data.
	if !errors.Is(err, ctf.ErrMissingBackend) && !errors.Is(err, ctf.ErrMissingTransactor) {
		return clobtypes.BalanceAllowanceResponse{}, err
	}

	return c.client.BalanceAllowance(ctx, &clobtypes.BalanceAllowanceRequest{
		AssetType: clobtypes.AssetTypeCollateral,
	})
}

func (c *orderManager) GetMarketBalance(ctx context.Context, market prediction.Market) (map[prediction.OutcomeID]clobtypes.BalanceAllowanceResponse, error) {
	balances := make(map[prediction.OutcomeID]clobtypes.BalanceAllowanceResponse, len(market.Outcomes))

	for _, outcome := range market.Outcomes {
		tokenID := outcome.OutcomeID.String()

		// Notify the CLOB to refresh its on-chain ERC-1155 balance cache for this
		// token before reading. Required when tokens were minted directly via
		// SplitPosition (EOA path) rather than bought through the order book.
		// The endpoint returns HTTP 200 with an empty body — ignore EOF errors.
		_, err := c.client.UpdateBalanceAllowance(ctx, &clobtypes.BalanceAllowanceUpdateRequest{
			AssetType: clobtypes.AssetTypeConditional,
			TokenID:   tokenID,
		})
		if err != nil && !isEmptyBodyErr(err) {
			return nil, fmt.Errorf("refresh balance for token %s: %w", tokenID, err)
		}

		response, err := c.client.BalanceAllowance(ctx, &clobtypes.BalanceAllowanceRequest{
			AssetType: clobtypes.AssetTypeConditional,
			TokenID:   tokenID,
		})
		if err != nil {
			return nil, err
		}

		balances[outcome.OutcomeID] = response
	}

	return balances, nil
}

