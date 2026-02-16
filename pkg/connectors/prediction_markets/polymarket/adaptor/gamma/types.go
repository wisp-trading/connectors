package gamma

import (
	"context"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/gamma"
)

// GammaClient defines the interface for interacting with the Polymarket Gamma metadata service.
type GammaClient interface {
	GetMarketBySlug(ctx context.Context, slug string) (*gamma.Market, error)
}
