package connector_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Market Data Tests", func() {
	var runner *TestRunner

	BeforeEach(func() {
		var err error
		runner, err = NewTestRunner(testConnectorName, getConnectorConfig(testConnectorName))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if runner != nil {
			runner.Cleanup()
		}
	})

	Context("FetchPrice", func() {
		It("should fetch current price", func() {
			conn := runner.GetConnector()
			perpSymbol := conn.GetPerpSymbol(CreateAsset(testSymbol))

			price, err := conn.FetchPrice(perpSymbol)
			AssertNoError(err, "FetchPrice should succeed")
			Expect(price).ToNot(BeNil())
			Expect(price.Price.IsPositive()).To(BeTrue())

			LogSuccess("Price for %s: %s", perpSymbol, price.Price.String())
		})
	})

	Context("FetchKlines", func() {
		It("should fetch historical klines", func() {
			conn := runner.GetConnector()
			perpSymbol := conn.GetPerpSymbol(CreateAsset(testSymbol))

			klines, err := conn.FetchKlines(perpSymbol, "1m", 10)
			AssertNoError(err, "FetchKlines should succeed")
			Expect(klines).ToNot(BeEmpty())

			LogSuccess("Fetched %d klines for %s", len(klines), perpSymbol)
			if len(klines) > 0 {
				latest := klines[len(klines)-1]
				LogInfo("Latest: O=%.8f H=%.8f L=%.8f C=%.8f",
					latest.Open, latest.High,
					latest.Low, latest.Close)
			}
		})
	})

	Context("FetchOrderBook", func() {
		It("should fetch order book", func() {
			conn := runner.GetConnector()
			asset := CreateAsset(testSymbol)

			ob, err := conn.FetchOrderBook(asset, testInstrumentType, 10)
			AssertNoError(err, "FetchOrderBook should succeed")
			Expect(ob).ToNot(BeNil())
			Expect(ob.Bids).ToNot(BeEmpty())
			Expect(ob.Asks).ToNot(BeEmpty())

			LogSuccess("OrderBook fetched: %d bids, %d asks", len(ob.Bids), len(ob.Asks))
			LogInfo("Best Bid: %s @ %s", ob.Bids[0].Quantity.String(), ob.Bids[0].Price.String())
			LogInfo("Best Ask: %s @ %s", ob.Asks[0].Quantity.String(), ob.Asks[0].Price.String())
		})
	})

	Context("FetchRecentTrades", func() {
		It("should fetch recent trades", func() {
			conn := runner.GetConnector()
			perpSymbol := conn.GetPerpSymbol(CreateAsset(testSymbol))

			trades, err := conn.FetchRecentTrades(perpSymbol, 10)
			AssertNoError(err, "FetchRecentTrades should succeed")
			Expect(trades).ToNot(BeNil())

			LogSuccess("Fetched %d recent trades for %s", len(trades), perpSymbol)
		})
	})

	Context("FetchAvailableAssets", func() {
		It("should fetch available perpetual assets", func() {
			conn := runner.GetConnector()

			if !conn.SupportsPerpetuals() {
				Skip("Connector does not support perpetuals")
			}

			assets, err := conn.FetchAvailablePerpetualAssets()
			AssertNoError(err, "FetchAvailablePerpetualAssets should succeed")
			Expect(assets).ToNot(BeEmpty())

			LogSuccess("Found %d perpetual assets", len(assets))
		})

		It("should fetch available spot assets", func() {
			conn := runner.GetConnector()

			if !conn.SupportsSpot() {
				Skip("Connector does not support spot")
			}

			assets, err := conn.FetchAvailableSpotAssets()
			AssertNoError(err, "FetchAvailableSpotAssets should succeed")
			Expect(assets).ToNot(BeEmpty())

			LogSuccess("Found %d spot assets", len(assets))
		})
	})

	Context("FetchFundingRates", func() {
		It("should fetch funding rate for asset", func() {
			conn := runner.GetConnector()

			if !conn.SupportsFundingRates() {
				Skip("Connector does not support funding rates")
			}

			asset := CreateAsset(testSymbol)
			fr, err := conn.FetchFundingRate(asset)
			AssertNoError(err, "FetchFundingRate should succeed")
			Expect(fr).ToNot(BeNil())

			LogSuccess("Funding rate for %s: %s", testSymbol, fr.CurrentRate.String())
		})

		It("should fetch all current funding rates", func() {
			conn := runner.GetConnector()

			if !conn.SupportsFundingRates() {
				Skip("Connector does not support funding rates")
			}

			rates, err := conn.FetchCurrentFundingRates()
			AssertNoError(err, "FetchCurrentFundingRates should succeed")
			Expect(rates).ToNot(BeEmpty())

			LogSuccess("Fetched funding rates for %d assets", len(rates))
		})
	})
})
