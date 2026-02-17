package polymarket

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/ws"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/gamma"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// parseOrderbookEvent converts a websocket order book event to a connector.OrderBook struct
func (p *polymarket) parseOrderbookEvent(msg ws.OrderbookEvent, market prediction.Market) connector.OrderBook {
	pair := prediction.NewPredictionPair(market.Slug, msg.AssetID, getQuoteAsset())

	orderbook := connector.OrderBook{
		Pair: pair.Pair,
		Bids: []connector.PriceLevel{},
		Asks: []connector.PriceLevel{},
	}

	bids, err := p.parseOrderbookLevel(msg.Bids)
	if err != nil {
		fmt.Printf("Error converting bids: %v\n", err)
		return orderbook
	}

	asks, err := p.parseOrderbookLevel(msg.Asks)

	if err != nil {
		fmt.Printf("Error converting asks: %v\n", err)
		return orderbook
	}

	orderbook.Bids = bids
	orderbook.Asks = asks

	return orderbook
}

// parseOrderbookLevel converts websocket order book levels to connector.PriceLevel slice
func (p *polymarket) parseOrderbookLevel(levels []ws.OrderbookLevel) ([]connector.PriceLevel, error) {
	result := make([]connector.PriceLevel, 0, len(levels))

	for _, level := range levels {
		price, err := numerical.NewFromString(level.Price)
		if err != nil {
			return nil, fmt.Errorf("failed to parse price %s: %w", level.Price, err)
		}

		quantity, err := numerical.NewFromString(level.Size)
		if err != nil {
			return nil, fmt.Errorf("failed to parse quantity %s: %w", level.Size, err)
		}

		result = append(result, connector.PriceLevel{
			Price:    price,
			Quantity: quantity,
		})
	}

	return result, nil
}

func (p *polymarket) parseOrderbook(
	msg clobtypes.OrderBookResponse,
	market prediction.Market,
	outcome prediction.Outcome,
) connector.OrderBook {
	orderbook := connector.OrderBook{
		Pair: outcome.Pair.Pair,
		Bids: []connector.PriceLevel{},
		Asks: []connector.PriceLevel{},
	}

	bids, err := p.parsePriceLevel(msg.Bids)
	if err != nil {
		fmt.Printf("Error converting bids: %v\n", err)
		return orderbook
	}

	asks, err := p.parsePriceLevel(msg.Asks)

	if err != nil {
		fmt.Printf("Error converting asks: %v\n", err)
		return orderbook
	}

	orderbook.Bids = bids
	orderbook.Asks = asks

	return orderbook
}

func (p *polymarket) parsePriceLevel(levels []clobtypes.PriceLevel) ([]connector.PriceLevel, error) {
	var priceLevels []connector.PriceLevel
	for _, level := range levels {
		price, err := numerical.NewFromString(level.Price)
		if err != nil {
			return []connector.PriceLevel{}, fmt.Errorf("failed to parse price %s: %w", level.Price, err)
		}

		quantity, err := numerical.NewFromString(level.Size)
		if err != nil {
			return []connector.PriceLevel{}, fmt.Errorf("failed to parse quantity %s: %w", level.Size, err)
		}

		priceLevels = append(priceLevels, connector.PriceLevel{
			Price:    price,
			Quantity: quantity,
		})
	}

	return priceLevels, nil
}

// parsePriceChange converts a websocket price change event to a prediction.PriceChange struct
func (p *polymarket) parsePriceChange(msg ws.PriceChangeEvent, market prediction.Market) (prediction.PriceChange, error) {
	outcome, err := market.FindOutcomeById(msg.AssetId)
	if err != nil {
		return prediction.PriceChange{}, err
	}

	price, err := numerical.NewFromString(msg.Price)
	if err != nil {
		fmt.Printf("Error converting price change: %v\n", err)
		return prediction.PriceChange{}, err
	}

	return prediction.PriceChange{
		Outcome: *outcome,
		Price:   price,
		Side:    msg.Side,
	}, nil
}

