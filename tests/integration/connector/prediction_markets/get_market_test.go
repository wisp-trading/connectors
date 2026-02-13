package prediction_markets_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
)

var _ = Describe("Prediction Market Connector Tests", func() {
	var runner *connector_test.PredictionMarketTestRunner

	BeforeEach(func() {
		var err error
		runner, err = connector_test.NewPredictionMarketTestRunner(
			connector_test.GetTestPredictionMarketConnectorName(),
			connector_test.GetPredictionMarketConnectorConfig(),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if runner != nil {
			runner.Cleanup()
		}
	})

	Describe("Getting Market Data", func() {
		Context("Getting a Market by slug", func() {
			It("should fetch a market by slug", func() {
				conn := runner.GetPredictionMarketConnector()

				market, err := conn.GetMarket("btc-updown-15m-1770952500")
				if err != nil {
					return
				}

				Expect(market.MarketId).To(Equal("1770952500"))
				Expect(market.Slug).To(Equal("btc-updown-15m-1770952500"))
				Expect(market.Exchange).To(Equal("Polymarket"))
				Expect(market.OutcomeType).To(Equal("BINARY"))
				Expect(market.Active).To(BeTrue())
				Expect(market.Closed).To(BeFalse())
				Expect(market.EndDate.Unix()).To(Equal(int64(1770952500)))

			})
		})
	})
})
