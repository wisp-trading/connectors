package clob

import (
	"bytes"
	"context"
	"crypto/ecdsa"

	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/types"
)

// doRequest is a helper method to execute HTTP requests with proper authentication and error handling
func (c *polymarketClient) doRequest(ctx context.Context, method, endpoint string, body interface{}, result interface{}) error {
	c.mu.RLock()
	baseURL := c.baseURL
	apiKey := c.apiKey
	apiSecret := c.apiSecret
	passphrase := c.passphrase
	privateKey := c.privateKey
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

	// Add L1 authentication headers (if you have API keys)
	if apiKey != "" {
		req.Header.Set("POLY_API_KEY", apiKey)
		req.Header.Set("POLY_SECRET", apiSecret)
		req.Header.Set("POLY_PASSPHRASE", passphrase)
	}

	// Add L2 authentication (wallet-based auth)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	message := timestamp + method + endpoint + bodyStr

	// Sign the request message
	hash := ethcrypto.Keccak256Hash([]byte(message))
	signature, err := ethcrypto.Sign(hash.Bytes(), privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	// Adjust V value
	if signature[64] < 27 {
		signature[64] += 27
	}

	// Derive address from private key
	publicKey := privateKey.Public()
	publicKeyECDSA, _ := publicKey.(*ecdsa.PublicKey)
	address := ethcrypto.PubkeyToAddress(*publicKeyECDSA)

	req.Header.Set("POLY_SIGNATURE", "0x"+hex.EncodeToString(signature))
	req.Header.Set("POLY_TIMESTAMP", timestamp)
	req.Header.Set("POLY_ADDRESS", address.Hex())

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
