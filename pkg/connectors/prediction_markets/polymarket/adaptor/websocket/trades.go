package websocket

import (
	"context"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/ws"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
)

func (w websocket) SubscribeTrades(market prediction.Market) (<-chan ws.TradeEvent, error) {
	ctx := context.Background()
	trades, err := w.client.SubscribeUserTrades(ctx, []string{market.MarketId})
	if err != nil {
		return nil, err
	}

	return trades, nil
}
