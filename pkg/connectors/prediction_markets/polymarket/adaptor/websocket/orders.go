package websocket

import (
	"context"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/ws"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

func (w websocket) SubscribeUserOrders(ctx context.Context, market prediction.Market) (<-chan ws.OrderEvent, error) {
	orderStream, err := w.client.SubscribeUserOrders(ctx, []string{market.MarketID.String()})
	if err != nil {
		return nil, err
	}
	return orderStream, nil
}
