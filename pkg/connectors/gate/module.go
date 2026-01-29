package gate

import (
	"github.com/wisp-trading/connectors/pkg/connectors/gate/spot"
	"go.uber.org/fx"
)

// Module includes Gate.io connector modules (spot and perpetuals)
var Module = fx.Options(
	spot.Module,
)
