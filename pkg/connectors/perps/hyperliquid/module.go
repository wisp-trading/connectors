package hyperliquid

import (
	hyperliquid2 "github.com/wisp-trading/connectors/pkg/connectors/adaptors/hyperliquid"
	"github.com/wisp-trading/connectors/pkg/connectors/perps/hyperliquid/rest"
	"github.com/wisp-trading/connectors/pkg/connectors/perps/hyperliquid/websocket"
	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	"go.uber.org/fx"
)

// Module is the main Hyperliquid connector module
var Module = fx.Options(
	websocket.WebSocketModule,

	fx.Provide(
		hyperliquid2.NewExchangeClient,
		hyperliquid2.NewInfoClient,
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
	reg.RegisterPerp(types.Hyperliquid, hyperliquidConn)
}
