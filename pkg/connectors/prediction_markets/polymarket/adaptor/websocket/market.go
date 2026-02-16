package websocket

import (
	"context"

	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
)

func (w websocket) UnsubscribeMarket(market prediction.Market) error {
	assetIds := make([]string, len(market.Outcomes))
	for i, outcome := range market.Outcomes {
		assetIds[i] = outcome.OutcomeId
	}

	ctx := context.Background()
	return w.client.UnsubscribeMarketAssets(ctx, assetIds)
}
