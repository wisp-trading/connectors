package hyperliquid

import (
	"fmt"
	"strconv"
	"time"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// PlaceLimitOrder implements connector.OrderExecutor
func (h *hyperliquidSpot) PlaceLimitOrder(pair portfolio.Pair, side connector.OrderSide, quantity, price numerical.Decimal) (*connector.OrderResponse, error) {
	coin := h.normaliseAssetName(pair.Base())
	size, _ := quantity.Float64()
	px, _ := price.Float64()

	var result interface{ }
	var err error

	switch side {
	case connector.OrderSideBuy:
		result, err = h.trading.PlaceBuyLimitOrder(coin, size, px)
	case connector.OrderSideSell:
		result, err = h.trading.PlaceSellLimitOrder(coin, size, px)
	default:
		return nil, fmt.Errorf("unknown order side: %s", side)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to place %s limit order: %w", side, err)
	}

	return &connector.OrderResponse{
		OrderID:   fmt.Sprintf("%v", result),
		Symbol:    coin,
		Side:      side,
		Type:      connector.OrderTypeLimit,
		Price:     price,
		Quantity:  quantity,
		Status:    connector.OrderStatusNew,
		Timestamp: time.Now(),
	}, nil
}

// PlaceMarketOrder implements connector.OrderExecutor
func (h *hyperliquidSpot) PlaceMarketOrder(pair portfolio.Pair, side connector.OrderSide, quantity numerical.Decimal) (*connector.OrderResponse, error) {
	coin := h.normaliseAssetName(pair.Base())
	size, _ := quantity.Float64()

	var err error

	switch side {
	case connector.OrderSideBuy:
		_, err = h.trading.PlaceBuyMarketOrder(coin, size, h.config.DefaultSlippage)
	case connector.OrderSideSell:
		_, err = h.trading.PlaceSellMarketOrder(coin, size, h.config.DefaultSlippage)
	default:
		return nil, fmt.Errorf("unknown order side: %s", side)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to place %s market order: %w", side, err)
	}

	return &connector.OrderResponse{
		OrderID:   fmt.Sprintf("%d", h.timeProvider.Now().UnixNano()),
		Symbol:    coin,
		Side:      side,
		Type:      connector.OrderTypeMarket,
		Quantity:  quantity,
		Status:    connector.OrderStatusFilled,
		Timestamp: time.Now(),
	}, nil
}

// CancelOrder implements connector.OrderExecutor
func (h *hyperliquidSpot) CancelOrder(orderID string, pair ...portfolio.Pair) (*connector.CancelResponse, error) {
	if len(pair) == 0 {
		return nil, fmt.Errorf("pair is required for Hyperliquid spot cancel")
	}

	coin := h.normaliseAssetName(pair[0].Base())
	oid, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid order ID: %w", err)
	}

	_, err = h.trading.CancelOrderByID(coin, oid)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	return &connector.CancelResponse{
		OrderID: orderID,
		Status:  "cancelled",
	}, nil
}

// GetOpenOrders implements connector.OrderExecutor
func (h *hyperliquidSpot) GetOpenOrders(pair ...portfolio.Pair) ([]connector.Order, error) {
	return nil, fmt.Errorf("GetOpenOrders not yet implemented for Hyperliquid spot")
}

// GetOrderStatus implements connector.OrderExecutor
func (h *hyperliquidSpot) GetOrderStatus(orderID string, pair ...portfolio.Pair) (*connector.Order, error) {
	return nil, fmt.Errorf("GetOrderStatus not yet implemented for Hyperliquid spot")
}
