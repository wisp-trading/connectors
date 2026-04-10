package order_manager

import (
	"context"
	"errors"

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
		balanceRequest := &clobtypes.BalanceAllowanceRequest{
			AssetType: clobtypes.AssetTypeConditional,
			TokenID:   outcome.OutcomeID.String(),
		}

		response, err := c.client.BalanceAllowance(ctx, balanceRequest)
		if err != nil {
			return nil, err
		}

		balances[outcome.OutcomeID] = response
	}

	return balances, nil
}
