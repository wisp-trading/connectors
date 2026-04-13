package hyperliquid

import (
	"fmt"
	"sync"

	"github.com/wisp-trading/connectors/pkg/connectors/adaptors/hyperliquid"
	"github.com/wisp-trading/connectors/pkg/connectors/spot/hyperliquid/rest"
	"github.com/wisp-trading/connectors/pkg/connectors/spot/hyperliquid/websocket"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	spotconnector "github.com/wisp-trading/sdk/pkg/types/connector/spot"
	"github.com/wisp-trading/sdk/pkg/types/logging"
	"github.com/wisp-trading/sdk/pkg/types/temporal"
)

// hyperliquidSpot implements the spot.WebSocketConnector interface
// for Hyperliquid's spot market. It shares the same API clients as
// the perp connector (same endpoint, same keys) but routes orders
// through spot asset indices (10000+ offset) and reads spot balances
// via the SpotUserState endpoint.
type hyperliquidSpot struct {
	exchangeClient hyperliquid.ExchangeClient
	infoClient     hyperliquid.InfoClient
	marketData     rest.SpotMarketDataService
	trading        rest.SpotTradingService
	realTime       websocket.SpotRealTimeService
	config         *Config
	appLogger      logging.ApplicationLogger
	tradingLogger  logging.TradingLogger
	timeProvider   temporal.TimeProvider
	initialized    bool

	// WebSocket channels
	tradeCh   chan connector.Trade
	balanceCh chan connector.AssetBalance
	errorCh   chan error

	// Separate channels per orderbook subscription (key: "HYPE", "PURR", etc.)
	orderBookChannels map[string]chan connector.OrderBook
	orderBookMu       sync.RWMutex

	// Separate channels per kline subscription (key: "HYPE:1m", "PURR:5m", etc.)
	klineChannels map[string]chan connector.Kline
	klineMu       sync.RWMutex

	// Subscription tracking
	subscriptions map[string]int
	subMu         sync.RWMutex
}

// Ensure hyperliquidSpot implements all interfaces at compile time
var _ spotconnector.WebSocketConnector = (*hyperliquidSpot)(nil)

// NewHyperliquidSpot creates a new Hyperliquid spot connector.
// It reuses the same adaptors (ExchangeClient, InfoClient) as the perp connector
// since both hit the same Hyperliquid API endpoints.
func NewHyperliquidSpot(
	exchangeClient hyperliquid.ExchangeClient,
	infoClient hyperliquid.InfoClient,
	tradingService rest.SpotTradingService,
	marketDataService rest.SpotMarketDataService,
	realTimeService websocket.SpotRealTimeService,
	appLogger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
	timeProvider temporal.TimeProvider,
) spotconnector.Connector {
	return &hyperliquidSpot{
		exchangeClient:    exchangeClient,
		infoClient:        infoClient,
		trading:           tradingService,
		marketData:        marketDataService,
		realTime:          realTimeService,
		config:            nil,
		appLogger:         appLogger,
		tradingLogger:     tradingLogger,
		timeProvider:      timeProvider,
		initialized:       false,
		tradeCh:           make(chan connector.Trade, 100),
		balanceCh:         make(chan connector.AssetBalance, 100),
		orderBookChannels: make(map[string]chan connector.OrderBook),
		klineChannels:     make(map[string]chan connector.Kline),
		errorCh:           make(chan error, 100),
		subscriptions:     make(map[string]int),
	}
}

// Initialize implements connector.Connector
func (h *hyperliquidSpot) Initialize(config connector.Config) error {
	if h.initialized {
		return fmt.Errorf("connector already initialized")
	}

	hlConfig, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for Hyperliquid spot connector: expected *spot.Config, got %T", config)
	}

	if err := hlConfig.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Configure the shared API clients if not already configured by the perp connector.
	if !h.exchangeClient.IsConfigured() {
		if err := h.exchangeClient.Configure(hlConfig.BaseURL, hlConfig.PrivateKey, hlConfig.VaultAddress, hlConfig.AccountAddress); err != nil {
			return fmt.Errorf("failed to configure exchange client: %w", err)
		}
	}

	if !h.infoClient.IsConfigured() {
		if err := h.infoClient.Configure(hlConfig.BaseURL); err != nil {
			return fmt.Errorf("failed to configure info client: %w", err)
		}
	}

	h.config = hlConfig
	h.initialized = true
	h.appLogger.Info("Hyperliquid spot connector initialized", "base_url", hlConfig.BaseURL)
	return nil
}

// IsInitialized implements connector.Connector
func (h *hyperliquidSpot) IsInitialized() bool {
	return h.initialized
}