// parseTradeEvent converts a websocket trade event to a connector.Trade struct
func (p *polymarket) parseTrade(market prediction.Market, tradeEvent ws.TradeEvent) (connector.Trade, bool) {
	outcome, err := market.FindOutcomeById(tradeEvent.AssetID)
	if err != nil {
		p.appLogger.Error("Failed to find outcome for trade event: %v", err)
		return connector.Trade{}, true
	}

	price, err := numerical.NewFromString(tradeEvent.Price)
	if err != nil {
		p.appLogger.Error("Failed to parse price for trade event: %v", err)
		return connector.Trade{}, true
	}

	quantity, err := numerical.NewFromString(tradeEvent.Size)
	if err != nil {
		p.appLogger.Error("Failed to parse quantity for trade event: %v", err)
		return connector.Trade{}, true
	}

	timeStamp := time.Unix(tradeEvent.Timestamp, 0)

	trade := connector.Trade{
		ID:        tradeEvent.ID,
		Pair:      outcome.Pair.Pair,
		Price:     price,
		Quantity:  quantity,
		Timestamp: timeStamp,
	}
	return trade, false
}

// parseOrder converts a websocket order event to a connector.Order struct
func (p *polymarket) parseOrder(market prediction.Market, event ws.OrderEvent) (connector.Order, bool) {
	outcome, err := market.FindOutcomeById(event.AssetID)
	if err != nil {
		p.appLogger.Error("Failed to find outcome for order event: %v", err)
		return connector.Order{}, true
	}

	price, err := numerical.NewFromString(event.Price)
	if err != nil {
		p.appLogger.Error("Failed to parse price for order event: %v", err)
		return connector.Order{}, true
	}

	quantity, err := numerical.NewFromString(event.OriginalSize)
	if err != nil {
		p.appLogger.Error("Failed to parse quantity for order event: %v", err)
		return connector.Order{}, true
	}

	// Timestamp is a string in milliseconds, need to parse it
	timestampMs, err := strconv.ParseInt(event.Timestamp, 10, 64)
	if err != nil {
		p.appLogger.Error("Failed to parse timestamp for order event: %v", err)
		return connector.Order{}, true
	}
	timeStamp := time.UnixMilli(timestampMs)

	// Map Polymarket status to your connector status
	status := mapPolymarketStatus(event.Status)

	order := connector.Order{
		ID:        event.ID,
		Pair:      outcome.Pair.Pair,
		Price:     price,
		Quantity:  quantity,
		CreatedAt: timeStamp,
		UpdatedAt: timeStamp,
		Status:    status,
	}

	return order, false
}

func mapPolymarketStatus(pmStatus string) connector.OrderStatus {
	switch pmStatus {
	case "LIVE":
		return connector.OrderStatusOpen
	case "MATCHED":
		return connector.OrderStatusFilled
	case "CANCELED":
		return connector.OrderStatusCanceled
	default:
		return connector.OrderStatus(pmStatus) // fallback
	}
}

// parseClobTokenIds parses the JSON string of token IDs into a slice.
// Returns nil if the field is empty or cannot be parsed.
func parseClobTokenIds(m gamma.Market) []string {
	if m.ClobTokenIds == "" {
		return nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(m.ClobTokenIds), &ids); err != nil {
		return nil
	}
	return ids
}

// ParseOutcomes parses the JSON string of outcome labels into a slice.
// Returns nil if the field is empty or cannot be parsed.
func parseOutcomes(m gamma.Market) []string {
	if m.Outcomes == "" {
		return nil
	}
	var outcomes []string
	if err := json.Unmarshal([]byte(m.Outcomes), &outcomes); err != nil {
		return nil
	}
	return outcomes
}

func getQuoteAsset() portfolio.Asset {
	return portfolio.NewAsset("USDC")
}
