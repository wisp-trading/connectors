package prediction_markets_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
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

				// Get market and subscribe to orderbook
				market, obChan, err := getMarketAndSubscribeOrderbook(conn, "will-jesus-christ-return-before-2027")
				Expect(err).ToNot(HaveOccurred())

				// Place limit order at best bid to avoid immediate fill
				amount := numerical.NewFromFloat(5.0)
				orderResponse, err := placeLimitOrderAtBestBid(runner, conn, market, obChan, amount, 0)
				Expect(err).ToNot(HaveOccurred(), "Order placement should succeed")
				Expect(orderResponse).ToNot(BeNil())
				Expect(orderResponse.OrderID).ToNot(BeEmpty(), "Should receive order ID")

				// Cancel order and verify
				resp, err := cancelOrderAndVerify(conn, orderResponse.OrderID)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.OrderID).To(Equal(orderResponse.OrderID), "Cancel response should match order ID")
			})
		})
	})
})
