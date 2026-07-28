package connector

import (
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
	wispTypes "github.com/wisp-trading/sdk/pkg/types/wisp"
)

// PairMarketTestRunner is the store-aware surface for spot/perp integration tests.
// Data path under test:
//
//	WatchPair → CollectNow (batch ingestor) → MarketStore → wisp.Spot()/Perp() facade
type PairMarketTestRunner interface {
	BaseTestRunner

	ExchangeName() connector.ExchangeName
	GetWisp() wispTypes.Wisp

	// WatchPair registers the pair on the domain watchlist (required before CollectNow).
	WatchPair(pair portfolio.Pair)
	// CollectNow triggers batch ingestors: connector fetch → store write.
	CollectNow()

	// SDK* reads through the strategy-facing facade (same path strategies use).
	SDKPrice(pair portfolio.Pair) (numerical.Decimal, bool)
	SDKOrderBook(pair portfolio.Pair) (*connector.OrderBook, bool)
	SDKKlines(pair portfolio.Pair, interval string, limit int) []connector.Kline

	// Store* reads the domain MarketStore directly (prove ingestor wrote there).
	StorePrice(pair portfolio.Pair) *connector.Price
	StoreOrderBook(pair portfolio.Pair) *connector.OrderBook
	StoreKlines(pair portfolio.Pair, interval string, limit int) []connector.Kline
}
