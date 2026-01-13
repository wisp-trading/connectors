package connector_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"

	"github.com/backtesting-org/kronos-sdk/kronos"
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	"github.com/backtesting-org/live-trading/pkg/connectors"
)

// TestRunner manages the lifecycle of connector tests
type TestRunner struct {
	app    *fx.App
	reg    registry.ConnectorRegistry
	conn   connector.Connector
	wsConn connector.WebSocketConnector
	ctx    context.Context
	cancel context.CancelFunc
}

// NewTestRunner creates a new test runner
func NewTestRunner(connectorName connector.ExchangeName, config connector.Config) (*TestRunner, error) {
	var reg registry.ConnectorRegistry

	// Create fx app with SDK + all connectors
	app := fx.New(
		kronos.Module,
		connectors.Module,
		fx.Invoke(func(conn connector.Connector, r registry.ConnectorRegistry) {
			r.RegisterConnector(connectorName, conn)
		}),
		fx.Populate(&reg),
		fx.NopLogger,
	)

	// Start fx app
	startCtx, startCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer startCancel()

	if err := app.Start(startCtx); err != nil {
		return nil, fmt.Errorf("failed to start fx app: %w", err)
	}

	// Get connector from registry
	conn, exists := reg.GetConnector(connectorName)
	if !exists {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("connector %s not found in registry", connectorName)
	}

	// Initialize connector
	if err := conn.Initialize(config); err != nil {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("failed to initialize connector: %w", err)
	}

	// Try to get WebSocket interface
	wsConn, _ := conn.(connector.WebSocketConnector)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	return &TestRunner{
		app:    app,
		reg:    reg,
		conn:   conn,
		wsConn: wsConn,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Cleanup stops the test runner
func (tr *TestRunner) Cleanup() {
	if tr.wsConn != nil && tr.wsConn.IsWebSocketConnected() {
		_ = tr.wsConn.StopWebSocket()
	}

	if tr.cancel != nil {
		tr.cancel()
	}

	if tr.app != nil {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer stopCancel()
		_ = tr.app.Stop(stopCtx)
	}
}

// GetConnector returns the connector instance
func (tr *TestRunner) GetConnector() connector.Connector {
	return tr.conn
}

// GetWebSocketConnector returns the WebSocket connector instance
func (tr *TestRunner) GetWebSocketConnector() connector.WebSocketConnector {
	return tr.wsConn
}

// GetContext returns the test context
func (tr *TestRunner) GetContext() context.Context {
	return tr.ctx
}

// HasWebSocketSupport checks if connector supports WebSocket
func (tr *TestRunner) HasWebSocketSupport() bool {
	return tr.wsConn != nil
}

// CreateAsset creates a portfolio asset from a symbol string
func CreateAsset(symbol string) portfolio.Asset {
	return portfolio.NewAsset(symbol)
}

// AssertNoError is a helper to assert no error occurred
func AssertNoError(err error, message string) {
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), message)
}

// AssertPositive checks if a numerical value is positive
func AssertPositive(value interface{}, message string) {
	// This will work with numerical.Decimal's IsPositive() method
	ExpectWithOffset(1, value).To(BeTrue(), message)
}

// LogSuccess prints a success message
func LogSuccess(format string, args ...interface{}) {
	GinkgoWriter.Printf("✓ "+format+"\n", args...)
}

// LogInfo prints an info message
func LogInfo(format string, args ...interface{}) {
	GinkgoWriter.Printf("  "+format+"\n", args...)
}
