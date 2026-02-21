package websocket

import (
	"context"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/ws"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
)

func (w websocket) SubscribeOrders(market prediction.Market) (<-chan ws.OrderEvent, error) {
	ctx := context.Background()

	orderStream, err := w.client.SubscribeUserOrders(ctx, []string{market.MarketID.String()})

	if err != nil {
		return nil, err
	}
	return orderStream, nil
}
