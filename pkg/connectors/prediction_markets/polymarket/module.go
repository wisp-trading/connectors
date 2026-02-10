package polymarket

import (
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/gamma"
	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	"go.uber.org/fx"
)

// Module is the Polymarket Spot connector module
var Module = fx.Options(
	fx.Provide(
		gamma.NewPolymarketClient,
		fx.Annotate(
			NewPolymarket,
			fx.ParamTags(``, ``, ``, ``, ``), // No special tags - use auto-wiring
			fx.ResultTags(`name:"polymarket"`),
		),
	),
	fx.Invoke(fx.Annotate(
		registerPolymarket,
		fx.ParamTags(`name:"polymarket"`),
	)),
)

// registerPolymarket registers the Polymarket Spot connector with the SDK's ConnectorRegistry
func registerPolymarket(polymarketConn prediction.Connector, reg registry.ConnectorRegistry) {
	reg.RegisterPredictionMarketConnector(types.Polymarket, polymarketConn)
}
