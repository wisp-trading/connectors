package adaptor

import (
	"context"
	"fmt"
	"sync"

	bybit "github.com/bybit-exchange/bybit.go.api"
)

// PerpClient wraps Bybit Perpetual API with lazy configuration
type PerpClient interface {
	Configure(apiKey, apiSecret, baseURL string, isTestnet bool) error
	IsConfigured() bool
	GetClient() (*bybit.Client, error)
	GetAPIContext() context.Context
}

type perpClient struct {
	client     *bybit.Client
	apiContext context.Context
	configured bool
	mu         sync.RWMutex
}

func NewPerpClient() PerpClient {
	return &perpClient{configured: false}
}

func (p *perpClient) Configure(apiKey, apiSecret, baseURL string, isTestnet bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.configured {
		return fmt.Errorf("client already configured")
	}

	p.client = bybit.NewBybitHttpClient(apiKey, apiSecret, bybit.WithBaseURL(baseURL))
	p.apiContext = context.Background()
	p.configured = true
	return nil
}

func (p *perpClient) IsConfigured() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.configured
}

func (p *perpClient) GetClient() (*bybit.Client, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if !p.configured {
		return nil, fmt.Errorf("perp client not configured")
	}
	return p.client, nil
}

func (p *perpClient) GetAPIContext() context.Context {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.apiContext
}
