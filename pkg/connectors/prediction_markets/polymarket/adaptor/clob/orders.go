package clob

import (
	"context"
	"fmt"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
)

// PlaceOrder places an order on Polymarket
func (c *polymarketClient) PlaceOrder(ctx context.Context, limitOrder prediction.LimitOrder) error {
	if !c.IsConfigured() {
		return fmt.Errorf("client not configured")
	}

	side := "BUY"
	if limitOrder.Side == connector.OrderSideSell {
		side = "SELL"
	}

	// Build the order using the SDK builder
	signableOrder, err := clob.NewOrderBuilder(c.client, c.signer).
		TokenID(limitOrder.Outcome.OutcomeId).
		Side(side).
		Price(limitOrder.Price.InexactFloat64()).
		Size(limitOrder.Amount.InexactFloat64()).
		OrderType(clobtypes.OrderTypeGTC).
		FeeRateBps(0).
		TickSize("0.1").
		Maker(c.polymarketAddress).
		UseSafe().
		Build()

	if err != nil {
		return fmt.Errorf("failed to build order: %w", err)
	}

	// Submit the order
	resp, err := c.client.CreateOrder(ctx, signableOrder)
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	fmt.Printf("Order placed: %v\n", resp)
	return nil
}
