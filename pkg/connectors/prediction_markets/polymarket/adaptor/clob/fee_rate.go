package clob

import (
	"context"
	"fmt"
)

func (c *polymarketClient) getFeeRate(ctx context.Context, assetID string) (int, error) {
	endpoint := fmt.Sprintf("%s?asset_id=%s", getFeeRateEndpoint, assetID)

	var response struct {
		FeeRateBps int `json:"fee_rate_bps"`
	}

	if err := c.doRequest(ctx, "GET", endpoint, nil, &response); err != nil {
		return 0, fmt.Errorf("failed to fetch fee rate: %w", err)
	}

	return response.FeeRateBps, nil
}
