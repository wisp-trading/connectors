package spot

import (
	"github.com/wisp-trading/connectors/pkg/connectors/gate/adaptor"
	"github.com/wisp-trading/connectors/pkg/connectors/gate/websocket"
	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector/spot"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	"go.uber.org/fx"
)

// Module is the Gate Spot connector module
var Module = fx.Options(
	websocket.WebSocketModule,
	fx.Provide(
		adaptor.NewSpotClient,
		fx.Annotate(
			NewGateSpot,
			fx.ParamTags(``, ``, ``, ``, ``), // No special tags - use auto-wiring
			fx.ResultTags(`name:"gate_spot"`),
		),
	),
	fx.Invoke(fx.Annotate(
		registerGateSpot,
		fx.ParamTags(`name:"gate_spot"`),
	)),
)

// registerGateSpot registers the Gate Spot connector with the SDK's ConnectorRegistry
func registerGateSpot(gateSpotConn spot.Connector, reg registry.ConnectorRegistry) {
	reg.RegisterSpotConnector(types.GateSpot, gateSpotConn)
}
