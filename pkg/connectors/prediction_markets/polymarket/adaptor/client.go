package adaptor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket"
)

const (
	// API endpoints
	ordersEndpoint        = "/orders"
	cancelOrderEndpoint   = "/orders" // DELETE /orders/:orderID
	cancelAllEndpoint     = "/orders" // DELETE /orders with params
	getOrderEndpoint      = "/orders" // GET /orders/:orderID
	getOpenOrdersEndpoint = "/orders" // GET /orders with params

	// HTTP timeouts
	defaultTimeout = 30 * time.Second
)

// PolymarketClient wraps Polymarket CLOB API with lazy configuration
type PolymarketClient interface {
	Configure(config *polymarket.Config) error
	IsConfigured() bool

	// Trading (requires authentication)
	PlaceOrder(ctx context.Context, order OrderRequest) (*OrderResponse, error)
	CancelOrder(ctx context.Context, orderID string) (*CancelOrderResponse, error)
	CancelAllOrders(ctx context.Context, req CancelAllOrdersRequest) (*CancelAllOrdersResponse, error)
	GetOrder(ctx context.Context, orderID string) (*OrderResponse, error)
	GetOpenOrders(ctx context.Context, assetID string) ([]OrderResponse, error)
}

// polymarketClient implementation
type polymarketClient struct {
	baseURL    string
	apiKey     string
	apiSecret  string
	passphrase string
	signer     *OrderSigner
	httpClient *http.Client
	configured bool
	mu         sync.RWMutex
}

// NewPolymarketClient creates an unconfigured Polymarket client
func NewPolymarketClient() PolymarketClient {
	return &polymarketClient{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		configured: false,
	}
}

// Configure sets up the client with runtime config
func (c *polymarketClient) Configure(config *polymarket.Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.configured {
		return fmt.Errorf("client already configured")
	}

	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Create order signer
	signer, err := NewOrderSigner(config.PrivateKey, config.ChainID)
	if err != nil {
		return fmt.Errorf("failed to create order signer: %w", err)
	}

	c.baseURL = config.BaseURL
	c.apiKey = config.APIKey
	c.apiSecret = config.APISecret
	c.passphrase = config.Passphrase
	c.signer = signer
	c.configured = true

	return nil
}

// IsConfigured returns whether the client is configured
func (c *polymarketClient) IsConfigured() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.configured
}

// PlaceOrder places an order on Polymarket
func (c *polymarketClient) PlaceOrder(ctx context.Context, order OrderRequest) (*OrderResponse, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("client not configured")
	}

	// Validate order using validator tags
	if err := ValidateStruct(order); err != nil {
		return nil, fmt.Errorf("invalid order: %w", err)
	}

	// Sign the order
	orderToSign := Order{
		Maker:         order.Maker,
		Signer:        order.Signer,
		Taker:         order.Taker,
		TokenID:       order.TokenID,
		MakerAmount:   order.MakerAmount,
		TakerAmount:   order.TakerAmount,
		Side:          order.Side,
		FeeRateBps:    order.FeeRateBps,
		Nonce:         order.Nonce,
		SignatureType: order.SignatureType,
		Expiration:    order.Expiration,
	}

	orderToSign.Salt = c.signer.GenerateSalt()

	signature, err := c.signer.SignOrder(orderToSign)
	if err != nil {
		return nil, fmt.Errorf("failed to sign order: %w", err)
	}
	order.Signature = signature
	order.Salt = orderToSign.Salt

	// Make HTTP request
	var response OrderResponse
	if err := c.doRequest(ctx, "POST", ordersEndpoint, order, &response); err != nil {
		return nil, fmt.Errorf("failed to place order: %w", err)
	}

	return &response, nil
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
func (c *polymarketClient) GetOrder(ctx context.Context, orderID string) (*OrderResponse, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("client not configured")
	}

	if orderID == "" {
		return nil, fmt.Errorf("orderID is required")
	}

	endpoint := fmt.Sprintf("%s/%s", getOrderEndpoint, orderID)

	var response OrderResponse
	if err := c.doRequest(ctx, "GET", endpoint, nil, &response); err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return &response, nil
}

// GetOpenOrders retrieves all open orders for an asset
func (c *polymarketClient) GetOpenOrders(ctx context.Context, assetID string) ([]OrderResponse, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("client not configured")
	}

	endpoint := getOpenOrdersEndpoint
	if assetID != "" {
		endpoint = fmt.Sprintf("%s?assetID=%s", endpoint, assetID)
	}

	var response []OrderResponse
	if err := c.doRequest(ctx, "GET", endpoint, nil, &response); err != nil {
		return nil, fmt.Errorf("failed to get open orders: %w", err)
	}

	return response, nil
}

// doRequest performs an HTTP request with authentication
// doRequest performs an HTTP request with L2 authentication
func (c *polymarketClient) doRequest(ctx context.Context, method, endpoint string, body interface{}, result interface{}) error {
	c.mu.RLock()
	baseURL := c.baseURL
	apiKey := c.apiKey
	apiSecret := c.apiSecret
	passphrase := c.passphrase
	signer := c.signer
	c.mu.RUnlock()

	// Build URL
	url := baseURL + endpoint

	// Prepare request body
	var reqBody io.Reader
	var bodyStr string
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyStr = string(jsonData)
		reqBody = bytes.NewBuffer(jsonData)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Add L1 authentication headers
	req.Header.Set("POLY_API_KEY", apiKey)
	req.Header.Set("POLY_SECRET", apiSecret)
	req.Header.Set("POLY_PASSPHRASE", passphrase)

	// Add L2 authentication headers
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	// Message format: timestamp + method + path + body
	message := timestamp + method + endpoint + bodyStr
	signature, err := signer.SignMessage(message)
	if err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	req.Header.Set("POLY_SIGNATURE", signature)
	req.Header.Set("POLY_TIMESTAMP", timestamp)
	req.Header.Set("POLY_ADDRESS", signer.GetAddress())

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			return fmt.Errorf("API error (status %d): %s - %s", resp.StatusCode, errResp.Error, errResp.Message)
		}
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}
