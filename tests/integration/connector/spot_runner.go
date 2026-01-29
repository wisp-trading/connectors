package connector

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"

	"github.com/wisp-trading/connectors/pkg/connectors"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/spot"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	"github.com/wisp-trading/sdk/wisp"
)

// SpotTestRunner manages the lifecycle of spot connector tests
type SpotTestRunner struct {
	*BaseRunnerImpl
	conn   spot.Connector
	wsConn spot.WebSocketConnector
}

// NewSpotTestRunner creates a new test runner for spot connectors
func NewSpotTestRunner(connectorName connector.ExchangeName, config connector.Config) (*SpotTestRunner, error) {
	var reg registry.ConnectorRegistry

	app := fx.New(
		wisp.Module,
		connectors.Module,
		fx.Populate(&reg),
		fx.NopLogger,
	)

	startCtx, startCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer startCancel()

	if err := app.Start(startCtx); err != nil {
		return nil, fmt.Errorf("failed to start fx app: %w", err)
	}

	// Get SPOT connector from registry
	conn, exists := reg.GetSpotConnector(connectorName)
	if !exists {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("spot connector %s not found in registry", connectorName)
	}

	// Initialize connector
	if err := conn.Initialize(config); err != nil {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("failed to initialize connector: %w", err)
	}

	// Try to get WebSocket interface
	wsConn, _ := conn.(spot.WebSocketConnector)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	return &SpotTestRunner{
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

// GetSpotConnector returns the spot connector instance
func (tr *SpotTestRunner) GetSpotConnector() spot.Connector {
	return tr.conn
}

// GetBaseConnector returns the base connector for shared tests
func (tr *SpotTestRunner) GetBaseConnector() connector.Connector {
	return tr.conn // spot.Connector embeds connector.Connector
}

// GetWebSocketConnector returns the WebSocket connector instance
func (tr *SpotTestRunner) GetWebSocketConnector() spot.WebSocketConnector {
	return tr.wsConn
}

// HasWebSocketSupport checks if connector supports WebSocket
func (tr *SpotTestRunner) HasWebSocketSupport() bool {
	return tr.wsConn != nil
}

// GetWebSocketCapable returns the base WebSocket capability
func (tr *SpotTestRunner) GetWebSocketCapable() connector.WebSocketCapable {
	if tr.wsConn == nil {
		return nil
	}
	return tr.wsConn
}
