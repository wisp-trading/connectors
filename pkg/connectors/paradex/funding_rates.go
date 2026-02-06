package paradex

import (
	"fmt"

	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
)

func (p *paradex) SubscribeFundingRates(pair portfolio.Pair) error {
	//TODO implement me
	panic("implement me")
}

func (p *paradex) UnsubscribeFundingRates(pair portfolio.Pair) error {
	//TODO implement me
	panic("implement me")
}

func (p *paradex) FundingRateUpdates() <-chan perp.FundingRate {
	//TODO implement me
	panic("implement me")
}

func (p *paradex) FetchCurrentFundingRates() (map[portfolio.Pair]perp.FundingRate, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *paradex) FetchFundingRate(pair portfolio.Pair) (*perp.FundingRate, error) {
	return nil, fmt.Errorf("not implemented")

}

func (p *paradex) FetchHistoricalFundingRates(pair portfolio.Pair, startTime, endTime int64) ([]perp.HistoricalFundingRate, error) {
	return nil, fmt.Errorf("not implemented")
}
