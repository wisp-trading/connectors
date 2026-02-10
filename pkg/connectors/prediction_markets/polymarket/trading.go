package polymarket

import (
	"context"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (p *polymarket) PlaceLimitOrder(order prediction.LimitOrder) (*connector.OrderResponse, error) {
	ctx := context.Background()

	// Validate order using validator tags
	//if err := ValidateStruct(order); err != nil {
	//	return nil, fmt.Errorf("invalid order: %w", err)
	//}

	err := p.clobClient.PlaceOrder(ctx, order)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (p *polymarket) PlaceLimitOrders(orders []prediction.LimitOrder) ([]*connector.OrderResponse, error) {
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

func (p *polymarket) CancelOrder(orderID string, pair ...portfolio.Pair) (*connector.CancelResponse, error) {
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

func (p *polymarket) GetSettlementHistory(limit int) ([]prediction.Settlement, error) {
	//TODO implement me
	panic("implement me")
}
