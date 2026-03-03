package websocket

import (
	"context"

	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

func (w websocket) UnsubscribeMarket(market prediction.Market) error {
	assetIds := make([]string, len(market.Outcomes))
	for i, outcome := range market.Outcomes {
		assetIds[i] = outcome.OutcomeID.String()
	}

	ctx := context.Background()
	return w.client.UnsubscribeMarketAssets(ctx, assetIds)
}
