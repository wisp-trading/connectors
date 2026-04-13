package hyperliquid

import (
	"github.com/wisp-trading/connectors/pkg/connectors/adaptors/hyperliquid"
	"github.com/wisp-trading/connectors/pkg/connectors/spot/hyperliquid/rest"
	"github.com/wisp-trading/connectors/pkg/connectors/spot/hyperliquid/websocket"
	"github.com/wisp-trading/connectors/pkg/connectors/types"
	spotconnector "github.com/wisp-trading/sdk/pkg/types/connector/spot"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	"go.uber.org/fx"
)

// Module is the Hyperliquid spot connector module.
// It reuses the adaptors (ExchangeClient, InfoClient) provided by the perps module
// and wires its own WebSocket connection with spot-specific named deps.
var Module = fx.Options(
	websocket.WebSocketModule,

	fx.Provide(
		rest.NewSpotTradingService,
		rest.NewSpotMarketDataService,
		fx.Annotate(
			NewHyperliquidSpot,
			fx.ResultTags(`name:"hyperliquid_spot"`),
		),
	),

	fx.Invoke(fx.Annotate(
		registerHyperliquidSpot,
		fx.ParamTags(`name:"hyperliquid_spot"`),
	)),
)

// registerHyperliquidSpot registers the spot connector with the SDK's ConnectorRegistry
func registerHyperliquidSpot(spotConn spotconnector.Connector, reg registry.ConnectorRegistry) {
	reg.RegisterSpot(types.HyperliquidSpot, spotConn)
}

// SharedAdaptorsModule provides the shared API clients.
// Only needed if the perps module is not loaded — when both are loaded,
// fx deduplicates the adaptors automatically.
var SharedAdaptorsModule = fx.Options(
	fx.Provide(
		hyperliquid.NewExchangeClient,
		hyperliquid.NewInfoClient,
	),
)
