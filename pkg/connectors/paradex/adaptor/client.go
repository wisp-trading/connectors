package adaptor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/trishtzy/go-paradex/client/authentication"
	"github.com/wisp-trading/sdk/pkg/types/logging"

	"github.com/trishtzy/go-paradex/auth"
	"github.com/trishtzy/go-paradex/client"
	"github.com/trishtzy/go-paradex/client/system"
	"github.com/trishtzy/go-paradex/models"
)

type Client struct {
	api               *client.ParadexRESTAPI
	logger            logging.ApplicationLogger
	ethPrivateKey     string
	dexPrivateKey     string
	dexPublicKey      string
	dexAccountAddress string
	ethereumAddress   string
	systemConfig      *models.ResponsesSystemConfigResponse
	jwtToken          string
	tokenExpiry       time.Time
	useTestnet        bool
	mu                sync.RWMutex
}

type Config struct {
	BaseURL       string
	StarknetRPC   string
	EthPrivateKey string
	Network       string
}

func NewClient(cfg *Config, logger logging.ApplicationLogger) (*Client, error) {
	host := "api.prod.paradex.trade"
	if cfg.Network == "testnet" {
		host = "api.testnet.paradex.trade"
	}

	clientConfig := client.DefaultTransportConfig().
		WithHost(host).
		WithBasePath("/v1").
		WithSchemes([]string{"https"})
	api := client.NewHTTPClientWithConfig(nil, clientConfig)

	sysParams := system.NewGetSystemConfigParams()
	sysResp, err := api.System.GetSystemConfig(sysParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get system config: %w", err)
	}

	systemConfig := sysResp.GetPayload()

	dexPrivKey, dexPubKey, dexAccountAddr := auth.GenerateParadexAccount(
		*systemConfig,
		cfg.EthPrivateKey,
	)
	_, ethAddress := auth.GetEthereumAccount(cfg.EthPrivateKey)

	return &Client{
		api:               api,
		logger:            logger,
		ethPrivateKey:     cfg.EthPrivateKey,
		dexPrivateKey:     dexPrivKey,
		dexPublicKey:      dexPubKey,
		dexAccountAddress: dexAccountAddr,
		ethereumAddress:   ethAddress,
		systemConfig:      systemConfig,
		useTestnet:        cfg.Network == "testnet",
	}, nil
}

// Authenticate obtains a new JWT if needed.
func (c *Client) Authenticate(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.jwtToken != "" && time.Now().Before(c.tokenExpiry) {
		return nil
	}
	now := time.Now().Unix()
	timestamp := fmt.Sprintf("%d", now)
	expiration := fmt.Sprintf("%d", now+auth.DEFAULT_EXPIRY_IN_SECONDS)
	sig := auth.SignSNTypedData(auth.SignerParams{
		MessageType:       "auth",
		DexAccountAddress: c.dexAccountAddress,
		DexPrivateKey:     c.dexPrivateKey,
		SysConfig:         *c.systemConfig,
		Params: map[string]interface{}{
			"timestamp":  timestamp,
			"expiration": expiration,
		},
	})
	authParams := authentication.NewAuthParams().WithContext(ctx)
	authParams.SetPARADEXSTARKNETSIGNATURE(sig)
	authParams.SetPARADEXSTARKNETACCOUNT(c.dexAccountAddress)
	authParams.SetPARADEXTIMESTAMP(timestamp)
	authParams.SetPARADEXSIGNATUREEXPIRATION(&expiration)
	resp, err := c.api.Authentication.Auth(authParams)
	if err != nil {
		c.logger.Error("authentication failed", "error", err)
		return fmt.Errorf("authentication failed: %w", err)
	}
	c.jwtToken = resp.Payload.JwtToken
	c.tokenExpiry = time.Now().Add(time.Duration(auth.DEFAULT_EXPIRY_IN_SECONDS-30) * time.Second)
	c.logger.Info("authentication successful")
	return nil
}

func (c *Client) Onboard(ctx context.Context) error {
	sig := auth.SignSNTypedData(auth.SignerParams{
		MessageType:       "onboarding",
		DexAccountAddress: c.dexAccountAddress,
		DexPrivateKey:     c.dexPrivateKey,
		SysConfig:         *c.systemConfig,
	})

	params := authentication.NewOnboardingParams().WithContext(ctx)
	params.SetPARADEXETHEREUMACCOUNT(c.ethereumAddress)
	params.SetPARADEXSTARKNETACCOUNT(c.dexAccountAddress)
	params.SetPARADEXSTARKNETSIGNATURE(sig)
	params.SetRequest(&models.RequestsOnboarding{
		PublicKey: c.dexPublicKey,
	})

	onboarded, err := c.api.Authentication.Onboarding(params)
	if err != nil {
		return fmt.Errorf("onboarding failed: %w", err)
	}

	c.logger.Info("onboarding successful", onboarded.String())
	return nil
}

func (c *Client) GetJWTToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.jwtToken
}

func (c *Client) GetAuthHeaders() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return map[string]string{
		"Authorization":            fmt.Sprintf("Bearer %s", c.jwtToken),
		"PARADEX-STARKNET-ACCOUNT": c.dexAccountAddress,
	}
}

func (c *Client) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.jwtToken != "" && time.Now().Before(c.tokenExpiry)
}

func (c *Client) GetTokenExpiry() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tokenExpiry
}

func (c *Client) GetDexAccountAddress() string                           { return c.dexAccountAddress }
func (c *Client) GetDexPrivateKey() string                               { return c.dexPrivateKey }
func (c *Client) GetDexPublicKey() string                                { return c.dexPublicKey }
func (c *Client) GetEthereumAddress() string                             { return c.ethereumAddress }
func (c *Client) GetSystemConfig() *models.ResponsesSystemConfigResponse { return c.systemConfig }
func (c *Client) API() *client.ParadexRESTAPI                            { return c.api }
func (c *Client) IsTestnet() bool                                        { return c.useTestnet }
func (c *Client) GetBaseURL() string {
	if c.useTestnet {
		return "https://api.testnet.paradex.trade/v1"
	}
	return "https://api.prod.paradex.trade/v1"
}

func convertParadexChainID(paradexChainID string) string {
	switch paradexChainID {
	case "PRIVATE_SN_PARACLEAR_MAINNET":
		return "0x534e5f4d41494e" // Starknet Mainnet
	case "PRIVATE_SN_PARACLEAR_SEPOLIA":
		return "0x534e5f5345504f4c4941" // Starknet Sepolia
	default:
		// Default to mainnet if unknown
		return "0x534e5f4d41494e"
	}
}
