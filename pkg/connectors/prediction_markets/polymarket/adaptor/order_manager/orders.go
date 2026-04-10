package order_manager

import (
	"context"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector"
)

// toClobOrderType maps the SDK TimeInForce to the Polymarket CLOB order type.
// FOK ensures the entire order fills immediately or is cancelled — no resting on the book.
// FAK (IOC) fills what it can and cancels the rest. Default is GTC.
func toClobOrderType(tif connector.TimeInForce) clobtypes.OrderType {
	switch tif {
	case connector.TimeInForceFOK:
		return clobtypes.OrderTypeFOK
	case connector.TimeInForceFAK:
		return clobtypes.OrderTypeFAK
	default:
		return clobtypes.OrderTypeGTC
	}
}

// PlaceOrder places an order on Polymarket
func (c *orderManager) PlaceOrder(ctx context.Context, order prediction.LimitOrder) (clobtypes.OrderResponse, error) {
	side := "BUY"
	if order.Side == connector.OrderSideSell {
		side = "SELL"
	}

	size, err := c.client.TickSize(ctx, &clobtypes.TickSizeRequest{
		TokenID: order.Outcome.OutcomeID.String(),
	})
	if err != nil {
		return clobtypes.OrderResponse{}, err
	}

	// Build the order using the SDK builder
	signableOrder, err := clob.NewOrderBuilder(c.client, c.signer).
		TokenID(order.Outcome.OutcomeID.String()).
		Side(side).
		Price(order.Price.InexactFloat64()).
		Size(order.Amount.InexactFloat64()).
		OrderType(toClobOrderType(order.TimeInForce)).
		TickSize(size.MinimumTickSize).
		Build()

	if err != nil {
		return clobtypes.OrderResponse{}, err
	}

	// Submit the order
	resp, err := c.client.CreateOrder(ctx, signableOrder)
	if err != nil {
		return clobtypes.OrderResponse{}, err
	}

	return resp, nil
}

func (c *orderManager) CancelOrder(ctx context.Context, orderID string) (clobtypes.CancelResponse, error) {
	resp, err := c.client.CancelOrder(ctx, &clobtypes.CancelOrderRequest{
		OrderID: orderID,
	})

	if err != nil {
		return clobtypes.CancelResponse{}, err
	}

	return resp, nil
}
