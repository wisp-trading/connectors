package adaptor

import (
	"context"
	"fmt"
	"sync"

	gateapi "github.com/gate/gateapi-go/v7"
)

// SpotClient wraps Gate.io Spot API with lazy configuration
type SpotClient interface {
	Configure(apiKey, apiSecret, baseURL string) error
	IsConfigured() bool
	GetSpotApi() (*gateapi.APIClient, error)
	GetAPIContext() context.Context
}

// FuturesClient wraps Gate.io Futures API with lazy configuration
type FuturesClient interface {
	Configure(apiKey, apiSecret, baseURL, settle string) error
	IsConfigured() bool
	GetFuturesApi() (*gateapi.APIClient, error)
	GetAPIContext() context.Context
	GetSettle() string
}

// spotClient implementation
type spotClient struct {
	client     *gateapi.APIClient
	apiContext context.Context
	configured bool
	mu         sync.RWMutex
}

// futuresClient implementation
type futuresClient struct {
	client     *gateapi.APIClient
	apiContext context.Context
	settle     string
	configured bool
	mu         sync.RWMutex
}

// NewSpotClient creates an unconfigured spot client
func NewSpotClient() SpotClient {
	return &spotClient{
		configured: false,
	}
}

// NewFuturesClient creates an unconfigured futures client
func NewFuturesClient() FuturesClient {
	return &futuresClient{
		configured: false,
	}
}

// Configure sets up the spot client with runtime config
func (s *spotClient) Configure(apiKey, apiSecret, baseURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.configured {
		return fmt.Errorf("client already configured")
	}

	if baseURL == "" {
		baseURL = "https://api.gateio.ws/api/v4"
	}

	config := gateapi.NewConfiguration()
	config.BasePath = baseURL
	config.Key = apiKey
	config.Secret = apiSecret

	s.client = gateapi.NewAPIClient(config)
	s.apiContext = context.WithValue(context.Background(), gateapi.ContextGateAPIV4, gateapi.GateAPIV4{
		Key:    apiKey,
		Secret: apiSecret,
	})
	s.configured = true

	return nil
}

func (s *spotClient) IsConfigured() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.configured
}

func (s *spotClient) GetSpotApi() (*gateapi.APIClient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.configured {
		return nil, fmt.Errorf("spot client not configured")
	}
	return s.client, nil
}

func (s *spotClient) GetAPIContext() context.Context {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.apiContext
}

// Configure sets up the futures client with runtime config
func (f *futuresClient) Configure(apiKey, apiSecret, baseURL, settle string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.configured {
		return fmt.Errorf("client already configured")
	}

	if baseURL == "" {
		baseURL = "https://api.gateio.ws/api/v4"
	}

	if settle == "" {
		settle = "usdt"
	}

	config := gateapi.NewConfiguration()
	config.BasePath = baseURL
	config.Key = apiKey
	config.Secret = apiSecret

	f.client = gateapi.NewAPIClient(config)
	f.apiContext = context.WithValue(context.Background(), gateapi.ContextGateAPIV4, gateapi.GateAPIV4{
		Key:    apiKey,
		Secret: apiSecret,
	})
	f.settle = settle
	f.configured = true

	return nil
}

func (f *futuresClient) IsConfigured() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.configured
}

func (f *futuresClient) GetFuturesApi() (*gateapi.APIClient, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.configured {
		return nil, fmt.Errorf("futures client not configured")
	}
	return f.client, nil
}

func (f *futuresClient) GetAPIContext() context.Context {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.apiContext
}

func (f *futuresClient) GetSettle() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.settle
}
