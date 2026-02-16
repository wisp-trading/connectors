package prediction_markets_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

var _ = Describe("Prediction Market Order Placement Tests", func() {
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

	Describe("Placing an order", func() {
		Context("Order is a Limit Order", func() {
			It("Should place a $1 limit order successfully", func() {
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

				// Get current recurring market
				market, err := conn.GetMarket("will-jesus-christ-return-before-2027")
				//market, err := conn.GetRecurringMarket("btc-updown-15m", prediction.Recurrence15Min)
				Expect(err).ToNot(HaveOccurred())
				Expect(market.Outcomes).ToNot(BeEmpty(), "Market should have outcomes")

				// Subscribe to orderbook
				err = conn.SubscribeOrderBook(market)
				Expect(err).ToNot(HaveOccurred())

				orderbookChannels := conn.GetOrderbookChannels()
				Expect(orderbookChannels).ToNot(BeNil())

				outcome, exists := orderbookChannels[market.Slug]
				Expect(exists).To(BeTrue())
				Expect(outcome).ToNot(BeNil())

				// Wait for orderbook data
				orderBook := runner.VerifyOrderBookData(outcome, 30*time.Second)
				Expect(orderBook.Bids).ToNot(BeEmpty(), "Should have bids")
				Expect(orderBook.Asks).ToNot(BeEmpty(), "Should have asks")

				// Get best bid to ensure our order doesn't immediately fill
				bestBid := orderBook.Bids[0].Price // e.g., "0.65"

				price := bestBid
				amount := numerical.NewFromFloat(5.0)

				fmt.Printf("Placing limit order with price %s and amount %s\n", price.String(), amount.String())

				order := prediction.LimitOrder{
					Outcome:    market.Outcomes[0],
					Side:       connector.OrderSideBuy,
					Price:      price,
					Amount:     amount,
					Expiration: time.Now().Add(1 * time.Hour).Unix(),
				}

				orderResponse, err := conn.PlaceLimitOrder(order)
				fmt.Printf("Order response: %+v, error: %v\n", orderResponse, err)
				Expect(err).ToNot(HaveOccurred(), "Order placement should succeed")
				Expect(orderResponse).ToNot(BeNil())
				Expect(orderResponse.OrderID).ToNot(BeEmpty(), "Should receive order ID")

				resp, err := conn.CancelOrder(orderResponse.OrderID)
				fmt.Printf("Cancel response: %+v\n", resp)
				Expect(resp.OrderID).To(Equal(orderResponse.OrderID), "Cancel response should match order ID")
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
