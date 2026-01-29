package bybit

import (
	"github.com/wisp-trading/connectors/pkg/connectors/bybit/data"
	"github.com/wisp-trading/connectors/pkg/connectors/bybit/data/real_time"
	"github.com/wisp-trading/connectors/pkg/connectors/bybit/trading"
	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		trading.NewTradingService,
		data.NewMarketDataService,
		real_time.NewRealTimeService,
		fx.Annotate(
			NewBybit,
			fx.ResultTags(`name:"bybit"`),
		),
	),
	fx.Invoke(fx.Annotate(
		registerBybit,
		fx.ParamTags(`name:"bybit"`),
	)),
)

func registerBybit(bybitConn perp.Connector, reg registry.ConnectorRegistry) {
	reg.RegisterPerpConnector(types.Bybit, bybitConn)
}
