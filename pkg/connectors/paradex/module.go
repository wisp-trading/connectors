package paradex

import (
	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		fx.Annotate(
			NewParadex,
			fx.ResultTags(`name:"paradex"`),
		),
	),
	// Automatically register hyperliquid with the SDK registry at startup
	fx.Invoke(fx.Annotate(
		registerParadex,
		fx.ParamTags(`name:"paradex"`),
	)),
)

// registerParadex registers the paradex connector with the SDK's ConnectorRegistry
func registerParadex(paradexConn perp.WebSocketConnector, reg registry.ConnectorRegistry) {
	// Register the connector
	reg.RegisterPerp(types.Paradex, perp.Connector(paradexConn))
}
