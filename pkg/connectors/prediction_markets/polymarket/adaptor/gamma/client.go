package gamma

import "github.com/GoPolymarket/polymarket-go-sdk/pkg/gamma"

// gammaClient implementation
type gammaClient struct {
	client gamma.Client
}

// NewGammaClient creates a new Gamma client wrapper
func NewGammaClient(client gamma.Client) GammaClient {
	return &gammaClient{
		client: client,
	}
}
