package adaptor

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"

	polymarket "github.com/GoPolymarket/polymarket-go-sdk"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/auth"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/ctf"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
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

	// Create fully configured signer - SDK handles all Safe/EOA logic
	sigType := auth.SignatureType(config.SignatureType)
	configuredSigner, err := auth.NewConfiguredSignerFromConfig(config.PrivateKey, sigType, 137)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create signer: %w", err)
	}

	signer := configuredSigner.GetSigner()
	safeAddr := configuredSigner.GetOperationAddress()

	// Log Safe mode if applicable
	if configuredSigner.IsSafeMode() {
		fmt.Printf("[polymarket] Safe mode: address %s (owner: %s)\n", safeAddr.Hex(), configuredSigner.GetSigningAddress().Hex())
	}

	clientOpts := []polymarket.Option{polymarket.WithUseServerTime(true)}

	// When a Polygon RPC URL is configured, build a CTF client with a NegRisk backend
	// so SplitPosition and MergePositions (on-chain EVM calls) can be executed.
	// Without this, ctf.NewClient() is used — a lightweight client that only computes
	// IDs client-side and cannot submit transactions (CTF-002 backend required error).
	if config.PolygonRPCURL != "" {
		ctfClient, err := buildCTFClient(config, safeAddr)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to build CTF client: %w", err)
		}
		clientOpts = append(clientOpts, polymarket.WithCTF(ctfClient))
	}

	client := polymarket.NewClient(clientOpts...)

	// Set signature type BEFORE deriving API key - Polymarket needs to know
	// which address context to use (EOA vs Safe)
	key, err := client.CLOB.WithAuth(signer, nil).WithSignatureType(sigType).DeriveAPIKey(context.Background())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("derive API key: %w", err)
	}

	creds := &auth.APIKey{
		Key:        key.APIKey,
		Secret:     key.Secret,
		Passphrase: key.Passphrase,
	}

	// Set up the actual clobClient with the proper signer (SafeSigner for Safe mode)
	// and configure funder for Safe wallets
	clobClient := client.CLOB.WithAuth(signer, creds).WithSignatureType(sigType)
	if sigType != auth.SignatureEOA {
		clobClient = clobClient.WithFunder(safeAddr)
	}
	clobWebsocket := client.CLOBWS
	tokenManager := client.CTF

	clobWebsocket.Authenticate(signer, creds)
	orderManager := order_manager.NewOrderManager(clobClient, tokenManager, signer, config.PolygonRPCURL, sigType, safeAddr)
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

// buildCTFClient creates a CTF client backed by an Ethereum RPC connection.
// This is required for on-chain operations (SplitPosition / MergePositions).
// The private key is used to derive the transactor; the chain ID is always
// Polygon mainnet (137) since Polymarket is deployed there.
// For Safe wallets, the safeAddr is passed for reference (Safe support requires
// additional configuration beyond private key signing).
func buildCTFClient(cfg *config.Config, safeAddr common.Address) (ctf.Client, error) {
	backend, err := ethclient.Dial(cfg.PolygonRPCURL)
	if err != nil {
		return nil, fmt.Errorf("dial polygon rpc %q: %w", cfg.PolygonRPCURL, err)
	}

	privKeyHex := strings.TrimPrefix(cfg.PrivateKey, "0x")
	key, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	txOpts, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(ctf.PolygonChainID))
	if err != nil {
		return nil, fmt.Errorf("create transactor: %w", err)
	}

	// For Safe mode: set txOpts.From to the Safe address so that transactions
	// are credited to the Safe in the CLOB's accounting (matching CLOB orders
	// which use SafeSigner). Polymarket's infrastructure routes these correctly
	// through the Safe contract when signature_type is GNOSIS_SAFE.
	sigType := auth.SignatureType(cfg.SignatureType)
	if sigType != auth.SignatureEOA {
		fmt.Printf("[polymarket:ctf] Safe mode: using Safe address %s for transactions\n", safeAddr.Hex())
		txOpts.From = safeAddr
	}

	return ctf.NewClientWithNegRisk(backend, txOpts, int64(ctf.PolygonChainID))
}
