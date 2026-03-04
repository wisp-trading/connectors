package websocket

import (
	"context"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/ws"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

func (w websocket) SubscribeTrades(market prediction.Market) (<-chan ws.TradeEvent, error) {
	ctx := context.Background()
	trades, err := w.client.SubscribeUserTrades(ctx, []string{market.MarketID.String()})
	if err != nil {
		return nil, err
	}

	return trades, nil
}
