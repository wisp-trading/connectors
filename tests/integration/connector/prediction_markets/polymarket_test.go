package prediction_markets_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"

	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
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

	// ========================================
	// ORDER PLACEMENT TESTS (IMPLEMENTED)
	// ========================================

	Describe("Order Placement", func() {
		Context("PlaceLimitOrder", func() {
			It("should place a limit order successfully", func() {
				conn := runner.GetPredictionMarketConnector()
				tokenID := connector_test.GetPredictionMarketTokenIDs()

				makerAmount, _ := numerical.NewFromString("10.0")
				takerAmount, _ := numerical.NewFromString("5.0")

				market := prediction.Market{
					MarketId:    "0xb51ef0ffaaca4559f39359ae9793cba168b1b1fa2376b696b3046d6a27bce6be",
					Slug:        "test-market-slug",
					Exchange:    connector.ExchangeName("Polymarket"),
					OutcomeType: prediction.OutcomeTypeBinary,
					Outcomes: []prediction.Outcome{
						{
							OutcomeId: tokenID[0],
						},
						{
							OutcomeId: tokenID[1],
						},
					},
					Active: true,
					Closed: false,
				}

				// Create a limit order
				// Create a limit order
				order := prediction.LimitOrder{
					Market:       market,
					MakerAddress: "", // TODO: Set maker address from test config
					TakerAddress: "", // TODO: Set taker address or empty for public order
					MakerAmount:  makerAmount,
					TakerAmount:  takerAmount,
					Side:         connector.OrderSideBuy,
					IsMaker:      true,
					Expiration:   0,
				}

				response, err := conn.PlaceLimitOrder(order)
				Expect(err).ToNot(HaveOccurred())
				Expect(response).ToNot(BeNil())

				connector_test.LogSuccess("Limit order placed successfully for token %s", tokenID)
			})
		})

		Context("PlaceMarketOrder", func() {
			It("should place a market order successfully", func() {
				Skip("Market orders not yet implemented")
			})
		})
	})

	// ========================================
	// ORDER MANAGEMENT STUBS (FUTURE)
	// ========================================

	Describe("Order Management", func() {
		Context("CancelOrder", func() {
			It("should cancel an open order", func() {
				Skip("Order cancellation not yet implemented")
			})

			It("should handle cancelling non-existent order", func() {
				Skip("Order cancellation not yet implemented")
			})
		})

		Context("GetOpenOrders", func() {
			It("should fetch all open orders", func() {
				Skip("GetOpenOrders not yet implemented")
			})

			It("should return empty list when no orders", func() {
				Skip("GetOpenOrders not yet implemented")
			})
		})

		Context("GetOrderStatus", func() {
			It("should fetch order status", func() {
				Skip("GetOrderStatus not yet implemented")
			})

			It("should handle non-existent order", func() {
				Skip("GetOrderStatus not yet implemented")
			})
		})
	})

	// ========================================
	// POSITIONS STUBS (FUTURE)
	// ========================================

	Describe("Position Management", func() {
		Context("GetPositions", func() {
			It("should fetch all positions", func() {
				Skip("GetPositions not yet implemented")
			})

			It("should return empty list when no positions", func() {
				Skip("GetPositions not yet implemented")
			})
		})

		Context("GetPositionsByMarket", func() {
			It("should fetch positions for specific market", func() {
				Skip("GetPositionsByMarket not yet implemented")
			})

			It("should handle invalid market ID", func() {
				Skip("GetPositionsByMarket not yet implemented")
			})
		})
	})

	// ========================================
	// MARKET DATA STUBS (FUTURE)
	// ========================================

	Describe("Market Data", func() {
		Context("FetchPrice", func() {
			It("should fetch current market price", func() {
				Skip("Market data not yet implemented")
			})
		})

		Context("FetchOrderBook", func() {
			It("should fetch order book for market", func() {
				Skip("Order book not yet implemented")
			})
		})

		Context("FetchMarket", func() {
			It("should fetch market details", func() {
				conn := runner.GetPredictionMarketConnector()
				slug := "us-strike-on-somalia-by-february-14"

				conn.GetMarket(slug)

			})
		})
	})

	// ========================================
	// TRADING HISTORY STUBS (FUTURE)
	// ========================================

	Describe("Trading History", func() {
		Context("GetTradingHistory", func() {
			It("should fetch trading history", func() {
				Skip("Trading history not yet implemented")
			})
		})

		Context("GetSettlementHistory", func() {
			It("should fetch settlement history", func() {
				Skip("Settlement history not yet implemented")
			})
		})

		Context("FetchRecentTrades", func() {
			It("should fetch recent trades", func() {
				Skip("Recent trades not yet implemented")
			})
		})
	})

	// ========================================
	// ACCOUNT DATA STUBS (FUTURE)
	// ========================================

	Describe("Account Data", func() {
		Context("GetBalance", func() {
			It("should fetch account balance", func() {
				Skip("Account balance not yet implemented")
			})
		})
	})

	// ========================================
	// WEBSOCKET STUBS (FUTURE)
	// ========================================

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

				outcomes := []prediction.Outcome{
					{
						Pair: prediction.NewPredictionPair(
							"btc-updown-4h",
							"Yes",
							portfolio.NewAsset("USDC"),
						),
						OutcomeId: "55465499552240998868444058452917756997929695703966881733232094915253142184919",
					},
					//{
					//	Pair: prediction.NewPredictionPair(
					//		"btc-updown-4h",
					//		"No",
					//		portfolio.NewAsset("USDC"),
					//	),
					//	OutcomeId: "6757703472668573966175902785925764600818133786181136162144102885621254094181",
					//},
				}

				market := prediction.Market{
					MarketId: "0x049e9f5ee242baad05476a24f9c9a3ea64e4c297f81dbc9c5c60756864c526e1",
					Outcomes: outcomes,
					Slug:     "btc-updown-4h",
				}

				err = conn.SubscribeOrderBook(market)
				Expect(err).ToNot(HaveOccurred())

				channels := conn.GetOrderbookChannels()
				Expect(channels).ToNot(BeNil(), "Market channels should not be nil")

				outcome1, exists := channels[market.Outcomes[0].Pair.Symbol()]
				Expect(exists).To(BeTrue(), "Market book channel should exist for subscribed market")
				Expect(outcome1).ToNot(BeNil(), "Market book channel should not be nil")

				//outcome2, exists := channels[market.Outcomes[1].Pair.Symbol()]
				//Expect(exists).To(BeTrue(), "Market book channel should exist for subscribed market")
				//Expect(outcome2).ToNot(BeNil(), "Market book channel should not be nil")

				// Wait for order book data with timeout
				Expect(exists).To(BeTrue())

				// Use helper to verify order book
				orderBook := runner.VerifyOrderBookData(outcome1, 30*time.Second)
				Expect(orderBook).ToNot(BeNil())

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

			It("should subscribe to price changes", func() {
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

				outcomes := []prediction.Outcome{
					{
						Pair: prediction.NewPredictionPair(
							"btc-updown-4h",
							"Yes",
							portfolio.NewAsset("USDC"),
						),
						OutcomeId: "33602105978995322122109139507127572886653318971583142116018047077584613020792",
					},
					//{
					//	Pair: prediction.NewPredictionPair(
					//		"btc-updown-4h",
					//		"No",
					//		portfolio.NewAsset("USDC"),
					//	),
					//	OutcomeId: "6757703472668573966175902785925764600818133786181136162144102885621254094181",
					//},
				}

				market := prediction.Market{
					MarketId: "0x049e9f5ee242baad05476a24f9c9a3ea64e4c297f81dbc9c5c60756864c526e1",
					Outcomes: outcomes,
				}

				err = conn.SubscribeOrderBook(market)
				Expect(err).ToNot(HaveOccurred())

				channels := conn.GetOrderbookChannels()
				Expect(channels).ToNot(BeNil(), "Market channels should not be nil")

				connector_test.LogSuccess(
					"Received order book data for market %s",
					market.MarketId,
				)
				time.Sleep(5 * time.Second) // Sleep to allow any additional messages to be received before test ends
			})
		})

		Context("SubscribeTrades", func() {
			It("should subscribe to trade updates", func() {
				Skip("WebSocket subscriptions not yet implemented")
			})
		})

		Context("SubscribePositions", func() {
			It("should subscribe to position updates", func() {
				Skip("WebSocket subscriptions not yet implemented")
			})
		})
	})
})
