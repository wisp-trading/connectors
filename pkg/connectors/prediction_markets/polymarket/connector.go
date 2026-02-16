package polymarket

import (
	"context"
	"fmt"
	"sync"

	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/gamma"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/order_manager"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/websocket"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/config"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
	"github.com/wisp-trading/sdk/pkg/types/logging"
	"github.com/wisp-trading/sdk/pkg/types/temporal"
)

type polymarket struct {
	client adaptor.Client

	config        *config.Config
	appLogger     logging.ApplicationLogger
	tradingLogger logging.TradingLogger
	timeProvider  temporal.TimeProvider
	ctx           context.Context
	initialized   bool

	// WebSocket state management
	wsContext context.Context
	wsCancel  context.CancelFunc
	wsMutex   sync.RWMutex

	tradeCh chan connector.Trade

	// Separate channels per outcome subscription (key: "btc-updown-4h:YES-USDC")
	orderBookChannels map[string]chan connector.OrderBook
	orderBookMu       sync.RWMutex

	priceChangeChannels map[string]chan prediction.PriceChange
	priceChangeMu       sync.RWMutex

	tradesChannel chan connector.Trade

	// Separate channels per outcome subscription for klines (key: "btc-updown-4h:YES-USDC")
	klineChannels map[string]chan connector.Kline
	klineMu       sync.RWMutex
	orderManager  order_manager.OrderManager
	clobWebsocket websocket.Websocket
	gammaClient   gamma.GammaClient
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
	client adaptor.Client,
	appLogger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
	timeProvider temporal.TimeProvider,
) prediction.WebSocketConnector {

	return &polymarket{
		client:              client,
		config:              nil, // Will be set during initialization
		appLogger:           appLogger,
		tradingLogger:       tradingLogger,
		timeProvider:        timeProvider,
		ctx:                 context.Background(),
		initialized:         false,
		orderBookChannels:   make(map[string]chan connector.OrderBook),
		priceChangeChannels: make(map[string]chan prediction.PriceChange),
		tradesChannel:       make(chan connector.Trade, 100),
		klineChannels:       make(map[string]chan connector.Kline),
		tradeCh:             make(chan connector.Trade, 100),
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

	var err error
	p.orderManager, p.clobWebsocket, p.gammaClient, err = p.client.Configure(polymarketConfig)
	if err != nil {
		return err
	}

	p.config = polymarketConfig

	p.initialized = true
	p.appLogger.Info("Polymarket connector initialized")
	return nil
}

// IsInitialized implements Initializable interface
func (p *polymarket) IsInitialized() bool {
	return p.initialized
}
