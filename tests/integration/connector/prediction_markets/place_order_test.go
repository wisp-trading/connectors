package prediction_markets_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// orderTestMarketSlug is the market used for order placement tests.
// It must be an active NegRisk market with bids and asks on at least the first outcome.
const orderTestMarketSlug = "will-jesus-christ-return-before-2027"

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
		Context("BUY Limit Order", func(){
			It("should place a $1.10 limit buy order and cancel it", func() {
				conn := runner.GetWebSocketCapable()
				err := conn.StartWebSocket()
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					if stopErr := conn.StopWebSocket(); stopErr != nil {
						connector_test.LogError("StopWebSocket: %v", stopErr)
					}
				}()

				market, obChan, err := getMarketAndSubscribeOrderbook(conn, orderTestMarketSlug)
				Expect(err).ToNot(HaveOccurred())

				// Place limit order at best bid to avoid immediate fill.
				// Use a $1.10 target so the order always clears the $1.00 exchange minimum.
				targetValue := numerical.NewFromFloat(1.10)
				orderResponse, err := placeLimitOrderAtBestBidValue(runner, conn, market, obChan, targetValue, 0)
				Expect(err).ToNot(HaveOccurred(), "order placement should succeed")
				Expect(orderResponse).ToNot(BeNil())
				Expect(orderResponse.OrderID).ToNot(BeEmpty(), "should receive order ID")

				resp, err := cancelOrderAndVerify(conn, orderResponse.OrderID)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.OrderID).To(Equal(orderResponse.OrderID), "cancel response should match order ID")
			})
		})

		Context("SELL Limit Order", func() {
			It("should place a SELL order for outcome tokens and cancel it", func() {
				conn := runner.GetWebSocketCapable()
				Expect(conn).ToNot(BeNil(), "connector must implement WebSocketConnector")
				balancePreflightCheck(conn)

				err := conn.StartWebSocket()
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					if stopErr := conn.StopWebSocket(); stopErr != nil {
						connector_test.LogError("StopWebSocket: %v", stopErr)
					}
				}()

				market, obChan, err := getMarketAndSubscribeOrderbook(conn, orderTestMarketSlug)
				Expect(err).ToNot(HaveOccurred())

				orderBook := runner.VerifyOrderBookData(obChan, 30*time.Second)
				Expect(len(orderBook.Asks)).ToNot(BeZero(), "need asks to price the SELL")

				bestAsk := orderBook.Asks[0].Price
				// Minimum shares to clear Polymarket's $1.00 order value floor.
				sharesToSell := numerical.NewFromFloat(1).Div(bestAsk).RoundUp(0)

				fmt.Fprintf(GinkgoWriter, "bestAsk: %s  sharesToSell: %s\n", bestAsk, sharesToSell)

				// Place a resting SELL limit order at best ask — should not cross.
				sellParams := OrderPlacementParams{
					Market:     market,
					OutcomeIdx: 0,
					Side:       connector.OrderSideSell,
					Price:      bestAsk,
					Amount:     sharesToSell,
					Expiration: 1 * time.Hour,
				}
				orderResp, err := placeLimitOrderAtPrice(conn, sellParams)
				Expect(err).ToNot(HaveOccurred(), "CLOB should accept SELL order")
				Expect(orderResp.OrderID).ToNot(BeEmpty())

				fmt.Fprintf(GinkgoWriter, "SELL order accepted: orderID=%s\n", orderResp.OrderID)

				cancelResp, err := cancelOrderAndVerify(conn, orderResp.OrderID)
				Expect(err).ToNot(HaveOccurred())
				Expect(cancelResp.OrderID).To(Equal(orderResp.OrderID))
			})
		})
	})
})
