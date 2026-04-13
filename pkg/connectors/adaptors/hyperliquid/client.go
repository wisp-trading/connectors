package hyperliquid

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sonirico/go-hyperliquid"
)

// ExchangeClient interface for trading operations with lazy configuration
type ExchangeClient interface {
	Configure(baseURL, privateKey, vaultAddr, accountAddr string) error
	IsConfigured() bool
	GetExchange() (*hyperliquid.Exchange, error)
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
	info := hyperliquid.NewInfo(context.Background(), baseURL, true, nil, nil, nil)

	meta, err := info.Meta(context.Background())
	if err != nil {
		return fmt.Errorf("failed to fetch meta: %w", err)
	}

	spotMeta, err := info.SpotMeta(context.Background())
	if err != nil {
		return fmt.Errorf("failed to fetch spot meta: %w", err)
	}

	e.exchange = hyperliquid.NewExchange(
		context.Background(),
		privateKeyECDSA,
		baseURL,
		meta,
		vaultAddr,
		accountAddr,
		spotMeta,
		nil,
	)
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

// Configure sets up the info client with runtime config
func (i *infoClient) Configure(baseURL string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.configured {
		return fmt.Errorf("client already configured")
	}

	i.info = hyperliquid.NewInfo(context.Background(), baseURL, true, nil, nil, nil)
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
