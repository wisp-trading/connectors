package websocket

import (
	"context"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/ws"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

func (w websocket) SubscribePriceChanges(market prediction.Market) (<-chan ws.PriceChangeEvent, error) {
	assetIds := make([]string, len(market.Outcomes))
	for i, outcome := range market.Outcomes {
		assetIds[i] = outcome.OutcomeID.String()
	}

	ctx := context.Background()
	stream, err := w.client.SubscribePrices(ctx, assetIds)
	if err != nil {
		return nil, err
	}

	return stream, nil
}
