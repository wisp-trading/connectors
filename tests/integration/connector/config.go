package connector

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/joho/godotenv"
	deribitconfig "github.com/wisp-trading/connectors/pkg/connectors/options/deribit"
	hyperliquid "github.com/wisp-trading/connectors/pkg/connectors/perps/hyperliquid"
	hyperliquidspot "github.com/wisp-trading/connectors/pkg/connectors/spot/hyperliquid"
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
	testSpotConnectorName = types.HyperliquidSpot
	testSpotSymbol        = "HYPE"
	testSpotQuote         = "USDC"
)

// GetTestSpotConnectorName returns the spot connector name for tests
func GetTestSpotConnectorName() connector.ExchangeName {
	return testSpotConnectorName
}

// GetSpotSymbol returns the spot symbol for tests
func GetSpotSymbol() string {
	return testSpotSymbol
}

// GetSpotQuote returns the spot quote currency for tests
func GetSpotQuote() string {
	return testSpotQuote
}

// GetSpotConnectorConfig returns the config for the spot connector under test
func GetSpotConnectorConfig() connector.Config {
	return getHyperliquidSpotConfig()
}

// IsSpotTradingEnabled returns whether live trading tests should run
func IsSpotTradingEnabled() bool {
	enabled, _ := strconv.ParseBool(os.Getenv("ENABLE_SPOT_TRADING_TESTS"))
	return enabled
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

// getHyperliquidSpotConfig creates a Hyperliquid Spot config from environment variables.
// Shares the same credentials as the perp connector — same API, same keys.
func getHyperliquidSpotConfig() *hyperliquidspot.Config {
	testnet, _ := strconv.ParseBool(os.Getenv("HYPERLIQUID_TESTNET"))
	return &hyperliquidspot.Config{
		AccountAddress: os.Getenv("HYPERLIQUID_ACCOUNT_ADDRESS"),
		PrivateKey:     os.Getenv("HYPERLIQUID_PRIVATE_KEY"),
		VaultAddress:   os.Getenv("HYPERLIQUID_VAULT_ADDRESS"),
		BaseURL:        os.Getenv("HYPERLIQUID_BASE_URL"),
		UseTestnet:     testnet,
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
	signatureType, _ := strconv.Atoi(os.Getenv("POLYMARKET_SIGNATURE_TYPE"))

	return &polymarketconfig.Config{
		PrivateKey:        os.Getenv("POLYMARKET_PRIVATE_KEY"),
		PolymarketAddress: os.Getenv("POLYMARKET_ADDRESS"),
		SignatureType:     polymarketconfig.SignatureType(signatureType),
		PolygonRPCURL:     os.Getenv("POLYGON_RPC_URL"),
	}
}

// ========================================
// OPTIONS CONNECTOR CONFIGURATION
// ========================================
const (
	testOptionsConnectorName = types.DeribitOptions
	testOptionsSymbol        = "BTC"
)

// GetTestOptionsConnectorName returns the options connector name for tests
func GetTestOptionsConnectorName() connector.ExchangeName {
	return testOptionsConnectorName
}

// GetOptionsConnectorConfig returns the config for the options connector under test
func GetOptionsConnectorConfig() connector.Config {
	return getDeribitOptionsConfig()
}

// getDeribitOptionsConfig creates a Deribit Options config from environment variables
func getDeribitOptionsConfig() *deribitconfig.Config {
	useTestnet, _ := strconv.ParseBool(os.Getenv("DERIBIT_TESTNET"))

	// Set defaults if not in environment
	baseURL := os.Getenv("DERIBIT_BASE_URL")
	if baseURL == "" {
		baseURL = "https://test.deribit.com/api/v2"
	}
	wsURL := os.Getenv("DERIBIT_WS_URL")
	if wsURL == "" {
		wsURL = "wss://test.deribit.com/ws/api/v2"
	}

	// If not explicitly set, default to testnet for integration tests
	if os.Getenv("DERIBIT_TESTNET") == "" {
		useTestnet = true
	}

	return &deribitconfig.Config{
		ClientID:        os.Getenv("DERIBIT_CLIENT_ID"),
		ClientSecret:    os.Getenv("DERIBIT_CLIENT_SECRET"),
		BaseURL:         baseURL,
		WebSocketURL:    wsURL,
		UseTestnet:      useTestnet,
		DefaultSlippage: 0.001, // 0.1% default slippage
	}
}
