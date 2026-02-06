package connector

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
)

// CreatePair creates a portfolio.Asset for testing
func CreatePair(symbol string) portfolio.Pair {
	base := portfolio.NewAsset(symbol)
	quote := portfolio.NewAsset("USDT")

	return portfolio.NewPair(
		base,
		quote,
	)
}

// MarketDataBehavior defines shared market data test behaviors
// Use this in both spot and perp test files
func MarketDataBehavior(getRunner func() BaseTestRunner, getPair func() portfolio.Pair) {

	Describe("Market Data (Shared)", func() {

		Context("FetchPrice", func() {
			It("should fetch current price", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector().(connector.MarketDataReader)
				symbol := getPair()

				price, err := conn.FetchPrice(symbol)
				Expect(err).ToNot(HaveOccurred())
				Expect(price).ToNot(BeNil())
				Expect(price.Price.IsPositive()).To(BeTrue())

				LogSuccess("Price for %s: %s", symbol, price.Price.String())
			})
		})

		Context("FetchKlines", func() {
			It("should fetch historical klines", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector().(connector.MarketDataReader)
				symbol := getPair()

				klines, err := conn.FetchKlines(symbol, "1m", 10)
				Expect(err).ToNot(HaveOccurred())
				Expect(klines).ToNot(BeEmpty())

				LogSuccess("Fetched %d klines for %s", len(klines), symbol)
			})
		})

		Context("FetchOrderBook", func() {
			It("should fetch order book", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector().(connector.MarketDataReader)
				pair := getPair()

				ob, err := conn.FetchOrderBook(pair, 10)
				Expect(err).ToNot(HaveOccurred())
				Expect(ob).ToNot(BeNil())
				Expect(ob.Bids).ToNot(BeEmpty())
				Expect(ob.Asks).ToNot(BeEmpty())

				LogSuccess("OrderBook fetched: %d bids, %d asks", len(ob.Bids), len(ob.Asks))
			})
		})

		Context("FetchRecentTrades", func() {
			It("should fetch recent trades", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector().(connector.MarketDataReader)
				symbol := getPair()

				trades, err := conn.FetchRecentTrades(symbol, 10)
				Expect(err).ToNot(HaveOccurred())
				Expect(trades).ToNot(BeNil())

				LogSuccess("Fetched %d recent trades", len(trades))
			})
		})
	})
}

// AccountBehavior defines shared account test behaviors
func AccountBehavior(getRunner func() BaseTestRunner) {

	Describe("Account Data (Shared)", func() {

		Context("GetAccountBalance", func() {
			It("should fetch account balance", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector().(connector.AccountReader)

				balance, err := conn.GetBalance(portfolio.NewAsset("USDC"))
				Expect(err).ToNot(HaveOccurred())
				Expect(balance).ToNot(BeNil())
				Expect(balance.Asset.Symbol()).ToNot(BeEmpty())

				LogSuccess("Account Balance: %s %s", balance.Total.String(), balance.Asset.Symbol())
			})
		})
	})
}

// WebSocketLifecycleBehavior defines shared WebSocket lifecycle tests
func WebSocketLifecycleBehavior(getRunner func() BaseTestRunner) {

	Describe("WebSocket Lifecycle (Shared)", func() {

		Context("StartWebSocket", func() {
			It("should establish connection", func() {
				runner := getRunner()
				if !runner.HasWebSocketSupport() {
					Skip("Connector does not support WebSocket")
				}

				wsConn := runner.GetWebSocketCapable()

				err := wsConn.StartWebSocket()
				Expect(err).ToNot(HaveOccurred())

				Eventually(wsConn.IsWebSocketConnected, "10s", "500ms").
					Should(BeTrue(), "WebSocket should connect")

				LogSuccess("WebSocket connected")
			})
		})

		Context("StopWebSocket", func() {
			It("should disconnect cleanly", func() {
				runner := getRunner()
				if !runner.HasWebSocketSupport() {
					Skip("Connector does not support WebSocket")
				}

				wsConn := runner.GetWebSocketCapable()

				// Start first
				err := wsConn.StartWebSocket()
				Expect(err).ToNot(HaveOccurred())
				Eventually(wsConn.IsWebSocketConnected, "10s").Should(BeTrue())

				// Stop
				err = wsConn.StopWebSocket()
				Expect(err).ToNot(HaveOccurred())

				Eventually(wsConn.IsWebSocketConnected, "5s", "500ms").
					Should(BeFalse(), "WebSocket should disconnect")

				LogSuccess("WebSocket disconnected")
			})
		})
	})
}
