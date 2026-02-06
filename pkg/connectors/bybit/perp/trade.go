package perp

import (
	"context"
	"fmt"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (b *bybit) PlaceLimitOrder(pair portfolio.Pair, side connector.OrderSide, quantity, price numerical.Decimal) (*connector.OrderResponse, error) {
	client, err := b.client.GetClient()

	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
	}

	params := map[string]interface{}{
		"category":    "linear",
		"symbol":      b.GetPerpSymbol(pair),
		"side":        string(side),
		"orderType":   "Limit",
		"qty":         quantity.String(),
		"price":       price.String(),
		"timeInForce": "GTC",
	}

	result, err := client.NewUtaBybitServiceWithParams(params).PlaceOrder(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to place limit order: %w", err)
	}

	var orderID string
	if result != nil && result.Result != nil {
		if ordData, ok := result.Result.(map[string]interface{}); ok {
			if id, ok := ordData["orderId"].(string); ok {
				orderID = id
			}
		}
	}

	return &connector.OrderResponse{
		OrderID:   orderID,
		Symbol:    pair.Symbol(),
		Status:    connector.OrderStatusNew,
		Side:      side,
		Type:      connector.OrderTypeLimit,
		Quantity:  quantity,
		Price:     price,
		Timestamp: b.timeProvider.Now(),
	}, nil
}

func (b *bybit) PlaceMarketOrder(pair portfolio.Pair, side connector.OrderSide, quantity numerical.Decimal) (*connector.OrderResponse, error) {
	client, err := b.client.GetClient()

	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
	}

	params := map[string]interface{}{
		"category":  "linear",
		"symbol":    b.GetPerpSymbol(pair),
		"side":      string(side),
		"orderType": "Market",
		"qty":       quantity.String(),
	}

	result, err := client.NewUtaBybitServiceWithParams(params).PlaceOrder(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to place market order: %w", err)
	}

	var orderID string
	if result != nil && result.Result != nil {
		if ordData, ok := result.Result.(map[string]interface{}); ok {
			if id, ok := ordData["orderId"].(string); ok {
				orderID = id
			}
		}
	}

	return &connector.OrderResponse{
		OrderID:   orderID,
		Symbol:    pair.Symbol(),
		Status:    connector.OrderStatusNew,
		Side:      side,
		Type:      connector.OrderTypeMarket,
		Quantity:  quantity,
		Timestamp: b.timeProvider.Now(),
	}, nil
}

func (b *bybit) CancelOrder(pair portfolio.Pair, orderID string) (*connector.CancelResponse, error) {
	client, err := b.client.GetClient()

	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
	}

	params := map[string]interface{}{
		"category": "linear",
		"symbol":   b.GetPerpSymbol(pair),
		"orderId":  orderID,
	}

	_, err = client.NewUtaBybitServiceWithParams(params).CancelOrder(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	return &connector.CancelResponse{

		OrderID:   orderID,
		Timestamp: b.timeProvider.Now(),
	}, nil
}

func (b *bybit) GetOpenOrders() ([]connector.Order, error) {
	client, err := b.client.GetClient()

	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
	}

	params := map[string]interface{}{
		"category": "linear",
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetOpenOrders(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders: %w", err)
	}

	var orders []connector.Order
	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if list, ok := resultData["list"].([]interface{}); ok {
				for _, item := range list {
					if orderData, ok := item.(map[string]interface{}); ok {
						order := b.parseOrder(orderData)
						orders = append(orders, order)
					}
				}
			}
		}
	}

	return orders, nil
}

func (b *bybit) GetOrderStatus(orderID string, pair ...portfolio.Pair) (*connector.Order, error) {
	client, err := b.client.GetClient()

	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
	}

	params := map[string]interface{}{
		"category": "linear",
		"orderId":  orderID,
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetOrderHistory(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get order status: %w", err)
	}

	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if list, ok := resultData["list"].([]interface{}); ok {
				if len(list) > 0 {
					if orderData, ok := list[0].(map[string]interface{}); ok {
						order := b.parseOrder(orderData)
						return &order, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("order not found")
}

func (b *bybit) parseTrade(data map[string]interface{}, pair portfolio.Pair) connector.Trade {
	trade := connector.Trade{
		Pair:      pair,
		Timestamp: b.timeProvider.Now(),
	}

	if side, ok := data["side"].(string); ok {
		trade.Side = connector.OrderSide(side)
	}
	if price, ok := data["price"].(string); ok {
		if val, err := numerical.NewFromString(price); err == nil {
			trade.Price = val
		}
	}
	if quantity, ok := data["size"].(string); ok {
		if val, err := numerical.NewFromString(quantity); err == nil {
			trade.Quantity = val
		}
	}

	return trade
}
