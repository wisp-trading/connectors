package connectors

import (
	"strings"

	"github.com/backtesting-org/kronos-sdk/pkg/types/config"
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/live-trading/pkg/connectors/bybit"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid"
	"github.com/backtesting-org/live-trading/pkg/connectors/paradex"
	"github.com/backtesting-org/live-trading/pkg/connectors/types"
)

// AvailableConnectors is a map of all available exchange connectors with their config types
var AvailableConnectors = map[connector.ExchangeName]connector.Config{
	types.Paradex:     &paradex.Config{},
	types.Hyperliquid: &hyperliquid.Config{},
	types.Bybit:       &bybit.Config{},
}

type connectorAvailability struct{}

// NewConnectorAvailabilityService creates a new ConnectorAvailability service
func NewConnectorAvailabilityService() config.ConnectorAvailability {
	return &connectorAvailability{}
}

// IsAvailable checks if a connector is available for the given exchange
func (c *connectorAvailability) IsAvailable(exchange connector.ExchangeName) bool {
	normalizedExchange := connector.ExchangeName(strings.ToLower(string(exchange)))
	_, exists := AvailableConnectors[normalizedExchange]
	return exists
}

// ListAvailable returns a list of all available exchange names
func (c *connectorAvailability) ListAvailable() []connector.ExchangeName {
	exchanges := make([]connector.ExchangeName, 0, len(AvailableConnectors))
	for exchange := range AvailableConnectors {
		exchanges = append(exchanges, exchange)
	}
	return exchanges
}

// GetConfigType returns the config type for a given exchange
func (c *connectorAvailability) GetConfigType(exchange connector.ExchangeName) connector.Config {
	return AvailableConnectors[exchange]
}
