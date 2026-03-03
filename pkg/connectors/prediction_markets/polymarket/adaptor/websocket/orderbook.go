package websocket

import (
	"context"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/ws"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

func (w websocket) SubscribeOrderbook(market prediction.Market) (<-chan ws.OrderbookEvent, error) {
	assetIds := make([]string, len(market.Outcomes))
	for i, outcome := range market.Outcomes {
		assetIds[i] = outcome.OutcomeID.String()
	}

	ctx := context.Background()
	return w.client.SubscribeOrderbook(ctx, assetIds)
}
