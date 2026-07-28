package uniswap_v3

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	onchainconnector "github.com/wisp-trading/sdk/pkg/types/connector/onchain"
	"github.com/wisp-trading/sdk/pkg/types/logging"
	"github.com/wisp-trading/sdk/pkg/types/temporal"
)

type tokenMeta struct {
	address  common.Address
	decimals uint8
}

// uniswapV3 is the UniV3 EVM onchain connector.
type uniswapV3 struct {
	appLogger     logging.ApplicationLogger
	tradingLogger logging.TradingLogger
	timeProvider  temporal.TimeProvider

	cfg         *Config
	client      *ethclient.Client
	initialized bool

	mu     sync.RWMutex
	tokens map[string]tokenMeta // lower(symbol) -> meta
}

var _ onchainconnector.Connector = (*uniswapV3)(nil)

// NewUniswapV3 constructs an uninitialized connector (fx).
func NewUniswapV3(
	appLogger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
	timeProvider temporal.TimeProvider,
) onchainconnector.Connector {
	return &uniswapV3{
		appLogger:     appLogger,
		tradingLogger: tradingLogger,
		timeProvider:  timeProvider,
		tokens:        make(map[string]tokenMeta),
	}
}

func (u *uniswapV3) NewConfig() connector.Config {
	return &Config{}
}

func (u *uniswapV3) GetConnectorInfo() *connector.Info {
	return &connector.Info{
		Name:                types.UniswapV3,
		TradingEnabled:      true,
		WebSocketEnabled:    false,
		SupportedOrderTypes: []connector.OrderType{connector.OrderTypeMarket},
		QuoteCurrency:       "WETH",
	}
}

func (u *uniswapV3) SupportsTradingOperations() bool { return true }
func (u *uniswapV3) SupportsRealTimeData() bool      { return false }

func (u *uniswapV3) IsInitialized() bool { return u.initialized }

func (u *uniswapV3) Initialize(config connector.Config) error {
	if u.initialized {
		return fmt.Errorf("uniswap_v3 already initialized")
	}
	cfg, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for uniswap_v3: %T", config)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	client, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("rpc dial: %w", err)
	}

	// Verify chain id when possible.
	ctx := context.Background()
	if id, err := client.ChainID(ctx); err == nil {
		if id.Uint64() != cfg.ChainID {
			client.Close()
			return fmt.Errorf("rpc chain id %d != config chain_id %d", id.Uint64(), cfg.ChainID)
		}
	}

	u.client = client
	u.cfg = cfg
	u.initialized = true

	// Seed WETH if configured.
	if cfg.WETH != "" {
		_ = u.RegisterToken("WETH", cfg.WETH, 18)
		_ = u.RegisterToken("ETH", cfg.WETH, 18)
	}

	u.appLogger.Info("uniswap_v3 initialized chain_id=%d dry_run=%v", cfg.ChainID, cfg.DryRun)
	return nil
}

func (u *uniswapV3) Close() error {
	if u.client != nil {
		u.client.Close()
	}
	u.initialized = false
	return nil
}

func (u *uniswapV3) ChainID() uint64 {
	if u.cfg == nil {
		return 0
	}
	return u.cfg.ChainID
}

func (u *uniswapV3) NativeWrapped() string { return "WETH" }

func (u *uniswapV3) RegisterToken(symbol string, address string, decimals uint8) error {
	if symbol == "" {
		return fmt.Errorf("symbol required")
	}
	if !common.IsHexAddress(address) {
		return fmt.Errorf("invalid token address: %s", address)
	}
	key := strings.ToLower(strings.TrimSpace(symbol))
	u.mu.Lock()
	u.tokens[key] = tokenMeta{address: common.HexToAddress(address), decimals: decimals}
	// also index by address for resolve-by-0x
	u.tokens[strings.ToLower(common.HexToAddress(address).Hex())] = tokenMeta{
		address:  common.HexToAddress(address),
		decimals: decimals,
	}
	u.mu.Unlock()
	return nil
}

func (u *uniswapV3) ResolveToken(symbol string) (address string, decimals uint8, ok bool) {
	u.mu.RLock()
	defer u.mu.RUnlock()
	meta, ok := u.tokens[strings.ToLower(strings.TrimSpace(symbol))]
	if !ok {
		// bare hex address not yet registered — allow as 18-dec provisional
		if common.IsHexAddress(symbol) {
			return common.HexToAddress(symbol).Hex(), 18, true
		}
		return "", 0, false
	}
	return meta.address.Hex(), meta.decimals, true
}

func (u *uniswapV3) requireInit() error {
	if !u.initialized || u.cfg == nil {
		return fmt.Errorf("uniswap_v3 not initialized")
	}
	return nil
}
