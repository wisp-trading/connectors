package spot

import (
	"context"
	"fmt"
	"sync"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/backtesting-org/kronos-sdk/pkg/types/temporal"
	"github.com/backtesting-org/live-trading/pkg/connectors/gate/adaptor"
	"github.com/backtesting-org/live-trading/pkg/connectors/gate/websocket"
)

// gateSpot implements Connector and WebSocketConnector interfaces
type gateSpot struct {
	spotClient    adaptor.SpotClient
	wsService     websocket.RealTimeService
	config        *Config
	appLogger     logging.ApplicationLogger
	tradingLogger logging.TradingLogger
	timeProvider  temporal.TimeProvider
	ctx           context.Context
	initialized   bool

	// WebSocket channels
	tradeCh    chan connector.Trade
	positionCh chan connector.Position
	balanceCh  chan connector.AccountBalance
	errorCh    chan error

	// Separate channels per orderbook subscription (key: "BTC_USDT", "ETH_USDT", etc.)
	orderBookChannels map[string]chan connector.OrderBook
	orderBookMu       sync.RWMutex

	// Separate channels per kline subscription (key: "BTC_USDT:1m", "ETH_USDT:5m", etc.)
	klineChannels map[string]chan connector.Kline
	klineMu       sync.RWMutex

	// Subscription tracking
	subscriptions map[string]int
	subMu         sync.RWMutex
}

// Ensure gateSpot implements all interfaces at compile time
var _ connector.Connector = (*gateSpot)(nil)
var _ connector.WebSocketConnector = (*gateSpot)(nil)

// NewGateSpot creates a new Gate.io Spot connector
func NewGateSpot(
	spotClient adaptor.SpotClient,
	wsService websocket.RealTimeService,
	appLogger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
	timeProvider temporal.TimeProvider,
) connector.Connector {
	return &gateSpot{
		spotClient:        spotClient,
		wsService:         wsService,
		config:            nil, // Will be set during initialization
		appLogger:         appLogger,
		tradingLogger:     tradingLogger,
		timeProvider:      timeProvider,
		ctx:               context.Background(),
		initialized:       false,
		tradeCh:           make(chan connector.Trade, 100),
		positionCh:        make(chan connector.Position, 100),
		balanceCh:         make(chan connector.AccountBalance, 100),
		errorCh:           make(chan error, 100),
		orderBookChannels: make(map[string]chan connector.OrderBook),
		klineChannels:     make(map[string]chan connector.Kline),
		subscriptions:     make(map[string]int),
	}
}

// NewConfig returns a new config instance
func (g *gateSpot) NewConfig() connector.Config {
	return &Config{}
}

// Initialize implements Connector interface
func (g *gateSpot) Initialize(config connector.Config) error {
	if g.initialized {
		return fmt.Errorf("connector already initialized")
	}

	gateConfig, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for Gate Spot connector: expected *spot.Config, got %T", config)
	}

	// Validate config
	if err := gateConfig.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Configure the spot client with runtime config
	if err := g.spotClient.Configure(gateConfig.APIKey, gateConfig.APISecret, gateConfig.BaseURL); err != nil {
		return fmt.Errorf("failed to configure spot client: %w", err)
	}

	g.config = gateConfig
	g.initialized = true

	g.appLogger.Info("Gate Spot connector initialized successfully")
	return nil
}

// Close implements Connector interface
func (g *gateSpot) Close() error {
	if !g.initialized {
		return fmt.Errorf("connector not initialized")
	}

	// Close all channels
	close(g.tradeCh)
	close(g.positionCh)
	close(g.balanceCh)
	close(g.errorCh)

	g.orderBookMu.Lock()
	for _, ch := range g.orderBookChannels {
		close(ch)
	}
	g.orderBookMu.Unlock()

	g.klineMu.Lock()
	for _, ch := range g.klineChannels {
		close(ch)
	}
	g.klineMu.Unlock()

	g.initialized = false
	g.appLogger.Info("Gate Spot connector closed")
	return nil
}

