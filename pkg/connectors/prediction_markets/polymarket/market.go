package polymarket

import (
	"context"
	"fmt"
)

func (p *polymarket) GetMarket(slug string) {
	ctx := context.Background()
	market, err := p.gammaClient.GetMarket(ctx, slug)

	if err != nil {
		return
	}

	fmt.Printf("Market: %+v\n", market)
}
