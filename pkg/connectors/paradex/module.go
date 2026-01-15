package paradex

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector/perp"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	"github.com/backtesting-org/live-trading/pkg/connectors/types"
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
func registerParadex(paradexConn perp.Connector, reg registry.ConnectorRegistry) {
	// Register the connector
	reg.RegisterPerpConnector(types.Paradex, paradexConn)
}
