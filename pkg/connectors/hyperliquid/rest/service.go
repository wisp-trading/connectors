package rest

import (
	"fmt"

	hyperliquid "github.com/sonirico/go-hyperliquid"
	"github.com/wisp-trading/connectors/pkg/connectors/hyperliquid/adaptors"
)

// TradingService interface for trading operations
type TradingService interface {
	ModifyOrder(orderID int64, coin string, size, price float64, isBuy bool) (hyperliquid.OrderStatus, error)
	PlaceBulkOrders(orders []hyperliquid.CreateOrderRequest) (*hyperliquid.APIResponse[hyperliquid.OrderResponse], error)

	// Buy operations
	PlaceBuyLimitOrder(coin string, size, price float64) (hyperliquid.OrderStatus, error)
	PlaceBuyMarketOrder(coin string, size, slippage float64) (hyperliquid.OrderStatus, error)
	PlaceBuyStopLoss(coin string, size, triggerPrice float64) (hyperliquid.OrderStatus, error)
	PlaceBuyTakeProfit(coin string, size, triggerPrice float64) (hyperliquid.OrderStatus, error)
	PlaceBuyLimitOrderWithCustomRef(coin string, size, price float64, customRef string) (hyperliquid.OrderStatus, error)

	// Sell operations
	PlaceSellLimitOrder(coin string, size, price float64) (hyperliquid.OrderStatus, error)
	PlaceSellMarketOrder(coin string, size, slippage float64) (hyperliquid.OrderStatus, error)
	PlaceSellStopLoss(coin string, size, triggerPrice float64) (hyperliquid.OrderStatus, error)
	PlaceSellTakeProfit(coin string, size, triggerPrice float64) (hyperliquid.OrderStatus, error)
	PlaceSellLimitOrderWithCustomRef(coin string, size, price float64, customRef string) (hyperliquid.OrderStatus, error)

	// Close operations
	ClosePosition(coin string, size *float64, slippage float64) (hyperliquid.OrderStatus, error)
	CloseEntirePosition(coin string, slippage float64) (hyperliquid.OrderStatus, error)

	// Cancel operations
	CancelOrderByID(coin string, orderID int64) (*hyperliquid.APIResponse[hyperliquid.CancelOrderResponse], error)
	CancelOrderByCustomRef(coin, customRef string) (*hyperliquid.APIResponse[hyperliquid.CancelOrderResponse], error)
}

// tradingService implementation
type tradingService struct {
	client         adaptors.ExchangeClient
	infoClient     adaptors.InfoClient
	priceValidator PriceValidator
}

// NewTradingService creates a new trading service
func NewTradingService(
	client adaptors.ExchangeClient,
	infoClient adaptors.InfoClient,
	priceValidator PriceValidator,
) TradingService {
	return &tradingService{
		client:         client,
		infoClient:     infoClient,
		priceValidator: priceValidator,
	}
}

// Initialize loads asset metadata for price validation
// This should be called after both clients are configured
func (t *tradingService) Initialize() error {
	info, err := t.infoClient.GetInfo()
	if err != nil {
		return fmt.Errorf("failed to get info client: %w", err)
	}

	meta, err := info.Meta()
	if err != nil {
		return fmt.Errorf("failed to get meta: %w", err)
	}

	return t.priceValidator.LoadAssetInfo(meta)
}

func (t *tradingService) ModifyOrder(orderID int64, coin string, size, price float64, isBuy bool) (hyperliquid.OrderStatus, error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return hyperliquid.OrderStatus{}, fmt.Errorf("exchange not configured: %w", err)
	}

	oid := &orderID
	req := hyperliquid.ModifyOrderRequest{
		Oid: oid,
		Order: hyperliquid.CreateOrderRequest{
			Coin:       coin,
			IsBuy:      isBuy,
			Price:      price,
			Size:       size,
			ReduceOnly: false,
			OrderType: hyperliquid.OrderType{
				Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc},
			},
		},
	}
	return ex.ModifyOrder(req)
}

func (t *tradingService) PlaceBulkOrders(orders []hyperliquid.CreateOrderRequest) (*hyperliquid.APIResponse[hyperliquid.OrderResponse], error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return nil, fmt.Errorf("exchange not configured: %w", err)
	}
	return ex.BulkOrders(orders, nil)
}
