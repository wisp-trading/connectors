package rest

import (
	"fmt"

	hyperliquid "github.com/sonirico/go-hyperliquid"
)

func (t *spotTradingService) PlaceBuyLimitOrder(coin string, size, price float64) (hyperliquid.OrderStatus, error) {
	return t.placeLimitOrder(coin, size, price, true)
}

func (t *spotTradingService) PlaceSellLimitOrder(coin string, size, price float64) (hyperliquid.OrderStatus, error) {
	return t.placeLimitOrder(coin, size, price, false)
}

func (t *spotTradingService) PlaceBuyMarketOrder(coin string, size, slippage float64) (hyperliquid.OrderStatus, error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return hyperliquid.OrderStatus{}, fmt.Errorf("exchange not configured: %w", err)
	}
	return ex.MarketOpen(coin, true, size, nil, slippage, nil, nil)
}

func (t *spotTradingService) PlaceSellMarketOrder(coin string, size, slippage float64) (hyperliquid.OrderStatus, error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return hyperliquid.OrderStatus{}, fmt.Errorf("exchange not configured: %w", err)
	}
	return ex.MarketOpen(coin, false, size, nil, slippage, nil, nil)
}

func (t *spotTradingService) CancelOrderByID(coin string, orderID int64) (*hyperliquid.APIResponse[hyperliquid.CancelOrderResponse], error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return nil, fmt.Errorf("exchange not configured: %w", err)
	}
	return ex.Cancel(coin, orderID)
}

func (t *spotTradingService) PlaceBulkOrders(orders []hyperliquid.CreateOrderRequest) (*hyperliquid.APIResponse[hyperliquid.OrderResponse], error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return nil, fmt.Errorf("exchange not configured: %w", err)
	}
	return ex.BulkOrders(orders, nil)
}

// placeLimitOrder places a limit order via the Go SDK.
// The SDK resolves coin -> asset index internally, including the spot offset.
func (t *spotTradingService) placeLimitOrder(coin string, size, price float64, isBuy bool) (hyperliquid.OrderStatus, error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return hyperliquid.OrderStatus{}, fmt.Errorf("exchange not configured: %w", err)
	}

	req := hyperliquid.CreateOrderRequest{
		Coin:       coin,
		IsBuy:      isBuy,
		Price:      price,
		Size:       size,
		ReduceOnly: false,
		OrderType: hyperliquid.OrderType{
			Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc},
		},
	}

	return ex.Order(req, nil)
}
