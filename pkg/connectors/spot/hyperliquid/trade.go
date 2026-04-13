package hyperliquid

import (
	"fmt"
	"strconv"
	"time"

	hyperliquid "github.com/sonirico/go-hyperliquid"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// PlaceLimitOrder implements connector.OrderExecutor
func (h *hyperliquidSpot) PlaceLimitOrder(pair portfolio.Pair, side connector.OrderSide, quantity, price numerical.Decimal) (*connector.OrderResponse, error) {
	coin := h.normaliseAssetName(pair.Base())
	size, _ := quantity.Float64()
	px, _ := price.Float64()

	var status hyperliquid.OrderStatus
	var err error

	switch side {
	case connector.OrderSideBuy:
		status, err = h.trading.PlaceBuyLimitOrder(coin, size, px)
	case connector.OrderSideSell:
		status, err = h.trading.PlaceSellLimitOrder(coin, size, px)
	default:
		return nil, fmt.Errorf("unknown order side: %s", side)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to place %s limit order: %w", side, err)
	}

	orderID, orderStatus := extractOrderInfo(status)

	return &connector.OrderResponse{
		OrderID:   orderID,
		Symbol:    coin,
		Side:      side,
		Type:      connector.OrderTypeLimit,
		Price:     price,
		Quantity:  quantity,
		Status:    orderStatus,
		Timestamp: time.Now(),
	}, nil
}

// extractOrderInfo pulls the order ID and status from the Hyperliquid OrderStatus response.
func extractOrderInfo(status hyperliquid.OrderStatus) (string, connector.OrderStatus) {
	if status.Resting != nil {
		return fmt.Sprintf("%d", status.Resting.Oid), connector.OrderStatusOpen
	}
	if status.Filled != nil {
		return fmt.Sprintf("%d", status.Filled.Oid), connector.OrderStatusFilled
	}
	if status.Error != nil {
		return "", connector.OrderStatusRejected
	}
	return "", connector.OrderStatusPending
}

// PlaceMarketOrder implements connector.OrderExecutor
func (h *hyperliquidSpot) PlaceMarketOrder(pair portfolio.Pair, side connector.OrderSide, quantity numerical.Decimal) (*connector.OrderResponse, error) {
	coin := h.normaliseAssetName(pair.Base())
	size, _ := quantity.Float64()

	var status hyperliquid.OrderStatus
	var err error

	switch side {
	case connector.OrderSideBuy:
		status, err = h.trading.PlaceBuyMarketOrder(coin, size, h.config.DefaultSlippage)
	case connector.OrderSideSell:
		status, err = h.trading.PlaceSellMarketOrder(coin, size, h.config.DefaultSlippage)
	default:
		return nil, fmt.Errorf("unknown order side: %s", side)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to place %s market order: %w", side, err)
	}

	orderID, orderStatus := extractOrderInfo(status)

	return &connector.OrderResponse{
		OrderID:   orderID,
		Symbol:    coin,
		Side:      side,
		Type:      connector.OrderTypeMarket,
		Quantity:  quantity,
		Status:    orderStatus,
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

	if err := h.trading.CancelOrderByID(coin, oid); err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	return &connector.CancelResponse{
		OrderID: orderID,
		Status:  "cancelled",
	}, nil
}

// GetOpenOrders implements connector.OrderExecutor
func (h *hyperliquidSpot) GetOpenOrders(pair ...portfolio.Pair) ([]connector.Order, error) {
	orders, err := h.marketData.GetOpenOrders(h.effectiveAddress())
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders: %w", err)
	}

	// Build a set of filter coins from the pair arguments.
	// If no pairs are given, all orders are returned.
	filterCoins := make(map[string]bool, len(pair))
	for _, p := range pair {
		filterCoins[h.normaliseAssetName(p.Base())] = true
	}

	connectorOrders := make([]connector.Order, 0, len(orders))
	for _, order := range orders {
		if len(filterCoins) > 0 && !filterCoins[order.Coin] {
			continue
		}
		connectorOrders = append(connectorOrders, connector.Order{
			ID:        fmt.Sprintf("%d", order.Oid),
			Pair:      h.coinToPair(order.Coin),
			Side:      connector.FromString(order.Side),
			Quantity:  numerical.NewFromFloat(order.Size),
			Price:     numerical.NewFromFloat(order.LimitPx),
			CreatedAt: time.Unix(order.Timestamp/1000, 0),
		})
	}

	return connectorOrders, nil
}

// GetOrderStatus implements connector.OrderExecutor
func (h *hyperliquidSpot) GetOrderStatus(orderID string, pair ...portfolio.Pair) (*connector.Order, error) {
	oid, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid order ID %q: %w", orderID, err)
	}

	order, err := h.marketData.GetOrderByOid(h.effectiveAddress(), oid)
	if err != nil {
		return nil, fmt.Errorf("failed to get order status: %w", err)
	}

	return &connector.Order{
		ID:        fmt.Sprintf("%d", order.Oid),
		Pair:      h.coinToPair(order.Coin),
		Side:      connector.FromString(order.Side),
		Quantity:  numerical.NewFromFloat(order.Size),
		Price:     numerical.NewFromFloat(order.LimitPx),
		CreatedAt: time.Unix(order.Timestamp/1000, 0),
	}, nil
}
