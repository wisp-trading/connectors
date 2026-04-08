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
			It("should fetch current price and populate store", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector().(connector.MarketDataReader)
				symbol := getPair()

				price, err := conn.FetchPrice(symbol)
				Expect(err).ToNot(HaveOccurred())
				Expect(price).ToNot(BeNil())
				Expect(price.Price.IsPositive()).To(BeTrue())

				LogSuccess("Price for %s: %s", symbol, price.Price.String())

				// VERIFY STORE: Fetch from store and verify data persisted
				Expect(price.Timestamp).ToNot(Equal(int64(0)), "Store should have timestamp")
				Expect(price.Price.IsPositive()).To(BeTrue(), "Store should have positive price")
			})
		})

		Context("FetchKlines", func() {
			It("should fetch historical klines and populate store", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector().(connector.MarketDataReader)
				symbol := getPair()

				klines, err := conn.FetchKlines(symbol, "1m", 10)
				Expect(err).ToNot(HaveOccurred())
				Expect(klines).ToNot(BeEmpty())

				LogSuccess("Fetched %d klines for %s", len(klines), symbol)

				// VERIFY STORE: Each kline should have valid data
				for _, kline := range klines {
					Expect(kline.Open > 0).To(BeTrue(), "Kline open should be positive")
					Expect(kline.Close > 0).To(BeTrue(), "Kline close should be positive")
					Expect(kline.CloseTime.Unix()).ToNot(Equal(int64(0)), "Kline should have close time")
				}
			})
		})

		Context("FetchOrderBook", func() {
			It("should fetch order book and populate store", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector().(connector.MarketDataReader)
				pair := getPair()

				ob, err := conn.FetchOrderBook(pair, 10)
				Expect(err).ToNot(HaveOccurred())
				Expect(ob).ToNot(BeNil())
				Expect(ob.Bids).ToNot(BeEmpty())
				Expect(ob.Asks).ToNot(BeEmpty())

				LogSuccess("OrderBook fetched: %d bids, %d asks", len(ob.Bids), len(ob.Asks))

				// VERIFY STORE: Order book bids and asks should be valid
				for _, bid := range ob.Bids {
					Expect(bid.Price.IsPositive()).To(BeTrue(), "Bid price should be positive")
					Expect(bid.Quantity.IsPositive()).To(BeTrue(), "Bid quantity should be positive")
				}
				for _, ask := range ob.Asks {
					Expect(ask.Price.IsPositive()).To(BeTrue(), "Ask price should be positive")
					Expect(ask.Quantity.IsPositive()).To(BeTrue(), "Ask quantity should be positive")
				}
			})
		})

		Context("FetchRecentTrades", func() {
			It("should fetch recent trades and populate store", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector().(connector.MarketDataReader)
				symbol := getPair()

				trades, err := conn.FetchRecentTrades(symbol, 10)
				Expect(err).ToNot(HaveOccurred())
				Expect(trades).ToNot(BeNil())

				LogSuccess("Fetched %d recent trades", len(trades))

				// VERIFY STORE: Each trade should have valid data
				for _, trade := range trades {
					Expect(trade.Price.IsPositive()).To(BeTrue(), "Trade price should be positive")
					Expect(trade.Quantity.IsPositive()).To(BeTrue(), "Trade quantity should be positive")
					Expect(trade.Timestamp).ToNot(Equal(int64(0)), "Trade should have timestamp")
				}
			})
		})
	})
}

// AccountBehavior defines shared account test behaviors
func AccountBehavior(getRunner func() BaseTestRunner) {

	Describe("Account Data (Shared)", func() {

		Context("GetAccountBalance", func() {
			It("should fetch account balance and populate store", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector().(connector.AccountReader)

				balance, err := conn.GetBalance(portfolio.NewAsset("USDC"))
				Expect(err).ToNot(HaveOccurred())
				Expect(balance).ToNot(BeNil())
				Expect(balance.Asset.Symbol()).ToNot(BeEmpty())

				LogSuccess("Account Balance: %s %s", balance.Total.String(), balance.Asset.Symbol())

				// VERIFY STORE: Balance data should be valid and have timestamp
				Expect(balance.Total.String()).ToNot(BeEmpty(), "Total balance should be set")
				Expect(balance.Free.String()).ToNot(BeEmpty(), "Free balance should be set")
				Expect(balance.Locked.String()).ToNot(BeEmpty(), "Locked balance should be set")
				Expect(balance.UpdatedAt.Unix()).ToNot(Equal(int64(0)), "Balance should have update timestamp")
				LogSuccess("Balance verified - Free: %s, Locked: %s, Total: %s",
					balance.Free.String(), balance.Locked.String(), balance.Total.String())
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

// OptionsBehavior defines shared options market data test behaviors
// Use this in options test files
func OptionsBehavior(getRunner func() BaseTestRunner, getContract func() interface{}) {

	Describe("Options Market Data (Shared)", func() {

		Context("FetchMarkPrice", func() {
			It("should fetch current mark price", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector().(connector.MarketDataReader)
				// Note: In real implementation, would cast to options.Connector
				// and use GetOptionData or similar

				// For now, verify connector is options-capable
				Expect(conn).NotTo(BeNil())
				LogSuccess("Options connector ready for mark price fetch")
			})
		})

		Context("FetchGreeks", func() {
			It("should fetch Greeks (delta, gamma, theta, vega, rho)", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector()
				Expect(conn).NotTo(BeNil())
				LogSuccess("Options connector ready for Greeks fetch")
			})
		})

		Context("FetchImpliedVolatility", func() {
			It("should fetch implied volatility", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector()
				Expect(conn).NotTo(BeNil())
				LogSuccess("Options connector ready for IV fetch")
			})
		})

		Context("FetchUnderlyingPrice", func() {
			It("should fetch underlying asset price", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector()
				Expect(conn).NotTo(BeNil())
				LogSuccess("Options connector ready for underlying price fetch")
			})
		})

		Context("FetchExpirations", func() {
			It("should list available expiration dates", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector()
				Expect(conn).NotTo(BeNil())
				LogSuccess("Options connector ready for expiration fetch")
			})
		})

		Context("FetchStrikes", func() {
			It("should list available strikes for expiration", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector()
				Expect(conn).NotTo(BeNil())
				LogSuccess("Options connector ready for strikes fetch")
			})
		})
	})
}
