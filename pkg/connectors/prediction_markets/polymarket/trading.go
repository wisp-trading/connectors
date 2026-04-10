package polymarket

import (
	"context"
	"fmt"

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
	//TODO implement me
	panic("implement me")
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

func (p *polymarket) UnsubscribeUserMarket(market prediction.Market) error {
	//TODO implement me
	panic("implement me")
}

func (p *polymarket) GetOutcome(marketID, outcomeID string) prediction.Outcome {
	//TODO implement me
	panic("implement me")
}

func (p *polymarket) PlaceMarketOrder(pair portfolio.Pair, side connector.OrderSide, quantity numerical.Decimal) (*connector.OrderResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (p *polymarket) GetOpenOrders(pair ...portfolio.Pair) ([]connector.Order, error) {
	//TODO implement me
	panic("implement me")
}

func (p *polymarket) GetOrderStatus(orderID string, pair ...portfolio.Pair) (*connector.Order, error) {
	//TODO implement me
	panic("implement me")
}

func (p *polymarket) GetTradingHistory(pair portfolio.Pair, limit int) ([]connector.Trade, error) {
	//TODO implement me
	panic("implement me")
}

func (p *polymarket) FetchRecentTrades(pair portfolio.Pair, limit int) ([]connector.Trade, error) {
	//TODO implement me
	panic("implement me")
}

func (p *polymarket) GetPositions() ([]prediction.Position, error) {
	//TODO implement me
	panic("implement me")
}

func (p *polymarket) GetPositionsByMarket(marketID string) ([]prediction.Position, error) {
	//TODO implement me
	panic("implement me")
}
