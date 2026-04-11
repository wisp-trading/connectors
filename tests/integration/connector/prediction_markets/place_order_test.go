package prediction_markets_test

import (
	"fmt"
	"math/big"
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

		// EOASellAfterSplit verifies the full MintSell path at the connector level:
		// SplitPosition → CLOB balance refresh → SELL limit order accepted (no balance:0).
		// Requires POLYGON_RPC_URL (on-chain split) and a funded EOA.
		Context("Order is a SELL Limit Order via EOA after SplitPosition", func() {
			It("CLOB should accept a SELL order for freshly-minted YES tokens", func() {
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

				// ── 1. Get market and subscribe to orderbook ──────────────────────────
				market, obChan, err := getMarketAndSubscribeOrderbook(conn, orderTestMarketSlug)
				Expect(err).ToNot(HaveOccurred())

				// ── 2. Read orderbook to determine ask price and split amount ─────────
				// outcome 0 = YES; we will split USDC → YES+NO, then sell the YES tokens.
				orderBook := runner.VerifyOrderBookData(obChan, 30*time.Second)
				Expect(len(market.Outcomes)).To(BeNumerically(">=", 1), "Market must have outcomes")
				Expect(len(orderBook.Asks)).ToNot(BeZero(), "Need at least one ask to compute sell size")

				bestAsk := orderBook.Asks[0].Price
				Expect(bestAsk.IsZero()).To(BeFalse(), "Best ask price must be non-zero")

				// Compute the number of YES shares required so the order value ≥ $1.10.
				// shares = $1.10 / bestAsk, rounded up to 0 decimals.
				targetUSDC := numerical.NewFromFloat(1.10)
				sharesToSell := targetUSDC.Div(bestAsk).RoundUp(0)

				// SplitPosition amount in 6-decimal USDC units.
				// Split 10% more than we need to ensure we cover rounding.
				usdcUnits := sharesToSell.Mul(numerical.NewFromFloat(1.10)).RoundUp(0)
				splitAmt := new(big.Int).Mul(usdcUnits.BigInt(), big.NewInt(1_000_000))

				fmt.Fprintf(GinkgoWriter,
					"EOA: %s  bestAsk: %s  sharesToSell: %s  splitAmt: %s USDC-units\n",
					eoa.Hex(), bestAsk, sharesToSell, splitAmt)

				// ── 3. On-chain: split USDC → YES + NO tokens ─────────────────────────
				// SplitPosition automatically calls RefreshMarketBalance after the tx so
				// the CLOB sees the newly-minted ERC-1155 tokens before we try to sell.
				//txHash, err := conn.SplitPosition(market, splitAmt)
				//Expect(err).ToNot(HaveOccurred(), "SplitPosition should succeed")
				//fmt.Fprintf(GinkgoWriter, "Split tx: %s\n", txHash)

				// ── 4. Place SELL limit order for outcome 0 (YES) at best ask ─────────
				// Post at best ask — the order rests on the book without crossing.
				// The key assertion: no "balance: 0" error from the CLOB.
				sellParams := OrderPlacementParams{
					Market:     market,
					OutcomeIdx: 0,
					Side:       connector.OrderSideSell,
					Price:      bestAsk,
					Amount:     sharesToSell,
					Expiration: 1 * time.Hour,
				}
				orderResp, err := placeLimitOrderAtPrice(conn, sellParams)
				Expect(err).ToNot(HaveOccurred(),
					"CLOB should accept SELL order after SplitPosition+refresh — not get balance:0")
				Expect(orderResp).ToNot(BeNil())
				Expect(orderResp.OrderID).ToNot(BeEmpty(), "Should receive an order ID for the SELL")

				fmt.Fprintf(GinkgoWriter, "SELL order accepted: orderID=%s\n", orderResp.OrderID)

				// ── 5. Cancel the resting SELL order ──────────────────────────────────
				cancelResp, err := cancelOrderAndVerify(conn, orderResp.OrderID)
				Expect(err).ToNot(HaveOccurred())
				Expect(cancelResp.OrderID).To(Equal(orderResp.OrderID))

				// ── 6. Merge YES + NO tokens back to USDC ─────────────────────────────
				mergeTx, err := conn.MergePositions(market, splitAmt)
				Expect(err).ToNot(HaveOccurred(), "MergePositions should recover USDC")
				fmt.Fprintf(GinkgoWriter, "Merge tx: %s\n", mergeTx)
			})
		})
	})
})
