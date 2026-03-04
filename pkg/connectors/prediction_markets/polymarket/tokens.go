package polymarket

import (
	"context"

	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

func (p *polymarket) Redeem(market prediction.Market) (string, error) {
	ctx := context.Background()

	hash, err := p.orderManager.RedeemPosition(ctx, market)
	if err != nil {
		return "", err
	}

	return hash, nil
}
