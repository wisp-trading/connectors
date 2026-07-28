package connector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/fx"

	"github.com/wisp-trading/connectors/pkg/connectors"
	realtimeTypes "github.com/wisp-trading/sdk/pkg/markets/base/types/ingestors/realtime"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
	predtypes "github.com/wisp-trading/sdk/pkg/markets/prediction/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
	"github.com/wisp-trading/sdk/wisp"
)

// PredictionMarketTestRunner manages prediction connector tests with full SDK wiring.
type PredictionMarketTestRunner struct {
	*BaseRunnerImpl
	conn                    prediction.Connector
	exchangeName            connector.ExchangeName
	predict                 predtypes.Predict
	store                   predtypes.MarketStore
	watchlist               predtypes.PredictionWatchlist
	batchIngestorFactory    predtypes.PredictionBatchIngestorFactory
	realtimeIngestorFactory predtypes.PredictionRealtimeIngestorFactory

	rtMu        sync.Mutex
	rtIngestors []realtimeTypes.RealtimeIngestor
}

// NewPredictionMarketTestRunner creates a prediction test runner.
func NewPredictionMarketTestRunner(connectorName connector.ExchangeName, config connector.Config) (*PredictionMarketTestRunner, error) {
	var reg registry.ConnectorRegistry
	var predictService predtypes.Predict
	var store predtypes.MarketStore
	var watchlist predtypes.PredictionWatchlist
	var batchFactory predtypes.PredictionBatchIngestorFactory
	var realtimeFactory predtypes.PredictionRealtimeIngestorFactory

	app := fx.New(
		wisp.Module,
		connectors.Module,
		fx.Populate(&reg, &predictService, &store, &watchlist, &batchFactory, &realtimeFactory),
		fx.NopLogger,
	)

	startCtx, startCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer startCancel()

	if err := app.Start(startCtx); err != nil {
		return nil, fmt.Errorf("failed to start fx app: %w", err)
	}

	conn, exists := reg.Prediction(connectorName)
	if !exists {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("prediction market connector %s not found in registry", connectorName)
	}

	if err := conn.Initialize(config); err != nil {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("failed to initialize connector: %w", err)
	}

	if err := reg.MarkReady(connectorName); err != nil {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("failed to mark connector ready: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	return &PredictionMarketTestRunner{
		BaseRunnerImpl: &BaseRunnerImpl{
			app:    app,
			ctx:    ctx,
			cancel: cancel,
			reg:    reg,
		},
		conn:                    conn,
		exchangeName:            connectorName,
		predict:                 predictService,
		store:                   store,
		watchlist:               watchlist,
		batchIngestorFactory:    batchFactory,
		realtimeIngestorFactory: realtimeFactory,
	}, nil
}

func (tr *PredictionMarketTestRunner) Cleanup() {
	_ = tr.StopRealtime()
	tr.BaseRunnerImpl.Cleanup()
}

func (tr *PredictionMarketTestRunner) GetPredictionMarketConnector() prediction.Connector {
	return tr.conn
}

func (tr *PredictionMarketTestRunner) GetPredict() predtypes.Predict { return tr.predict }

func (tr *PredictionMarketTestRunner) GetPredictionStore() predtypes.MarketStore { return tr.store }

func (tr *PredictionMarketTestRunner) ExchangeName() connector.ExchangeName { return tr.exchangeName }

func (tr *PredictionMarketTestRunner) GetBaseConnector() connector.Connector { return tr.conn }

func (tr *PredictionMarketTestRunner) HasWebSocketSupport() bool {
	_, ok := tr.conn.(connector.WebSocketCapable)
	return ok
}

func (tr *PredictionMarketTestRunner) GetWebSocketCapable() prediction.WebSocketConnector {
	ws, ok := tr.conn.(prediction.WebSocketConnector)
	if !ok {
		return nil
	}
	return ws
}

// WatchMarket adds a market to the prediction watchlist (required for realtime store path).
func (tr *PredictionMarketTestRunner) WatchMarket(market prediction.Market) {
	tr.watchlist.RequireMarket(tr.exchangeName, market)
	// Also via SDK so strategy path stays consistent
	tr.predict.WatchMarket(tr.exchangeName, market)
}

// CollectNow runs prediction batch ingestors (balances → store).
func (tr *PredictionMarketTestRunner) CollectNow() {
	for _, ingestor := range tr.batchIngestorFactory.CreateIngestors() {
		ingestor.CollectNow()
	}
}

func (tr *PredictionMarketTestRunner) StartRealtime(ctx context.Context) error {
	tr.rtMu.Lock()
	defer tr.rtMu.Unlock()
	if len(tr.rtIngestors) > 0 {
		return nil
	}
	ingestors := tr.realtimeIngestorFactory.CreateIngestors()
	if len(ingestors) == 0 {
		return fmt.Errorf("no prediction realtime ingestors")
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

func (tr *PredictionMarketTestRunner) StopRealtime() error {
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

// StoreOrderBook for a market outcome.
func (tr *PredictionMarketTestRunner) StoreOrderBook(marketID prediction.MarketID, outcomeID prediction.OutcomeID) *connector.OrderBook {
	return tr.store.GetOrderBook(tr.exchangeName, marketID, outcomeID)
}

// StoreBalance for an asset.
func (tr *PredictionMarketTestRunner) StoreBalance(asset portfolio.Asset) (numerical.Decimal, bool) {
	return tr.store.GetBalance(tr.exchangeName, asset)
}

// VerifyOrderBookData waits for order book data from channel with timeout
func (tr *PredictionMarketTestRunner) VerifyOrderBookData(
	obChan <-chan prediction.OrderBook,
	timeout time.Duration,
) prediction.OrderBook {
	select {
	case ob, ok := <-obChan:
		if !ok {
			return prediction.OrderBook{}
		}
		return ob
	case <-time.After(timeout):
		return prediction.OrderBook{}
	}
}

// VerifyPriceChangeData waits for price change data from channel with timeout
func (tr *PredictionMarketTestRunner) VerifyPriceChangeData(
	obChan <-chan prediction.PriceChange,
	timeout time.Duration,
) (prediction.PriceChange, error) {
	select {
	case ob, ok := <-obChan:
		if !ok {
			return prediction.PriceChange{}, fmt.Errorf("price change channel closed")
		}
		return ob, nil
	case <-time.After(timeout):
		return prediction.PriceChange{}, fmt.Errorf("timed out waiting for price change data")
	}
}

func (tr *PredictionMarketTestRunner) VerifyTradeData(channel <-chan connector.Trade, duration time.Duration) (connector.Trade, error) {
	timeout := time.After(duration)
	for {
		select {
		case trade, ok := <-channel:
			if !ok {
				return connector.Trade{}, fmt.Errorf("trade channel closed")
			}
			return trade, nil
		case <-timeout:
			return connector.Trade{}, fmt.Errorf("timed out waiting for trade data")
		}
	}
}

func (tr *PredictionMarketTestRunner) VerifyOrderData(channel <-chan connector.Order, duration time.Duration) (connector.Order, error) {
	timeout := time.After(duration)
	for {
		select {
		case order, ok := <-channel:
			if !ok {
				return connector.Order{}, fmt.Errorf("order channel closed")
			}
			return order, nil
		case <-timeout:
			return connector.Order{}, fmt.Errorf("timed out waiting for order data")
		}
	}
}
