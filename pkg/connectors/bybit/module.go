package bybit

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector/perp"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	"github.com/backtesting-org/live-trading/pkg/connectors/bybit/data"
	"github.com/backtesting-org/live-trading/pkg/connectors/bybit/data/real_time"
	"github.com/backtesting-org/live-trading/pkg/connectors/bybit/trading"
	"github.com/backtesting-org/live-trading/pkg/connectors/types"
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
