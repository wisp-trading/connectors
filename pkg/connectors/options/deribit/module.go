package deribit

import (
	"github.com/wisp-trading/connectors/pkg/connectors/options/deribit/adaptor"
	deribitWS "github.com/wisp-trading/connectors/pkg/connectors/options/deribit/websocket"
	"github.com/wisp-trading/connectors/pkg/connectors/types"
	optionsConnector "github.com/wisp-trading/sdk/pkg/types/connector/options"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	"go.uber.org/fx"
)

// Module provides the Deribit options connector, its HTTP client, and its WebSocket service.
var Module = fx.Options(
	// Wire all WebSocket infrastructure (connection manager, reconnect, base service, etc.)
	deribitWS.WebSocketModule,

	fx.Provide(
		adaptor.NewClient,
		fx.Annotate(
			NewDeribitOptions,
			// NewDeribitOptions params: client, appLogger, tradingLogger, timeProvider, wsService
			fx.ParamTags(``, ``, ``, ``, ``),
			fx.ResultTags(`name:"deribit_options"`),
		),
	),
	fx.Invoke(fx.Annotate(
		registerDeribitOptions,
		fx.ParamTags(`name:"deribit_options"`),
	)),
)

// registerDeribitOptions registers the connector with the SDK's ConnectorRegistry.
// We register as options.Connector (the base interface) so the registry type is satisfied;
// the full WebSocketConnector is accessible via type assertion on the registered value.
func registerDeribitOptions(deribitConn optionsConnector.WebSocketConnector, reg registry.ConnectorRegistry) {
	reg.RegisterOptions(types.DeribitOptions, deribitConn)
}
