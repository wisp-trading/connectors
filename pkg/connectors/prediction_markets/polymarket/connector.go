package polymarket

import (
	"context"
	"fmt"
	"sync"

	"github.com/wisp-trading/connectors/pkg/connectors/paradex/websocket"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/config"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
	"github.com/wisp-trading/sdk/pkg/types/logging"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/temporal"
)

type polymarket struct {
	client        adaptor.PolymarketClient
	config        *config.Config
	appLogger     logging.ApplicationLogger
	tradingLogger logging.TradingLogger
	timeProvider  temporal.TimeProvider
	ctx           context.Context
	initialized   bool

	// WebSocket service
	wsService websockets.WebSocketService

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

func (p *polymarket) GetConnectorInfo() *connector.Info {
	//TODO implement me
	panic("implement me")
}

func (p *polymarket) NewConfig() connector.Config {
	//TODO implement me
	panic("implement me")
}

func (p *polymarket) SupportsTradingOperations() bool {
	//TODO implement me
	panic("implement me")
}

func (p *polymarket) SupportsRealTimeData() bool {
	//TODO implement me
	panic("implement me")
}

// Ensure polymarket implements all interfaces at compile time
var _ prediction.Connector = (*polymarket)(nil)

func NewPolymarket(
	appLogger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
	timeProvider temporal.TimeProvider,
) prediction.Connector {
	client := adaptor.NewPolymarketClient()

	return &polymarket{
		client:            client,
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
func (p *polymarket) Initialize(conf connector.Config) error {
	if p.initialized {
		return fmt.Errorf("connector already initialized")
	}

	polymarketConfig, ok := conf.(*config.Config)
	if !ok {
		return fmt.Errorf("invalid conf type for Polymarket connector: expected *polymarket.Config, got %T", conf)
	}

	err := p.client.Configure(polymarketConfig)
	if err != nil {
		return err
	}

	p.config = polymarketConfig
	p.initialized = true
	p.appLogger.Info("Polymarket connector initialized", "base_url", polymarketConfig.BaseURL)
	return nil
}

// IsInitialized implements Initializable interface
func (p *polymarket) IsInitialized() bool {
	return p.initialized
}

// GetPredictionPair
func (p *polymarket) GetPredictionPair(marketID, outcomeID string) prediction.PredictionPair {
	return prediction.NewPredictionPair(marketID, outcomeID, portfolio.NewAsset("USDC"))
}
