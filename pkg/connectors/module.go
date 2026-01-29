package connectors

import (
	"github.com/wisp-trading/connectors/pkg/connectors/bybit"
	"github.com/wisp-trading/connectors/pkg/connectors/gate"
	"github.com/wisp-trading/connectors/pkg/connectors/hyperliquid"
	"github.com/wisp-trading/connectors/pkg/connectors/paradex"
	"go.uber.org/fx"
)

// Module includes all exchange connector modules
// Each connector module automatically registers itself via fx groups
var Module = fx.Options(
	paradex.Module,
	hyperliquid.Module,
	bybit.Module,
	gate.Module,
)
