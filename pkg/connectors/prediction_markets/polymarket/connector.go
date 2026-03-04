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
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector"
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

	subscribedMarkets map[prediction.MarketID]prediction.Market

	orderBookChannel chan prediction.OrderBook

	priceChangeChannels map[string]chan prediction.PriceChange
	priceChangeMu       sync.RWMutex

	tradeChannel chan connector.Trade
	orderChannel chan connector.Order

	orderManager  order_manager.OrderManager
	clobWebsocket websocket.Websocket
	gammaClient   gamma.GammaClient
}

func (p *polymarket) GetConnectorInfo() *connector.Info {
	return &connector.Info{
		Name: connector.ExchangeName("polymarket"),
	}
}

func (p *polymarket) NewConfig() connector.Config {
	return config.NewConfig()
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
		orderBookChannel:    make(chan prediction.OrderBook, 100),
		priceChangeChannels: make(map[string]chan prediction.PriceChange),
		tradeChannel:        make(chan connector.Trade, 100),
		orderChannel:        make(chan connector.Order, 100),
		subscribedMarkets:   make(map[prediction.MarketID]prediction.Market),
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
