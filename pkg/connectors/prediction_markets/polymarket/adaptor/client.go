package adaptor

import (
	"context"
	"fmt"
	"sync"

	"github.com/GoPolymarket/polymarket-go-sdk"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/auth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/gamma"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/order_manager"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/websocket"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/config"
)

type Client interface {
	Configure(config *config.Config) (order_manager.OrderManager, websocket.Websocket, gamma.GammaClient, error)
	IsConfigured() bool
}

// polymarketClient implementation
type polymarketClient struct {
	configured bool
	mu         sync.RWMutex
}

// NewPolymarketClient creates an unconfigured Polymarket client
func NewPolymarketClient() Client {
	return &polymarketClient{
		configured: false,
	}
}

// Configure sets up the client with runtime config
func (c *polymarketClient) Configure(config *config.Config) (order_manager.OrderManager, websocket.Websocket, gamma.GammaClient, error) {

	err := c.validate(config)
	if err != nil {
		return nil, nil, nil, err
	}

	signer, err := auth.NewPrivateKeySigner(config.PrivateKey, 137)

	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create signer: %w", err)
	}

	client := polymarket.NewClient(polymarket.WithUseServerTime(true))
	clobClient := client.CLOB.WithAuth(signer, nil)

	key, err := clobClient.DeriveAPIKey(context.Background())
	if err != nil {
		return nil, nil, nil, err
	}

	creds := &auth.APIKey{
		Key:        key.APIKey,
		Secret:     key.Secret,
		Passphrase: key.Passphrase,
	}

	polymarketAddress := common.HexToAddress(config.PolymarketAddress)
	clobClient = clobClient.WithAuth(signer, creds).WithFunder(polymarketAddress).WithSignatureType(auth.SignatureGnosisSafe)
	clobWebsocket := client.CLOBWS
	tokenManager := client.CTF

	clobWebsocket.Authenticate(signer, creds)
	orderManager := order_manager.NewOrderManager(clobClient, tokenManager, signer)
	websocketManager := websocket.NewWebsocket(clobWebsocket)
	gammaClient := gamma.NewGammaClient(client.Gamma)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.configured = true

	return orderManager, websocketManager, gammaClient, nil
}

func (c *polymarketClient) validate(config *config.Config) error {
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
	return nil
}

// IsConfigured returns whether the client is configured
func (c *polymarketClient) IsConfigured() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.configured
}
