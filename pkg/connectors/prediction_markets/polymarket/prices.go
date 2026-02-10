package polymarket

import (
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
)

func (p *polymarket) FetchPrice(pair portfolio.Pair) (*connector.Price, error) {
	//TODO implement me
	panic("implement me")
}

func (p *polymarket) FetchKlines(pair portfolio.Pair, interval string, limit int) ([]connector.Kline, error) {
	//TODO implement me
	panic("implement me")
}
