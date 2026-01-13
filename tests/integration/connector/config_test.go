package connector_test

import (
	"os"
	"path/filepath"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/live-trading/pkg/connectors/bybit"
	gatespot "github.com/backtesting-org/live-trading/pkg/connectors/gate/spot"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid"
	"github.com/backtesting-org/live-trading/pkg/connectors/paradex"
	"github.com/backtesting-org/live-trading/pkg/connectors/types"
	"github.com/joho/godotenv"
)

func init() {
	// Try to load .env file from the test directory
	// Ignore errors if file doesn't exist (env vars may be set directly)
	envPath := filepath.Join(".", ".env")
	_ = godotenv.Load(envPath)
}

// ========================================
// TEST CONFIGURATION - EDIT HERE
// ========================================
const (
	// Which connector to test
	testConnectorName = types.GateSpot // Change to types.Paradex or types.Bybit
	//testConnectorName = types.Hyperliquid // Change to types.Paradex or types.Bybit

	// Test asset
	testSymbol = "ETH"

	// Test instrument type
	testInstrumentType = connector.TypeSpot

	// Enable trading tests (DANGEROUS - only on testnet)
	enableTradingTests = true
)

// ========================================
// CONFIG LOADERS
// ========================================

func getConnectorConfig(name connector.ExchangeName) connector.Config {
	switch name {
	case types.Hyperliquid:
		return getHyperliquidConfig()
	case types.Paradex:
		return getParadexConfig()
	case types.Bybit:
		return getBybitConfig()
	case types.GateSpot:
		return getGateSpotConfig()
	default:
		panic("unknown connector: " + name)
	}
}

func getHyperliquidConfig() connector.Config {
	useTestnet := getEnv("HYPERLIQUID_TESTNET", "true") == "true"

	var baseURL string
	if envURL := os.Getenv("HYPERLIQUID_BASE_URL"); envURL != "" {
		baseURL = envURL
	} else if useTestnet {
		baseURL = "https://api.hyperliquid-testnet.xyz"
	} else {
		baseURL = "https://api.hyperliquid.xyz"
		println("⚠️  WARNING: Using Hyperliquid MAINNET - real money at risk!")
	}

	return &hyperliquid.Config{
		BaseURL:        baseURL,
		AccountAddress: mustGetEnv("HYPERLIQUID_ACCOUNT_ADDRESS"),
		PrivateKey:     mustGetEnv("HYPERLIQUID_PRIVATE_KEY"),
		VaultAddress:   getEnv("HYPERLIQUID_VAULT_ADDRESS", ""),
		UseTestnet:     useTestnet,
	}
}

func getParadexConfig() connector.Config {
	return &paradex.Config{
		BaseURL:        getEnv("PARADEX_BASE_URL", "https://api.testnet.paradex.trade/consumer"),
		WebSocketURL:   getEnv("PARADEX_WS_URL", "wss://ws.testnet.paradex.trade/v1"),
		StarknetRPC:    getEnv("PARADEX_STARKNET_RPC", "https://starknet-sepolia.public.blastapi.io"),
		AccountAddress: mustGetEnv("PARADEX_ACCOUNT_ADDRESS"),
		EthPrivateKey:  mustGetEnv("PARADEX_ETH_PRIVATE_KEY"),
		Network:        getEnv("PARADEX_NETWORK", "testnet"),
	}
}

func getBybitConfig() connector.Config {
	return &bybit.Config{
		APIKey:    mustGetEnv("BYBIT_API_KEY"),
		APISecret: mustGetEnv("BYBIT_API_SECRET"),
		IsTestnet: getEnv("BYBIT_TESTNET", "true") == "true",
	}
}

func getGateSpotConfig() connector.Config {
	useTestnet := getEnv("GATE_TESTNET", "true") == "true"

	var baseURL string
	if envURL := os.Getenv("GATE_BASE_URL"); envURL != "" {
		baseURL = envURL
	} else if useTestnet {
		baseURL = "https://api-testnet.gateapi.io/api/v4"
	} else {
		baseURL = "https://api.gateio.ws/api/v4"
		println("⚠️  WARNING: Using Gate.io MAINNET - real money at risk!")
	}

	return &gatespot.Config{
		APIKey:          mustGetEnv("GATE_API_KEY"),
		APISecret:       mustGetEnv("GATE_API_SECRET"),
		BaseURL:         baseURL,
		UseTestnet:      useTestnet,
		DefaultSlippage: 0.005, // 0.5% default slippage
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic("required environment variable not set: " + key)
	}
	return value
}
