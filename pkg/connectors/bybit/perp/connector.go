package perp

import (
	"context"
	"fmt"
	"sync"

	"github.com/wisp-trading/connectors/pkg/connectors/bybit/adaptor"
	"github.com/wisp-trading/connectors/pkg/connectors/bybit/websocket"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
	"github.com/wisp-trading/sdk/pkg/types/logging"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/temporal"
)

// bybit implements perp.WebSocketConnector interface
type bybit struct {
	client        adaptor.PerpClient
	wsService     websocket.RealTimeService
	config        *Config
	appLogger     logging.ApplicationLogger
	tradingLogger logging.TradingLogger
	timeProvider  temporal.TimeProvider
	ctx           context.Context
	initialized   bool

	// Separate channels per orderbook subscription (key: "BTC", "ETH", etc.)
	orderBookChannels map[string]chan connector.OrderBook
	orderBookMu       sync.RWMutex

	// Separate channels per kline subscription (key: "BTC:1m", "ETH:5m", etc.)
	klineChannels map[string]chan connector.Kline
	klineMu       sync.RWMutex

	// WebSocket channels
	tradeCh    chan connector.Trade
	positionCh chan perp.Position
	balanceCh  chan connector.AssetBalance
	errorCh    chan error

	// Subscription tracking
	subscriptions map[string]int
	subMu         sync.RWMutex
}

// Ensure bybit implements all interfaces at compile time
var _ perp.WebSocketConnector = (*bybit)(nil)

// NewBybit creates a new Bybit connector (Gate.io pattern)
func NewBybit(
	client adaptor.PerpClient,
	wsService websocket.RealTimeService,
	appLogger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
	timeProvider temporal.TimeProvider,
) perp.Connector {
	return &bybit{
		client:            client,
		wsService:         wsService,
		config:            nil, // Will be set during initialization
		appLogger:         appLogger,
		tradingLogger:     tradingLogger,
		timeProvider:      timeProvider,
		ctx:               context.Background(),
		initialized:       false,
		tradeCh:           make(chan connector.Trade, 100),
		positionCh:        make(chan perp.Position, 100),
		balanceCh:         make(chan connector.AssetBalance, 100),
		errorCh:           make(chan error, 100),
		orderBookChannels: make(map[string]chan connector.OrderBook),
		klineChannels:     make(map[string]chan connector.Kline),
		subscriptions:     make(map[string]int),
	}
}

// Initialize implements Connector interface
func (b *bybit) Initialize(config connector.Config) error {
	if b.initialized {
		return fmt.Errorf("connector already initialized")
	}

	bybitConfig, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for Bybit connector: expected *bybit.Config, got %T", config)
	}

	// Configure the client with runtime config (single configuration point)
	if err := b.client.Configure(bybitConfig.APIKey, bybitConfig.APISecret, bybitConfig.BaseURL); err != nil {
		return fmt.Errorf("failed to configure client: %w", err)
	}

	b.config = bybitConfig
	b.initialized = true
	b.appLogger.Info("Bybit connector initialized", "testnet", bybitConfig.IsTestnet)
	return nil
}

// Close implements Connector interface
func (b *bybit) Close() error {
	if !b.initialized {
		return fmt.Errorf("connector not initialized")
	}

	// Close all channels
	close(b.tradeCh)
	close(b.positionCh)
	close(b.balanceCh)
	close(b.errorCh)

	b.orderBookMu.Lock()
	for _, ch := range b.orderBookChannels {
		close(ch)
	}
	b.orderBookMu.Unlock()

	b.klineMu.Lock()
	for _, ch := range b.klineChannels {
		close(ch)
	}
	b.klineMu.Unlock()

	b.initialized = false
	b.appLogger.Info("Bybit connector closed")
	return nil
}

// IsInitialized implements Initializable interface
func (b *bybit) IsInitialized() bool {
	return b.initialized
}

func (b *bybit) Name() string {
	return "Bybit"
}

func (b *bybit) GetPerpSymbol(pair portfolio.Pair) string {
	return pair.Base().Symbol() + pair.Quote().Symbol()
}
