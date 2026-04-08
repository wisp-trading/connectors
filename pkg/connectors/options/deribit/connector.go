package deribit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wisp-trading/connectors/pkg/connectors/options/deribit/adaptor"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	optionsConnector "github.com/wisp-trading/sdk/pkg/types/connector/options"
	"github.com/wisp-trading/sdk/pkg/types/logging"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/temporal"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// deribitOptions implements the options.Connector interface
type deribitOptions struct {
	client        adaptor.Client
	config        *Config
	appLogger     logging.ApplicationLogger
	tradingLogger logging.TradingLogger
	timeProvider  temporal.TimeProvider
	initialized   bool
	mu            sync.RWMutex
}

// Ensure deribitOptions implements the options.Connector interface at compile time
var _ optionsConnector.Connector = (*deribitOptions)(nil)

// NewDeribitOptions creates a new Deribit options connector
func NewDeribitOptions(
	client adaptor.Client,
	appLogger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
	timeProvider temporal.TimeProvider,
) optionsConnector.Connector {
	return &deribitOptions{
		client:        client,
		appLogger:     appLogger,
		tradingLogger: tradingLogger,
		timeProvider:  timeProvider,
		initialized:   false,
	}
}

// NewConfig returns a new config instance
func (d *deribitOptions) NewConfig() connector.Config {
	return &Config{}
}

// Initialize configures the connector with API credentials
func (d *deribitOptions) Initialize(config connector.Config) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.initialized {
		return fmt.Errorf("connector already initialized")
	}

	deribitConfig, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for Deribit Options connector: expected *Config, got %T", config)
	}

	// Validate config
	if err := deribitConfig.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Configure the HTTP client
	if err := d.client.Configure(deribitConfig.ClientID, deribitConfig.ClientSecret, deribitConfig.BaseURL); err != nil {
		return fmt.Errorf("failed to configure client: %w", err)
	}

	d.config = deribitConfig
	d.initialized = true

	d.appLogger.Info("Deribit Options connector initialized successfully")

	return nil
}

// IsInitialized returns whether the connector has been initialized
func (d *deribitOptions) IsInitialized() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.initialized
}

// GetConnectorInfo returns metadata about the connector
func (d *deribitOptions) GetConnectorInfo() *connector.Info {
	return &connector.Info{
		Name:             "deribit_options",
		TradingEnabled:   true,
		WebSocketEnabled: true,
	}
}

// SupportsTradingOperations returns true as Deribit supports trading
func (d *deribitOptions) SupportsTradingOperations() bool {
	return true
}

// SupportsRealTimeData returns true as Deribit supports WebSocket
func (d *deribitOptions) SupportsRealTimeData() bool {
	return true
}

// GetExpirations returns all available expirations for a pair
func (d *deribitOptions) GetExpirations(pair portfolio.Pair) ([]time.Time, error) {
	d.mu.RLock()
	if !d.initialized {
		d.mu.RUnlock()
		return nil, fmt.Errorf("connector not initialized")
	}
	d.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return d.fetchExpirations(ctx, pair)
}

// GetStrikes returns available strikes for an expiration
func (d *deribitOptions) GetStrikes(pair portfolio.Pair, expiration time.Time) ([]float64, error) {
	d.mu.RLock()
	if !d.initialized {
		d.mu.RUnlock()
		return nil, fmt.Errorf("connector not initialized")
	}
	d.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return d.fetchStrikes(ctx, pair, expiration)
}

// GetOptionData returns mark price and Greeks for a specific option
func (d *deribitOptions) GetOptionData(contract optionsConnector.OptionContract) (optionsConnector.OptionData, error) {
	d.mu.RLock()
	if !d.initialized {
		d.mu.RUnlock()
		return optionsConnector.OptionData{}, fmt.Errorf("connector not initialized")
	}
	d.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Format contract to Deribit instrument name
	instrumentName := formatInstrumentName(contract)

	return d.fetchOptionData(ctx, instrumentName)
}

// GetExpirationData returns all option data for an expiration
func (d *deribitOptions) GetExpirationData(pair portfolio.Pair, expiration time.Time) (
	map[float64]map[string]optionsConnector.OptionData,
	error,
) {
	d.mu.RLock()
	if !d.initialized {
		d.mu.RUnlock()
		return nil, fmt.Errorf("connector not initialized")
	}
	d.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get strikes for this expiration
	strikes, err := d.fetchStrikes(ctx, pair, expiration)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch strikes: %w", err)
	}

	// Build result map: map[strike][callOrPut]OptionData
	result := make(map[float64]map[string]optionsConnector.OptionData)

	// Fetch data for each strike and option type
	for _, strike := range strikes {
		result[strike] = make(map[string]optionsConnector.OptionData)

		// Fetch CALL data
		callContract := optionsConnector.OptionContract{
			Pair:       pair,
			Strike:     strike,
			Expiration: expiration,
			OptionType: "CALL",
		}
		callInstrument := formatInstrumentName(callContract)
		callData, err := d.fetchOptionData(ctx, callInstrument)
		if err != nil {
			d.appLogger.Warn("failed to fetch CALL option data", map[string]interface{}{
				"instrument": callInstrument,
				"error":      err,
			})
		} else {
			result[strike]["CALL"] = callData
		}

		// Fetch PUT data
		putContract := optionsConnector.OptionContract{
			Pair:       pair,
			Strike:     strike,
			Expiration: expiration,
			OptionType: "PUT",
		}
		putInstrument := formatInstrumentName(putContract)
		putData, err := d.fetchOptionData(ctx, putInstrument)
		if err != nil {
			d.appLogger.Warn("failed to fetch PUT option data", map[string]interface{}{
				"instrument": putInstrument,
				"error":      err,
			})
		} else {
			result[strike]["PUT"] = putData
		}
	}

	return result, nil
}

