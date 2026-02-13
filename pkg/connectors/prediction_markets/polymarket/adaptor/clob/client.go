package clob

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/GoPolymarket/polymarket-go-sdk"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/auth"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob"
	"github.com/ethereum/go-ethereum/common"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/config"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
)

const (
	// API endpoints
	ordersEndpoint        = "/order"
	cancelOrderEndpoint   = "/orders"
	cancelAllEndpoint     = "/orders"
	getOrderEndpoint      = "/orders"
	getOpenOrdersEndpoint = "/orders"

	getFeeRateEndpoint = "fee-rate?token_id="

	// HTTP timeouts
	defaultTimeout = 30 * time.Second
)

// PolymarketClient wraps Polymarket CLOB API with lazy configuration
type PolymarketClient interface {
	Configure(config *config.Config) error
	IsConfigured() bool

	// Trading
	PlaceOrder(ctx context.Context, order prediction.LimitOrder) error
}

// polymarketClient implementation
type polymarketClient struct {
	httpClient        *http.Client
	configured        bool
	mu                sync.RWMutex
	client            clob.Client
	signer            *auth.PrivateKeySigner
	polymarketAddress common.Address
}

// NewPolymarketClient creates an unconfigured Polymarket client
func NewPolymarketClient() PolymarketClient {
	return &polymarketClient{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		configured: false,
	}
}

// Configure sets up the client with runtime config
func (c *polymarketClient) Configure(config *config.Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.configured {
		return fmt.Errorf("client already configured")
	}

	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	signer, err := auth.NewPrivateKeySigner(config.PrivateKey, 137)

	if err != nil {
		return fmt.Errorf("failed to create signer: %w", err)
	}

	c.signer = signer

	client := polymarket.NewClient(polymarket.WithUseServerTime(true))
	c.client = client.CLOB.WithAuth(signer, nil)

	key, err := c.client.DeriveAPIKey(context.Background())
	if err != nil {
		return err
	}

	creds := &auth.APIKey{
		Key:        key.APIKey,
		Secret:     key.Secret,
		Passphrase: key.Passphrase,
	}

	c.polymarketAddress = common.HexToAddress(config.PolymarketAddress)
	c.client = c.client.WithAuth(signer, creds).WithFunder(c.polymarketAddress).WithSignatureType(auth.SignatureGnosisSafe)

	c.configured = true

	return nil
}

// IsConfigured returns whether the client is configured
func (c *polymarketClient) IsConfigured() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.configured
}
