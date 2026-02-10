package websocket

// PolymarketWebsocket defines the WebSocket interface for Polymarket CLOB real-time market data
type PolymarketWebsocket interface {
	Connect(wsURL string) error
	Disconnect() error
	IsConnected() bool
	GetErrorChannel() <-chan error

	SubscribeToMarketBook(marketID string, callback func(*OrderBookMessage))
	UnsubscribeFromMarketBook(marketID string)
}

// SubscriptionHandler manages a subscription callback
type SubscriptionHandler struct {
	ID       int
	Channel  string
	Asset    string // market slug or token ID
	Callback interface{}
}
