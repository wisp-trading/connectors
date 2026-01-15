package hyperliquid

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector/perp"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/adaptors"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/rest"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/websocket"
	"github.com/backtesting-org/live-trading/pkg/connectors/types"
	"go.uber.org/fx"
)

// Module is the main Hyperliquid connector module
var Module = fx.Options(
	websocket.WebSocketModule,

	fx.Provide(
		adaptors.NewExchangeClient,
		adaptors.NewInfoClient,
		rest.NewPriceValidator,
		rest.NewTradingService,
		rest.NewMarketDataService,
		fx.Annotate(
			NewHyperliquid,
			fx.ResultTags(`name:"hyperliquid"`),
		),
	),

	fx.Invoke(fx.Annotate(
		registerHyperliquid,
		fx.ParamTags(`name:"hyperliquid"`),
	)),
)

// registerHyperliquid registers the hyperliquid connector with the SDK's ConnectorRegistry
func registerHyperliquid(hyperliquidConn perp.Connector, reg registry.ConnectorRegistry) {
	reg.RegisterPerpConnector(types.Hyperliquid, hyperliquidConn)
}
