package deribit

import (
	"github.com/wisp-trading/connectors/pkg/connectors/options/deribit/adaptor"
	"github.com/wisp-trading/sdk/pkg/types/logging"
	"github.com/wisp-trading/sdk/pkg/types/temporal"
	"go.uber.org/fx"
)

// Module provides the Deribit options connector and its dependencies
var Module = fx.Module(
	"deribit_options",
	fx.Provide(
		adaptor.NewClient,
		NewDeribitOptions,
	),
)

// ProvideConnector explicitly provides the options connector for dependency injection
// Usage: fx.Provide(deribit.ProvideConnector) in your main module
func ProvideConnector(
	client adaptor.Client,
	appLogger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
	timeProvider temporal.TimeProvider,
) interface{} {
	return NewDeribitOptions(client, appLogger, tradingLogger, timeProvider)
}
