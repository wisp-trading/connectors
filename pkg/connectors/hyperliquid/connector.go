package hyperliquid

import (
	"fmt"
	"sync"

	"github.com/wisp-trading/connectors/pkg/connectors/hyperliquid/adaptors"
	"github.com/wisp-trading/connectors/pkg/connectors/hyperliquid/rest"
	"github.com/wisp-trading/connectors/pkg/connectors/hyperliquid/websocket"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
	"github.com/wisp-trading/sdk/pkg/types/logging"
	"github.com/wisp-trading/sdk/pkg/types/temporal"
)

// hyperliquid implements Connector and Initializable interfaces
type hyperliquid struct {
	exchangeClient adaptors.ExchangeClient
	infoClient     adaptors.InfoClient
	marketData     rest.MarketDataService
	trading        rest.TradingService
	realTime       websocket.RealTimeService
	config         *Config
	appLogger      logging.ApplicationLogger
	tradingLogger  logging.TradingLogger
	timeProvider   temporal.TimeProvider
	initialized    bool

	// WebSocket channels
	tradeCh       chan connector.Trade
	positionCh    chan connector.Position
	balanceCh     chan connector.AccountBalance
	fundingRateCh chan perp.FundingRate
	errorCh       chan error

	// Separate channels per orderbook subscription (key: "BTC", "ETH", etc.)
	orderBookChannels map[string]chan connector.OrderBook
	orderBookMu       sync.RWMutex

	// Separate channels per kline subscription (key: "BTC:1m", "ETH:5m", etc.)
	klineChannels map[string]chan connector.Kline
	klineMu       sync.RWMutex

	// Subscription tracking
	subscriptions map[string]int
	subMu         sync.RWMutex
}

// Ensure hyperliquid implements all interfaces at compile time
var _ perp.WebSocketConnector = (*hyperliquid)(nil)

// NewHyperliquid creates a new Hyperliquid connector
func NewHyperliquid(
	exchangeClient adaptors.ExchangeClient,
	infoClient adaptors.InfoClient,
	tradingService rest.TradingService,
	marketDataService rest.MarketDataService,
	realTimeService websocket.RealTimeService,
	appLogger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
	timeProvider temporal.TimeProvider,
) perp.Connector {
	return &hyperliquid{
		exchangeClient:    exchangeClient,
		infoClient:        infoClient,
		trading:           tradingService,
		marketData:        marketDataService,
		realTime:          realTimeService,
		config:            nil, // Will be set during initialization
		appLogger:         appLogger,
		tradingLogger:     tradingLogger,
		timeProvider:      timeProvider,
		initialized:       false,
		tradeCh:           make(chan connector.Trade, 100),
		positionCh:        make(chan connector.Position, 100),
		balanceCh:         make(chan connector.AccountBalance, 100),
		fundingRateCh:     make(chan perp.FundingRate, 100),
		orderBookChannels: make(map[string]chan connector.OrderBook),
		klineChannels:     make(map[string]chan connector.Kline),
		errorCh:           make(chan error, 100),
		subscriptions:     make(map[string]int),
	}
}

// Initialize implements Initializable interface
func (h *hyperliquid) Initialize(config connector.Config) error {
	if h.initialized {
		return fmt.Errorf("connector already initialized")
	}

	hlConfig, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for Hyperliquid connector: expected *hyperliquid.Config, got %T", config)
	}

	hlConfig.Validate()

	// Configure the existing clients with runtime config
	if err := h.exchangeClient.Configure(hlConfig.BaseURL, hlConfig.PrivateKey, hlConfig.VaultAddress, hlConfig.AccountAddress); err != nil {
		return fmt.Errorf("failed to configure exchange client: %w", err)
	}

	if err := h.infoClient.Configure(hlConfig.BaseURL); err != nil {
		return fmt.Errorf("failed to configure info client: %w", err)
	}

	// Initialize trading service to load asset metadata for price validation
	if tradingService, ok := h.trading.(interface{ Initialize() error }); ok {
		if err := tradingService.Initialize(); err != nil {
			h.appLogger.Warn("Failed to initialize trading service price validation: %v", err)
		}
	}

	h.config = hlConfig
	h.initialized = true
	h.appLogger.Info("Hyperliquid connector initialized", "base_url", hlConfig.BaseURL)
	return nil
}

// IsInitialized implements Initializable interface
func (h *hyperliquid) IsInitialized() bool {
	return h.initialized
}
