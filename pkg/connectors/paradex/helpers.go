package paradex

import (
	"time"

	"github.com/trishtzy/go-paradex/models"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (p *paradex) convertParadexOrder(paradexOrder *models.ResponsesOrderResp) connector.Order {
	// Parse decimal values with error handling
	quantity, _ := numerical.NewFromString(paradexOrder.Size)
	price, _ := numerical.NewFromString(paradexOrder.Price)
	avgPrice, _ := numerical.NewFromString(paradexOrder.AvgFillPrice)
	remainingQty, _ := numerical.NewFromString(paradexOrder.RemainingSize)

	// Calculate filled quantity
	filledQty := quantity.Sub(remainingQty)

	// Convert timestamps
	createdAt := time.Unix(paradexOrder.CreatedAt/1000, (paradexOrder.CreatedAt%1000)*1000000)
	updatedAt := time.Unix(paradexOrder.LastUpdatedAt/1000, (paradexOrder.LastUpdatedAt%1000)*1000000)

	return connector.Order{
		ID:            paradexOrder.ID,
		ClientOrderID: paradexOrder.ClientID,
		Symbol:        paradexOrder.Market,
		Side:          p.convertOrderSide(paradexOrder.Side.ResponsesOrderSide),
		Type:          p.convertOrderType(paradexOrder.Type.ResponsesOrderType),
		Status:        p.convertOrderStatus(paradexOrder.Status.ResponsesOrderStatus),
		Quantity:      quantity,
		Price:         price,
		FilledQty:     filledQty,
		RemainingQty:  remainingQty,
		AvgPrice:      avgPrice,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}

func (p *paradex) convertOrderType(orderType models.ResponsesOrderType) connector.OrderType {
	switch orderType {
	case "MARKET":
		return connector.OrderTypeMarket
	case "LIMIT":
		return connector.OrderTypeLimit
	case "STOP_LIMIT":
		return connector.OrderTypeStopLimit
	case "STOP_MARKET":
		return connector.OrderTypeStopMarket
	case "TAKE_PROFIT_LIMIT":
		return connector.OrderTypeTakeProfitLimit
	case "TAKE_PROFIT_MARKET":
		return connector.OrderTypeTakeProfitMarket
	default:
		return connector.OrderTypeLimit
	}
}

func (p *paradex) convertOrderStatus(status models.ResponsesOrderStatus) connector.OrderStatus {
	switch status {
	case "OPEN":
		return connector.OrderStatusOpen
	case "FILLED":
		return connector.OrderStatusFilled
	case "PARTIALLY_FILLED":
		return connector.OrderStatusPartiallyFilled
	case "CANCELLED":
		return connector.OrderStatusCanceled
	case "REJECTED":
		return connector.OrderStatusRejected
	case "EXPIRED":
		return connector.OrderStatusExpired
	default:
		return connector.OrderStatusOpen
	}
}

func (p *paradex) convertOrderSide(side models.ResponsesOrderSide) connector.OrderSide {
	switch side {
	case "BUY":
		return connector.OrderSideBuy
	case "SELL":
		return connector.OrderSideSell
	default:
		return connector.OrderSideUnknown
	}
}
