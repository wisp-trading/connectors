package polymarket

import (
	"context"

	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (p *polymarket) PlaceLimitOrder(order prediction.LimitOrder) (*connector.OrderResponse, error) {
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
		return err
	}

	// Process trade events
	go func() {
		for tradeEvent := range tradeChannel {
			trade, done := p.parseTrade(market, tradeEvent)
			if done {
				return
			}
			p.tradeChannel <- trade
		}
	}()

	return nil
}

func (p *polymarket) SubscribeOrders(market prediction.Market) error {
	ctx := context.Background()
	orderChannel, err := p.clobWebsocket.SubscribeUserOrders(ctx, market)
	if err != nil {
		return err
	}

	// Process order events
	go func() {
		for orderEvent := range orderChannel {
			order, done := p.parseOrder(market, orderEvent)
			if done {
				return
			}
			p.orderChannel <- order
		}
	}()

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
