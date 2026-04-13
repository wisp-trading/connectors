package connector

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"

	"github.com/wisp-trading/connectors/pkg/connectors"
	spotTypes "github.com/wisp-trading/sdk/pkg/markets/spot/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/execution"
	lifecycleTypes "github.com/wisp-trading/sdk/pkg/types/lifecycle"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	"github.com/wisp-trading/sdk/pkg/types/strategy"
	wispTypes "github.com/wisp-trading/sdk/pkg/types/wisp"
	"github.com/wisp-trading/sdk/wisp"
)

// TestStrategyName is the strategy name used by the test runner when starting
// the SDK lifecycle. Signal builders in tests must use this same name.
const TestStrategyName = "integration-test"

// errCleanupTimeout is used for app.Stop in error paths during construction.
const errCleanupTimeout = 5 * time.Second

// SpotTestRunner manages the lifecycle of spot connector integration tests.
// It boots the full SDK stack so every assertion flows through the public
// wisp.Spot() API — exactly as a strategy would consume data.
//
// NO direct connector access is exposed. All reads go through
// Spot(), and all writes go through Spot().Signal().
type SpotTestRunner struct {
	app        *fx.App
	ctx        context.Context
	cancel     context.CancelFunc
	controller lifecycleTypes.Controller
	wisp       wispTypes.Wisp
	registry   registry.ConnectorRegistry
	exchange   connector.ExchangeName
}

// NewSpotTestRunner boots the full fx app (wisp.Module + connectors.Module),
// initialises the named connector, marks it ready, and starts the SDK
// lifecycle so that ingestors begin collecting data automatically.
func NewSpotTestRunner(connectorName connector.ExchangeName, config connector.Config) (*SpotTestRunner, error) {
	var reg registry.ConnectorRegistry
	var wispInstance wispTypes.Wisp
	var controller lifecycleTypes.Controller

	app := fx.New(
		wisp.Module,
		connectors.Module,
		fx.Populate(&reg, &wispInstance, &controller),
		fx.NopLogger,
	)

	startCtx, startCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer startCancel()

	if err := app.Start(startCtx); err != nil {
		return nil, fmt.Errorf("failed to start fx app: %w", err)
	}

	// Initialise and mark the connector ready BEFORE starting the lifecycle.
	// This ensures the coordinator's CreateIngestors() finds a ready connector.
	conn, exists := reg.Spot(connectorName)
	if !exists {
		stopOnError(app)
		return nil, fmt.Errorf("spot connector %s not found in registry", connectorName)
	}

	if err := conn.Initialize(config); err != nil {
		stopOnError(app)
		return nil, fmt.Errorf("failed to initialize connector: %w", err)
	}

	if err := reg.MarkReady(connectorName); err != nil {
		stopOnError(app)
		return nil, fmt.Errorf("failed to mark connector ready: %w", err)
	}

	// Start the SDK lifecycle — this creates domain coordinators which spawn
	// batch + realtime ingestors. The batch ingestors fire an immediate
	// CollectNow() on startup, then run on a 30-second timer.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	if err := controller.Start(ctx, TestStrategyName, nil); err != nil {
		cancel()
		stopOnError(app)
		return nil, fmt.Errorf("failed to start SDK lifecycle: %w", err)
	}

	return &SpotTestRunner{
		app:        app,
		ctx:        ctx,
		cancel:     cancel,
		controller: controller,
		wisp:       wispInstance,
		registry:   reg,
		exchange:   connectorName,
	}, nil
}

// ─── SDK Public API ───────────────────────────────────────────────────────

// Spot returns the SDK's public Spot domain service.
// All test assertions MUST go through this — never bypass to internal stores.
func (tr *SpotTestRunner) Spot() spotTypes.Spot {
	return tr.wisp.Spot()
}

// ExchangeName returns the exchange name of the initialised connector.
func (tr *SpotTestRunner) ExchangeName() connector.ExchangeName {
	return tr.exchange
}

// ─── Order Execution ─────────────────────────────────────────────────────

// Emit dispatches a built signal through the SDK's execution pipeline and
// returns the execution callback. Callers can Await or AwaitWithTimeout.
func (tr *SpotTestRunner) Emit(signal strategy.Signal) execution.ExecutionCallback {
	return tr.wisp.Emit(signal)
}

// CancelOrder cancels an order by ID via the connector's OrderExecutor.
// This bypasses the SDK signal pipeline because the Spot public API does
// not expose cancellation — it is a connector-level operation.
func (tr *SpotTestRunner) CancelOrder(orderID string, pair portfolio.Pair) (*connector.CancelResponse, error) {
	conn, exists := tr.registry.Spot(tr.exchange)
	if !exists {
		return nil, fmt.Errorf("spot connector %s not found in registry", tr.exchange)
	}
	return conn.CancelOrder(orderID, pair)
}

// GetOpenOrders queries the exchange for currently open orders.
func (tr *SpotTestRunner) GetOpenOrders(pair ...portfolio.Pair) ([]connector.Order, error) {
	conn, exists := tr.registry.Spot(tr.exchange)
	if !exists {
		return nil, fmt.Errorf("spot connector %s not found in registry", tr.exchange)
	}
	return conn.GetOpenOrders(pair...)
}

// ─── Watchlist ────────────────────────────────────────────────────────────

// WatchPair registers a pair via the SDK's public Spot().WatchPair() API,
// exactly as a strategy would.
func (tr *SpotTestRunner) WatchPair(pair portfolio.Pair) {
	tr.wisp.Spot().WatchPair(tr.exchange, pair)
}

// UnwatchPair removes a pair via the SDK's public Spot().UnwatchPair() API.
func (tr *SpotTestRunner) UnwatchPair(pair portfolio.Pair) {
	tr.wisp.Spot().UnwatchPair(tr.exchange, pair)
}

// ─── Lifecycle ────────────────────────────────────────────────────────────

// Cleanup releases all resources. Order: cancel context (signal goroutines) →
// drain the lifecycle controller → stop the fx app.
func (tr *SpotTestRunner) Cleanup() {
	// 1. Cancel the running context so all goroutines receive the stop signal.
	if tr.cancel != nil {
		tr.cancel()
	}

	// 2. Drain the lifecycle controller (ingestors, orchestrator).
	if tr.controller != nil {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer stopCancel()
		if err := tr.controller.Stop(stopCtx); err != nil {
			LogError("controller.Stop failed: %v", err)
		}
	}

	// 3. Stop the fx app (dependency teardown).
	if tr.app != nil {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer stopCancel()
		if err := tr.app.Stop(stopCtx); err != nil {
			LogError("app.Stop failed: %v", err)
		}
	}
}

// GetContext returns the test context.
func (tr *SpotTestRunner) GetContext() context.Context {
	return tr.ctx
}

// stopOnError is a helper for error paths in NewSpotTestRunner.
// It stops the fx app with a bounded timeout to avoid hanging.
func stopOnError(app *fx.App) {
	ctx, cancel := context.WithTimeout(context.Background(), errCleanupTimeout)
	defer cancel()
	if err := app.Stop(ctx); err != nil {
		LogError("app.Stop on error path failed: %v", err)
	}
}
