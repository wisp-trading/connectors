package connector

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/wisp-trading/connectors/pkg/connectors/bybit/perp"
	gatespot "github.com/wisp-trading/connectors/pkg/connectors/gate/spot"
	"github.com/wisp-trading/connectors/pkg/connectors/hyperliquid"
	"github.com/wisp-trading/connectors/pkg/connectors/paradex"
	polymarketconfig "github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/config"
	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
)

func init() {
	// Get the directory of this source file
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		dir := filepath.Dir(filename)
		// Load .env from the connector directory
		_ = godotenv.Load(filepath.Join(dir, ".env"))
	}
	// Fallback to current directory
	_ = godotenv.Load()
}

// ========================================
// SPOT CONNECTOR CONFIGURATION
// ========================================
const (
	testSpotConnectorName = types.GateSpot
	testSpotSymbol        = "ETH"
)

// GetTestSpotConnectorName returns the spot connector name for tests
func GetTestSpotConnectorName() connector.ExchangeName {
	return testSpotConnectorName
}

// GetSpotSymbol returns the spot symbol for tests
func GetSpotSymbol() string {
	return testSpotSymbol
}

// GetSpotConnectorConfig returns the config for the spot connector under test
func GetSpotConnectorConfig() connector.Config {
	return getGateSpotConfig()
}

// ========================================
// PERP CONNECTOR CONFIGURATION
// ========================================
const (
	testPerpConnectorName = types.Hyperliquid
	testPerpSymbol        = "ETH"
)

// GetTestPerpConnectorName returns the perp connector name for tests
func GetTestPerpConnectorName() connector.ExchangeName {
	return testPerpConnectorName
}

// GetPerpSymbol returns the perp symbol for tests
func GetPerpSymbol() string {
	return testPerpSymbol
}

// GetPerpConnectorConfig returns the config for the perp connector under test
func GetPerpConnectorConfig() connector.Config {
	return getHyperliquidConfig()
}

// ========================================
// TRADING TEST FLAGS
// ========================================
const (
	enableSpotTradingTests = true
	enablePerpTradingTests = true
)

// ========================================
// INDIVIDUAL CONNECTOR CONFIGS
// ========================================

// getHyperliquidConfig creates a Hyperliquid config from environment variables
func getHyperliquidConfig() *hyperliquid.Config {
	testnet, _ := strconv.ParseBool(os.Getenv("HYPERLIQUID_TESTNET"))
	return &hyperliquid.Config{
		AccountAddress: os.Getenv("HYPERLIQUID_ACCOUNT_ADDRESS"),
		PrivateKey:     os.Getenv("HYPERLIQUID_PRIVATE_KEY"),
		VaultAddress:   os.Getenv("HYPERLIQUID_VAULT_ADDRESS"),
		BaseURL:        os.Getenv("HYPERLIQUID_BASE_URL"),
		UseTestnet:     testnet,
	}
}

// getParadexConfig creates a Paradex config from environment variables
func getParadexConfig() *paradex.Config {
	return &paradex.Config{
		AccountAddress: os.Getenv("PARADEX_ACCOUNT_ADDRESS"),
		EthPrivateKey:  os.Getenv("PARADEX_ETH_PRIVATE_KEY"),
		Network:        os.Getenv("PARADEX_NETWORK"),
		BaseURL:        os.Getenv("PARADEX_BASE_URL"),
		WebSocketURL:   os.Getenv("PARADEX_WS_URL"),
		StarknetRPC:    os.Getenv("PARADEX_STARKNET_RPC"),
	}
}

// getBybitConfig creates a Bybit config from environment variables
func getBybitConfig() *perp.Config {
	testnet, _ := strconv.ParseBool(os.Getenv("BYBIT_TESTNET"))
	return &perp.Config{
		APIKey:    os.Getenv("BYBIT_API_KEY"),
		APISecret: os.Getenv("BYBIT_API_SECRET"),
		IsTestnet: testnet,
	}
}

// getGateSpotConfig creates a Gate.io Spot config from environment variables
func getGateSpotConfig() *gatespot.Config {
	testnet, _ := strconv.ParseBool(os.Getenv("GATE_TESTNET"))
	return &gatespot.Config{
		APIKey:     os.Getenv("GATE_API_KEY"),
		APISecret:  os.Getenv("GATE_API_SECRET"),
		UseTestnet: testnet,
	}
}

// ========================================
// PREDICTION MARKET CONNECTOR CONFIGURATION
// ========================================
const (
	testPredictionMarketConnectorName = types.Polymarket
)

var (
	testPredictionMarketTokenIDs = []string{
		"70308501195956323589797156800521969197358506986152833648253437673484286051597",
		"77385393614263738045377442390679465888613338149607876972436340566574399345181",
	}
)

// GetTestPredictionMarketConnectorName returns the prediction market connector name for tests
func GetTestPredictionMarketConnectorName() connector.ExchangeName {
	return testPredictionMarketConnectorName
}

// GetPredictionMarketTokenIDs returns the token ID for tests
func GetPredictionMarketTokenIDs() []string {
	return testPredictionMarketTokenIDs
}

// GetPredictionMarketConnectorConfig returns the config for the prediction market connector under test
func GetPredictionMarketConnectorConfig() connector.Config {
	return getPolymarketConfig()
}

// getPolymarketConfig creates a Polymarket config from environment variables
func getPolymarketConfig() *polymarketconfig.Config {
	chainID, _ := strconv.Atoi(os.Getenv("POLYMARKET_CHAIN_ID"))
	signatureType, _ := strconv.Atoi(os.Getenv("POLYMARKET_SIGNATURE_TYPE"))

	return &polymarketconfig.Config{
		APIKey:        os.Getenv("POLYMARKET_API_KEY"),
		APISecret:     os.Getenv("POLYMARKET_API_SECRET"),
		Passphrase:    os.Getenv("POLYMARKET_PASSPHRASE"),
		PrivateKey:    os.Getenv("POLYMARKET_PRIVATE_KEY"),
		FunderAddress: os.Getenv("POLYMARKET_FUNDER_ADDRESS"),
		BaseURL:       os.Getenv("POLYMARKET_BASE_URL"),
		GammaURL:      os.Getenv("POLYMARKET_GAMMA_URL"),
		WebSocketURL:  os.Getenv("POLYMARKET_WEBSOCKET_URL"),
		ChainID:       int64(chainID),
		SignatureType: signatureType,
	}
}
