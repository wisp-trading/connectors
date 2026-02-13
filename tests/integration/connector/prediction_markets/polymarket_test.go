package prediction_markets_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
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

	Describe("Market Data", func() {
		Context("FetchMarket", func() {
			It("should fetch market details", func() {
				conn := runner.GetPredictionMarketConnector()
				slug := "us-strike-on-somalia-by-february-14"

				conn.GetMarket(slug)

			})
		})
	})

	Describe("WebSocket Subscriptions", func() {
		Context("StartWebSocket", func() {
			It("should establish WebSocket connection", func() {
				conn := runner.GetWebSocketCapable()
				err := conn.StartWebSocket()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("Subscribing to market events", func() {
			It("should subscribe to order book updates and receive data", func() {
				conn := runner.GetWebSocketCapable()
				err := conn.StartWebSocket()
				defer func(conn prediction.WebSocketConnector) {
					err := conn.StopWebSocket()
					if err != nil {
						connector_test.LogError("Failed to stop WebSocket connection: %v", err)
						return
					}
				}(conn)
				Expect(err).ToNot(HaveOccurred())

				market, err := conn.GetRecurringMarket("btc-updown-15m", prediction.Recurrence15Min)

				err = conn.SubscribeOrderBook(market)
				Expect(err).ToNot(HaveOccurred())

				orderbookChannels := conn.GetOrderbookChannels()
				Expect(orderbookChannels).ToNot(BeNil(), "Market orderbookChannels should not be nil")

				priceChangeChannels := conn.GetPriceChangeChannels()
				Expect(priceChangeChannels).ToNot(BeNil(), "Market priceChangeChannels should not be nil")

				outcome, exists := orderbookChannels[market.Slug]
				Expect(exists).To(BeTrue(), "Market book channel should exist for subscribed market")
				Expect(outcome).ToNot(BeNil(), "Market book channel should not be nil")

				priceChangeChannel, exists := priceChangeChannels[market.Slug]
				Expect(exists).To(BeTrue(), "Market price change channel should exist for subscribed market")
				Expect(priceChangeChannel).ToNot(BeNil(), "Market price change channel should not be nil")
				Expect(exists).To(BeTrue())

				// Use helper to verify order book
				orderBook := runner.VerifyOrderBookData(outcome, 30*time.Second)
				Expect(orderBook.Bids).ToNot(BeNil())
				Expect(orderBook.Asks).ToNot(BeNil())

				priceChange := runner.VerifyPriceChangeData(priceChangeChannel, 30*time.Second)
				Expect(priceChange).ToNot(BeNil())
				Expect(len(priceChange)).To(
					BeNumerically(">", 0),
					"Should receive at least one price change update",
				)
				Expect(priceChange[0].Outcome.Pair.Market()).To(Equal(market.Slug))
				Expect(priceChange[0].Outcome.Pair.Outcome()).To(BeElementOf("Up", "Down"))

				connector_test.LogSuccess(
					"Received order book data for market %s with %d bids and %d asks",
					market.MarketId,
					len(orderBook.Bids),
					len(orderBook.Asks),
				)
				connector_test.LogSuccess(
					"Received order book data for market %s",
					market.MarketId,
				)
				time.Sleep(5 * time.Second) // Sleep to allow any additional messages to be received before test ends
			})
		})
	})
})
