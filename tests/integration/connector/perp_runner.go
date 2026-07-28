package connector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/fx"

	"github.com/wisp-trading/connectors/pkg/connectors"
	realtimeTypes "github.com/wisp-trading/sdk/pkg/markets/base/types/ingestors/realtime"
	perpTypes "github.com/wisp-trading/sdk/pkg/markets/perp/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
	wispTypes "github.com/wisp-trading/sdk/pkg/types/wisp"
	"github.com/wisp-trading/sdk/wisp"
)

// PerpTestRunner manages the lifecycle of perpetual connector tests with full SDK wiring.
type PerpTestRunner struct {
	*BaseRunnerImpl
	conn                    perp.Connector
	wsConn                  perp.WebSocketConnector
	exchangeName            connector.ExchangeName
	wisp                    wispTypes.Wisp
	store                   perpTypes.MarketStore
	watchlist               perpTypes.PerpWatchlist
	batchIngestorFactory    perpTypes.PerpBatchIngestorFactory
	realtimeIngestorFactory perpTypes.PerpRealtimeIngestorFactory

	rtMu        sync.Mutex
	rtIngestors []realtimeTypes.RealtimeIngestor
}

// NewPerpTestRunner creates a new test runner for perp connectors.
func NewPerpTestRunner(connectorName connector.ExchangeName, config connector.Config) (*PerpTestRunner, error) {
	var reg registry.ConnectorRegistry
	var wispInstance wispTypes.Wisp
	var store perpTypes.MarketStore
	var watchlist perpTypes.PerpWatchlist
	var batchFactory perpTypes.PerpBatchIngestorFactory
	var realtimeFactory perpTypes.PerpRealtimeIngestorFactory

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

	conn, exists := reg.Perp(connectorName)
	if !exists {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("perp connector %s not found in registry", connectorName)
	}

	if err := conn.Initialize(config); err != nil {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("failed to initialize connector: %w", err)
	}

	if err := reg.MarkReady(connectorName); err != nil {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("failed to mark connector ready: %w", err)
	}

	wsConn, _ := conn.(perp.WebSocketConnector)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	return &PerpTestRunner{
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

func (tr *PerpTestRunner) Cleanup() {
	_ = tr.StopRealtime()
	tr.BaseRunnerImpl.Cleanup()
}

func (tr *PerpTestRunner) GetPerpConnector() perp.Connector { return tr.conn }

func (tr *PerpTestRunner) GetBaseConnector() connector.Connector { return tr.conn }

func (tr *PerpTestRunner) GetWebSocketConnector() perp.WebSocketConnector { return tr.wsConn }

func (tr *PerpTestRunner) HasWebSocketSupport() bool { return tr.wsConn != nil }

func (tr *PerpTestRunner) GetWebSocketCapable() connector.WebSocketCapable {
	if tr.wsConn == nil {
		return nil
	}
	return tr.wsConn
}

func (tr *PerpTestRunner) GetWisp() wispTypes.Wisp { return tr.wisp }

func (tr *PerpTestRunner) GetPerpStore() perpTypes.MarketStore { return tr.store }

func (tr *PerpTestRunner) ExchangeName() connector.ExchangeName { return tr.exchangeName }

func (tr *PerpTestRunner) WatchPair(pair portfolio.Pair) {
	tr.watchlist.RequirePair(tr.exchangeName, pair)
}

func (tr *PerpTestRunner) CollectNow() {
	for _, ingestor := range tr.batchIngestorFactory.CreateIngestors() {
		ingestor.CollectNow()
	}
}

func (tr *PerpTestRunner) StartRealtime(ctx context.Context) error {
	tr.rtMu.Lock()
	defer tr.rtMu.Unlock()
	if len(tr.rtIngestors) > 0 {
		return nil
	}
	ingestors := tr.realtimeIngestorFactory.CreateIngestors()
	if len(ingestors) == 0 {
		return fmt.Errorf("no perp realtime ingestors (need MarkReady + WebSocket connector)")
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

func (tr *PerpTestRunner) StopRealtime() error {
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

func (tr *PerpTestRunner) SDKPrice(pair portfolio.Pair) (numerical.Decimal, bool) {
	return tr.wisp.Perp().Price(tr.exchangeName, pair)
}

func (tr *PerpTestRunner) SDKOrderBook(pair portfolio.Pair) (*connector.OrderBook, bool) {
	return tr.wisp.Perp().OrderBook(tr.exchangeName, pair)
}

func (tr *PerpTestRunner) SDKKlines(pair portfolio.Pair, interval string, limit int) []connector.Kline {
	return tr.wisp.Perp().Klines(tr.exchangeName, pair, interval, limit)
}

func (tr *PerpTestRunner) StorePrice(pair portfolio.Pair) *connector.Price {
	return tr.store.GetPairPrice(pair, tr.exchangeName)
}

func (tr *PerpTestRunner) StoreOrderBook(pair portfolio.Pair) *connector.OrderBook {
	return tr.store.GetOrderBook(pair, tr.exchangeName)
}

func (tr *PerpTestRunner) StoreKlines(pair portfolio.Pair, interval string, limit int) []connector.Kline {
	return tr.store.GetKlines(pair, tr.exchangeName, interval, limit)
}

func (tr *PerpTestRunner) GetPerpSymbol(asset portfolio.Pair) string {
	return tr.conn.GetPerpSymbol(asset)
}

var _ PairMarketTestRunner = (*PerpTestRunner)(nil)
