package order_manager

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/auth"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/ctf"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
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

// GetNativeBalance queries the Polygon RPC for the signing address's native MATIC balance.
func (c *orderManager) GetNativeBalance(ctx context.Context) (*big.Int, error) {
	if c.rpcURL == "" {
		return big.NewInt(0), nil
	}

	client, err := ethclient.DialContext(ctx, c.rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial polygon rpc: %w", err)
	}
	defer client.Close()

	addr := c.resolveBalanceAddress()
	balance, err := client.BalanceAt(ctx, addr, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch MATIC balance for %s: %w", addr.Hex(), err)
	}
	return balance, nil
}

// resolveBalanceAddress returns the address whose balances should be queried.
// For Safe wallets, capital sits on the Safe address; for EOA, it's the signer address.
func (c *orderManager) resolveBalanceAddress() common.Address {
	if c.sigType != auth.SignatureEOA && c.safeAddr != (common.Address{}) {
		return c.safeAddr
	}
	return common.HexToAddress(c.signer.Address().String())
}

