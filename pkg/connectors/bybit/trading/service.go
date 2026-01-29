package trading

import (
	"context"
	"fmt"
	"sync"

	bybit "github.com/bybit-exchange/bybit.go.api"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/temporal"
)

type Config struct {
	APIKey          string
	APISecret       string
	BaseURL         string
	IsTestnet       bool
	DefaultSlippage float64
}

type TradingService interface {
	Initialize(config *Config) error
	PlaceLimitOrder(symbol string, side connector.OrderSide, quantity, price numerical.Decimal) (*connector.OrderResponse, error)
	PlaceMarketOrder(symbol string, side connector.OrderSide, quantity numerical.Decimal) (*connector.OrderResponse, error)
	CancelOrder(symbol, orderID string) (*connector.CancelResponse, error)
	GetOpenOrders() ([]connector.Order, error)
	GetOrderStatus(orderID string) (*connector.Order, error)
	GetAccountBalance() (*connector.AccountBalance, error)
	GetPositions() ([]connector.Position, error)
	GetTradingHistory(symbol string, limit int) ([]connector.Trade, error)
}

type tradingService struct {
	client       *bybit.Client
	config       *Config
	timeProvider temporal.TimeProvider
	mu           sync.RWMutex
}

func NewTradingService(timeProvider temporal.TimeProvider) TradingService {
	return &tradingService{
		timeProvider: timeProvider,
	}
}

func (t *tradingService) Initialize(config *Config) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.client != nil {
		return fmt.Errorf("trading service already initialized")
	}

	t.config = config
	t.client = bybit.NewBybitHttpClient(config.APIKey, config.APISecret, bybit.WithBaseURL(config.BaseURL))
	return nil
}

func (t *tradingService) PlaceLimitOrder(symbol string, side connector.OrderSide, quantity, price numerical.Decimal) (*connector.OrderResponse, error) {
	t.mu.RLock()
	client := t.client
	t.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("trading service not initialized")
	}

	params := map[string]interface{}{
		"category":    "linear",
		"symbol":      symbol,
		"side":        string(side),
		"orderType":   "Limit",
		"qty":         quantity.String(),
		"price":       price.String(),
		"timeInForce": "GTC",
	}

	result, err := client.NewUtaBybitServiceWithParams(params).PlaceOrder(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to place limit order: %w", err)
	}

	var orderID string
	if result != nil && result.Result != nil {
		if ordData, ok := result.Result.(map[string]interface{}); ok {
			if id, ok := ordData["orderId"].(string); ok {
				orderID = id
			}
		}
	}

	return &connector.OrderResponse{
		OrderID:   orderID,
		Symbol:    symbol,
		Status:    connector.OrderStatusNew,
		Side:      side,
		Type:      connector.OrderTypeLimit,
		Quantity:  quantity,
		Price:     price,
		Timestamp: t.timeProvider.Now(),
	}, nil
}

func (t *tradingService) PlaceMarketOrder(symbol string, side connector.OrderSide, quantity numerical.Decimal) (*connector.OrderResponse, error) {
	t.mu.RLock()
	client := t.client
	t.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("trading service not initialized")
	}

	params := map[string]interface{}{
		"category":  "linear",
		"symbol":    symbol,
		"side":      string(side),
		"orderType": "Market",
		"qty":       quantity.String(),
	}

	result, err := client.NewUtaBybitServiceWithParams(params).PlaceOrder(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to place market order: %w", err)
	}

	var orderID string
	if result != nil && result.Result != nil {
		if ordData, ok := result.Result.(map[string]interface{}); ok {
			if id, ok := ordData["orderId"].(string); ok {
				orderID = id
			}
		}
	}

	return &connector.OrderResponse{
		OrderID:   orderID,
		Symbol:    symbol,
		Status:    connector.OrderStatusNew,
		Side:      side,
		Type:      connector.OrderTypeMarket,
		Quantity:  quantity,
		Timestamp: t.timeProvider.Now(),
	}, nil
}

func (t *tradingService) CancelOrder(symbol, orderID string) (*connector.CancelResponse, error) {
	t.mu.RLock()
	client := t.client
	t.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("trading service not initialized")
	}

	params := map[string]interface{}{
		"category": "linear",
		"symbol":   symbol,
		"orderId":  orderID,
	}

	_, err := client.NewUtaBybitServiceWithParams(params).CancelOrder(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	return &connector.CancelResponse{
		OrderID:   orderID,
		Symbol:    symbol,
		Status:    connector.OrderStatusCanceled,
		Timestamp: t.timeProvider.Now(),
	}, nil
}

func (t *tradingService) GetOpenOrders() ([]connector.Order, error) {
	t.mu.RLock()
	client := t.client
	t.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("trading service not initialized")
	}

	params := map[string]interface{}{
		"category": "linear",
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetOpenOrders(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders: %w", err)
	}

	var orders []connector.Order
	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if list, ok := resultData["list"].([]interface{}); ok {
				for _, item := range list {
					if orderData, ok := item.(map[string]interface{}); ok {
						order := t.parseOrder(orderData)
						orders = append(orders, order)
					}
				}
			}
		}
	}

	return orders, nil
}

