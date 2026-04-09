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

func (c *orderManager) GetOrderBooks(ctx context.Context, outcomes []prediction.Outcome) ([]clobtypes.OrderBook, error) {
	requests := make([]clobtypes.BookRequest, 0, len(outcomes))
	for _, outcome := range outcomes {
		requests = append(requests, clobtypes.BookRequest{
			TokenID: outcome.OutcomeID.String(),
			Side:    outcome.Side.ToString(),
		})
	}

	books, err := c.client.OrderBooks(ctx, &clobtypes.BooksRequest{
		Requests: requests,
	})
	if err != nil {
		return nil, err
	}

	return books, nil
}
