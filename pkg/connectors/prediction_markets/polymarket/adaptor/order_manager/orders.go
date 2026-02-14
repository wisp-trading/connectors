package order_manager

import (
	"context"
	"fmt"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
)

// PlaceOrder places an order on Polymarket
func (c *orderManager) PlaceOrder(ctx context.Context, order prediction.LimitOrder) (clobtypes.OrderResponse, error) {
	side := "BUY"
	if order.Side == connector.OrderSideSell {
		side = "SELL"
	}

	size, err := c.client.TickSize(ctx, &clobtypes.TickSizeRequest{
		TokenID: order.Outcome.OutcomeId,
	})
	if err != nil {
		return clobtypes.OrderResponse{}, err
	}

	fmt.Printf("Tick size for token %s: %f\n", order.Outcome.OutcomeId, size.MinimumTickSize)

	// Build the order using the SDK builder
	signableOrder, err := clob.NewOrderBuilder(c.client, c.signer).
		TokenID(order.Outcome.OutcomeId).
		Side(side).
		Price(order.Price.InexactFloat64()).
		Size(order.Amount.InexactFloat64()).
		OrderType(clobtypes.OrderTypeGTC).
		TickSize(size.MinimumTickSize).
		Build()

	if err != nil {
		return clobtypes.OrderResponse{}, err
	}

	// Submit the order
	resp, err := c.client.CreateOrder(ctx, signableOrder)
	if err != nil {
		return clobtypes.OrderResponse{}, nil
	}

	fmt.Printf("Order placed: %v\n", resp)
	return resp, nil
}