// PlaceLimitOrder places a limit order
func (d *deribitOptions) PlaceLimitOrder(pair portfolio.Pair, side connector.OrderSide, quantity, price numerical.Decimal) (*connector.OrderResponse, error) {
	d.mu.RLock()
	if !d.initialized {
		d.mu.RUnlock()
		return nil, fmt.Errorf("connector not initialized")
	}
	d.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return d.placeLimitOrder(ctx, pair, side, quantity, price)
}

// PlaceMarketOrder places a market order
func (d *deribitOptions) PlaceMarketOrder(pair portfolio.Pair, side connector.OrderSide, quantity numerical.Decimal) (*connector.OrderResponse, error) {
	d.mu.RLock()
	if !d.initialized {
		d.mu.RUnlock()
		return nil, fmt.Errorf("connector not initialized")
	}
	d.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return d.placeMarketOrder(ctx, pair, side, quantity)
}

// CancelOrder cancels an existing order
func (d *deribitOptions) CancelOrder(orderID string, pair ...portfolio.Pair) (*connector.CancelResponse, error) {
	d.mu.RLock()
	if !d.initialized {
		d.mu.RUnlock()
		return nil, fmt.Errorf("connector not initialized")
	}
	d.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return d.cancelOrder(ctx, orderID)
}

// GetOpenOrders returns all open orders
func (d *deribitOptions) GetOpenOrders(pair ...portfolio.Pair) ([]connector.Order, error) {
	d.mu.RLock()
	if !d.initialized {
		d.mu.RUnlock()
		return nil, fmt.Errorf("connector not initialized")
	}
	d.mu.RUnlock()

	// TODO: Implement open orders fetching
	return nil, fmt.Errorf("not implemented")
}

// GetOrderStatus returns the status of a specific order
func (d *deribitOptions) GetOrderStatus(orderID string, pair ...portfolio.Pair) (*connector.Order, error) {
	d.mu.RLock()
	if !d.initialized {
		d.mu.RUnlock()
		return nil, fmt.Errorf("connector not initialized")
	}
	d.mu.RUnlock()

	// TODO: Implement order status fetching
	return nil, fmt.Errorf("not implemented")
}

// GetBalances returns all account balances
func (d *deribitOptions) GetBalances() ([]connector.AssetBalance, error) {
	d.mu.RLock()
	if !d.initialized {
		d.mu.RUnlock()
		return nil, fmt.Errorf("connector not initialized")
	}
	d.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	account, err := d.fetchAccountSummary(ctx)
	if err != nil {
		return nil, err
	}

	// For options trading on Deribit, typically only USDT is used
	usdt := portfolio.NewAsset("USDT")
	balance := d.buildAssetBalance(account, usdt)
	if balance == nil {
		return []connector.AssetBalance{}, nil
	}

	return []connector.AssetBalance{*balance}, nil
}

// GetBalance returns the balance for a specific asset
func (d *deribitOptions) GetBalance(asset portfolio.Asset) (*connector.AssetBalance, error) {
	d.mu.RLock()
	if !d.initialized {
		d.mu.RUnlock()
		return nil, fmt.Errorf("connector not initialized")
	}
	d.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	account, err := d.fetchAccountSummary(ctx)
	if err != nil {
		return nil, err
	}

	return d.buildAssetBalance(account, asset), nil
}

// GetTradingHistory returns recent trades
func (d *deribitOptions) GetTradingHistory(pair portfolio.Pair, limit int) ([]connector.Trade, error) {
	d.mu.RLock()
	if !d.initialized {
		d.mu.RUnlock()
		return nil, fmt.Errorf("connector not initialized")
	}
	d.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// For options, we fetch a sample instrument to get trading history
	// In production, would need to aggregate across multiple pairs/expirations
	instrumentName := formatInstrumentName(optionsConnector.OptionContract{
		Pair: pair,
		// Note: This would need full contract details in real usage
	})

	trades, err := d.fetchUserTrades(ctx, instrumentName, limit)
	if err != nil {
		return nil, err
	}

	return convertTradeToConnectorTrade(trades, pair), nil
}

