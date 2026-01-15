package paradex

import (
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector/perp"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
)

func (p *paradex) FetchCurrentFundingRates() (map[portfolio.Asset]perp.FundingRate, error) {
	return nil, fmt.Errorf("current funding rates not needed for MM strategy")
}

func (p *paradex) FetchFundingRate(asset portfolio.Asset) (*perp.FundingRate, error) {
	return nil, fmt.Errorf("funding rate not needed for MM strategy")

}

func (p *paradex) FetchHistoricalFundingRates(asset portfolio.Asset, startTime, endTime int64) ([]perp.HistoricalFundingRate, error) {
	return nil, fmt.Errorf("historical funding rates not needed for MM strategy")
}
