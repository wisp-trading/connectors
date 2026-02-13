package clob

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/types"
)

func (c *polymarketClient) doRequest(ctx context.Context, method, endpoint string, body interface{}, result interface{}) error {
	c.mu.RLock()
	baseURL := c.baseURL
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

	// L2 HMAC Authentication
	timestamp := time.Now().Unix()

	// Create HMAC-SHA256 signature
	signature, err := c.signL2Request(timestamp, method, endpoint, bodyStr)

	// Set auth headers
	req.Header["POLY_ADDRESS"] = []string{c.signerAddress}
	req.Header["POLY_SIGNATURE"] = []string{signature}
	req.Header["POLY_TIMESTAMP"] = []string{fmt.Sprintf("%d", timestamp)}
	req.Header["POLY_API_KEY"] = []string{c.apiKey}
	req.Header["POLY_PASSPHRASE"] = []string{c.passphrase}

	fmt.Printf("=== ACTUAL HEADERS ===\n")
	for k, v := range req.Header {
		fmt.Printf("%s: %v\n", k, v)
	}
	fmt.Printf("======================\n")

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
		var errResp types.ErrorResponse
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
