package gamma

import (
	"context"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/gamma"
)

func (c *gammaClient) GetMarketBySlug(ctx context.Context, slug string) (*gamma.Market, error) {
	return c.client.MarketBySlug(ctx, &gamma.MarketBySlugRequest{
		Slug: slug,
	})
}
