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
)

// Client wraps Deribit API with request/response handling and authentication
// Follows Deribit JSON-RPC 2.0 specification exactly
type Client interface {
	Configure(clientID, clientSecret, baseURL string) error
	IsConfigured() bool
	GetBaseURL() string

	// Call makes a JSON-RPC call to public or private endpoints
	// Returns the full JSON-RPC response including result/error
	Call(ctx context.Context, method string, params map[string]interface{}) (json.RawMessage, error)

	// GetAccessToken returns a valid access token, refreshing if needed
	GetAccessToken(ctx context.Context) (string, error)
}

// JSONRPCRequest is a JSON-RPC 2.0 request per Deribit spec
type JSONRPCRequest struct {
	Jsonrpc string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
	ID      int64                  `json:"id"`
}

// JSONRPCResponse is a JSON-RPC 2.0 response per Deribit spec
type JSONRPCResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	ID      int64 `json:"id"`
	Testnet bool  `json:"testnet"`
	UsIn    int64 `json:"usIn"`   // Server receive time (microseconds)
	UsOut   int64 `json:"usOut"`  // Server send time (microseconds)
	UsDiff  int64 `json:"usDiff"` // Server processing time (microseconds)
}

// client implements the Client interface
type client struct {
	clientID     string
	clientSecret string
	baseURL      string
	httpClient   *http.Client
	accessToken  string
	tokenExpiry  time.Time
	configured   bool
	mu           sync.RWMutex
}

// NewClient creates an unconfigured Deribit client
func NewClient() Client {
	return &client{
		configured: false,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Configure sets up the client with API credentials (OAuth 2.0 client credentials)
func (c *client) Configure(clientID, clientSecret, baseURL string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.configured {
		return fmt.Errorf("client already configured")
	}

	if clientID == "" || clientSecret == "" || baseURL == "" {
		return fmt.Errorf("clientID, clientSecret, and baseURL are required")
	}

	c.clientID = clientID
	c.clientSecret = clientSecret
	c.baseURL = baseURL
	c.configured = true

	return nil
}

// IsConfigured returns whether the client has been configured
func (c *client) IsConfigured() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.configured
}

// GetBaseURL returns the configured base URL
func (c *client) GetBaseURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.baseURL
}

// GetAccessToken retrieves a valid access token, refreshing if necessary
// Implements OAuth 2.0 client credentials flow per Deribit spec
func (c *client) GetAccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.configured {
		return "", fmt.Errorf("client not configured")
	}

	// Check if current token is still valid (with 30 second buffer for safety)
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry.Add(-30*time.Second)) {
		return c.accessToken, nil
	}

	// Get new token via public/auth endpoint
	token, expiresIn, err := c.authenticateWithClientCredentials(ctx)
	if err != nil {
		return "", err
	}

	c.accessToken = token
	// expires_in is in seconds per Deribit spec
	c.tokenExpiry = time.Now().Add(time.Duration(expiresIn) * time.Second)

	return token, nil
}

// authenticateWithClientCredentials performs OAuth 2.0 client credentials grant
// per Deribit spec: https://docs.deribit.com/api-reference/authentication/public-auth
func (c *client) authenticateWithClientCredentials(ctx context.Context) (string, int, error) {
	// Prepare JSON-RPC request to public/auth
	req := JSONRPCRequest{
		Jsonrpc: "2.0",
		ID:      generateRequestID(),
		Method:  "public/auth",
		Params: map[string]interface{}{
			"grant_type":    "client_credentials",
			"client_id":     c.clientID,
			"client_secret": c.clientSecret,
		},
	}

	respBody, err := c.makeRequest(ctx, req)
	if err != nil {
		return "", 0, fmt.Errorf("authentication request failed: %w", err)
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", 0, fmt.Errorf("failed to parse auth response: %w", err)
	}

	if result.AccessToken == "" {
		return "", 0, fmt.Errorf("no access_token in response")
	}

	if result.ExpiresIn <= 0 {
		return "", 0, fmt.Errorf("invalid expires_in value: %d", result.ExpiresIn)
	}

	return result.AccessToken, result.ExpiresIn, nil
}

// Call makes a JSON-RPC call to Deribit API (public or private)
// Returns the result field from JSON-RPC response
func (c *client) Call(ctx context.Context, method string, params map[string]interface{}) (json.RawMessage, error) {
	c.mu.RLock()
	if !c.configured {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client not configured")
	}
	c.mu.RUnlock()

	// Determine if private method and get token if needed
	var authToken string
	if isPrivateMethod(method) {
		token, err := c.GetAccessToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get access token: %w", err)
		}
		authToken = token
	}

	// Build JSON-RPC request per spec
	req := JSONRPCRequest{
		Jsonrpc: "2.0",
		ID:      generateRequestID(),
		Method:  method,
		Params:  params,
	}

	result, err := c.makeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// If this was a private method call, include the auth token
	if authToken != "" {
		// Token is included in makeRequest via header
		_ = authToken
	}

	return result, nil
}

// makeRequest makes an HTTP POST request to Deribit API
// Returns the result field from the JSON-RPC response
func (c *client) makeRequest(ctx context.Context, req JSONRPCRequest) (json.RawMessage, error) {
	c.mu.RLock()
	baseURL := c.baseURL
	token := c.accessToken // Get current token if available
	c.mu.RUnlock()

	// URL format per spec: https://www.deribit.com/api/v2/{method}
	url := fmt.Sprintf("%s/%s", baseURL, req.Method)

	// Marshal request to JSON per JSON-RPC 2.0 spec
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set required headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Add Bearer token for private methods
	if isPrivateMethod(req.Method) && token != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	// Execute HTTP request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close() // nolint:errcheck
	}()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse JSON-RPC response per spec
	var jsonResp JSONRPCResponse
	if err := json.Unmarshal(respBody, &jsonResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON-RPC response: %w", err)
	}

	// Check for JSON-RPC error
	if jsonResp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", jsonResp.Error.Code, jsonResp.Error.Message)
	}

	return jsonResp.Result, nil
}

// isPrivateMethod returns true if method requires authentication
func isPrivateMethod(method string) bool {
	return len(method) > 8 && method[:8] == "private/"
}

// generateRequestID creates a unique request ID for JSON-RPC
func generateRequestID() int64 {
	return time.Now().UnixNano()
}
