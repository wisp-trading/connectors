package gamma

import (
	"context"
	"fmt"
)

func (c *gammaClient) GetMarket(ctx context.Context, slug string) (*MarketResponse, error) {
	marketEndpoint := fmt.Sprintf("%s%s", getMarketEndpoint, slug)

	var response []MarketResponse
	if err := c.doRequest(ctx, "GET", marketEndpoint, nil, &response); err != nil {
		return nil, fmt.Errorf("failed to get market: %w", err)
	}

	if len(response) == 0 {
		return nil, fmt.Errorf("no market data found")
	}

	market := &response[0]
	market.Parse()

	return market, nil
}
