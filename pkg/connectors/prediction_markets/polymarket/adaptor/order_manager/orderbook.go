package order_manager

import (
	"context"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

func (c *orderManager) GetOrderBook(ctx context.Context, outcome prediction.Outcome) (clobtypes.OrderBookResponse, error) {
	books, err := c.client.OrderBook(ctx, &clobtypes.BookRequest{
		TokenID: outcome.OutcomeID.String(),
		Side:    outcome.Side.ToString(),
	})
	if err != nil {
		return clobtypes.OrderBookResponse{}, err
	}

	return books, nil
}
