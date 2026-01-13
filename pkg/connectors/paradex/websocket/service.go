package websockets

import (
	"context"
	"sync"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/temporal"
	"github.com/backtesting-org/live-trading/pkg/connectors/paradex/adaptor"
	"github.com/backtesting-org/live-trading/pkg/websocket/base"
	"github.com/backtesting-org/live-trading/pkg/websocket/connection"
	"github.com/backtesting-org/live-trading/pkg/websocket/performance"
	"github.com/backtesting-org/live-trading/pkg/websocket/security"
)

// Ensure service implements WebSocketService interface at compile time
var _ WebSocketService = (*service)(nil)

type service struct {
	connectionManager connection.ConnectionManager
	reconnectManager  connection.ReconnectManager
	handlerRegistry   *base.HandlerRegistry
	subManager        *subscriptionManager

	client            *adaptor.Client
	applicationLogger logging.ApplicationLogger
	tradingLogger     logging.TradingLogger
	timeProvider      temporal.TimeProvider

	requestID    int64
	requestMutex sync.Mutex
	writeMutex   sync.Mutex

	orderbookChan chan OrderbookUpdate
	tradeChan     chan TradeUpdate
	accountChan   chan AccountUpdate
	errorChan     chan error

	// Add kline builder
	klineBuilder *KlineBuilder
	klineChan    chan KlineUpdate
}

func NewService(
	client *adaptor.Client,
	webSocketURL string,
	logger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
	timeProvider temporal.TimeProvider,
) WebSocketService {
	connConfig := connection.TradingConfig(webSocketURL)
	authManager := security.NewAuthManager(&ParadexAuthProvider{client: client}, logger)
	metrics := performance.NewMetrics()
	dialer := connection.NewGorillaDialer(connConfig)

	connectionManager := connection.NewConnectionManager(connConfig, authManager, metrics, logger, dialer)
	reconnectStrategy := connection.NewExponentialBackoffStrategy(5*time.Second, 5*time.Minute, 10)
	reconnectManager := connection.NewReconnectManager(connectionManager, reconnectStrategy, logger)
	handlerRegistry := base.NewHandlerRegistry(logger)

	service := &service{
		connectionManager: connectionManager,
		reconnectManager:  reconnectManager,
		handlerRegistry:   handlerRegistry,
		subManager:        newSubscriptionManager(),
		client:            client,
		applicationLogger: logger,
		tradingLogger:     tradingLogger,
		timeProvider:      timeProvider,

		orderbookChan: make(chan OrderbookUpdate, 1000),
		tradeChan:     make(chan TradeUpdate, 1000),
		accountChan:   make(chan AccountUpdate, 100),
		errorChan:     make(chan error, 10),

		// Initialize kline builder
		klineBuilder: NewKlineBuilder(timeProvider),
		klineChan:    make(chan KlineUpdate, 1000),
	}

	service.setupCallbacks()
	service.registerHandlers()

	// Start feeding trades to kline builder
	go service.feedTradesToKlineBuilder()

	return service
}

func (s *service) Connect() error {
	ctx := context.Background()

	return s.connectionManager.Connect(ctx, nil)
}

func (s *service) Disconnect() error {
	return s.connectionManager.Disconnect()
}

func (s *service) IsConnected() bool {
	return s.connectionManager.GetState() == connection.StateConnected
}

func (s *service) GetMetrics() map[string]interface{} {
	return s.connectionManager.GetConnectionStats()
}

func (s *service) ErrorChannel() <-chan error {
	return s.errorChan
}

func (s *service) StartWebSocket() error {
	return s.Connect()
}

func (s *service) StopWebSocket() error {
	return s.Disconnect()
}

func (s *service) IsWebSocketConnected() bool {
	return s.IsConnected()
}

func (s *service) SubscribeOrderBook(asset string) error {
	return s.SubscribeOrderbook(asset)
}

func (s *service) SubscribeTrades(asset string) error {
	return s.SubscribeTradesForSymbol(asset)
}

func (s *service) SubscribeAccount() error {
	return s.SubscribeAccountUpdates()
}

// Thread-safe write method
func (s *service) safeWriteJSON(message interface{}) error {
	s.writeMutex.Lock()
	defer s.writeMutex.Unlock()
	return s.connectionManager.SendJSON(message)
}

// Feed trades to kline builder
func (s *service) feedTradesToKlineBuilder() {
	for trade := range s.tradeChan {
		if s.klineBuilder != nil {
			s.klineBuilder.ProcessTrade(trade)
		}
	}
}
