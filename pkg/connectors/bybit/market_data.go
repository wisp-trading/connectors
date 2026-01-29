package bybit

import (
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
)

func (b *bybit) FetchKlines(symbol, interval string, limit int) ([]connector.Kline, error) {
	return b.marketData.FetchKlines(symbol, interval, limit)
}

func (b *bybit) FetchPrice(symbol string) (*connector.Price, error) {
	return b.marketData.FetchPrice(symbol)
}

func (b *bybit) FetchOrderBook(symbol portfolio.Asset, depth int) (*connector.OrderBook, error) {
	return b.marketData.FetchOrderBook(symbol.Symbol()+"USDT", depth)
}

func (b *bybit) FetchRecentTrades(symbol string, limit int) ([]connector.Trade, error) {
	return b.marketData.FetchRecentTrades(symbol, limit)
}

func (b *bybit) FetchFundingRate(asset portfolio.Asset) (*perp.FundingRate, error) {
	return b.marketData.FetchFundingRate(asset.Symbol() + "USDT")
}
