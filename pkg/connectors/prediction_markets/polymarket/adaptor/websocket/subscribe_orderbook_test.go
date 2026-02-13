package websocket_test

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	pmwebsocket "github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/websocket"
)

var _ = Describe("OrderBook Subscription", func() {
	resolutionTime := time.Now().Add(24 * time.Hour)

	var (
		app                  *fxtest.App
		ws                   pmwebsocket.PolymarketWebsocket
		testServer           *TestWebSocketServer
		receivedBooks        []*pmwebsocket.OrderBookMessage
		receivedPriceChanges []*pmwebsocket.PriceChanges

		market = prediction.Market{
			MarketId:    "70308501195956323589797156800521969197358506986152833648253437673484286051597",
			Slug:        "test-market",
			Exchange:    "polymarket",
			OutcomeType: prediction.OutcomeTypeBinary,
			Outcomes: []prediction.Outcome{
				{
					Pair:      prediction.NewPredictionPair("test-market", "YES", portfolio.NewAsset("USDC")),
					OutcomeId: "outcome-yes",
				},
				{
					Pair:      prediction.NewPredictionPair("test-market", "NO", portfolio.NewAsset("USDC")),
					OutcomeId: "outcome-no",
				},
			},
			Active:         true,
			Closed:         false,
			ResolutionTime: &resolutionTime,
		}
	)

	BeforeEach(func() {
		receivedBooks = make([]*pmwebsocket.OrderBookMessage, 0)

		app = fxtest.New(
			GinkgoT(),
			TestWebSocketModule,
			fx.Populate(&ws, &testServer),
			fx.NopLogger,
		)

		app.RequireStart()
	})

	AfterEach(func() {
		if ws != nil {
			ws.Disconnect()
		}
		if app != nil {
			app.RequireStop()
		}
	})

	Describe("SubscribeOrderBook", func() {
		It("should call callback when orderbook message received", func(ctx SpecContext) {
			assetID := "70308501195956323589797156800521969197358506986152833648253437673484286051597"
			called := false

			// Connect to test server
			err := ws.Connect(testServer.URL)
			Expect(err).ToNot(HaveOccurred())

			// Wait for connection
			Eventually(func() bool {
				return ws.IsConnected()
			}).WithTimeout(2 * time.Second).Should(BeTrue())

			// Subscribe
			ws.SubscribeToMarket(
				market,
				func(book *pmwebsocket.OrderBookMessage) {
					called = true
					receivedBooks = append(receivedBooks, book)
				},
				func(priceChange *pmwebsocket.PriceChanges) {
					receivedPriceChanges = append(receivedPriceChanges, priceChange)
				},
			)

			// Send orderbook message from server wrapped in array (Polymarket format)
			msg := map[string]interface{}{
				"event_type": "book",
				"asset_id":   assetID,
				"market":     "test-market",
				"bids":       []interface{}{map[string]interface{}{"price": "0.51", "size": "100"}},
				"asks":       []interface{}{map[string]interface{}{"price": "0.60", "size": "150"}},
				"timestamp":  "2026-02-10T08:30:00Z",
				"hash":       "test123",
			}

			// Polymarket sends messages as JSON arrays
			msgArray := []interface{}{msg}
			msgBytes, err := json.Marshal(msgArray)
			Expect(err).ToNot(HaveOccurred())

			err = testServer.SendMessage(msgBytes)
			Expect(err).ToNot(HaveOccurred())

			// Wait for callback
			Eventually(func() bool {
				return called
			}).WithTimeout(2 * time.Second).WithPolling(50 * time.Millisecond).Should(BeTrue())

			Expect(receivedBooks).To(HaveLen(1))
			Expect(receivedBooks[0].AssetID).To(Equal(assetID))
			Expect(receivedBooks[0].Market).To(Equal("test-market"))
			Expect(receivedBooks[0].Bids).To(HaveLen(1))
			Expect(receivedBooks[0].Bids[0].Price).To(Equal("0.51"))
		}, SpecTimeout(5*time.Second))

		It("should not trigger callback for different asset", func(ctx SpecContext) {
			differentAssetID := "77385393614263738045377442390679465888613338149607876972436340566574399345181"
			called := false

			err := ws.Connect(testServer.URL)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				return ws.IsConnected()
			}).WithTimeout(2 * time.Second).Should(BeTrue())

			ws.SubscribeToMarket(
				market,
				func(book *pmwebsocket.OrderBookMessage) {
					called = true
				},
				func(priceChange *pmwebsocket.PriceChanges) {
					// No-op for this test
				},
			)

			// Send message for different asset wrapped in array
			msg := map[string]interface{}{
				"event_type": "book",
				"asset_id":   differentAssetID,
				"market":     "other-market",
				"bids":       []interface{}{},
				"asks":       []interface{}{},
				"timestamp":  "2026-02-10T08:30:00Z",
			}

			msgArray := []interface{}{msg}
			msgBytes, _ := json.Marshal(msgArray)
			testServer.SendMessage(msgBytes)

			Consistently(func() bool {
				return called
			}).WithTimeout(500 * time.Millisecond).Should(BeFalse())
		}, SpecTimeout(3*time.Second))

		It("should handle unsubscribe correctly", func(ctx SpecContext) {
			assetID := "70308501195956323589797156800521969197358506986152833648253437673484286051597"
			called := false

			err := ws.Connect(testServer.URL)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				return ws.IsConnected()
			}).WithTimeout(2 * time.Second).Should(BeTrue())

			ws.SubscribeToMarket(
				market,
				func(book *pmwebsocket.OrderBookMessage) {
					called = true
				},
				func(priceChange *pmwebsocket.PriceChanges) {
					// No-op for this test
				},
			)

			ws.UnsubscribeFromMarket(market)

			msg := map[string]interface{}{
				"event_type": "book",
				"asset_id":   assetID,
				"market":     "test",
				"bids":       []interface{}{},
				"asks":       []interface{}{},
				"timestamp":  "2026-02-10T08:30:00Z",
			}

			msgArray := []interface{}{msg}
			msgBytes, _ := json.Marshal(msgArray)
			testServer.SendMessage(msgBytes)

			Consistently(func() bool {
				return called
			}).WithTimeout(500 * time.Millisecond).Should(BeFalse())
		}, SpecTimeout(3*time.Second))
	})
})
