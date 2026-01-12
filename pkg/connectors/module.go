package connectors

import (
	"github.com/backtesting-org/live-trading/pkg/connectors/bybit"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid"
	"github.com/backtesting-org/live-trading/pkg/connectors/paradex"
	"go.uber.org/fx"
)

// Module includes all exchange connector modules
// Each connector module automatically registers itself via fx groups
var Module = fx.Options(
	paradex.Module,
	hyperliquid.Module,
	bybit.Module,
)
