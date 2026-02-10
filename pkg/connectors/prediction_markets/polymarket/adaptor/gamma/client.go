package gamma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor"
)

const (
	baseUrl           = "https://gamma-api.polymarket.com"
	getMarketEndpoint = "/markets?slug=" // GET market by slug
	
	// HTTP timeouts
	defaultTimeout = 30 * time.Second
)

// gammaClient implementation
type gammaClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewPolymarketClient creates an unconfigured Polymarket client
func NewPolymarketClient() GammaClient {
	return &gammaClient{
		baseURL: baseUrl,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// doRequest performs an HTTP request with L2 authentication
func (c *gammaClient) doRequest(ctx context.Context, method, endpoint string, body interface{}, result interface{}) error {
	baseURL := c.baseURL

	// Build URL
	url := baseURL + endpoint

	// Prepare request body
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
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
		var errResp adaptor.ErrorResponse
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
