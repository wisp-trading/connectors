package connector

import (
	"context"
	"fmt"
	"time"

	optionsTypes "github.com/wisp-trading/sdk/pkg/types/connector/options"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	"go.uber.org/fx"
)

// BaseTestRunner provides common test runner functionality for perp, options,
// and prediction runners that still use the legacy shared-behaviors approach.
// The spot runner uses a standalone design that routes everything through the
// SDK public API instead.
type BaseTestRunner interface {
	Cleanup()
	GetContext() context.Context
	GetBaseConnector() connector.Connector
	HasWebSocketSupport() bool
	GetWebSocketCapable() connector.WebSocketCapable
}

// BaseRunnerImpl contains shared implementation for perp, options, and
// prediction test runners.
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

// ─── Logging Helpers ─────────────────────────────────────────────────────

// LogSuccess logs a successful test action with formatted message
func LogSuccess(format string, args ...interface{}) {
	fmt.Printf("[SUCCESS] "+format+"\n", args...)
}

// LogInfo logs an info message with formatted output
func LogInfo(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

// LogWarning logs a warning message with formatted output
func LogWarning(format string, args ...interface{}) {
	fmt.Printf("[WARNING] "+format+"\n", args...)
}

// LogDebug logs a debug message with formatted output
func LogDebug(format string, args ...interface{}) {
	fmt.Printf("[DEBUG] "+format+"\n", args...)
}

// LogError logs an error message with formatted output
func LogError(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

// ─── Test Data Constructors ──────────────────────────────────────────────

// CreateOptionsContract creates a test options contract
func CreateOptionsContract(symbol string, strike float64, optionType string) optionsTypes.OptionContract {
	pair := portfolio.NewPair(
		portfolio.NewAsset(symbol),
		portfolio.NewAsset("USDT"),
	)

	return optionsTypes.OptionContract{
		Pair:       pair,
		Strike:     strike,
		Expiration: time.Now().AddDate(0, 0, 30), // 30 days out
		OptionType: optionType,
	}
}
