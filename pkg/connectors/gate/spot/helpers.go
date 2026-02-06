package spot

import (
	"strings"
	"time"

	"github.com/gate/gateapi-go/v7"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (g *gateSpot) GetSpotSymbol(pair portfolio.Pair) string {
	// Gate.io uses "BTC_USDT" format for currency pairs
	return pair.Base().Symbol() + "_" + pair.Quote().Symbol()
}

func (g *gateSpot) getWispPair(currencyPair string) portfolio.Pair {
	// Gate.io uses "BTC_USDT" format for currency pairs
	parts := strings.Split(currencyPair, "_")
	if len(parts) != 2 {
		return portfolio.Pair{}
	}

	return portfolio.NewPair(portfolio.NewAsset(parts[0]), portfolio.NewAsset(parts[1]))
}

// Helper function to convert connector.OrderSide to Gate.io side string for requests
func convertSide(side connector.OrderSide) string {
	if side == connector.OrderSideBuy {
		return "buy"
	}
	return "sell"
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
		Pair:         g.getWispPair(gateOrder.CurrencyPair),
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

// convertGateTradeToConnector converts a Gate.io Trade to connector.Trade
func (g *gateSpot) convertGateTradeToConnector(trade *gateapi.Trade) connector.Trade {
	price, _ := numerical.NewFromString(trade.Price)
	quantity, _ := numerical.NewFromString(trade.Amount)
	fee, _ := numerical.NewFromString(trade.Fee)

	// Parse timestamp - CreateTimeMs is string representation of milliseconds
	timestamp := g.timeProvider.Now()
	if trade.CreateTimeMs != "" {
		if ms, err := numerical.NewFromString(trade.CreateTimeMs); err == nil {
			milliseconds := ms.IntPart()
			timestamp = time.UnixMilli(milliseconds)
		}
	}

	return connector.Trade{
		ID:        trade.Id,
		OrderID:   trade.OrderId,
		Pair:      g.getWispPair(trade.CurrencyPair),
		Exchange:  g.GetConnectorInfo().Name,
		Price:     price,
		Quantity:  quantity,
		Side:      g.convertGateOrderSide(trade.Side),
		IsMaker:   trade.Role == "maker",
		Fee:       fee,
		Timestamp: timestamp,
	}
}
