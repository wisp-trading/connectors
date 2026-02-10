package connectors

import (
	"github.com/wisp-trading/connectors/pkg/connectors/bybit/perp"
	"github.com/wisp-trading/connectors/pkg/connectors/gate"
	"github.com/wisp-trading/connectors/pkg/connectors/hyperliquid"
	"github.com/wisp-trading/connectors/pkg/connectors/paradex"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket"
	"go.uber.org/fx"
)

// Module includes all exchange connector modules
// Each connector module automatically registers itself via fx groups
var Module = fx.Options(
	paradex.Module,
	hyperliquid.Module,
	perp.Module,
	gate.Module,
	polymarket.Module,
)
