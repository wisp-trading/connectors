package connector

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"

	"github.com/backtesting-org/kronos-sdk/kronos"
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector/perp"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	"github.com/backtesting-org/live-trading/pkg/connectors"
)

// PerpTestRunner manages the lifecycle of perpetual connector tests
type PerpTestRunner struct {
	*BaseRunnerImpl
	conn   perp.Connector
	wsConn perp.WebSocketConnector
}

// NewPerpTestRunner creates a new test runner for perp connectors
func NewPerpTestRunner(connectorName connector.ExchangeName, config connector.Config) (*PerpTestRunner, error) {
	var reg registry.ConnectorRegistry

	app := fx.New(
		kronos.Module,
		connectors.Module,
		fx.Populate(&reg),
		fx.NopLogger,
	)

	startCtx, startCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer startCancel()

	if err := app.Start(startCtx); err != nil {
		return nil, fmt.Errorf("failed to start fx app: %w", err)
	}

	// Get PERP connector from registry
	conn, exists := reg.GetPerpConnector(connectorName)
	if !exists {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("perp connector %s not found in registry", connectorName)
	}

	// Initialize connector
	if err := conn.Initialize(config); err != nil {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("failed to initialize connector: %w", err)
	}

	// Try to get WebSocket interface
	wsConn, _ := conn.(perp.WebSocketConnector)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	return &PerpTestRunner{
		BaseRunnerImpl: &BaseRunnerImpl{
			app:    app,
			ctx:    ctx,
			cancel: cancel,
			reg:    reg,
		},
		conn:   conn,
		wsConn: wsConn,
	}, nil
}

// GetPerpConnector returns the perp connector instance
func (tr *PerpTestRunner) GetPerpConnector() perp.Connector {
	return tr.conn
}

// GetBaseConnector returns the base connector for shared tests
func (tr *PerpTestRunner) GetBaseConnector() connector.Connector {
	return tr.conn // perp.Connector embeds connector.Connector
}

// GetWebSocketConnector returns the WebSocket connector instance
func (tr *PerpTestRunner) GetWebSocketConnector() perp.WebSocketConnector {
	return tr.wsConn
}

// HasWebSocketSupport checks if connector supports WebSocket
func (tr *PerpTestRunner) HasWebSocketSupport() bool {
	return tr.wsConn != nil
}

// GetWebSocketCapable returns the base WebSocket capability
func (tr *PerpTestRunner) GetWebSocketCapable() connector.WebSocketCapable {
	if tr.wsConn == nil {
		return nil
	}
	return tr.wsConn
}

// Perp-specific helpers

// GetPerpSymbol returns the perp symbol for an asset
func (tr *PerpTestRunner) GetPerpSymbol(asset portfolio.Asset) string {
	return tr.conn.GetPerpSymbol(asset)
}
