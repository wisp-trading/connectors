package polymarket

import (
	"context"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (p *polymarket) GetBalances() ([]connector.AssetBalance, error) {
	//TODO implement me
	panic("implement me")
}

func (p *polymarket) GetBalance(_ portfolio.Asset) (*connector.AssetBalance, error) {
	ctx := context.Background()

	response, err := p.orderManager.GetBalance(ctx)

	if err != nil {
		return nil, err
	}

	balance, err := numerical.NewFromString(response.Balance)

	if err != nil {
		return nil, err
	}

	// Polymarket returns balance in smallest units (1e6 = 1 USD)
	normalizedBalance := balance.Div(numerical.NewFromInt(1_000_000))

	return &connector.AssetBalance{
		Asset: portfolio.NewAsset("USD"),
		Free:  normalizedBalance,
	}, nil
}
