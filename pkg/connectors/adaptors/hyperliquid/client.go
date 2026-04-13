package hyperliquid

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sonirico/go-hyperliquid"
)

// ExchangeClient interface for trading operations with lazy configuration
type ExchangeClient interface {
	Configure(baseURL, privateKey, vaultAddr, accountAddr string) error
	IsConfigured() bool
	GetExchange() (*hyperliquid.Exchange, error)

	// CancelOrder cancels a resting order by asset index and order ID.
	// This bypasses the go-hyperliquid SDK v0.5.0 bug where CancelOrderWire
	// serialises the OID as a JSON string instead of an integer.
	CancelOrder(assetIndex int, orderID int64) error
}

// InfoClient interface for market data queries with lazy configuration
type InfoClient interface {
	Configure(baseURL string) error
	IsConfigured() bool
	GetInfo() (*hyperliquid.Info, error)
}

// exchangeClient implementation
type exchangeClient struct {
	exchange   *hyperliquid.Exchange
	privateKey *ecdsa.PrivateKey
	vaultAddr  string
	baseURL    string
	info       *hyperliquid.Info
	configured bool
	mu         sync.RWMutex
}

// infoClient implementation
type infoClient struct {
	info       *hyperliquid.Info
	configured bool
	mu         sync.RWMutex
}

// NewExchangeClient creates an unconfigured exchange client
func NewExchangeClient() ExchangeClient {
	return &exchangeClient{
		configured: false,
	}
}

// NewInfoClient creates an unconfigured info client
func NewInfoClient() InfoClient {
	return &infoClient{
		configured: false,
	}
}

// Configure sets up the exchange client with runtime config
func (e *exchangeClient) Configure(baseURL, privateKey, vaultAddr, accountAddr string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.configured {
		return fmt.Errorf("client already configured")
	}

	privateKeyECDSA, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	// Fetch Meta and SpotMeta before creating Exchange
	// This is required for the Exchange to map coin symbols to asset indices
	info := hyperliquid.NewInfo(baseURL, true, nil, nil)

	meta, err := info.Meta()
	if err != nil {
		return fmt.Errorf("failed to fetch meta: %w", err)
	}

	spotMeta, err := info.SpotMeta()
	if err != nil {
		return fmt.Errorf("failed to fetch spot meta: %w", err)
	}

	e.exchange = hyperliquid.NewExchange(
		privateKeyECDSA,
		baseURL,
		meta,
		vaultAddr,
		accountAddr,
		spotMeta,
	)
	e.privateKey = privateKeyECDSA
	e.vaultAddr = vaultAddr
	e.baseURL = baseURL
	e.info = info
	e.configured = true
	return nil
}

func (e *exchangeClient) IsConfigured() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.configured
}

func (e *exchangeClient) GetExchange() (*hyperliquid.Exchange, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.configured {
		return nil, fmt.Errorf("exchange client not configured")
	}
	return e.exchange, nil
}

// cancelOrderWire is a corrected wire format for the cancel action.
// The go-hyperliquid v0.5.0 SDK defines OrderID as string, but the
// Hyperliquid API requires it as an integer. This struct fixes that.
type cancelOrderWire struct {
	Asset   int   `json:"a"`
	OrderID int64 `json:"o"`
}

type cancelAction struct {
	Type    string            `json:"type"`
	Cancels []cancelOrderWire `json:"cancels"`
}

// CancelOrder cancels a resting order, working around the SDK v0.5.0 bug
// where the order ID is serialised as a JSON string instead of an integer.
func (e *exchangeClient) CancelOrder(assetIndex int, orderID int64) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.configured {
		return fmt.Errorf("exchange client not configured")
	}

	action := cancelAction{
		Type: "cancel",
		Cancels: []cancelOrderWire{
			{Asset: assetIndex, OrderID: orderID},
		},
	}

	timestamp := time.Now().UnixMilli()
	isMainnet := e.baseURL == hyperliquid.MainnetAPIURL

	sig, err := hyperliquid.SignL1Action(
		e.privateKey,
		action,
		e.vaultAddr,
		timestamp,
		nil,
		isMainnet,
	)
	if err != nil {
		return fmt.Errorf("failed to sign cancel action: %w", err)
	}

	payload := map[string]any{
		"action":    action,
		"nonce":     timestamp,
		"signature": sig,
	}
	if e.vaultAddr != "" {
		payload["vaultAddress"] = e.vaultAddr
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal cancel payload: %w", err)
	}

	resp, err := http.Post(e.baseURL+"/exchange", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("cancel request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read cancel response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("cancel failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Status   string `json:"status"`
		Response string `json:"response"`
	}
	if err := json.Unmarshal(respBody, &result); err == nil && result.Status == "err" {
		return fmt.Errorf("cancel rejected: %s", result.Response)
	}

	return nil
}

// Configure sets up the info client with runtime config
func (i *infoClient) Configure(baseURL string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.configured {
		return fmt.Errorf("client already configured")
	}

	i.info = hyperliquid.NewInfo(baseURL, true, nil, nil)
	i.configured = true
	return nil
}

func (i *infoClient) IsConfigured() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.configured
}

func (i *infoClient) GetInfo() (*hyperliquid.Info, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if !i.configured {
		return nil, fmt.Errorf("info client not configured")
	}
	return i.info, nil
}
