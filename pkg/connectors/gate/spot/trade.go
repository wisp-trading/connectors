package spot

import (
	"fmt"

	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// PlaceLimitOrder places a limit order on Gate.io Spot
func (g *gateSpot) PlaceLimitOrder(pair portfolio.Pair, side connector.OrderSide, quantity, price numerical.Decimal) (*connector.OrderResponse, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	client, err := g.spotClient.GetSpotApi()
	if err != nil {
		return nil, fmt.Errorf("failed to get spot API client: %w", err)
	}

	ctx := g.spotClient.GetAPIContext()
	currencyPair := g.GetSpotSymbol(pair)

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
		Symbol:    pair.Symbol(),
		Status:    g.convertGateOrderStatus(result.Status),
		Side:      side,
		Type:      connector.OrderTypeLimit,
		Quantity:  quantity,
		Price:     price,
		Timestamp: g.timeProvider.Now(),
	}, nil
}

// PlaceMarketOrder places a market order on Gate.io Spot
func (g *gateSpot) PlaceMarketOrder(pair portfolio.Pair, side connector.OrderSide, quantity numerical.Decimal) (*connector.OrderResponse, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	// For market orders, fetch current price and apply slippage
	currentPrice, err := g.FetchPrice(pair)
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
	currencyPair := g.GetSpotSymbol(pair)

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
		Symbol:    pair.Symbol(),
		Status:    g.convertGateOrderStatus(result.Status),
		Side:      side,
		Type:      connector.OrderTypeMarket,
		Quantity:  quantity,
		Timestamp: g.timeProvider.Now(),
	}, nil
}

// CancelOrder cancels an existing order on Gate.io
func (g *gateSpot) CancelOrder(orderID string, pair ...portfolio.Pair) (*connector.CancelResponse, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	client, err := g.spotClient.GetSpotApi()
	if err != nil {
		return nil, fmt.Errorf("failed to get spot API client: %w", err)
	}

	if len(pair) == 0 {
		return nil, fmt.Errorf("pair is required to cancel order")
	}

	ctx := g.spotClient.GetAPIContext()
	currencyPair := g.GetSpotSymbol(pair[0])

	_, _, err = client.SpotApi.CancelOrder(ctx, orderID, currencyPair, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	return &connector.CancelResponse{
		Pair:      pair[0],
		OrderID:   orderID,
		Status:    connector.OrderStatusCanceled,
		Timestamp: g.timeProvider.Now(),
	}, nil
}

// GetOrderStatus retrieves the status of a specific order
func (g *gateSpot) GetOrderStatus(orderID string, pair ...portfolio.Pair) (*connector.Order, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	client, err := g.spotClient.GetSpotApi()
	if err != nil {
		return nil, fmt.Errorf("failed to get spot API client: %w", err)
	}

	ctx := g.spotClient.GetAPIContext()

	if len(pair) == 0 {
		return nil, fmt.Errorf("pair is required to get order status")
	}

	currencyPair := g.GetSpotSymbol(pair[0])

	order, _, err := client.SpotApi.GetOrder(ctx, orderID, currencyPair, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get order status: %w", err)
	}

	connectorOrder := g.convertGateOrderToConnector(&order)
	return &connectorOrder, nil
}
