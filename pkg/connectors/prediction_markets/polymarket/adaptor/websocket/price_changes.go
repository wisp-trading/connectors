package websocket

import (
	"context"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/ws"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

func (w websocket) SubscribePrices(ctx context.Context, market prediction.Market) (<-chan ws.PriceEvent, error) {
	assetIds := make([]string, len(market.Outcomes))
	for i, outcome := range market.Outcomes {
		assetIds[i] = outcome.OutcomeID.String()
	}

	stream, err := w.client.SubscribePrices(ctx, assetIds)
	if err != nil {
		return nil, err
	}

	return stream, nil
}
