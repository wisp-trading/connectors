package adaptors

import (
	"context"
	"fmt"
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

// Configure sets up the exchange client with runtime config.
// Aligned to go-hyperliquid v0.35+ (context-aware NewInfo / NewExchange).
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// NewInfo with nil meta/spotMeta loads them (requires live network).
	info := hyperliquid.NewInfo(ctx, baseURL, true, nil, nil, nil)

	meta, err := info.Meta(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch meta: %w", err)
	}

	spotMeta, err := info.SpotMeta(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch spot meta: %w", err)
	}

	e.exchange = hyperliquid.NewExchange(
		ctx,
		privateKeyECDSA,
		baseURL,
		meta,
		vaultAddr,
		accountAddr,
		spotMeta,
		nil, // default perp dexs
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

// Configure sets up the info client with runtime config.
func (i *infoClient) Configure(baseURL string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.configured {
		return fmt.Errorf("client already configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	i.info = hyperliquid.NewInfo(ctx, baseURL, true, nil, nil, nil)
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
