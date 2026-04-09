package gamma

import (
	"context"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/gamma"
)

// GetMarketBySlug retrieves a single market by slug
func (c *gammaClient) GetMarketBySlug(ctx context.Context, slug string) (*gamma.Market, error) {
	return c.client.MarketBySlug(ctx, &gamma.MarketBySlugRequest{
		Slug: slug,
	})
}

// MarketsAll retrieves all markets with automatic pagination
func (c *gammaClient) MarketsAll(ctx context.Context, req *gamma.MarketsRequest) ([]gamma.Market, error) {
	return c.client.MarketsAll(ctx, req)
}

// EventsAll retrieves all events with automatic pagination
func (c *gammaClient) EventsAll(ctx context.Context, req *gamma.EventsRequest) ([]gamma.Event, error) {
	return c.client.EventsAll(ctx, req)
}
