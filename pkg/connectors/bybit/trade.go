package bybit

import (
	"fmt"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (b *bybit) PlaceLimitOrder(symbol string, side connector.OrderSide, quantity, price numerical.Decimal) (*connector.OrderResponse, error) {
	if !b.SupportsTradingOperations() {
		return nil, fmt.Errorf("trading operations not supported")
	}
	return b.trading.PlaceLimitOrder(symbol, side, quantity, price)
}

func (b *bybit) PlaceMarketOrder(symbol string, side connector.OrderSide, quantity numerical.Decimal) (*connector.OrderResponse, error) {
	if !b.SupportsTradingOperations() {
		return nil, fmt.Errorf("trading operations not supported")
	}
	return b.trading.PlaceMarketOrder(symbol, side, quantity)
}

func (b *bybit) CancelOrder(symbol, orderID string) (*connector.CancelResponse, error) {
	if !b.SupportsTradingOperations() {
		return nil, fmt.Errorf("trading operations not supported")
	}
	return b.trading.CancelOrder(symbol, orderID)
}

func (b *bybit) GetOpenOrders() ([]connector.Order, error) {
	return b.trading.GetOpenOrders()
}

func (b *bybit) GetOrderStatus(orderID string) (*connector.Order, error) {
	return b.trading.GetOrderStatus(orderID)
}
