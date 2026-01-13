package spot

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	"github.com/backtesting-org/live-trading/pkg/connectors/gate/adaptor"
	"github.com/backtesting-org/live-trading/pkg/connectors/gate/websocket"
	"github.com/backtesting-org/live-trading/pkg/connectors/types"
	"go.uber.org/fx"
)

// Module is the Gate Spot connector module
var Module = fx.Options(
	websocket.WebSocketModule,
	fx.Provide(
		adaptor.NewSpotClient,
		fx.Annotate(
			NewGateSpot,
			fx.ResultTags(`name:"gate_spot"`),
		),
	),
	fx.Invoke(fx.Annotate(
		registerGateSpot,
		fx.ParamTags(`name:"gate_spot"`),
	)),
)

// registerGateSpot registers the Gate Spot connector with the SDK's ConnectorRegistry
func registerGateSpot(gateSpotConn connector.Connector, reg registry.ConnectorRegistry) {
	reg.RegisterConnector(types.GateSpot, gateSpotConn)
}
