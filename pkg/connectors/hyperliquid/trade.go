package hyperliquid

import (
	"fmt"
	"strconv"
	"time"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// PlaceLimitOrder places a limit order on Hyperliquid
func (h *hyperliquid) PlaceLimitOrder(symbol string, side connector.OrderSide, quantity, price numerical.Decimal) (*connector.OrderResponse, error) {
	if !h.SupportsTradingOperations() {
		return nil, fmt.Errorf("trading operations not supported")
	}

	var result interface{}
	var err error

	if side == connector.OrderSideBuy {
		result, err = h.trading.PlaceBuyLimitOrder(symbol, quantity.InexactFloat64(), price.InexactFloat64())
	} else {
		result, err = h.trading.PlaceSellLimitOrder(symbol, quantity.InexactFloat64(), price.InexactFloat64())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to place %s limit order: %w", side, err)
	}

	return &connector.OrderResponse{
		OrderID:   h.extractOrderID(result),
		Symbol:    symbol,
		Status:    connector.OrderStatusNew,
		Side:      side,
		Type:      connector.OrderTypeLimit,
		Quantity:  quantity,
		Price:     price,
		Timestamp: h.timeProvider.Now(),
	}, nil
}

// PlaceMarketOrder places a market order on Hyperliquid
func (h *hyperliquid) PlaceMarketOrder(symbol string, side connector.OrderSide, quantity numerical.Decimal) (*connector.OrderResponse, error) {
	if !h.SupportsTradingOperations() {
		return nil, fmt.Errorf("trading operations not supported")
	}

	var result interface{}
	var err error

	if side == connector.OrderSideBuy {
		result, err = h.trading.PlaceBuyMarketOrder(symbol, quantity.InexactFloat64(), h.config.DefaultSlippage)
	} else {
		result, err = h.trading.PlaceSellMarketOrder(symbol, quantity.InexactFloat64(), h.config.DefaultSlippage)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to place %s market order: %w", side, err)
	}

	return &connector.OrderResponse{
		OrderID:   h.extractOrderID(result),
		Symbol:    symbol,
		Status:    connector.OrderStatusNew,
		Side:      side,
		Type:      connector.OrderTypeMarket,
		Quantity:  quantity,
		Timestamp: h.timeProvider.Now(),
	}, nil
}

// CancelOrder cancels an existing order on Hyperliquid
func (h *hyperliquid) CancelOrder(symbol, orderID string) (*connector.CancelResponse, error) {
	if !h.SupportsTradingOperations() {
		return nil, fmt.Errorf("trading operations not supported")
	}

	oid, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid order ID format: %w", err)
	}

	_, err = h.trading.CancelOrderByID(symbol, oid)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	return &connector.CancelResponse{
		OrderID:   orderID,
		Symbol:    symbol,
		Status:    connector.OrderStatusCanceled,
		Timestamp: h.timeProvider.Now(),
	}, nil
}

// GetOpenOrders retrieves current open orders
func (h *hyperliquid) GetOpenOrders() ([]connector.Order, error) {
	orders, err := h.marketData.GetOpenOrders(h.config.AccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders: %w", err)
	}

	var connectorOrders []connector.Order
	for _, order := range orders {
		connectorOrder := connector.Order{
			ID:        fmt.Sprintf("%d", order.Oid),
			Symbol:    order.Coin,
			Side:      connector.FromString(order.Side),
			Quantity:  numerical.NewFromFloat(order.Size),
			Price:     numerical.NewFromFloat(order.LimitPx),
			CreatedAt: time.Unix(order.Timestamp/1000, 0),
		}
		connectorOrders = append(connectorOrders, connectorOrder)
	}

	return connectorOrders, nil
}

// GetOrderStatus retrieves the status of a specific order
func (h *hyperliquid) GetOrderStatus(orderID string) (*connector.Order, error) {
	return nil, fmt.Errorf("GetOrderStatus not yet implemented for Hyperliquid")
}
