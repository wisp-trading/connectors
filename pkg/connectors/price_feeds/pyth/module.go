package pyth

import "go.uber.org/fx"

// Module provides Pyth price feed integration via fx
var Module = fx.Module(
	"pyth",
	fx.Provide(NewService),
)
