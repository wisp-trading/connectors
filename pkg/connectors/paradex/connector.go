package paradex

import (
	"context"
	"fmt"
	"sync"

	"github.com/wisp-trading/connectors/pkg/connectors/paradex/adaptor"
	"github.com/wisp-trading/connectors/pkg/connectors/paradex/requests"
	websockets2 "github.com/wisp-trading/connectors/pkg/connectors/paradex/websocket"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
	"github.com/wisp-trading/sdk/pkg/types/logging"
	"github.com/wisp-trading/sdk/pkg/types/temporal"
)

// paradex implements Connector, WebSocketConnector, and Initializable interfaces
type paradex struct {
	paradexService *requests.Service
	config         *Config
	appLogger      logging.ApplicationLogger
	tradingLogger  logging.TradingLogger
	timeProvider   temporal.TimeProvider
	ctx            context.Context
	initialized    bool

	// WebSocket service
	wsService websockets2.WebSocketService

	// WebSocket state management
	wsContext context.Context
	wsCancel  context.CancelFunc
	wsMutex   sync.RWMutex

	tradeCh chan connector.Trade

	// Separate channels per orderbook subscription (key: "BTC", "ETH", etc.)
	orderBookChannels map[string]chan connector.OrderBook
	orderBookMu       sync.RWMutex

	// Separate channels per kline subscription (key: "BTC:1m", "ETH:5m", etc.)
	klineChannels map[string]chan connector.Kline
	klineMu       sync.RWMutex
}

// Ensure paradex implements all interfaces at compile time
var _ connector.Connector = (*paradex)(nil)
var _ perp.Connector = (*paradex)(nil)

func NewParadex(
	appLogger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
	timeProvider temporal.TimeProvider,
) perp.Connector {
	return &paradex{
		paradexService:    nil, // Will be created during initialization
		wsService:         nil, // Will be created during initialization
		config:            nil, // Will be set during initialization
		appLogger:         appLogger,
		tradingLogger:     tradingLogger,
		timeProvider:      timeProvider,
		ctx:               context.Background(),
		initialized:       false,
		orderBookChannels: make(map[string]chan connector.OrderBook),
		klineChannels:     make(map[string]chan connector.Kline),
		tradeCh:           make(chan connector.Trade, 100),
	}
}

// Initialize implements Initializable interface
func (p *paradex) Initialize(config connector.Config) error {
	if p.initialized {
		return fmt.Errorf("connector already initialized")
	}

	paradexConfig, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for Paradex connector: expected *paradex.Config, got %T", config)
	}

	// Create adaptor client
	adaptorConfig := &adaptor.Config{
		BaseURL:       paradexConfig.BaseURL,
		StarknetRPC:   paradexConfig.StarknetRPC,
		EthPrivateKey: paradexConfig.EthPrivateKey,
		Network:       paradexConfig.Network,
	}

	client, err := adaptor.NewClient(adaptorConfig, p.appLogger)
	if err != nil {
		return fmt.Errorf("failed to create Paradex client: %w", err)
	}

	// Create services
	p.paradexService = requests.NewService(client, p.appLogger)
	p.wsService = websockets2.NewService(client, paradexConfig.WebSocketURL, p.appLogger, p.tradingLogger, p.timeProvider)

	p.config = paradexConfig
	p.initialized = true
	p.appLogger.Info("Paradex connector initialized", "base_url", paradexConfig.BaseURL)
	return nil
}

// IsInitialized implements Initializable interface
func (p *paradex) IsInitialized() bool {
	return p.initialized
}
