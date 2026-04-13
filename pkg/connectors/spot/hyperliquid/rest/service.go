package rest

import (
	hyperliquid "github.com/sonirico/go-hyperliquid"
	hyperliquid2 "github.com/wisp-trading/connectors/pkg/connectors/adaptors/hyperliquid"
)

// SpotTradingService handles spot order placement and management.
// Spot orders use the same Hyperliquid API as perp orders but with
// spot asset indices (10000 + token index in SpotMeta).
type SpotTradingService interface {
	// Limit orders
	PlaceBuyLimitOrder(coin string, size, price float64) (hyperliquid.OrderStatus, error)
	PlaceSellLimitOrder(coin string, size, price float64) (hyperliquid.OrderStatus, error)

	// Market orders
	PlaceBuyMarketOrder(coin string, size, slippage float64) (hyperliquid.OrderStatus, error)
	PlaceSellMarketOrder(coin string, size, slippage float64) (hyperliquid.OrderStatus, error)

	// Cancel
	CancelOrderByID(coin string, orderID int64) (*hyperliquid.APIResponse[hyperliquid.CancelOrderResponse], error)

	// Bulk
	PlaceBulkOrders(orders []hyperliquid.CreateOrderRequest) (*hyperliquid.APIResponse[hyperliquid.OrderResponse], error)
}

// SpotMarketDataService handles spot market data queries.
type SpotMarketDataService interface {
	// FetchSpotMeta returns metadata for all spot tokens.
	FetchSpotMeta() (*hyperliquid.SpotMeta, error)

	// FetchSpotMetaAndAssetCtxs returns spot metadata with current pricing context.
	FetchSpotMetaAndAssetCtxs() (map[string]any, error)

	// FetchSpotUserState returns spot balances for the given address.
	FetchSpotUserState(address string) (*hyperliquid.UserState, error)

	// FetchL2Book fetches the L2 order book for a spot asset.
	FetchL2Book(coin string) (*hyperliquid.L2Book, error)

	// GetCandles fetches historical candlestick data.
	GetCandles(coin, interval string, startTime, endTime int64) ([]hyperliquid.Candle, error)

	// GetOpenOrders fetches open orders for the given address.
	GetOpenOrders(user string) ([]hyperliquid.OpenOrder, error)

	// GetUserFills fetches trade fills for the given address.
	GetUserFills(user string) ([]hyperliquid.Fill, error)

	// GetOrderByOid fetches a specific order by its order ID.
	GetOrderByOid(user string, oid int64) (*hyperliquid.OpenOrder, error)
}

// spotTradingService implementation
type spotTradingService struct {
	client     hyperliquid2.ExchangeClient
	infoClient hyperliquid2.InfoClient
}

// NewSpotTradingService creates a new spot trading service
func NewSpotTradingService(
	client hyperliquid2.ExchangeClient,
	infoClient hyperliquid2.InfoClient,
) SpotTradingService {
	return &spotTradingService{
		client:     client,
		infoClient: infoClient,
	}
}

// spotMarketDataService implementation
type spotMarketDataService struct {
	infoClient hyperliquid2.InfoClient
}

// NewSpotMarketDataService creates a new spot market data service
func NewSpotMarketDataService(infoClient hyperliquid2.InfoClient) SpotMarketDataService {
	return &spotMarketDataService{
		infoClient: infoClient,
	}
}
