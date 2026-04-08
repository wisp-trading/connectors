package deribit

import (
	"github.com/wisp-trading/connectors/pkg/connectors/options/deribit/adaptor"
	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector/options"
	"github.com/wisp-trading/sdk/pkg/types/logging"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	"github.com/wisp-trading/sdk/pkg/types/temporal"
	"go.uber.org/fx"
)

// Module provides the Deribit options connector and its dependencies
var Module = fx.Options(
	fx.Provide(
		adaptor.NewClient,
		fx.Annotate(
			NewDeribitOptions,
			fx.ParamTags(``, ``, ``, ``),
			fx.ResultTags(`name:"deribit_options"`),
		),
	),
	fx.Invoke(fx.Annotate(
		registerDeribitOptions,
		fx.ParamTags(`name:"deribit_options"`),
	)),
)

// registerDeribitOptions registers the Deribit Options connector with the SDK's ConnectorRegistry
func registerDeribitOptions(deribitConn options.Connector, reg registry.ConnectorRegistry) {
	reg.RegisterOptions(types.DeribitOptions, deribitConn)
}

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
