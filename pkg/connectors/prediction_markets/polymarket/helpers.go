package polymarket

import (
	"encoding/json"
	"fmt"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/ws"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/gamma"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (p *polymarket) convertToOrderBook(msg ws.OrderbookEvent, market prediction.Market) connector.OrderBook {
	pair := prediction.NewPredictionPair(market.Slug, msg.AssetID, getQuoteAsset())

	orderbook := connector.OrderBook{
		Pair: pair.Pair,
		Bids: []connector.PriceLevel{},
		Asks: []connector.PriceLevel{},
	}

	bids, err := p.convertPriceLevels(msg.Bids)
	if err != nil {
		fmt.Printf("Error converting bids: %v\n", err)
		return orderbook
	}

	asks, err := p.convertPriceLevels(msg.Asks)

	if err != nil {
		fmt.Printf("Error converting asks: %v\n", err)
		return orderbook
	}

	orderbook.Bids = bids
	orderbook.Asks = asks

	return orderbook
}

// convertPriceLevels converts websocket order book levels to connector.PriceLevel slice
func (p *polymarket) convertPriceLevels(levels []ws.OrderbookLevel) ([]connector.PriceLevel, error) {
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

// convertToPriceChange converts a websocket price change event to a prediction.PriceChange struct
func (p *polymarket) convertToPriceChange(msg ws.PriceChangeEvent, market prediction.Market) (prediction.PriceChange, error) {
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
