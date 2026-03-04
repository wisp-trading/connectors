package order_manager

import (
	"context"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
)

func (c *orderManager) GetBalance(ctx context.Context) (clobtypes.BalanceAllowanceResponse, error) {
	balanceRequest := &clobtypes.BalanceAllowanceRequest{
		AssetType: clobtypes.AssetTypeCollateral,
	}
	return c.client.BalanceAllowance(ctx, balanceRequest)
}
