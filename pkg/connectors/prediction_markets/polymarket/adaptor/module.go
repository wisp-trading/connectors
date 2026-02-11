package adaptor

import (
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/clob"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/gamma"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/websocket"
	"go.uber.org/fx"
)

var Module = fx.Options(
	websocket.WebSocketModule,
	
	fx.Provide(
		gamma.NewPolymarketClient,
		clob.NewPolymarketClient,
	),
)
