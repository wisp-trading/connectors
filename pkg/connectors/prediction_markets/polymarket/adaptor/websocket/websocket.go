package websocket

import (
	"context"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/ws"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

type Websocket interface {
	SubscribeOrderbook(ctx context.Context, market prediction.Market) (<-chan ws.OrderbookEvent, error)
	SubscribePrices(ctx context.Context, market prediction.Market) (<-chan ws.PriceChangeEvent, error)
	SubscribeUserTrades(ctx context.Context, market prediction.Market) (<-chan ws.TradeEvent, error)
	SubscribeUserOrders(ctx context.Context, market prediction.Market) (<-chan ws.OrderEvent, error)
	Close()
}

type websocket struct {
	client ws.Client
}

func NewWebsocket(client ws.Client) Websocket {
	return websocket{
		client: client,
	}
}

func (w websocket) Close() {
	w.client = w.client.Deauthenticate()
}
