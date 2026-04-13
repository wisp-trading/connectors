package websocket

import (
	// Reuse the perps websocket infrastructure — Hyperliquid uses the same
	// WS protocol and message format for both spot and perps markets.
	perpws "github.com/wisp-trading/connectors/pkg/connectors/perps/hyperliquid/websocket"
	"github.com/wisp-trading/connectors/pkg/websocket/base"
	"github.com/wisp-trading/connectors/pkg/websocket/connection"
	"github.com/wisp-trading/sdk/pkg/types/logging"
	"github.com/wisp-trading/sdk/pkg/types/temporal"
)

// SpotRealTimeService handles WebSocket connections for spot market data.
// Hyperliquid uses the same WS endpoint and message format for spot and perp,
// so this wraps the perps RealTimeService with a spot-specific interface.
type SpotRealTimeService interface {
	Connect(wsURL string) error
	Disconnect() error
	IsConnected() bool
	GetErrorChannel() <-chan error

	// Orderbook subscriptions
	SubscribeToOrderBook(coin string, callback func(*perpws.OrderBookMessage)) (int, error)
	UnsubscribeFromOrderBook(coin string, subscriptionID int) error

	// Trade subscriptions
	SubscribeToTrades(coin string, callback func([]perpws.TradeMessage)) (int, error)
	UnsubscribeFromTrades(coin string, subscriptionID int) error

	// Kline subscriptions
	SubscribeToKlines(coin, interval string, callback func(*perpws.KlineMessage)) (int, error)
	UnsubscribeFromKlines(coin, interval string, subscriptionID int) error

	// Account balance subscriptions
	SubscribeToAccountBalance(user string, callback func(*perpws.AccountBalanceMessage)) (int, error)
}

// spotRealTimeService delegates to the perps WebSocketService, providing
// a thin adapter that accepts a plain string URL (the spot config format)
// rather than a *string pointer.
type spotRealTimeService struct {
	inner perpws.RealTimeService
}

// NewSpotRealTimeService creates a spot WebSocket service backed by the
// shared Hyperliquid WebSocket infrastructure.
func NewSpotRealTimeService(
	connManager connection.ConnectionManager,
	reconnectMgr connection.ReconnectManager,
	baseService base.BaseService,
	logger logging.ApplicationLogger,
	timeProvider temporal.TimeProvider,
) (SpotRealTimeService, error) {
	parser := perpws.NewParser(logger, timeProvider)
	inner, err := perpws.NewWebSocketService(connManager, reconnectMgr, baseService, logger, parser)
	if err != nil {
		return nil, err
	}
	return &spotRealTimeService{inner: inner}, nil
}

func (s *spotRealTimeService) Connect(wsURL string) error {
	return s.inner.Connect(&wsURL)
}

func (s *spotRealTimeService) Disconnect() error {
	return s.inner.Disconnect()
}

func (s *spotRealTimeService) IsConnected() bool {
	return s.inner.IsConnected()
}

func (s *spotRealTimeService) GetErrorChannel() <-chan error {
	return s.inner.GetErrorChannel()
}

func (s *spotRealTimeService) SubscribeToOrderBook(coin string, callback func(*perpws.OrderBookMessage)) (int, error) {
	return s.inner.SubscribeToOrderBook(coin, callback)
}

func (s *spotRealTimeService) UnsubscribeFromOrderBook(coin string, subscriptionID int) error {
	return s.inner.UnsubscribeFromOrderBook(coin, subscriptionID)
}

func (s *spotRealTimeService) SubscribeToTrades(coin string, callback func([]perpws.TradeMessage)) (int, error) {
	return s.inner.SubscribeToTrades(coin, callback)
}

func (s *spotRealTimeService) UnsubscribeFromTrades(coin string, subscriptionID int) error {
	return s.inner.UnsubscribeFromTrades(coin, subscriptionID)
}

func (s *spotRealTimeService) SubscribeToKlines(coin, interval string, callback func(*perpws.KlineMessage)) (int, error) {
	return s.inner.SubscribeToKlines(coin, interval, callback)
}

func (s *spotRealTimeService) UnsubscribeFromKlines(coin, interval string, subscriptionID int) error {
	return s.inner.UnsubscribeFromKlines(coin, interval, subscriptionID)
}

func (s *spotRealTimeService) SubscribeToAccountBalance(user string, callback func(*perpws.AccountBalanceMessage)) (int, error) {
	return s.inner.SubscribeToAccountBalance(user, callback)
}
