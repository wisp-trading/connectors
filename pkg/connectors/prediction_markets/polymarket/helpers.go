package polymarket

import (
	"fmt"

	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/websocket"
	"github.com/wisp-trading/sdk/pkg/types/connector"
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
