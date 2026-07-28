package uniswap_v3

import (
	"github.com/wisp-trading/connectors/pkg/connectors/types"
	onchainconnector "github.com/wisp-trading/sdk/pkg/types/connector/onchain"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	"go.uber.org/fx"
)

// Module registers the UniV3 onchain connector.
var Module = fx.Options(
	fx.Provide(
		fx.Annotate(
			NewUniswapV3,
			fx.ResultTags(`name:"uniswap_v3"`),
		),
	),
	fx.Invoke(fx.Annotate(
		registerUniswapV3,
		fx.ParamTags(`name:"uniswap_v3"`),
	)),
)

func registerUniswapV3(conn onchainconnector.Connector, reg registry.ConnectorRegistry) {
	reg.RegisterOnchain(types.UniswapV3, conn)
}
