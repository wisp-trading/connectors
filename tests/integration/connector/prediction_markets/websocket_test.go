package prediction_markets_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
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

				market, err := conn.GetMarket(slug)
				Expect(err).ToNot(HaveOccurred())
				Expect(market.MarketID).ToNot(BeEmpty())
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
					Expect(err).ToNot(HaveOccurred())

					err = conn.SubscribeOrderBook(market)
					Expect(err).ToNot(HaveOccurred())

					orderbookChannel := conn.GetOrderBookUpdates()
					Expect(orderbookChannel).ToNot(BeNil(), "Market orderbookChannels should not be nil")

					// Verify order book data
					orderBook := runner.VerifyOrderBookData(orderbookChannel, 3*time.Second)
					Expect(orderBook.Bids).ToNot(BeNil())
					Expect(orderBook.Asks).ToNot(BeNil())

					connector_test.LogSuccess(
						"Received order book data for market %s with %d bids and %d asks",
						market.MarketID,
						len(orderBook.Bids),
						len(orderBook.Asks),
					)

					time.Sleep(2 * time.Second) // Allow additional messages
				})

				It("should subscribe to price changes and receive data", func() {
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
					Expect(err).ToNot(HaveOccurred())

					err = conn.SubscribePriceChanges(market)
					Expect(err).ToNot(HaveOccurred())

					priceChangeChannels := conn.GetPriceChangeChannels()
					Expect(priceChangeChannels).ToNot(BeNil(), "Market priceChangeChannels should not be nil")

					priceChangeChannel, exists := priceChangeChannels[market.Slug]
					Expect(exists).To(BeTrue(), "Market price change channel should exist for subscribed market")
					Expect(priceChangeChannel).ToNot(BeNil(), "Market price change channel should not be nil")

					// Verify price change data
					priceChange, err := runner.VerifyPriceChangeData(priceChangeChannel, 3*time.Second)
					Expect(err).ToNot(HaveOccurred())
					Expect(priceChange.Outcome.Pair.Market()).To(Equal(market.Slug))
					Expect(priceChange.Outcome.Pair.Outcome()).To(BeElementOf("Up", "Down"))

					connector_test.LogSuccess(
						"Received price change data for market %s, outcome %s",
						market.MarketID,
						priceChange.Outcome.Pair.Outcome(),
					)

					time.Sleep(2 * time.Second) // Allow additional messages
				})
			})
		})

		Context("Subscribing to orders", func() {
			It("should subscribe to order updates without placing orders", func() {
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

				// Use recurring market to ensure active volume
				market, err := conn.GetRecurringMarket("btc-updown-5m", prediction.Recurrence5Min)
				Expect(err).ToNot(HaveOccurred())
				Expect(market.Outcomes).ToNot(BeEmpty(), "Market should have outcomes")

				err = conn.SubscribeOrders(market)
				Expect(err).ToNot(HaveOccurred())

				ordersChannel := conn.GetOrdersChannel()
				Expect(ordersChannel).ToNot(BeNil(), "Orders updates channel should not be nil")

				// Verify we can fetch orderbook data (validates market is active)
				orderBook, err := conn.FetchOrderBooks(market, market.Outcomes[0])
				Expect(err).ToNot(HaveOccurred())
				Expect(orderBook.Bids).ToNot(BeEmpty(), "Should have bids in orderbook")
				Expect(orderBook.Asks).ToNot(BeEmpty(), "Should have asks in orderbook")

				connector_test.LogSuccess(
					"Successfully subscribed to orders for market %s with %d bids and %d asks",
					market.MarketID,
					len(orderBook.Bids),
					len(orderBook.Asks),
				)

				time.Sleep(2 * time.Second) // Allow subscription to stabilize
			})
		})
	})
})
