package connector_test

import (
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("WebSocket Subscription Tests", func() {
	var runner *TestRunner

	BeforeEach(func() {
		var err error
		runner, err = NewTestRunner(testConnectorName, getConnectorConfig(testConnectorName))
		Expect(err).ToNot(HaveOccurred())

		if !runner.HasWebSocketSupport() {
			Skip("Connector does not support WebSocket")
		}

		// Start WebSocket
		wsConn := runner.GetWebSocketConnector()
		err = wsConn.StartWebSocket()
		Expect(err).ToNot(HaveOccurred())
		Eventually(wsConn.IsWebSocketConnected, "10s").Should(BeTrue())
	})

	AfterEach(func() {
		if runner != nil {
			runner.Cleanup()
		}
	})

	Context("OrderBook Subscription", func() {
		It("should subscribe and receive updates", func() {
			wsConn := runner.GetWebSocketConnector()
			asset := CreateAsset(testSymbol)

			err := wsConn.SubscribeOrderBook(asset, testInstrumentType)
			AssertNoError(err, "SubscribeOrderBook should succeed")

			channels := wsConn.GetOrderBookChannels()
			Expect(channels).ToNot(BeEmpty())

			// Find the channel for our asset
			var obCh <-chan connector.OrderBook
			for key, ch := range channels {
				if key == testSymbol || key == asset.Symbol() {
					obCh = ch
					break
				}
			}
			Expect(obCh).ToNot(BeNil(), "OrderBook channel should exist")

			var ob connector.OrderBook
			Eventually(obCh, "15s").Should(Receive(&ob),
				"Should receive orderbook update")

			Expect(ob.Bids).ToNot(BeEmpty())
			Expect(ob.Asks).ToNot(BeEmpty())

			LogSuccess("OrderBook subscription working")
			LogInfo("Best Bid: %.8f @ %.8f", ob.Bids[0].Quantity, ob.Bids[0].Price)
			LogInfo("Best Ask: %.8f @ %.8f", ob.Asks[0].Quantity, ob.Asks[0].Price)
		})
	})

	Context("Klines Subscription", func() {
		It("should subscribe and receive updates", func() {
			wsConn := runner.GetWebSocketConnector()
			asset := CreateAsset(testSymbol)

			err := wsConn.SubscribeKlines(asset, "1m")
			AssertNoError(err, "SubscribeKlines should succeed")

			channels := wsConn.GetKlineChannels()
			Expect(channels).ToNot(BeEmpty())

			// Find the channel for our asset
			var klineCh <-chan connector.Kline
			for key, ch := range channels {
				if key == testSymbol || key == asset.Symbol() ||
					key == fmt.Sprintf("%s:1m", testSymbol) {
					klineCh = ch
					break
				}
			}
			Expect(klineCh).ToNot(BeNil(), "Kline channel should exist")

			var kline connector.Kline
			Eventually(klineCh, "90s").Should(Receive(&kline),
				"Should receive kline update (may take up to 90s)")

			Expect(kline.Open).To(BeNumerically(">", 0))

			LogSuccess("Kline subscription working")
			LogInfo("OHLC: %.8f / %.8f / %.8f / %.8f",
				kline.Open, kline.High,
				kline.Low, kline.Close)
		})
	})

	Context("Trades Subscription", func() {
		It("should subscribe to user trades", func() {
			wsConn := runner.GetWebSocketConnector()
			asset := CreateAsset(testSymbol)

			err := wsConn.SubscribeTrades(asset, testInstrumentType)
			AssertNoError(err, "SubscribeTrades should succeed")

			tradeCh := wsConn.TradeUpdates()
			Expect(tradeCh).ToNot(BeNil())

			LogSuccess("Trade subscription active (trades appear when orders execute)")
		})
	})

	Context("Position Subscription", func() {
		It("should subscribe to positions", func() {
			wsConn := runner.GetWebSocketConnector()
			asset := CreateAsset(testSymbol)

			err := wsConn.SubscribePositions(asset, testInstrumentType)
			AssertNoError(err, "SubscribePositions should succeed")

			posCh := wsConn.PositionUpdates()
			Expect(posCh).ToNot(BeNil())

			LogSuccess("Position subscription active")
		})
	})

	Context("Account Balance Subscription", func() {
		It("should subscribe to account balance", func() {
			wsConn := runner.GetWebSocketConnector()

			err := wsConn.SubscribeAccountBalance()
			AssertNoError(err, "SubscribeAccountBalance should succeed")

			balanceCh := wsConn.AccountBalanceUpdates()
			Expect(balanceCh).ToNot(BeNil())

			LogSuccess("Account balance subscription active")
		})
	})

	Context("Unsubscribe", func() {
		It("should unsubscribe from orderbook", func() {
			wsConn := runner.GetWebSocketConnector()
			asset := CreateAsset(testSymbol)

			// Subscribe first
			err := wsConn.SubscribeOrderBook(asset, testInstrumentType)
			Expect(err).ToNot(HaveOccurred())

			// Unsubscribe
			err = wsConn.UnsubscribeOrderBook(asset, testInstrumentType)
			AssertNoError(err, "UnsubscribeOrderBook should succeed")

			LogSuccess("Unsubscribed from orderbook")
		})
	})
})
