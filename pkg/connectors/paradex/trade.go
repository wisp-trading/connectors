package paradex

import (
	"context"
	"fmt"
	"time"

	"github.com/wisp-trading/connectors/pkg/connectors/paradex/requests"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (p *paradex) PlaceLimitOrder(symbol string, side connector.OrderSide, quantity, price numerical.Decimal) (*connector.OrderResponse, error) {
	orderReq := requests.PlaceOrderParams{
		Market:    symbol + "-USD-PERP",
		Side:      string(side), // "BUY" or "SELL"
		Size:      quantity.String(),
		Price:     price.String(),
		OrderType: "LIMIT",
		ClientID:  "", // Optional
	}

	resp, err := p.paradexService.PlaceOrder(p.ctx, orderReq)
	if err != nil {
		return nil, err
	}

	return &connector.OrderResponse{
		OrderID:   resp.ID,
		Symbol:    resp.Market,
		Status:    connector.OrderStatusNew,
		Side:      side,
		Type:      connector.OrderTypeLimit,
		Quantity:  quantity,
		Price:     price,
		FilledQty: numerical.Zero(), // Update if fill info is available
		Timestamp: time.Now(),       // Or resp.CreatedAt if available
	}, nil
}

func (p *paradex) PlaceMarketOrder(symbol string, side connector.OrderSide, quantity numerical.Decimal) (*connector.OrderResponse, error) {
	orderReq := requests.PlaceOrderParams{
		Market:    symbol,
		Side:      string(side),
		Size:      quantity.String(),
		OrderType: "MARKET",
		Price:     "", // Empty for MARKET orders - will be omitted by PlaceOrder
		ClientID:  "", // Optional
	}

	resp, err := p.paradexService.PlaceOrder(p.ctx, orderReq)
	if err != nil {
		return nil, err
	}

	return &connector.OrderResponse{
		OrderID:   resp.ID,
		Symbol:    resp.Market,
		Status:    connector.OrderStatusNew,
		Side:      side,
		Type:      connector.OrderTypeMarket,
		Quantity:  quantity,
		FilledQty: numerical.Zero(),
		Timestamp: time.Now(),
	}, nil
}

func (p *paradex) CancelOrder(symbol, orderID string) (*connector.CancelResponse, error) {
	p.tradingLogger.OrderLifecycle("Cancelling order %s for symbol %s", orderID, symbol)
	err := p.paradexService.CancelOrder(p.ctx, orderID)
	if err != nil {
		p.tradingLogger.OrderLifecycle("Failed to cancel order %s for symbol %s: %v", orderID, symbol, err)
		return nil, err
	}

	return &connector.CancelResponse{
		OrderID: orderID,
		Symbol:  symbol,
		Status:  connector.OrderCancellationRequested,
	}, nil
}

func (p *paradex) GetOpenOrders() ([]connector.Order, error) {
	ctx := context.Background()
	paradexOrders, err := p.paradexService.GetOpenOrders(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders from paradex: %w", err)
	}

	// Convert paradex orders to connector format
	orders := make([]connector.Order, 0, len(paradexOrders))

	for _, paradexOrder := range paradexOrders {
		if paradexOrder == nil {
			continue
		}

		convertedOrder := p.convertParadexOrder(paradexOrder)
		orders = append(orders, convertedOrder)
	}

	p.appLogger.Debug("Retrieved %d open orders from paradex", len(orders))
	return orders, nil
}

func (p *paradex) GetOrderStatus(orderID string) (*connector.Order, error) {
	order, err := p.paradexService.GetOrder(p.ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order == nil {
		return nil, fmt.Errorf("order %s not found", orderID)
	}

	// Convert paradex order to connector format
	convertedOrder := p.convertParadexOrder(order)

	return &convertedOrder, nil
}

// GetRawOrder returns the raw paradex order details
func (p *paradex) GetRawOrder(ctx context.Context, orderID string) (interface{}, error) {
	return p.paradexService.GetOrder(ctx, orderID)
}

// GetRawOpenOrders returns open orders with optional market filter
func (p *paradex) GetRawOpenOrders(ctx context.Context, market *string) (interface{}, error) {
	return p.paradexService.GetOpenOrders(ctx, market)
}

// GetAccountSummary returns the account summary
func (p *paradex) GetAccountSummary(ctx context.Context) (interface{}, error) {
	return p.paradexService.GetAccount(ctx)
}

// GetBalances returns account balances
func (p *paradex) GetBalances(ctx context.Context) (interface{}, error) {
	return p.paradexService.GetAssetBalances(ctx)
}

// GetUserPositions returns current positions
func (p *paradex) GetUserPositions(ctx context.Context) (interface{}, error) {
	return p.paradexService.GetUserPositions(ctx)
}

// GetTradeHistory returns trade history with optional market filter
func (p *paradex) GetTradeHistory(ctx context.Context, market *string) (interface{}, error) {
	return p.paradexService.GetTradeHistory(ctx, market, nil)
}

func (p *paradex) GetTradingHistory(symbol string, limit int) ([]connector.Trade, error) {
	return nil, fmt.Errorf("trading history not needed for MM strategy")
}
