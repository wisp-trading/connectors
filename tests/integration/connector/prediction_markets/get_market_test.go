package prediction_markets_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector"
)

var _ = Describe("Prediction Market Data Tests", func() {
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

				market, err := conn.GetMarket("will-jesus-christ-return-before-2027")
				if err != nil {
					return
				}

				Expect(market.MarketID).To(Equal(prediction.MarketID("0x0b4cc3b739e1dfe5d73274740e7308b6fb389c5af040c3a174923d928d134bee")))
				Expect(market.Slug).To(Equal("will-jesus-christ-return-before-2027"))
				Expect(market.Exchange).To(Equal(connector.ExchangeName("polymarket")))
				Expect(market.OutcomeType).To(Equal(prediction.OutcomeTypeBinary))
				Expect(market.Active).To(BeTrue())
				Expect(market.Closed).To(BeFalse())
				Expect(market.ResolutionTime.Unix()).To(Equal(int64(1798675200)))
			})
		})

		Context("Market is recurring", func() {
			It("Should fetch the current market data", func() {
				conn := runner.GetPredictionMarketConnector()

				market, err := conn.GetRecurringMarket("btc-updown-15m", prediction.Recurrence15Min)
				Expect(err).ToNot(HaveOccurred())

				// Verify market ID exists
				Expect(market.MarketID).ToNot(BeEmpty())

				// Verify recurrence interval is set correctly
				Expect(market.RecurringMarket.RecurrenceInterval).To(Equal(prediction.Recurrence15Min))

				// Verify EndDate aligns to 15-minute boundary
				endTimestamp := market.ResolutionTime.Unix()
				intervalSeconds := int64(15 * 60) // 15 minutes in seconds
				Expect(endTimestamp%intervalSeconds).To(
					Equal(int64(0)),
					"EndDate should align to 15-minute boundary",
				)

				// Verify EndDate is in the near future (within 15 minutes)
				now := time.Now()
				timeUntilClose := market.ResolutionTime.Sub(now)
				Expect(timeUntilClose).To(BeNumerically(">", 0), "Market should not be closed yet")
				Expect(timeUntilClose).To(BeNumerically("<=", 15*time.Minute), "Market should close within 15 minutes")

				// Verify market is active
				Expect(market.Active).To(BeTrue())
				Expect(market.Closed).To(BeFalse())

				// Verify outcomes exist (should be binary: UP/DOWN)
				Expect(market.Outcomes).To(HaveLen(2))

				// Verify each outcome has an OutcomeId (asset ID for orderbook)
				for _, outcome := range market.Outcomes {
					Expect(outcome.OutcomeID).ToNot(BeEmpty())
					Expect(outcome.Pair.Symbol()).ToNot(BeEmpty())
				}
			})
		})
	})
})
