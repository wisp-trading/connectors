package order_manager

import (
	"context"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

func (c *orderManager) GetBalance(ctx context.Context) (clobtypes.BalanceAllowanceResponse, error) {
	balanceRequest := &clobtypes.BalanceAllowanceRequest{
		AssetType: clobtypes.AssetTypeCollateral,
	}
	return c.client.BalanceAllowance(ctx, balanceRequest)
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
