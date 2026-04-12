package order_manager

import (
	"context"
	"fmt"
	"math"
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

	// For BUY orders the CLOB must know the EOA's on-chain USDC balance.
	// UpdateBalanceAllowance triggers an async re-read on Polymarket's side;
	// we poll BalanceAllowance until they confirm a non-zero balance before
	// submitting, otherwise orders are rejected with "balance: 0".
	if order.Side == connector.OrderSideBuy {
		if err := confirmCollateralBalance(ctx, c); err != nil {
			fmt.Printf("[polymarket:order] warn: collateral balance confirmation timed out, proceeding anyway: %v\n", err)
		}
	}

	size, err := c.client.TickSize(ctx, &clobtypes.TickSizeRequest{
		TokenID: order.Outcome.OutcomeID.String(),
	})
	if err != nil {
		return clobtypes.OrderResponse{}, err
	}

	// CLOB balance cache staleness workaround: after instant-fill buys (status=matched),
	// Polymarket's internal balance cache is ~1% stale. Selling 100% fails with "balance: 0"
	// but selling 99.99% succeeds because it stays under the stale ceiling.
	// See: https://github.com/GoPolymarket/py-clob-client/issues/XXX
	orderSize := order.Amount.InexactFloat64()
	if order.Side == connector.OrderSideSell {
		orderSize *= 0.9999 // Leave ~0.01% dust that settles at market resolution
		// Round to 2 decimal places (Polymarket CLOB max precision)
		orderSize = math.Round(orderSize*100) / 100
	}

	// Build the order using the SDK builder
	signableOrder, err := clob.NewOrderBuilder(c.client, c.signer).
		TokenID(order.Outcome.OutcomeID.String()).
		Side(side).
		Price(order.Price.InexactFloat64()).
		Size(orderSize).
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

// PlaceOrders builds, signs, and submits multiple orders in a single CLOB batch request.
// Collateral balance is confirmed once for the batch if any BUY orders are present.
func (c *orderManager) PlaceOrders(ctx context.Context, orders []prediction.LimitOrder) (clobtypes.PostOrdersResponse, error) {
	if len(orders) == 0 {
		return nil, nil
	}

	// Confirm collateral once for the batch if any leg is a BUY.
	hasBuy := false
	for _, o := range orders {
		if o.Side == connector.OrderSideBuy {
			hasBuy = true
			break
		}
	}
	if hasBuy {
		if err := confirmCollateralBalance(ctx, c); err != nil {
			fmt.Printf("[polymarket:order] warn: collateral balance confirmation timed out, proceeding anyway: %v\n", err)
		}
	}

	signed := make([]clobtypes.SignedOrder, 0, len(orders))
	for _, order := range orders {
		side := "BUY"
		if order.Side == connector.OrderSideSell {
			side = "SELL"
		}

		size, err := c.client.TickSize(ctx, &clobtypes.TickSizeRequest{
			TokenID: order.Outcome.OutcomeID.String(),
		})
		if err != nil {
			return nil, fmt.Errorf("tick size lookup failed for %s: %w", order.Outcome.OutcomeID, err)
		}

		orderSize := order.Amount.InexactFloat64()
		if order.Side == connector.OrderSideSell {
			orderSize *= 0.9999
			orderSize = math.Round(orderSize*100) / 100
		}

		built, err := clob.NewOrderBuilder(c.client, c.signer).
			TokenID(order.Outcome.OutcomeID.String()).
			Side(side).
			Price(order.Price.InexactFloat64()).
			Size(orderSize).
			OrderType(toClobOrderType(order.TimeInForce)).
			TickSize(size.MinimumTickSize).
			Build()
		if err != nil {
			return nil, fmt.Errorf("order build failed for %s: %w", order.Outcome.OutcomeID, err)
		}

		signerAddr := c.signer.Address()
		tokenID := order.Outcome.OutcomeID.String()
		negRisk := built.NegRisk
		sigType := 0
		if built.SignatureType != nil {
			sigType = *built.SignatureType
		}
		fmt.Printf("[polymarket:batch] side=%s token=%s signer=%s maker=%s sig_type=%d neg_risk=%v\n",
			side, tokenID, signerAddr.Hex(), built.Maker.Hex(), sigType, negRisk)

		s, err := c.client.SignOrder(built)
		if err != nil {
			return nil, fmt.Errorf("order signing failed for %s: %w", order.Outcome.OutcomeID, err)
		}

		// Set order type on the signed order (FOK/FAK/GTC).
		s.OrderType = toClobOrderType(order.TimeInForce)
		signed = append(signed, *s)
	}

	resp, err := c.client.PostOrders(ctx, &clobtypes.SignedOrders{Orders: signed})
	if err != nil {
		return nil, fmt.Errorf("batch order submission failed: %w", err)
	}

	// Check FOK statuses — any cancelled/unmatched FOK order is an error.
	for i, r := range resp {
		if i < len(orders) && orders[i].TimeInForce == connector.TimeInForceFOK {
			s := strings.ToLower(strings.TrimSpace(r.Status))
			if strings.Contains(s, "cancel") || strings.Contains(s, "unmatch") {
				return resp, fmt.Errorf(
					"fok order %d not filled: clob returned status %q for token %s",
					i, r.Status, orders[i].Outcome.OutcomeID,
				)
			}
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

func (c *orderManager) GetOrder(ctx context.Context, orderID string) (clobtypes.OrderResponse, error) {
	return c.client.Order(ctx, orderID)
}

func (c *orderManager) GetOpenOrders(ctx context.Context, market string) ([]clobtypes.OrderResponse, error) {
	req := &clobtypes.OrdersRequest{}
	if market != "" {
		req.Market = market
	}

	resp, err := c.client.Orders(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}
