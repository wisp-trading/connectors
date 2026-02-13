package clob

import (
	"context"
	"fmt"

	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
)

// PlaceOrder places an order on Polymarket
func (c *polymarketClient) PlaceOrder(ctx context.Context, limitOrder prediction.LimitOrder) error {
	if !c.IsConfigured() {
		return fmt.Errorf("client not configured")
	}

	order := OrderRequest{}.FromLimitOrder(limitOrder)

	// You are ALWAYS the maker when placing a limit order
	order.Maker = c.publicAddress
	order.Signer = c.publicAddress

	order.Taker = "0x0000000000000000000000000000000000000000"

	order.SignatureType = 712
	order.Nonce = "0"
	order.FeeRateBps = "0"

	signedOrder, err := c.SignOrder(order)
	if err != nil {
		return fmt.Errorf("failed to sign order: %w", err)
	}

	// Make HTTP request
	var response interface{}
	if err := c.doRequest(ctx, "POST", ordersEndpoint, signedOrder, &response); err != nil {
		return fmt.Errorf("failed to place order: %w", err)
	}

	return nil
}

// CancelOrder cancels an order by ID
func (c *polymarketClient) CancelOrder(ctx context.Context, orderID string) (*CancelOrderResponse, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("client not configured")
	}

	if orderID == "" {
		return nil, fmt.Errorf("orderID is required")
	}

	endpoint := fmt.Sprintf("%s/%s", cancelOrderEndpoint, orderID)

	var response CancelOrderResponse
	if err := c.doRequest(ctx, "DELETE", endpoint, nil, &response); err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	return &response, nil
}

// CancelAllOrders cancels all orders for a market or asset
func (c *polymarketClient) CancelAllOrders(ctx context.Context, req CancelAllOrdersRequest) (*CancelAllOrdersResponse, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("client not configured")
	}

	// Validate request using validator tags
	if err := ValidateStruct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	var response CancelAllOrdersResponse
	if err := c.doRequest(ctx, "DELETE", cancelAllEndpoint, req, &response); err != nil {
		return nil, fmt.Errorf("failed to cancel all orders: %w", err)
	}

	return &response, nil
}

// GetOrder retrieves an order by ID
func (c *polymarketClient) GetOrder(ctx context.Context, orderID string) error {
	if !c.IsConfigured() {
		return fmt.Errorf("client not configured")
	}

	if orderID == "" {
		return fmt.Errorf("orderID is required")
	}

	endpoint := fmt.Sprintf("%s/%s", getOrderEndpoint, orderID)

	var response interface{}
	if err := c.doRequest(ctx, "GET", endpoint, nil, &response); err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	return nil
}

// GetOpenOrders retrieves all open orders for an asset
func (c *polymarketClient) GetOpenOrders(ctx context.Context, assetID string) error {
	if !c.IsConfigured() {
		return fmt.Errorf("client not configured")
	}

	endpoint := getOpenOrdersEndpoint
	if assetID != "" {
		endpoint = fmt.Sprintf("%s?assetID=%s", endpoint, assetID)
	}

	var response []interface{}
	if err := c.doRequest(ctx, "GET", endpoint, nil, &response); err != nil {
		return fmt.Errorf("failed to get open orders: %w", err)
	}

	return nil
}