// FetchContracts returns available contracts (spot doesn't have contracts)
func (g *gateSpot) FetchContracts() ([]connector.ContractInfo, error) {
	// Spot trading doesn't have contracts
	return []connector.ContractInfo{}, nil
}

// FetchRiskFundBalance returns risk fund balance (not applicable for spot)
func (g *gateSpot) FetchRiskFundBalance(_ string) (*connector.RiskFundBalance, error) {
	return nil, fmt.Errorf("risk fund balance not supported for spot trading")
}

// FetchCurrentFundingRates returns current funding rates (not applicable for spot)
func (g *gateSpot) FetchCurrentFundingRates() (map[portfolio.Asset]connector.FundingRate, error) {
	return nil, fmt.Errorf("funding rates not supported for spot trading")
}

// FetchFundingRate returns funding rate for a specific asset (not applicable for spot)
func (g *gateSpot) FetchFundingRate(asset portfolio.Asset) (*connector.FundingRate, error) {
	return nil, fmt.Errorf("funding rates not supported for spot trading")
}

// SupportsFundingRates returns whether funding rates are supported
func (g *gateSpot) SupportsFundingRates() bool {
	return false
}

// FetchHistoricalFundingRates returns historical funding rates (not applicable for spot)
func (g *gateSpot) FetchHistoricalFundingRates(asset portfolio.Asset, startTime int64, endTime int64) ([]connector.HistoricalFundingRate, error) {
	return nil, fmt.Errorf("funding rates not supported for spot trading")
}

// GetTradingHistory retrieves trading history
func (g *gateSpot) GetTradingHistory(_ string, _ int) ([]connector.Trade, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}
	// TODO: Implement trading history when needed
	return []connector.Trade{}, nil
}

// IsInitialized returns whether the connector is initialized
func (g *gateSpot) IsInitialized() bool {
	return g.initialized
}

// StartWebSocket starts the WebSocket connection
func (g *gateSpot) StartWebSocket() error {
	return g.ConnectWebSocket()
}

// StopWebSocket stops the WebSocket connection
func (g *gateSpot) StopWebSocket() error {
	return g.DisconnectWebSocket()
}

// IsWebSocketConnected returns whether WebSocket is connected
func (g *gateSpot) IsWebSocketConnected() bool {
	if g.wsService == nil {
		return false
	}
	return g.wsService.IsConnected()
}

// SubscribeAccountBalance subscribes to account balance updates
func (g *gateSpot) SubscribeAccountBalance() error {
	if !g.initialized {
		return fmt.Errorf("connector not initialized")
	}

	_ = g.AccountBalanceUpdates()
	return nil
}

// UnsubscribeAccountBalance unsubscribes from account balance updates
func (g *gateSpot) UnsubscribeAccountBalance() error {
	// TODO: Implement proper unsubscription tracking
	g.appLogger.Info("Account balance unsubscription requested")
	return nil
}

// SubscribeTrades subscribes to trade updates for an asset
func (g *gateSpot) SubscribeTrades(asset portfolio.Asset, _ connector.Instrument) error {
	if !g.initialized {
		return fmt.Errorf("connector not initialized")
	}

	// TradeUpdates() returns just a channel, no error
	_ = g.TradeUpdates()
	return nil
}

// UnsubscribeTrades unsubscribes from trade updates
func (g *gateSpot) UnsubscribeTrades(_ portfolio.Asset, _ connector.Instrument) error {
	// TODO: Implement proper unsubscription tracking
	g.appLogger.Info("Trades unsubscription requested")
	return nil
}

// SubscribePositions subscribes to position updates (not applicable for spot)
func (g *gateSpot) SubscribePositions(_ portfolio.Asset, _ connector.Instrument) error {
	return fmt.Errorf("positions not supported for spot trading")
}

// UnsubscribePositions unsubscribes from position updates (not applicable for spot)
func (g *gateSpot) UnsubscribePositions(_ portfolio.Asset, _ connector.Instrument) error {
	return fmt.Errorf("positions not supported for spot trading")
}
