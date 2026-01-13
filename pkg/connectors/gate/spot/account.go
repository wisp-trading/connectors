package spot

import (
	"fmt"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/kronos/numerical"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/gate/gateapi-go/v7"
)

// GetAccountBalance retrieves the account balance for spot trading
func (g *gateSpot) GetAccountBalance() (*connector.AccountBalance, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	client, err := g.spotClient.GetSpotApi()
	if err != nil {
		return nil, fmt.Errorf("failed to get spot API client: %w", err)
	}

	ctx := g.spotClient.GetAPIContext()

	// Get spot account balances
	accounts, _, err := client.SpotApi.ListSpotAccounts(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get spot accounts: %w", err)
	}

	totalBalance := numerical.Zero()
	availableBalance := numerical.Zero()

	// Sum up all currency balances (converted to USDT equivalent would be ideal)
	for _, account := range accounts {
		if account.Currency == "USDT" {
			available, _ := numerical.NewFromString(account.Available)
			locked, _ := numerical.NewFromString(account.Locked)
			totalBalance = totalBalance.Add(available).Add(locked)
			availableBalance = availableBalance.Add(available)
		}
	}

	balance := &connector.AccountBalance{
		TotalBalance:     totalBalance,
		AvailableBalance: availableBalance,
		UsedMargin:       numerical.Zero(), // Spot doesn't use margin
		UnrealizedPnL:    numerical.Zero(), // Spot doesn't have unrealized PnL
		Currency:         "USDT",
		UpdatedAt:        g.timeProvider.Now(),
	}

	return balance, nil
}

// GetPositions returns empty slice for spot (no positions in spot trading)
func (g *gateSpot) GetPositions() ([]connector.Position, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	// Spot trading doesn't have positions
	return []connector.Position{}, nil
}

// GetOpenOrders retrieves all open orders
func (g *gateSpot) GetOpenOrders() ([]connector.Order, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	client, err := g.spotClient.GetSpotApi()
	if err != nil {
		return nil, fmt.Errorf("failed to get spot API client: %w", err)
	}

	ctx := g.spotClient.GetAPIContext()

	// Get all open orders (empty currency pair means all)
	orders, _, err := client.SpotApi.ListOrders(ctx, "", "open", &gateapi.ListOrdersOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders: %w", err)
	}

	var connectorOrders []connector.Order
	for _, order := range orders {
		connectorOrder := g.convertGateOrderToConnector(&order)
		connectorOrders = append(connectorOrders, connectorOrder)
	}

	return connectorOrders, nil
}

// convertGateOrderToConnector converts Gate.io order to connector.Order
func (g *gateSpot) convertGateOrderToConnector(gateOrder *gateapi.Order) connector.Order {
	qty, _ := numerical.NewFromString(gateOrder.Amount)
	price, _ := numerical.NewFromString(gateOrder.Price)
	filledQty, _ := numerical.NewFromString(gateOrder.FilledAmount)
	avgPrice, _ := numerical.NewFromString(gateOrder.AvgDealPrice)

	// Parse timestamps (Gate uses string timestamps)
	var createdAt, updatedAt time.Time
	if gateOrder.CreateTimeMs > 0 {
		createdAt = time.UnixMilli(gateOrder.CreateTimeMs)
	}
	if gateOrder.UpdateTimeMs > 0 {
		updatedAt = time.UnixMilli(gateOrder.UpdateTimeMs)
	}

	return connector.Order{
		ID:           gateOrder.Id,
		Symbol:       g.parseSymbol(gateOrder.CurrencyPair).Symbol(),
		Status:       g.convertGateOrderStatus(gateOrder.Status),
		Side:         g.convertGateOrderSide(gateOrder.Side),
		Type:         g.convertGateOrderType(gateOrder.Type),
		Quantity:     qty,
		Price:        price,
		FilledQty:    filledQty,
		RemainingQty: qty.Sub(filledQty),
		AvgPrice:     avgPrice,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}

// convertGateOrderStatus converts Gate.io order status to connector.OrderStatus
func (g *gateSpot) convertGateOrderStatus(status string) connector.OrderStatus {
	switch status {
	case "open":
		return connector.OrderStatusOpen
	case "closed":
		return connector.OrderStatusFilled
	case "cancelled":
		return connector.OrderStatusCanceled
	default:
		return connector.OrderStatusOpen
	}
}

// convertGateOrderSide converts Gate.io order side to connector.OrderSide
func (g *gateSpot) convertGateOrderSide(side string) connector.OrderSide {
	switch side {
	case "buy":
		return connector.OrderSideBuy
	case "sell":
		return connector.OrderSideSell
	default:
		return connector.OrderSideUnknown
	}
}

// convertGateOrderType converts Gate.io order type to connector.OrderType
func (g *gateSpot) convertGateOrderType(orderType string) connector.OrderType {
	switch orderType {
	case "limit":
		return connector.OrderTypeLimit
	case "market":
		return connector.OrderTypeMarket
	default:
		return connector.OrderTypeLimit
	}
}

// formatSymbol converts symbol from Kronos format to Gate format
// Example: ETH -> ETH_USDT
func (g *gateSpot) formatSymbol(symbol string) string {
	return symbol + "_USDT"
}

// parseSymbol converts Gate symbol format to Kronos format
// Example: ETH_USDT -> ETH
func (g *gateSpot) parseSymbol(gateSymbol string) portfolio.Asset {
	// Simple implementation - split on underscore and take first part
	for i, c := range gateSymbol {
		if c == '_' {
			return portfolio.NewAsset(gateSymbol[:i])
		}
	}
	return portfolio.NewAsset(gateSymbol)
}
