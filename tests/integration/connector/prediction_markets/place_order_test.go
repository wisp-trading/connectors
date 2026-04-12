package prediction_markets_test

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
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
		Context("Order is a BUY Limit Order via EOA", func() {
			It("Should place a $1.10 limit buy order and cancel it", func() {
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
				market, obChan, err := getMarketAndSubscribeOrderbook(conn, orderTestMarketSlug)
				Expect(err).ToNot(HaveOccurred())

				// Place limit order at best bid to avoid immediate fill.
				// Use a $1.10 target so the order always clears the $1.00 exchange minimum.
				targetValue := numerical.NewFromFloat(1.10)
				orderResponse, err := placeLimitOrderAtBestBidValue(runner, conn, market, obChan, targetValue, 0)
				Expect(err).ToNot(HaveOccurred(), "Order placement should succeed")
				Expect(orderResponse).ToNot(BeNil())
				Expect(orderResponse.OrderID).ToNot(BeEmpty(), "Should receive order ID")

				// Cancel order and verify
				resp, err := cancelOrderAndVerify(conn, orderResponse.OrderID)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.OrderID).To(Equal(orderResponse.OrderID), "Cancel response should match order ID")
			})
		})

		// EOASell verifies the CLOB accepts a SELL limit order when the EOA holds
		// YES tokens on-chain. Requires POLYGON_RPC_URL and existing token balance
		// (run the split_merge_test or a prior strategy pass to fund the EOA first).
		Context("Order is a SELL Limit Order via EOA", func() {
			It("CLOB should accept a SELL order for YES tokens held by the EOA", func() {
				if os.Getenv("POLYGON_RPC_URL") == "" {
					Skip("POLYGON_RPC_URL not set — skipping on-chain EOA SELL test")
				}

				eoa := ctfPreflightCheck()

				conn := runner.GetWebSocketCapable()
				Expect(conn).ToNot(BeNil(), "connector must implement WebSocketConnector")

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

				fmt.Fprintf(GinkgoWriter,
					"EOA: %s  bestAsk: %s  sharesToSell: %s\n",
					eoa.Hex(), bestAsk, sharesToSell)

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
				Expect(err).ToNot(HaveOccurred(), "CLOB should accept SELL order — no balance:0")
				Expect(orderResp.OrderID).ToNot(BeEmpty())

				fmt.Fprintf(GinkgoWriter, "SELL order accepted: orderID=%s\n", orderResp.OrderID)

				cancelResp, err := cancelOrderAndVerify(conn, orderResp.OrderID)
				Expect(err).ToNot(HaveOccurred())
				Expect(cancelResp.OrderID).To(Equal(orderResp.OrderID))
			})
		})
	})
})
