package websocket

import "github.com/wisp-trading/sdk/pkg/types/connector/prediction"

// PolymarketWebsocket defines the WebSocket interface for Polymarket CLOB real-time market data
type PolymarketWebsocket interface {
	Connect(wsURL string) error
	Disconnect() error
	IsConnected() bool
	GetErrorChannel() <-chan error

	SubscribeToMarket(
		market prediction.Market,
		orderbookCallback func(*OrderBookMessage),
		priceChangeCallback func(changes *PriceChanges),
	)
	UnsubscribeFromMarket(market prediction.Market)
}

// SubscriptionHandler manages a subscription callback
type SubscriptionHandler struct {
	ID       int
	Channel  string
	Asset    string // market slug or token ID
	Callback interface{}
}
