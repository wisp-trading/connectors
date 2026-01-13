package spot

import (
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/kronos/numerical"
	gateapi "github.com/gate/gateapi-go/v7"
)

// PlaceLimitOrder places a limit order on Gate.io Spot
func (g *gateSpot) PlaceLimitOrder(symbol string, side connector.OrderSide, quantity, price numerical.Decimal) (*connector.OrderResponse, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	client, err := g.spotClient.GetSpotApi()
	if err != nil {
		return nil, fmt.Errorf("failed to get spot API client: %w", err)
	}

	ctx := g.spotClient.GetAPIContext()
	currencyPair := g.formatSymbol(symbol)

	order := gateapi.Order{
		CurrencyPair: currencyPair,
		Side:         convertSide(side),
		Amount:       quantity.String(),
		Price:        price.String(),
		Type:         "limit",
	}

	result, _, err := client.SpotApi.CreateOrder(ctx, order, &gateapi.CreateOrderOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to place limit order: %w", err)
	}

	return &connector.OrderResponse{
		OrderID:   result.Id,
		Symbol:    symbol,
		Status:    g.convertGateOrderStatus(result.Status),
		Side:      side,
		Type:      connector.OrderTypeLimit,
		Quantity:  quantity,
		Price:     price,
		Timestamp: g.timeProvider.Now(),
	}, nil
}

// PlaceMarketOrder places a market order on Gate.io Spot
func (g *gateSpot) PlaceMarketOrder(symbol string, side connector.OrderSide, quantity numerical.Decimal) (*connector.OrderResponse, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	// For market orders, fetch current price and apply slippage
	currentPrice, err := g.FetchPrice(symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch current price: %w", err)
	}

	// Apply slippage
	var limitPrice numerical.Decimal
	if side == connector.OrderSideBuy {
		// Buy higher
		slippage := numerical.NewFromFloat(1.0 + g.config.DefaultSlippage)
		limitPrice = currentPrice.Price.Mul(slippage)
	} else {
		// Sell lower
		slippage := numerical.NewFromFloat(1.0 - g.config.DefaultSlippage)
		limitPrice = currentPrice.Price.Mul(slippage)
	}

	client, err := g.spotClient.GetSpotApi()
	if err != nil {
		return nil, fmt.Errorf("failed to get spot API client: %w", err)
	}

	ctx := g.spotClient.GetAPIContext()
	currencyPair := g.formatSymbol(symbol)

	order := gateapi.Order{
		CurrencyPair: currencyPair,
		Side:         convertSide(side),
		Amount:       quantity.String(),
		Price:        limitPrice.String(),
		Type:         "limit", // Gate.io doesn't have true market orders in spot, use limit with slippage
		TimeInForce:  "ioc",   // Immediate or cancel
	}

	result, _, err := client.SpotApi.CreateOrder(ctx, order, &gateapi.CreateOrderOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to place market order: %w", err)
	}

	return &connector.OrderResponse{
		OrderID:   result.Id,
		Symbol:    symbol,
		Status:    g.convertGateOrderStatus(result.Status),
		Side:      side,
		Type:      connector.OrderTypeMarket,
		Quantity:  quantity,
		Timestamp: g.timeProvider.Now(),
	}, nil
}

// CancelOrder cancels an existing order on Gate.io
func (g *gateSpot) CancelOrder(symbol, orderID string) (*connector.CancelResponse, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	client, err := g.spotClient.GetSpotApi()
	if err != nil {
		return nil, fmt.Errorf("failed to get spot API client: %w", err)
	}

	ctx := g.spotClient.GetAPIContext()
	currencyPair := g.formatSymbol(symbol)

	_, _, err = client.SpotApi.CancelOrder(ctx, orderID, currencyPair, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	return &connector.CancelResponse{
		OrderID:   orderID,
		Symbol:    symbol,
		Status:    connector.OrderStatusCanceled,
		Timestamp: g.timeProvider.Now(),
	}, nil
}

// GetOrderStatus retrieves the status of a specific order
func (g *gateSpot) GetOrderStatus(orderID string) (*connector.Order, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	client, err := g.spotClient.GetSpotApi()
	if err != nil {
		return nil, fmt.Errorf("failed to get spot API client: %w", err)
	}

	ctx := g.spotClient.GetAPIContext()

	// Note: Gate.io requires currency_pair for GetOrder
	// We'll need to iterate through potential pairs or store it
	// For now, we'll try with a common pair pattern
	order, _, err := client.SpotApi.GetOrder(ctx, orderID, "", &gateapi.GetOrderOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to get order status: %w", err)
	}

	connectorOrder := g.convertGateOrderToConnector(&order)
	return &connectorOrder, nil
}

// Helper function to convert connector.OrderSide to Gate.io side string for requests
func convertSide(side connector.OrderSide) string {
	if side == connector.OrderSideBuy {
		return "buy"
	}
	return "sell"
}
