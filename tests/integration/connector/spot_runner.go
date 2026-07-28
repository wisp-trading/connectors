package connector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/fx"

	"github.com/wisp-trading/connectors/pkg/connectors"
	realtimeTypes "github.com/wisp-trading/sdk/pkg/markets/base/types/ingestors/realtime"
	spotTypes "github.com/wisp-trading/sdk/pkg/markets/spot/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/spot"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
	wispTypes "github.com/wisp-trading/sdk/pkg/types/wisp"
	"github.com/wisp-trading/sdk/wisp"
)

// SpotTestRunner manages the lifecycle of spot connector tests with full SDK wiring.
type SpotTestRunner struct {
	*BaseRunnerImpl
	conn                    spot.Connector
	wsConn                  spot.WebSocketConnector
	exchangeName            connector.ExchangeName
	wisp                    wispTypes.Wisp
	store                   spotTypes.MarketStore
	watchlist               spotTypes.SpotWatchlist
	batchIngestorFactory    spotTypes.SpotBatchIngestorFactory
	realtimeIngestorFactory spotTypes.SpotRealtimeIngestorFactory

	rtMu        sync.Mutex
	rtIngestors []realtimeTypes.RealtimeIngestor
}

// NewSpotTestRunner creates a new test runner for spot connectors.
func NewSpotTestRunner(connectorName connector.ExchangeName, config connector.Config) (*SpotTestRunner, error) {
	var reg registry.ConnectorRegistry
	var wispInstance wispTypes.Wisp
	var store spotTypes.MarketStore
	var watchlist spotTypes.SpotWatchlist
	var batchFactory spotTypes.SpotBatchIngestorFactory
	var realtimeFactory spotTypes.SpotRealtimeIngestorFactory

	app := fx.New(
		wisp.Module,
		connectors.Module,
		fx.Populate(&reg, &wispInstance, &store, &watchlist, &batchFactory, &realtimeFactory),
		fx.NopLogger,
	)

	startCtx, startCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer startCancel()

	if err := app.Start(startCtx); err != nil {
		return nil, fmt.Errorf("failed to start fx app: %w", err)
	}

	conn, exists := reg.Spot(connectorName)
	if !exists {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("spot connector %s not found in registry", connectorName)
	}

	if err := conn.Initialize(config); err != nil {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("failed to initialize connector: %w", err)
	}

	// ReadyOnly filters require MarkReady — without this CreateIngestors returns empty.
	if err := reg.MarkReady(connectorName); err != nil {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("failed to mark connector ready: %w", err)
	}

	wsConn, _ := conn.(spot.WebSocketConnector)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	return &SpotTestRunner{
		BaseRunnerImpl: &BaseRunnerImpl{
			app:    app,
			ctx:    ctx,
			cancel: cancel,
			reg:    reg,
		},
		conn:                    conn,
		wsConn:                  wsConn,
		exchangeName:            connectorName,
		wisp:                    wispInstance,
		store:                   store,
		watchlist:               watchlist,
		batchIngestorFactory:    batchFactory,
		realtimeIngestorFactory: realtimeFactory,
	}, nil
}

// Cleanup stops realtime ingestors then tears down fx.
func (tr *SpotTestRunner) Cleanup() {
	_ = tr.StopRealtime()
	tr.BaseRunnerImpl.Cleanup()
}

func (tr *SpotTestRunner) GetSpotConnector() spot.Connector { return tr.conn }

func (tr *SpotTestRunner) GetBaseConnector() connector.Connector { return tr.conn }

func (tr *SpotTestRunner) GetWebSocketConnector() spot.WebSocketConnector { return tr.wsConn }

func (tr *SpotTestRunner) HasWebSocketSupport() bool { return tr.wsConn != nil }

func (tr *SpotTestRunner) GetWebSocketCapable() connector.WebSocketCapable {
	if tr.wsConn == nil {
		return nil
	}
	return tr.wsConn
}

func (tr *SpotTestRunner) GetWisp() wispTypes.Wisp { return tr.wisp }

func (tr *SpotTestRunner) GetSpotStore() spotTypes.MarketStore { return tr.store }

func (tr *SpotTestRunner) ExchangeName() connector.ExchangeName { return tr.exchangeName }

// WatchPair registers a pair so batch ingestors will collect it.
func (tr *SpotTestRunner) WatchPair(pair portfolio.Pair) {
	tr.watchlist.RequirePair(tr.exchangeName, pair)
}

// CollectNow runs connector → batch ingestor → store for watched pairs.
func (tr *SpotTestRunner) CollectNow() {
	for _, ingestor := range tr.batchIngestorFactory.CreateIngestors() {
		ingestor.CollectNow()
	}
}

// StartRealtime starts WS ingestors that write order book / klines into the store.
func (tr *SpotTestRunner) StartRealtime(ctx context.Context) error {
	tr.rtMu.Lock()
	defer tr.rtMu.Unlock()
	if len(tr.rtIngestors) > 0 {
		return nil
	}
	ingestors := tr.realtimeIngestorFactory.CreateIngestors()
	if len(ingestors) == 0 {
		return fmt.Errorf("no spot realtime ingestors (need MarkReady + WebSocket connector)")
	}
	for _, ing := range ingestors {
		if err := ing.Start(ctx); err != nil {
			for _, started := range tr.rtIngestors {
				_ = started.Stop()
			}
			tr.rtIngestors = nil
			return err
		}
		tr.rtIngestors = append(tr.rtIngestors, ing)
	}
	return nil
}

// StopRealtime stops WS ingestors started by StartRealtime.
func (tr *SpotTestRunner) StopRealtime() error {
	tr.rtMu.Lock()
	defer tr.rtMu.Unlock()
	var first error
	for _, ing := range tr.rtIngestors {
		if err := ing.Stop(); err != nil && first == nil {
			first = err
		}
	}
	tr.rtIngestors = nil
	return first
}

// SDKPrice reads price via the strategy-facing facade (store-backed).
func (tr *SpotTestRunner) SDKPrice(pair portfolio.Pair) (numerical.Decimal, bool) {
	return tr.wisp.Spot().Price(tr.exchangeName, pair)
}

// SDKOrderBook reads order book via the strategy-facing facade.
func (tr *SpotTestRunner) SDKOrderBook(pair portfolio.Pair) (*connector.OrderBook, bool) {
	return tr.wisp.Spot().OrderBook(tr.exchangeName, pair)
}

// SDKKlines reads klines via the strategy-facing facade.
func (tr *SpotTestRunner) SDKKlines(pair portfolio.Pair, interval string, limit int) []connector.Kline {
	return tr.wisp.Spot().Klines(tr.exchangeName, pair, interval, limit)
}

// StorePrice returns the raw store entry (bypasses facade) for dual assertion.
func (tr *SpotTestRunner) StorePrice(pair portfolio.Pair) *connector.Price {
	return tr.store.GetPairPrice(pair, tr.exchangeName)
}

// StoreOrderBook returns the raw store order book.
func (tr *SpotTestRunner) StoreOrderBook(pair portfolio.Pair) *connector.OrderBook {
	return tr.store.GetOrderBook(pair, tr.exchangeName)
}

// StoreKlines returns raw store klines.
func (tr *SpotTestRunner) StoreKlines(pair portfolio.Pair, interval string, limit int) []connector.Kline {
	return tr.store.GetKlines(pair, tr.exchangeName, interval, limit)
}

// Ensure SpotTestRunner satisfies PairMarketTestRunner.
var _ PairMarketTestRunner = (*SpotTestRunner)(nil)
