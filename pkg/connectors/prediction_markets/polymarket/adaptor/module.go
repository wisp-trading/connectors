package adaptor

import (
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/gamma"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/order_manager"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/websocket"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		NewPolymarketClient,
		gamma.NewGammaClient,
		order_manager.NewOrderManager,
		websocket.NewWebsocket,
	),
)
