package perp

import (
	"github.com/wisp-trading/connectors/pkg/connectors/bybit/adaptor"
	"github.com/wisp-trading/connectors/pkg/connectors/bybit/websocket"
	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	"go.uber.org/fx"
)

var Module = fx.Options(
	websocket.WebSocketModule,
	fx.Provide(
		adaptor.NewPerpClient,
		fx.Annotate(
			NewBybit,
			fx.ParamTags(``, ``, ``, ``, ``), // No special tags - use auto-wiring
			fx.ResultTags(`name:"bybit"`),
		),
	),
	fx.Invoke(fx.Annotate(
		registerBybit,
		fx.ParamTags(`name:"bybit"`),
	)),
)

func registerBybit(bybitConn perp.Connector, reg registry.ConnectorRegistry) {
	reg.RegisterPerp(types.Bybit, bybitConn)
}
