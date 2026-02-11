package websocket

import (
	"go.uber.org/fx"
)

const (
	// PolymarketWSURL is the default WebSocket URL for Polymarket CLOB
	PolymarketWSURL = "wss://ws-subscriptions-clob.polymarket.com/ws/market"
)

// WebSocketModule provides the WebSocket service factory for Polymarket
// The factory creates fully-configured services at runtime (after config is available)
var WebSocketModule = fx.Module("polymarket_websocket",
	fx.Provide(
		// Just provide the factory - no config dependencies!
		NewWebSocketServiceFactory,
	),
)
