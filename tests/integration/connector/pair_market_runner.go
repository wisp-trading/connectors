package connector

import (
	"context"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
	wispTypes "github.com/wisp-trading/sdk/pkg/types/wisp"
)

// PairMarketTestRunner is the store-aware surface for spot/perp integration tests.
//
// Batch path:
//
//	WatchPair → CollectNow → MarketStore → wisp.Spot()/Perp()
//
// Realtime path:
//
//	WatchPair → StartRealtime → WS update → MarketStore → wisp.Spot()/Perp()
type PairMarketTestRunner interface {
	BaseTestRunner

	ExchangeName() connector.ExchangeName
	GetWisp() wispTypes.Wisp

	// WatchPair registers the pair on the domain watchlist (required before CollectNow / StartRealtime).
	WatchPair(pair portfolio.Pair)
	// CollectNow triggers batch ingestors: connector fetch → store write.
	CollectNow()

	// StartRealtime starts WS ingestors (subscribe + process channels into store).
	// StopRealtime stops them. Safe to call when HasWebSocketSupport is false (no-op / error).
	StartRealtime(ctx context.Context) error
	StopRealtime() error

	// SDK* reads through the strategy-facing facade (same path strategies use).
	SDKPrice(pair portfolio.Pair) (numerical.Decimal, bool)
	SDKOrderBook(pair portfolio.Pair) (*connector.OrderBook, bool)
	SDKKlines(pair portfolio.Pair, interval string, limit int) []connector.Kline

	// Store* reads the domain MarketStore directly (prove ingestor wrote there).
	StorePrice(pair portfolio.Pair) *connector.Price
	StoreOrderBook(pair portfolio.Pair) *connector.OrderBook
	StoreKlines(pair portfolio.Pair, interval string, limit int) []connector.Kline
}
