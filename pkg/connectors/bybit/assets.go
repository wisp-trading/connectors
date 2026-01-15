package bybit

import (
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector/perp"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
)

func (b *bybit) FetchAvailablePerpetualAssets() ([]portfolio.Asset, error) {
	return b.marketData.FetchAvailablePerpetualAssets()
}

func (b *bybit) FetchAvailableSpotAssets() ([]portfolio.Asset, error) {
	return b.marketData.FetchAvailableSpotAssets()
}

func (b *bybit) FetchContracts() ([]connector.ContractInfo, error) {
	return nil, fmt.Errorf("FetchContracts not implemented for Bybit")
}

func (b *bybit) FetchCurrentFundingRates() (map[portfolio.Asset]perp.FundingRate, error) {
	return b.marketData.FetchCurrentFundingRates()
}

func (b *bybit) FetchHistoricalFundingRates(symbol portfolio.Asset, startTime, endTime int64) ([]perp.HistoricalFundingRate, error) {
	return b.marketData.FetchHistoricalFundingRates(symbol.Symbol()+"USDT", startTime, endTime)
}

func (b *bybit) FetchRiskFundBalance(symbol string) (*connector.RiskFundBalance, error) {
	return nil, fmt.Errorf("FetchRiskFundBalance not implemented for Bybit")
}

func (b *bybit) SupportsFundingRates() bool {
	return true
}
