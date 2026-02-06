package paradex

import (
	"fmt"
	"time"

	"github.com/trishtzy/go-paradex/models"
	"github.com/wisp-trading/connectors/pkg/connectors/paradex/requests"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (p *paradex) PlaceLimitOrder(pair portfolio.Pair, side connector.OrderSide, quantity, price numerical.Decimal) (*connector.OrderResponse, error) {
	market := p.GetPerpSymbol(pair)

	orderReq := requests.PlaceOrderParams{
		Market:    market,
		Side:      string(side), // "BUY" or "SELL"
		Size:      quantity.String(),
		Price:     price.String(),
		OrderType: "LIMIT",
		ClientID:  "", // Optional
	}

	resp, err := p.paradexService.PlaceOrder(p.ctx, orderReq)
	if err != nil {
		return nil, err
	}

	return &connector.OrderResponse{
		OrderID:   resp.ID,
		Symbol:    resp.Market,
		Status:    connector.OrderStatusNew,
		Side:      side,
		Type:      connector.OrderTypeLimit,
		Quantity:  quantity,
		Price:     price,
		FilledQty: numerical.Zero(), // Update if fill info is available
		Timestamp: time.Now(),       // Or resp.CreatedAt if available
	}, nil
}

func (p *paradex) PlaceMarketOrder(pair portfolio.Pair, side connector.OrderSide, quantity numerical.Decimal) (*connector.OrderResponse, error) {
	market := p.GetPerpSymbol(pair)

	orderReq := requests.PlaceOrderParams{
		Market:    market,
		Side:      string(side),
		Size:      quantity.String(),
		OrderType: "MARKET",
		Price:     "", // Empty for MARKET orders - will be omitted by PlaceOrder
		ClientID:  "", // Optional
	}

	resp, err := p.paradexService.PlaceOrder(p.ctx, orderReq)
	if err != nil {
		return nil, err
	}

	return &connector.OrderResponse{
		OrderID:   resp.ID,
		Symbol:    resp.Market,
		Status:    connector.OrderStatusNew,
		Side:      side,
		Type:      connector.OrderTypeMarket,
		Quantity:  quantity,
		FilledQty: numerical.Zero(),
		Timestamp: time.Now(),
	}, nil
}

func (p *paradex) CancelOrder(orderID string, pair ...portfolio.Pair) (*connector.CancelResponse, error) {
	var symbol string

	if len(pair) > 0 {
		symbol = pair[0].Symbol()
	}

	p.tradingLogger.OrderLifecycle(
		"Cancelling paradex order %s for symbol %s",
		orderID,
		symbol,
	)

	err := p.paradexService.CancelOrder(p.ctx, orderID)

	if err != nil {
		p.tradingLogger.OrderLifecycle("Failed to cancel order %s for symbol %s: %v", orderID, symbol, err)
		return nil, err
	}

	cancelResponse := &connector.CancelResponse{
		OrderID:   orderID,
		Status:    connector.OrderCancellationRequested,
		Timestamp: p.timeProvider.Now(),
	}

	if len(pair) > 0 {
		cancelResponse.Pair = pair[0]
	}

	p.tradingLogger.OrderLifecycle("Successfully requested cancellation for order %s on symbol %s", orderID, symbol)

	return cancelResponse, nil
}

func (p *paradex) GetOpenOrders(pair ...portfolio.Pair) ([]connector.Order, error) {
	var paradexOrders []*models.ResponsesOrderResp
	var err error

	paradexOrders, err = p.paradexService.FetchOpenOrders(pair...)

	if err != nil {
		return nil, fmt.Errorf("failed to get open orders from paradex: %w", err)
	}

	// Convert paradex orders to connector format
	orders := make([]connector.Order, 0, len(paradexOrders))

	for _, paradexOrder := range paradexOrders {
		if paradexOrder == nil {
			continue
		}

		convertedOrder := p.convertParadexOrder(paradexOrder)
		orders = append(orders, convertedOrder)
	}

	p.appLogger.Debug("Retrieved %d open orders from paradex", len(orders))
	return orders, nil
}

func (p *paradex) GetOrderStatus(orderID string, pair ...portfolio.Pair) (*connector.Order, error) {
	order, err := p.paradexService.GetOrder(p.ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order == nil {
		return nil, fmt.Errorf("order %s not found", orderID)
	}

	// Convert paradex order to connector format
	convertedOrder := p.convertParadexOrder(order)

	return &convertedOrder, nil
}
