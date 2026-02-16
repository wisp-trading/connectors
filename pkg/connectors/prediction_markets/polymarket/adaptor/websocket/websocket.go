package websocket

import (
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/ws"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
)

type Websocket interface {
	SubscribeOrderbook(market prediction.Market) (<-chan ws.OrderbookEvent, error)
	SubscribePriceChanges(market prediction.Market) (<-chan ws.PriceChangeEvent, error)
	SubscribeTrades(market prediction.Market) (<-chan ws.TradeEvent, error)
	UnsubscribeMarket(market prediction.Market) error
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
