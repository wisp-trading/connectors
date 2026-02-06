package bybit

import (
	"context"
	"fmt"
	"sync"

	"github.com/wisp-trading/connectors/pkg/connectors/bybit/data"
	"github.com/wisp-trading/connectors/pkg/connectors/bybit/data/real_time"
	"github.com/wisp-trading/connectors/pkg/connectors/bybit/trading"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
	"github.com/wisp-trading/sdk/pkg/types/logging"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/temporal"
)

type bybit struct {
	marketData    data.MarketDataService
	trading       trading.TradingService
	realTime      real_time.RealTimeService
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
	tradeCh   chan connector.Trade
	balanceCh chan connector.AssetBalance
	errorCh   chan error

	// Subscription tracking
	subscriptions map[string]int
	subMu         sync.RWMutex
}

var _ perp.Connector = (*bybit)(nil)

func NewBybit(
	tradingService trading.TradingService,
	marketDataService data.MarketDataService,
	realTimeService real_time.RealTimeService,
	appLogger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
	timeProvider temporal.TimeProvider,
) perp.WebSocketConnector {
	return &bybit{
		trading:       tradingService,
		marketData:    marketDataService,
		realTime:      realTimeService,
		config:        nil,
		appLogger:     appLogger,
		tradingLogger: tradingLogger,
		timeProvider:  timeProvider,
		ctx:           context.Background(),
		initialized:   false,
		tradeCh:       make(chan connector.Trade, 100),
		positionCh:    make(chan connector.Position, 100),
		balanceCh:     make(chan connector.AccountBalance, 100),
		errorCh:       make(chan error, 100),
		subscriptions: make(map[string]int),

		orderBookChannels: make(map[string]chan connector.OrderBook),
		klineChannels:     make(map[string]chan connector.Kline),
	}
}

func (b *bybit) Initialize(config connector.Config) error {
	if b.initialized {
		return fmt.Errorf("connector already initialized")
	}

	bybitConfig, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for Bybit connector: expected *bybit.Config, got %T", config)
	}

	tradingConfig := &trading.Config{
		APIKey:          bybitConfig.APIKey,
		APISecret:       bybitConfig.APISecret,
		BaseURL:         bybitConfig.BaseURL,
		IsTestnet:       bybitConfig.IsTestnet,
		DefaultSlippage: bybitConfig.DefaultSlippage,
	}

	dataConfig := &data.Config{
		APIKey:          bybitConfig.APIKey,
		APISecret:       bybitConfig.APISecret,
		BaseURL:         bybitConfig.BaseURL,
		IsTestnet:       bybitConfig.IsTestnet,
		DefaultSlippage: bybitConfig.DefaultSlippage,
	}

	realTimeConfig := &real_time.Config{
		APIKey:    bybitConfig.APIKey,
		APISecret: bybitConfig.APISecret,
		BaseURL:   bybitConfig.BaseURL,
	}

	if err := b.trading.Initialize(tradingConfig); err != nil {
		return fmt.Errorf("failed to initialize trading service: %w", err)
	}

	if err := b.marketData.Initialize(dataConfig); err != nil {
		return fmt.Errorf("failed to initialize market data service: %w", err)
	}

	if err := b.realTime.Initialize(realTimeConfig); err != nil {
		return fmt.Errorf("failed to initialize real-time service: %w", err)
	}

	b.config = bybitConfig
	b.initialized = true
	b.appLogger.Info("Bybit connector initialized", "testnet", bybitConfig.IsTestnet)
	return nil
}

// IsInitialized implements Initializable interface
func (b *bybit) IsInitialized() bool {
	return b.initialized
}

func (b *bybit) Name() string {
	return "Bybit"
}

func (b *bybit) SupportedInstruments() []connector.Instrument {
	return []connector.Instrument{
		connector.TypePerpetual,
		connector.TypeSpot,
	}
}

func (b *bybit) SupportsMarketData() bool {
	return true
}

func (b *bybit) GetPerpSymbol(asset portfolio.Asset) string {
	return asset.Symbol() + "USDT"
}