func (t *tradingService) GetOrderStatus(orderID string) (*connector.Order, error) {
	t.mu.RLock()
	client := t.client
	t.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("trading service not initialized")
	}

	params := map[string]interface{}{
		"category": "linear",
		"orderId":  orderID,
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetOrderHistory(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get order status: %w", err)
	}

	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if list, ok := resultData["list"].([]interface{}); ok {
				if len(list) > 0 {
					if orderData, ok := list[0].(map[string]interface{}); ok {
						order := t.parseOrder(orderData)
						return &order, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("order not found")
}

func (t *tradingService) parseOrder(data map[string]interface{}) connector.Order {
	orderID, _ := data["orderId"].(string)
	symbol, _ := data["symbol"].(string)
	sideStr, _ := data["side"].(string)
	qtyStr, _ := data["qty"].(string)
	priceStr, _ := data["price"].(string)

	qty, _ := numerical.NewFromString(qtyStr)
	price, _ := numerical.NewFromString(priceStr)

	return connector.Order{
		ID:        orderID,
		Symbol:    symbol,
		Side:      connector.FromString(sideStr),
		Quantity:  qty,
		Price:     price,
		CreatedAt: t.timeProvider.Now(),
	}
}

func (t *tradingService) GetAccountBalance() (*connector.AccountBalance, error) {
	t.mu.RLock()
	client := t.client
	t.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("trading service not initialized")
	}

	params := map[string]interface{}{
		"accountType": "UNIFIED",
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetAccountWallet(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet balance: %w", err)
	}

	balance := &connector.AccountBalance{
		Currency:  "USDT",
		UpdatedAt: t.timeProvider.Now(),
	}

	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok && len(listData) > 0 {
				if accountData, ok := listData[0].(map[string]interface{}); ok {
					if totalEquity, ok := accountData["totalEquity"].(string); ok {
						if val, err := numerical.NewFromString(totalEquity); err == nil {
							balance.TotalBalance = val
						}
					}
					if availableBalance, ok := accountData["totalAvailableBalance"].(string); ok {
						if val, err := numerical.NewFromString(availableBalance); err == nil {
							balance.AvailableBalance = val
						}
					}
					if totalMarginBalance, ok := accountData["totalMarginBalance"].(string); ok {
						if val, err := numerical.NewFromString(totalMarginBalance); err == nil {
							balance.UsedMargin = balance.TotalBalance.Sub(val)
						}
					}
					if totalPerpUPL, ok := accountData["totalPerpUPL"].(string); ok {
						if val, err := numerical.NewFromString(totalPerpUPL); err == nil {
							balance.UnrealizedPnL = val
						}
					}
				}
			}
		}
	}

	return balance, nil
}

func (t *tradingService) GetPositions() ([]connector.Position, error) {
	t.mu.RLock()
	client := t.client
	t.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("trading service not initialized")
	}

	params := map[string]interface{}{
		"category":   "linear",
		"settleCoin": "USDT",
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetPositionList(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	var positions []connector.Position

	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok {
				for _, item := range listData {
					if posData, ok := item.(map[string]interface{}); ok {
						pos := t.parsePosition(posData)
						if !pos.Size.IsZero() {
							positions = append(positions, pos)
						}
					}
				}
			}
		}
	}

	return positions, nil
}

func (t *tradingService) parsePosition(data map[string]interface{}) connector.Position {
	pos := connector.Position{
		UpdatedAt: t.timeProvider.Now(),
	}

	if symbol, ok := data["symbol"].(string); ok {
		pos.Symbol = portfolio.NewAsset(symbol)
	}
	if side, ok := data["side"].(string); ok {
		pos.Side = connector.OrderSide(side)
	}
	if size, ok := data["size"].(string); ok {
		if val, err := numerical.NewFromString(size); err == nil {
			pos.Size = val
		}
	}
	if avgPrice, ok := data["avgPrice"].(string); ok {
		if val, err := numerical.NewFromString(avgPrice); err == nil {
			pos.EntryPrice = val
		}
	}
	if markPrice, ok := data["markPrice"].(string); ok {
		if val, err := numerical.NewFromString(markPrice); err == nil {
			pos.MarkPrice = val
		}
	}
	if unrealizedPnl, ok := data["unrealisedPnl"].(string); ok {
		if val, err := numerical.NewFromString(unrealizedPnl); err == nil {
			pos.UnrealizedPnL = val
		}
	}
	if cumRealisedPnl, ok := data["cumRealisedPnl"].(string); ok {
		if val, err := numerical.NewFromString(cumRealisedPnl); err == nil {
			pos.RealizedPnL = val
		}
	}
	if leverage, ok := data["leverage"].(string); ok {
		if val, err := numerical.NewFromString(leverage); err == nil {
			pos.Leverage = val
		}
	}
	if liqPrice, ok := data["liqPrice"].(string); ok {
		if val, err := numerical.NewFromString(liqPrice); err == nil {
			pos.LiquidationPrice = val
		}
	}

	return pos
}

func (t *tradingService) GetTradingHistory(symbol string, limit int) ([]connector.Trade, error) {
	t.mu.RLock()
	client := t.client
	t.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("trading service not initialized")
	}

	params := map[string]interface{}{
		"category": "linear",
		"symbol":   symbol,
		"limit":    limit,
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetTransactionLog(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch trading history: %w", err)
	}

	var trades []connector.Trade

	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok {
				for _, item := range listData {
					if tradeData, ok := item.(map[string]interface{}); ok {
						trade := connector.Trade{
							Symbol:    symbol,
							Timestamp: t.timeProvider.Now(),
						}

						if side, ok := tradeData["side"].(string); ok {
							trade.Side = connector.OrderSide(side)
						}
						if price, ok := tradeData["execPrice"].(string); ok {
							if val, err := numerical.NewFromString(price); err == nil {
								trade.Price = val
							}
						}
						if qty, ok := tradeData["execQty"].(string); ok {
							if val, err := numerical.NewFromString(qty); err == nil {
								trade.Quantity = val
							}
						}
						if execID, ok := tradeData["execId"].(string); ok {
							trade.ID = execID
						}

						trades = append(trades, trade)
					}
				}
			}
		}
	}

	return trades, nil
}
