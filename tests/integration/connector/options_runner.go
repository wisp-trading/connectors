package connector

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"

	"github.com/wisp-trading/connectors/pkg/connectors"
	optionsTypes "github.com/wisp-trading/sdk/pkg/markets/options/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/options"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	wispTypes "github.com/wisp-trading/sdk/pkg/types/wisp"
	"github.com/wisp-trading/sdk/wisp"
)

// OptionsTestRunner manages the lifecycle of options connector tests
type OptionsTestRunner struct {
	*BaseRunnerImpl
	conn                 options.Connector
	wsConn               options.WebSocketConnector
	wisp                 wispTypes.Wisp
	optionsStore         optionsTypes.OptionsStore
	optionsWatchlist     optionsTypes.OptionsWatchlist
	batchIngestorFactory optionsTypes.OptionsBatchIngestorFactory
}

// NewOptionsTestRunner creates a new test runner for options connectors
func NewOptionsTestRunner(connectorName connector.ExchangeName, config connector.Config) (*OptionsTestRunner, error) {
	var reg registry.ConnectorRegistry
	var wispInstance wispTypes.Wisp
	var optionsStore optionsTypes.OptionsStore
	var optionsWatchlist optionsTypes.OptionsWatchlist
	var batchIngestorFactory optionsTypes.OptionsBatchIngestorFactory

	app := fx.New(
		wisp.Module,
		connectors.Module,
		fx.Populate(&reg, &wispInstance, &optionsStore, &optionsWatchlist, &batchIngestorFactory),
		fx.NopLogger,
	)

	startCtx, startCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer startCancel()

	if err := app.Start(startCtx); err != nil {
		return nil, fmt.Errorf("failed to start fx app: %w", err)
	}

	// Get OPTIONS connector from registry
	conn, exists := reg.Options(connectorName)
	if !exists {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("options connector %s not found in registry", connectorName)
	}

	// Initialize connector
	if err := conn.Initialize(config); err != nil {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("failed to initialize connector: %w", err)
	}

	// Mark connector ready so the ingestor factory can find it via FilterOptions(ReadyOnly())
	if err := reg.MarkReady(connectorName); err != nil {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("failed to mark connector ready: %w", err)
	}

	// Try to get WebSocket interface
	wsConn, _ := conn.(options.WebSocketConnector)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	return &OptionsTestRunner{
		BaseRunnerImpl: &BaseRunnerImpl{
			app:    app,
			ctx:    ctx,
			cancel: cancel,
			reg:    reg,
		},
		conn:                 conn,
		wsConn:               wsConn,
		wisp:                 wispInstance,
		optionsStore:         optionsStore,
		optionsWatchlist:     optionsWatchlist,
		batchIngestorFactory: batchIngestorFactory,
	}, nil
}

// GetOptionsConnector returns the options connector instance
func (tr *OptionsTestRunner) GetOptionsConnector() options.Connector {
	return tr.conn
}

// GetBaseConnector returns the base connector for shared tests
func (tr *OptionsTestRunner) GetBaseConnector() connector.Connector {
	return tr.conn // options.Connector embeds connector.Connector
}

// GetWebSocketConnector returns the WebSocket connector instance
func (tr *OptionsTestRunner) GetWebSocketConnector() options.WebSocketConnector {
	return tr.wsConn
}

// HasWebSocketSupport checks if connector supports WebSocket
func (tr *OptionsTestRunner) HasWebSocketSupport() bool {
	return tr.wsConn != nil
}

// GetWebSocketCapable returns the base WebSocket capability
func (tr *OptionsTestRunner) GetWebSocketCapable() connector.WebSocketCapable {
	if tr.wsConn == nil {
		return nil
	}
	return tr.wsConn
}

// GetWisp returns the Wisp SDK instance for strategy testing
// This is how strategies access market data via the SDK
func (tr *OptionsTestRunner) GetWisp() wispTypes.Wisp {
	return tr.wisp
}

// GetOptionsStore returns the options store for verification
// This allows tests to verify that connector data reaches the store
func (tr *OptionsTestRunner) GetOptionsStore() optionsTypes.OptionsStore {
	return tr.optionsStore
}

// WatchExpiration adds an expiration to the watchlist so the ingestor will collect its data.
// This must be called before CollectNow to populate the store.
func (tr *OptionsTestRunner) WatchExpiration(pair portfolio.Pair, expiration time.Time) error {
	exchangeName := connector.ExchangeName(tr.conn.GetConnectorInfo().Name)
	return tr.optionsWatchlist.RequireExpiration(exchangeName, pair, expiration)
}

// CollectNow creates ingestors from the factory and triggers an immediate collection,
// running the full chain: connector.GetExpirationData() → store.Set*()
func (tr *OptionsTestRunner) CollectNow() {
	ingestors := tr.batchIngestorFactory.CreateIngestors()
	for _, ingestor := range ingestors {
		ingestor.CollectNow()
	}
}
