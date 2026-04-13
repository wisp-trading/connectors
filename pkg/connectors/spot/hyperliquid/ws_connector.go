package hyperliquid

import (
	"fmt"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
)

// ─── WebSocket Lifecycle ────────────────────────────────────────────────────

// StartWebSocket implements connector.WebSocketCapable
func (h *hyperliquidSpot) StartWebSocket() error {
	if h.config == nil {
		return fmt.Errorf("connector not initialized")
	}
	return h.realTime.Connect(h.config.WebsocketURL)
}

// StopWebSocket implements connector.WebSocketCapable
func (h *hyperliquidSpot) StopWebSocket() error {
	return h.realTime.Disconnect()
}

// IsWebSocketConnected implements connector.WebSocketCapable
func (h *hyperliquidSpot) IsWebSocketConnected() bool {
	return h.realTime.IsConnected()
}

// ErrorChannel implements connector.WebSocketCapable
func (h *hyperliquidSpot) ErrorChannel() <-chan error {
	return h.errorCh
}

// ─── Subscriptions ──────────────────────────────────────────────────────────

// SubscribeOrderBook implements spot.WebSocketConnector
func (h *hyperliquidSpot) SubscribeOrderBook(pair portfolio.Pair) error {
	return fmt.Errorf("SubscribeOrderBook not yet implemented for Hyperliquid spot")
}

// UnsubscribeOrderBook implements spot.WebSocketConnector
func (h *hyperliquidSpot) UnsubscribeOrderBook(pair portfolio.Pair) error {
	return fmt.Errorf("UnsubscribeOrderBook not yet implemented for Hyperliquid spot")
}

// SubscribeTrades implements spot.WebSocketConnector
func (h *hyperliquidSpot) SubscribeTrades(pair portfolio.Pair) error {
	return fmt.Errorf("SubscribeTrades not yet implemented for Hyperliquid spot")
}

// UnsubscribeTrades implements spot.WebSocketConnector
func (h *hyperliquidSpot) UnsubscribeTrades(pair portfolio.Pair) error {
	return fmt.Errorf("UnsubscribeTrades not yet implemented for Hyperliquid spot")
}

// SubscribeKlines implements spot.WebSocketConnector
func (h *hyperliquidSpot) SubscribeKlines(pair portfolio.Pair, interval string) error {
	return fmt.Errorf("SubscribeKlines not yet implemented for Hyperliquid spot")
}

// UnsubscribeKlines implements spot.WebSocketConnector
func (h *hyperliquidSpot) UnsubscribeKlines(pair portfolio.Pair, interval string) error {
	return fmt.Errorf("UnsubscribeKlines not yet implemented for Hyperliquid spot")
}

// SubscribeAccountBalance implements spot.WebSocketConnector
func (h *hyperliquidSpot) SubscribeAccountBalance() error {
	return fmt.Errorf("SubscribeAccountBalance not yet implemented for Hyperliquid spot")
}

// UnsubscribeAccountBalance implements spot.WebSocketConnector
func (h *hyperliquidSpot) UnsubscribeAccountBalance() error {
	return fmt.Errorf("UnsubscribeAccountBalance not yet implemented for Hyperliquid spot")
}

// ─── Channel Accessors ──────────────────────────────────────────────────────

// GetOrderBookChannels implements spot.WebSocketConnector
func (h *hyperliquidSpot) GetOrderBookChannels() map[string]<-chan connector.OrderBook {
	h.orderBookMu.RLock()
	defer h.orderBookMu.RUnlock()

	result := make(map[string]<-chan connector.OrderBook, len(h.orderBookChannels))
	for k, v := range h.orderBookChannels {
		result[k] = v
	}
	return result
}

// GetKlineChannels implements spot.WebSocketConnector
func (h *hyperliquidSpot) GetKlineChannels() map[string]<-chan connector.Kline {
	h.klineMu.RLock()
	defer h.klineMu.RUnlock()

	result := make(map[string]<-chan connector.Kline, len(h.klineChannels))
	for k, v := range h.klineChannels {
		result[k] = v
	}
	return result
}

// TradeUpdates implements spot.WebSocketConnector
func (h *hyperliquidSpot) TradeUpdates() <-chan connector.Trade {
	return h.tradeCh
}

// AssetBalanceUpdates implements spot.WebSocketConnector
func (h *hyperliquidSpot) AssetBalanceUpdates() <-chan connector.AssetBalance {
	return h.balanceCh
}
