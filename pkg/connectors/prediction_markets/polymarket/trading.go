package polymarket

import (
	"context"
	"fmt"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// polymarketMinOrderSizeUSDC is the minimum order value enforced by Polymarket.
// Orders below this threshold are rejected by the API with a NET-002 error.
const polymarketMinOrderSizeUSDC = 1.0

func (p *polymarket) PlaceLimitOrder(order prediction.LimitOrder) (*connector.OrderResponse, error) {
	// Exchange-level pre-flight: validate minimum order size before calling the API.
	// For both BUY and SELL, Polymarket requires the USDC value (shares × price) ≥ $1.
	orderValueUSDC := order.Amount.Mul(order.Price)
	minSize := numerical.NewFromFloat(polymarketMinOrderSizeUSDC)
	if orderValueUSDC.LessThan(minSize) {
		return nil, fmt.Errorf(
			"order rejected: value $%.2f is below Polymarket minimum of $%.2f (%.4f shares @ $%.4f)",
			orderValueUSDC.InexactFloat64(),
			polymarketMinOrderSizeUSDC,
			order.Amount.InexactFloat64(),
			order.Price.InexactFloat64(),
		)
	}

	ctx := context.Background()

	resp, err := p.orderManager.PlaceOrder(ctx, order)
	if err != nil {
		return nil, err
	}

	return &connector.OrderResponse{
		OrderID: resp.ID,
		Status:  connector.OrderStatus(resp.Status),
	}, nil
}

func (p *polymarket) PlaceLimitOrders(orders []prediction.LimitOrder) ([]*connector.OrderResponse, error) {
	if len(orders) == 0 {
		return nil, nil
	}

	// Validate all orders upfront before submitting the batch.
	minSize := numerical.NewFromFloat(polymarketMinOrderSizeUSDC)
	for i, order := range orders {
		orderValueUSDC := order.Amount.Mul(order.Price)
		if orderValueUSDC.LessThan(minSize) {
			return nil, fmt.Errorf(
				"order %d rejected: value $%.2f is below Polymarket minimum of $%.2f",
				i, orderValueUSDC.InexactFloat64(), polymarketMinOrderSizeUSDC,
			)
		}
	}

	ctx := context.Background()
	batchResp, err := p.orderManager.PlaceOrders(ctx, orders)
	if err != nil {
		return nil, err
	}

	responses := make([]*connector.OrderResponse, 0, len(batchResp))
	for _, r := range batchResp {
		responses = append(responses, &connector.OrderResponse{
			OrderID: r.ID,
			Status:  connector.OrderStatus(r.Status),
		})
	}
	return responses, nil
}

func (p *polymarket) CancelOrder(orderID string, outcome ...prediction.Outcome) (*connector.CancelResponse, error) {
	ctx := context.Background()

	resp, err := p.orderManager.CancelOrder(ctx, orderID)

	if err != nil {
		return nil, err
	}

	response := &connector.CancelResponse{
		OrderID: orderID,
		Status:  connector.OrderStatus(resp.Status),
	}

	if len(outcome) > 0 {
		response.Pair = outcome[0].Pair.Pair
	}

	return response, nil
}

func (p *polymarket) SubscribeTrades(market prediction.Market) error {
	ctx := context.Background()
	tradeChannel, err := p.clobWebsocket.SubscribeUserTrades(ctx, market)
	if err != nil {
		return fmt.Errorf("failed to subscribe to user trades: %w", err)
	}

	// Process trade events
	go func() {
		defer func() {
			p.appLogger.Info("Unsubscribed from trades for market %s", market.Slug)
		}()

		for tradeEvent := range tradeChannel {
			trade, done := p.parseTrade(market, tradeEvent)
			if done {
				p.appLogger.Error("Failed to parse trade event for market %s, stopping subscription", market.Slug)
				return
			}

			select {
			case p.tradeChannel <- trade:
				// Successfully sent
			default:
				p.appLogger.Warn("Trade channel full for market %s, dropping message", market.Slug)
			}
		}
	}()

	p.appLogger.Info("Subscribed to trades for market %s", market.Slug)
	return nil
}

func (p *polymarket) SubscribeOrders(market prediction.Market) error {
	ctx := context.Background()
	orderChannel, err := p.clobWebsocket.SubscribeUserOrders(ctx, market)
	if err != nil {
		return fmt.Errorf("failed to subscribe to user orders: %w", err)
	}

	// Process order events
	go func() {
		defer func() {
			p.appLogger.Info("Unsubscribed from orders for market %s", market.Slug)
		}()

		for orderEvent := range orderChannel {
			order, done := p.parseOrder(market, orderEvent)
			if done {
				p.appLogger.Error("Failed to parse order event for market %s, stopping subscription", market.Slug)
				return
			}

			select {
			case p.orderChannel <- order:
				// Successfully sent
			default:
				p.appLogger.Warn("Order channel full for market %s, dropping message", market.Slug)
			}
		}
	}()

	p.appLogger.Info("Subscribed to orders for market %s", market.Slug)
	return nil
}

func (p *polymarket) UnsubscribeUserMarket(_ prediction.Market) error {
	// WebSocket unsubscription is handled by closing the goroutine via context cancellation.
	// Individual market unsubscribe is not yet supported by the Polymarket WS SDK.
	return nil
}

func (p *polymarket) GetOutcome(marketID, outcomeID string) prediction.Outcome {
	// Look up the outcome from subscribed markets first.
	for _, m := range p.subscribedMarkets {
		if m.MarketID.String() == marketID {
			for _, o := range m.Outcomes {
				if o.OutcomeID.String() == outcomeID {
					return o
				}
			}
		}
	}
	// Return a minimal outcome with just the IDs populated.
	return prediction.Outcome{
		OutcomeID: prediction.OutcomeID(outcomeID),
	}
}

func (p *polymarket) PlaceMarketOrder(_ portfolio.Pair, _ connector.OrderSide, _ numerical.Decimal) (*connector.OrderResponse, error) {
	return nil, fmt.Errorf("polymarket does not support market orders; use PlaceLimitOrder with FOK time-in-force")
}

func (p *polymarket) GetOpenOrders(_ ...portfolio.Pair) ([]connector.Order, error) {
	ctx := context.Background()
	clobOrders, err := p.orderManager.GetOpenOrders(ctx, "")
	if err != nil {
		return nil, err
	}
	orders := make([]connector.Order, 0, len(clobOrders))
	for _, o := range clobOrders {
		orders = append(orders, mapCLOBOrder(o))
	}
	return orders, nil
}

func (p *polymarket) GetOrderStatus(orderID string, _ ...portfolio.Pair) (*connector.Order, error) {
	ctx := context.Background()
	clobOrder, err := p.orderManager.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	order := mapCLOBOrder(clobOrder)
	return &order, nil
}

func (p *polymarket) GetTradingHistory(_ portfolio.Pair, _ int) ([]connector.Trade, error) {
	// Not implemented — prediction markets use position-based tracking, not trade history.
	return nil, nil
}

func (p *polymarket) FetchRecentTrades(_ portfolio.Pair, _ int) ([]connector.Trade, error) {
	// Not implemented — prediction market trades flow via WebSocket subscriptions.
	return nil, nil
}

func (p *polymarket) GetPositions() ([]prediction.Position, error) {
	// Positions are tracked by the Wisp SDK store via the balance ingestor and
	// order event stream. Direct CLOB API position queries are not available.
	return nil, nil
}

func (p *polymarket) GetPositionsByMarket(_ string) ([]prediction.Position, error) {
	// See GetPositions — positions are tracked at the SDK layer.
	return nil, nil
}

// mapCLOBOrder converts a Polymarket CLOB OrderResponse to the SDK connector.Order type.
func mapCLOBOrder(o clobtypes.OrderResponse) connector.Order {
	price, _ := numerical.NewFromString(o.Price)
	origSize, _ := numerical.NewFromString(o.OriginalSize)
	matched, _ := numerical.NewFromString(o.SizeMatched)

	side := connector.OrderSideBuy
	if o.Side == "SELL" {
		side = connector.OrderSideSell
	}

	return connector.Order{
		ID:           o.ID,
		Side:         side,
		Type:         connector.OrderTypeLimit,
		Status:       connector.OrderStatus(o.Status),
		Quantity:     origSize,
		Price:        price,
		FilledQty:    matched,
		RemainingQty: origSize.Sub(matched),
	}
}
