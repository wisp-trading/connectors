package gamma

import (
	"context"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/gamma"
)

// GammaClient defines the interface for interacting with the Polymarket Gamma metadata service.
type GammaClient interface {
	GetMarketBySlug(ctx context.Context, slug string) (*gamma.Market, error)
	MarketsAll(ctx context.Context, req *gamma.MarketsRequest) ([]gamma.Market, error)
	EventsAll(ctx context.Context, req *gamma.EventsRequest) ([]gamma.Event, error)
}
