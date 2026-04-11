package order_manager

import (
	"context"
	"fmt"
	"strings"

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

	// Log the resolved maker address and token ID before submission.
	// signableOrder.Maker is what the builder resolved — this is the address the CLOB
	// uses for balance checking and order crediting. c.signer.Address() is the EOA signer.
	// If these differ, orders are being routed through a proxy/Safe (sig_type != 0).
	signerAddr := c.signer.Address()
	tokenID := order.Outcome.OutcomeID.String()
	negRisk := signableOrder.NegRisk
	sigType := 0
	if signableOrder.SignatureType != nil {
		sigType = *signableOrder.SignatureType
	}
	fmt.Printf("[polymarket:order] side=%s token=%s signer=%s maker=%s sig_type=%d neg_risk=%v\n",
		side, tokenID, signerAddr.Hex(), signableOrder.Maker.Hex(), sigType, negRisk)

	// Submit the order
	resp, err := c.client.CreateOrder(ctx, signableOrder)
	if err != nil {
		return clobtypes.OrderResponse{}, err
	}

	// FOK orders must fill immediately — a cancelled response means no fill occurred.
	// The CLOB returns a non-error HTTP 200 with status "cancelled"/"unmatched" when
	// the order book has insufficient resting liquidity at the requested price.
	// Treat this as an error so callers can detect partial multi-leg failures early.
	if order.TimeInForce == connector.TimeInForceFOK {
		s := strings.ToLower(strings.TrimSpace(resp.Status))
		if strings.Contains(s, "cancel") || strings.Contains(s, "unmatch") {
			return clobtypes.OrderResponse{}, fmt.Errorf(
				"fok order not filled: clob returned status %q for token %s",
				resp.Status, order.Outcome.OutcomeID,
			)
		}
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
