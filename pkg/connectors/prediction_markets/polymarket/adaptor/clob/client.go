package clob

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto" // Alias to avoid clash
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/config"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
)

const (
	// API endpoints
	ordersEndpoint        = "/orders"
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
	CancelOrder(ctx context.Context, orderID string) (*CancelOrderResponse, error)
	CancelAllOrders(ctx context.Context, req CancelAllOrdersRequest) (*CancelAllOrdersResponse, error)
	GetOrder(ctx context.Context, orderID string) error
	GetOpenOrders(ctx context.Context, assetID string) error
}

// polymarketClient implementation
type polymarketClient struct {
	baseURL           string
	privateKey        *ecdsa.PrivateKey
	signerAddress     string
	polymarketAddress string
	privateKeyHex     string
	chainID           *big.Int
	httpClient        *http.Client
	configured        bool
	mu                sync.RWMutex
	apiKey            string
	apiSecret         string
	passphrase        string
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

	// Parse private key
	privateKeyHex := strings.TrimPrefix(config.PrivateKey, "0x")
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return fmt.Errorf("failed to decode private key: %w", err)
	}

	privateKey, err := ethcrypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	address := ethcrypto.PubkeyToAddress(*publicKey)

	c.baseURL = config.BaseURL
	c.privateKey = privateKey
	c.privateKeyHex = privateKeyHex
	c.signerAddress = strings.ToLower(address.Hex())
	c.polymarketAddress = config.PolymarketAddress
	c.chainID = big.NewInt(config.ChainID)

	credentials, err := c.deriveCredentials()
	if err != nil {
		return err
	}

	c.apiKey = credentials.APIKey
	c.apiSecret = credentials.Secret
	c.passphrase = credentials.Passphrase

	c.configured = true

	return nil
}

// IsConfigured returns whether the client is configured
func (c *polymarketClient) IsConfigured() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.configured
}
