package spot_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
)

// market_data_test.go — Verifies that prices, order books, and klines flow
// from the connector through the SDK pipeline and are readable via wisp.Spot().
//
// The SDK's batch ingestor runs on a 30-second timer with an immediate
// collection at startup. Tests use Eventually to wait for data to arrive
// through the automatic pipeline rather than manually triggering collection.

var _ = Describe("Spot — Market Data via SDK", func() {
	var runner *connector_test.SpotTestRunner

	BeforeEach(func() {
		var err error
		runner, err = connector_test.NewSpotTestRunner(
			connector_test.GetTestSpotConnectorName(),
			connector_test.GetSpotConnectorConfig(),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if runner != nil {
			runner.Cleanup()
		}
	})

	// ────────────────────────────────────────────────────────────────
	// Price
	// ────────────────────────────────────────────────────────────────

	Context("Price", func() {
		It("should flow from connector into Spot().Price()", func() {
			pair := testPair()
			exchange := runner.ExchangeName()

			runner.WatchPair(pair)

			Eventually(func() bool {
				price, ok := runner.Spot().Price(exchange, pair)
				return ok && price.IsPositive()
			}, "60s", "1s").Should(BeTrue(),
				"Spot().Price() should return a positive price after ingestor runs")

			price, _ := runner.Spot().Price(exchange, pair)
			connector_test.LogSuccess("Spot().Price(%s, %s) = %s", exchange, pair, price.String())
		})

		It("should return prices for a pair across all exchanges via Spot().Prices()", func() {
			pair := testPair()
			exchange := runner.ExchangeName()

			runner.WatchPair(pair)

			Eventually(func() bool {
				prices := runner.Spot().Prices(pair)
				if len(prices) == 0 {
					return false
				}
				p, ok := prices[exchange]
				return ok && p.IsPositive()
			}, "60s", "1s").Should(BeTrue(),
				"Spot().Prices() should include our exchange")

			prices := runner.Spot().Prices(pair)
			connector_test.LogSuccess("Spot().Prices(%s): %d exchanges", pair, len(prices))
		})

		It("should not find an unwatched pair", func() {
			pair := testPair()
			exchange := runner.ExchangeName()

			// Never called WatchPair
			_, ok := runner.Spot().Price(exchange, pair)
			Expect(ok).To(BeFalse(), "Unwatched pair should not be in the SDK")
		})
	})

	// ────────────────────────────────────────────────────────────────
	// Order Book
	// ────────────────────────────────────────────────────────────────

	Context("OrderBook", func() {
		It("should flow from connector into Spot().OrderBook()", func() {
			pair := testPair()
			exchange := runner.ExchangeName()

			runner.WatchPair(pair)

			Eventually(func() bool {
				ob, ok := runner.Spot().OrderBook(exchange, pair)
				return ok && ob != nil && len(ob.Bids) > 0 && len(ob.Asks) > 0
			}, "60s", "1s").Should(BeTrue(),
				"Spot().OrderBook() should have bids and asks after ingestor runs")

			ob, _ := runner.Spot().OrderBook(exchange, pair)

			for _, bid := range ob.Bids {
				Expect(bid.Price.IsPositive()).To(BeTrue())
				Expect(bid.Quantity.IsPositive()).To(BeTrue())
			}
			for _, ask := range ob.Asks {
				Expect(ask.Price.IsPositive()).To(BeTrue())
				Expect(ask.Quantity.IsPositive()).To(BeTrue())
			}

			connector_test.LogSuccess("Spot().OrderBook(): %d bids, %d asks",
				len(ob.Bids), len(ob.Asks))
		})

		It("best bid should be below best ask", func() {
			pair := testPair()
			exchange := runner.ExchangeName()

			runner.WatchPair(pair)

			Eventually(func() bool {
				ob, ok := runner.Spot().OrderBook(exchange, pair)
				return ok && ob != nil && len(ob.Bids) > 0 && len(ob.Asks) > 0
			}, "60s", "1s").Should(BeTrue())

			ob, _ := runner.Spot().OrderBook(exchange, pair)
			bestBid := ob.Bids[0].Price
			bestAsk := ob.Asks[0].Price

			Expect(bestBid.LessThan(bestAsk)).To(BeTrue(),
				"Best bid (%s) should be below best ask (%s)",
				bestBid.String(), bestAsk.String())

			connector_test.LogSuccess("Spread: bid=%s, ask=%s", bestBid.String(), bestAsk.String())
		})
	})

	// ────────────────────────────────────────────────────────────────
	// Klines
	// ────────────────────────────────────────────────────────────────

	Context("Klines", func() {
		It("should flow from connector into Spot().Klines()", func() {
			pair := testPair()
			exchange := runner.ExchangeName()

			runner.WatchPair(pair)

			Eventually(func() bool {
				klines := runner.Spot().Klines(exchange, pair, "1h", 10)
				return len(klines) > 0
			}, "60s", "1s").Should(BeTrue(),
				"Spot().Klines() should return candles after ingestor runs")

			klines := runner.Spot().Klines(exchange, pair, "1h", 10)
			for _, k := range klines {
				Expect(k.Open > 0).To(BeTrue(), "Kline open should be positive")
				Expect(k.Close > 0).To(BeTrue(), "Kline close should be positive")
			}

			connector_test.LogSuccess("Spot().Klines(): %d candles", len(klines))
		})
	})

	// ────────────────────────────────────────────────────────────────
	// Watchlist lifecycle
	// ────────────────────────────────────────────────────────────────

	Context("WatchPair / UnwatchPair", func() {
		It("should accept unwatch without error after data has arrived", func() {
			pair := testPair()
			exchange := runner.ExchangeName()

			runner.WatchPair(pair)

			// Wait for data to arrive
			Eventually(func() bool {
				_, ok := runner.Spot().Price(exchange, pair)
				return ok
			}, "60s", "1s").Should(BeTrue())

			// Unwatch — the ingestor will no longer collect for this pair.
			// Existing data may linger in the store, but new fetches stop.
			runner.UnwatchPair(pair)

			connector_test.LogSuccess("WatchPair/UnwatchPair lifecycle completed")
		})
	})
})
