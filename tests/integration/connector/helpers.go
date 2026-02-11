package connector

import (
	"context"
	"fmt"
	"time"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	"go.uber.org/fx"
)

// BaseTestRunner provides common test runner functionality
type BaseTestRunner interface {
	// Cleanup releases resources
	Cleanup()
	// GetContext returns the test context
	GetContext() context.Context
	// GetBaseConnector returns the base connector for shared tests
	GetBaseConnector() connector.Connector
	// HasWebSocketSupport checks if connector supports WebSocket
	HasWebSocketSupport() bool
	// GetWebSocketCapable returns the base WebSocket capability
	GetWebSocketCapable() connector.WebSocketCapable
}

// BaseRunnerImpl contains shared implementation for test runners
type BaseRunnerImpl struct {
	app    *fx.App
	ctx    context.Context
	cancel context.CancelFunc
	reg    registry.ConnectorRegistry
}

// Cleanup releases all resources
func (b *BaseRunnerImpl) Cleanup() {
	if b.cancel != nil {
		b.cancel()
	}
	if b.app != nil {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer stopCancel()
		_ = b.app.Stop(stopCtx)
	}
}

// GetContext returns the test context
func (b *BaseRunnerImpl) GetContext() context.Context {
	return b.ctx
}

// GetRegistry returns the connector registry
func (b *BaseRunnerImpl) GetRegistry() registry.ConnectorRegistry {
	return b.reg
}

// LogSuccess logs a successful test action with formatted message
func LogSuccess(format string, args ...interface{}) {
	fmt.Printf("[SUCCESS] "+format+"\n", args...)
}

// LogInfo logs an info message with formatted output
func LogInfo(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

// LogDebug logs a debug message with formatted output
func LogDebug(format string, args ...interface{}) {
	fmt.Printf("[DEBUG] "+format+"\n", args...)
}

func LogError(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}
