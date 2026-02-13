package clob

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"

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
	apiKey := c.apiKey
	apiSecret := c.apiSecret
	passphrase := c.passphrase
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

	// Build HMAC message: timestamp + METHOD + path + body
	message := fmt.Sprintf("%d%s%s%s", timestamp, strings.ToUpper(method), endpoint, bodyStr)

	// Create HMAC-SHA256 signature
	h := hmac.New(sha256.New, []byte(apiSecret))
	h.Write([]byte(message))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// Set auth headers
	req.Header.Set("POLY_ADDRESS", c.polymarketAddress)
	req.Header.Set("POLY_SIGNATURE", signature)
	req.Header.Set("POLY_TIMESTAMP", fmt.Sprintf("%d", timestamp))
	req.Header.Set("POLY_API_KEY", apiKey)
	req.Header.Set("POLY_PASSPHRASE", passphrase)

	fmt.Printf("=== AUTH DEBUG ===\n")
	fmt.Printf("Method: %s\n", method)
	fmt.Printf("Endpoint: %s\n", endpoint)
	fmt.Printf("Body: %s\n", bodyStr)
	fmt.Printf("Timestamp: %d\n", timestamp)
	fmt.Printf("Message: %s\n", message)
	fmt.Printf("API Key: %s\n", apiKey)
	fmt.Printf("Secret length: %d\n", len(apiSecret))
	fmt.Printf("Passphrase: %s\n", passphrase)
	fmt.Printf("Address: %s\n", c.polymarketAddress)
	fmt.Printf("Signature: %s\n", signature)
	fmt.Printf("Base URL: %s\n", baseURL)
	fmt.Printf("==================\n")

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
