package polymarket

import (
	"fmt"

	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/websocket"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// convertPriceLevels converts websocket order book levels to connector.PriceLevel slice
func convertPriceLevels(levels []websocket.PriceLevel) ([]connector.PriceLevel, error) {
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

func convertToOrderBook(msg *websocket.OrderBookMessage) connector.OrderBook {
	pair := prediction.NewPredictionPair(msg.Market, msg.AssetID, getQuoteAsset())

	orderbook := connector.OrderBook{
		Pair: pair.Pair,
		Bids: []connector.PriceLevel{},
		Asks: []connector.PriceLevel{},
	}

	bids, err := convertPriceLevels(msg.Bids)
	if err != nil {
		fmt.Printf("Error converting bids: %v\n", err)
		return orderbook
	}

	asks, err := convertPriceLevels(msg.Asks)

	if err != nil {
		fmt.Printf("Error converting asks: %v\n", err)
		return orderbook
	}

	orderbook.Bids = bids
	orderbook.Asks = asks

	return orderbook
}

func convertToPriceChange(market prediction.Market, msg *websocket.PriceChanges) []prediction.PriceChange {
	priceChanges := make([]prediction.PriceChange, 0, len(msg.PriceChange))

	for i, change := range msg.PriceChange {
		outcome, err := market.FindOutcomeById(change.AssetId)
		if err != nil {
			return nil
		}

		if change.Price == "" || change.Size == "" || change.Side == "" {
			fmt.Printf("Skipping price change %d due to missing fields\n", i)
			continue
		}

		priceChange := append(priceChanges, prediction.PriceChange{
			Outcome:   *outcome,
			Timestamp: msg.Timestamp,
			Price:     change.Price,
			Size:      change.Size,
			Side:      change.Side,
			BestBid:   change.BestBid,
			BestAsk:   change.BestAsk,
		})

		priceChanges = append(priceChanges, priceChange...)
	}

	return priceChanges
}

func getQuoteAsset() portfolio.Asset {
	return portfolio.NewAsset("USDC")
}
