package adaptor_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor"
)

var _ = Describe("PolymarketClient", func() {
	var (
		client     adaptor.PolymarketClient
		config     *polymarket.Config
		mockServer *httptest.Server
		privateKey string
	)

	BeforeEach(func() {
		// Test private key (do not use in production)
		privateKey = "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

		// Create mock HTTP server
		mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Default 200 OK response
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))

		// Create test config
		config = &polymarket.Config{
			BaseURL:       mockServer.URL,
			APIKey:        "test-api-key",
			APISecret:     "test-api-secret",
			Passphrase:    "test-passphrase",
			PrivateKey:    privateKey,
			FunderAddress: "0x1234567890123456789012345678901234567890",
			ChainID:       137,
		}

		// Create client
		client = adaptor.NewPolymarketClient()
	})

	AfterEach(func() {
		mockServer.Close()
	})

	Describe("NewPolymarketClient", func() {
		Context("when creating a new client", func() {
			It("should create an unconfigured client", func() {
				c := adaptor.NewPolymarketClient()
				Expect(c).ToNot(BeNil())
				Expect(c.IsConfigured()).To(BeFalse())
			})
		})
	})

	Describe("Configure", func() {
		Context("when given valid config", func() {
			It("should configure the client successfully", func() {
				err := client.Configure(config)
				Expect(err).ToNot(HaveOccurred())
				Expect(client.IsConfigured()).To(BeTrue())
			})
		})

		Context("when given nil config", func() {
			It("should return an error", func() {
				err := client.Configure(nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("config cannot be nil"))
			})
		})

		Context("when given invalid config", func() {
			It("should return an error for missing API key", func() {
				invalidConfig := *config
				invalidConfig.APIKey = ""
				err := client.Configure(&invalidConfig)
				Expect(err).To(HaveOccurred())
			})

			It("should return an error for invalid private key", func() {
				invalidConfig := *config
				invalidConfig.PrivateKey = "invalid"
				err := client.Configure(&invalidConfig)
				Expect(err).To(HaveOccurred())
			})

		})

		Context("when already configured", func() {
			It("should return an error", func() {
				err := client.Configure(config)
				Expect(err).ToNot(HaveOccurred())

				err = client.Configure(config)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("already configured"))
			})
		})
	})

	Describe("IsConfigured", func() {
		Context("when client is not configured", func() {
			It("should return false", func() {
				Expect(client.IsConfigured()).To(BeFalse())
			})
		})

		Context("when client is configured", func() {
			It("should return true", func() {
				err := client.Configure(config)
				Expect(err).ToNot(HaveOccurred())
				Expect(client.IsConfigured()).To(BeTrue())
			})
		})
	})

	Describe("PlaceOrder", func() {
		var (
			ctx   context.Context
			order adaptor.OrderRequest
		)

		BeforeEach(func() {
			ctx = context.Background()
			err := client.Configure(config)
			Expect(err).ToNot(HaveOccurred())

			order = adaptor.OrderRequest{
				Maker:         "0x1234567890123456789012345678901234567890",
				Signer:        "0x1234567890123456789012345678901234567890",
				Taker:         "0x0000000000000000000000000000000000000000",
				TokenID:       "12345",
				MakerAmount:   "1000000",
				TakerAmount:   "500000",
				Side:          "BUY",
				FeeRateBps:    "100",
				Nonce:         "1",
				SignatureType: 2,
				Expiration:    time.Now().Add(24 * time.Hour).Unix(),
			}
		})

		Context("when client is not configured", func() {
			It("should return an error", func() {
				unconfiguredClient := adaptor.NewPolymarketClient()
				_, err := unconfiguredClient.PlaceOrder(ctx, order)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not configured"))
			})
		})

		Context("when given valid order", func() {
			It("should sign and send the order", func() {
				mockServer.Close()
				mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify request headers
					Expect(r.Header.Get("POLY_API_KEY")).To(Equal("test-api-key"))
					Expect(r.Header.Get("POLY_SECRET")).To(Equal("test-api-secret"))
					Expect(r.Header.Get("POLY_PASSPHRASE")).To(Equal("test-passphrase"))
					Expect(r.Header.Get("POLY_SIGNATURE")).ToNot(BeEmpty())
					Expect(r.Header.Get("POLY_TIMESTAMP")).ToNot(BeEmpty())
					Expect(r.Header.Get("POLY_ADDRESS")).ToNot(BeEmpty())
					Expect(r.Header.Get("Content-Type")).To(Equal("application/json"))

					// Verify request body
					var reqBody adaptor.OrderRequest
					err := json.NewDecoder(r.Body).Decode(&reqBody)
					Expect(err).ToNot(HaveOccurred())
					Expect(reqBody.Signature).ToNot(BeEmpty())
					Expect(reqBody.Salt).ToNot(BeZero())

					// Return mock response
					response := adaptor.OrderResponse{
						OrderID: "test-order-id",
						Status:  "OPEN",
					}
					json.NewEncoder(w).Encode(response)
				}))
				config.BaseURL = mockServer.URL
				client = adaptor.NewPolymarketClient()
				err := client.Configure(config)
				Expect(err).ToNot(HaveOccurred())

				resp, err := client.PlaceOrder(ctx, order)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.OrderID).To(Equal("test-order-id"))
			})
		})

		Context("when given invalid order", func() {
			It("should return validation error for missing maker", func() {
				order.Maker = ""
				_, err := client.PlaceOrder(ctx, order)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid order"))
			})

			It("should return validation error for missing token ID", func() {
				order.TokenID = ""
				_, err := client.PlaceOrder(ctx, order)
				Expect(err).To(HaveOccurred())
			})

			It("should return validation error for missing maker amount", func() {
				order.MakerAmount = ""
				_, err := client.PlaceOrder(ctx, order)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when API returns error", func() {
			It("should return API error", func() {
				mockServer.Close()
				mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(adaptor.ErrorResponse{
						Error:   "INVALID_ORDER",
						Message: "Order validation failed",
					})
				}))
				config.BaseURL = mockServer.URL
				client = adaptor.NewPolymarketClient()
				err := client.Configure(config)
				Expect(err).ToNot(HaveOccurred())

				_, err = client.PlaceOrder(ctx, order)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("API error"))
				Expect(err.Error()).To(ContainSubstring("INVALID_ORDER"))
			})
		})

		Context("edge case: context cancellation", func() {
			It("should return context error", func() {
				cancelCtx, cancel := context.WithCancel(ctx)
				cancel()

				_, err := client.PlaceOrder(cancelCtx, order)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("CancelOrder", func() {
		var ctx context.Context

		BeforeEach(func() {
			ctx = context.Background()
			err := client.Configure(config)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when client is not configured", func() {
			It("should return an error", func() {
				unconfiguredClient := adaptor.NewPolymarketClient()
				_, err := unconfiguredClient.CancelOrder(ctx, "order-123")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not configured"))
			})
		})

		Context("when given valid order ID", func() {
			It("should cancel the order", func() {
				mockServer.Close()
				mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					Expect(r.Method).To(Equal("DELETE"))
					Expect(r.URL.Path).To(ContainSubstring("/orders/order-123"))

					response := adaptor.CancelOrderResponse{
						OrderID: "order-123",
						Status:  "CANCELLED",
					}
					json.NewEncoder(w).Encode(response)
				}))
				config.BaseURL = mockServer.URL
				client = adaptor.NewPolymarketClient()
				err := client.Configure(config)
				Expect(err).ToNot(HaveOccurred())

				resp, err := client.CancelOrder(ctx, "order-123")
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.OrderID).To(Equal("order-123"))
				Expect(resp.Status).To(Equal("CANCELLED"))
			})
		})

		Context("when given empty order ID", func() {
			It("should return an error", func() {
				_, err := client.CancelOrder(ctx, "")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("orderID is required"))
			})
		})
	})

	Describe("CancelAllOrders", func() {
		var (
			ctx context.Context
			req adaptor.CancelAllOrdersRequest
		)

		BeforeEach(func() {
			ctx = context.Background()
			err := client.Configure(config)
			Expect(err).ToNot(HaveOccurred())

			req = adaptor.CancelAllOrdersRequest{
				AssetID: "asset-123",
			}
		})

		Context("when client is not configured", func() {
			It("should return an error", func() {
				unconfiguredClient := adaptor.NewPolymarketClient()
				_, err := unconfiguredClient.CancelAllOrders(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not configured"))
			})
		})

		Context("when given valid request", func() {
			It("should cancel all orders", func() {
				mockServer.Close()
				mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					Expect(r.Method).To(Equal("DELETE"))

					response := adaptor.CancelAllOrdersResponse{
						Cancelled: []string{"order-1", "order-2"},
					}
					json.NewEncoder(w).Encode(response)
				}))
				config.BaseURL = mockServer.URL
				client = adaptor.NewPolymarketClient()
				err := client.Configure(config)
				Expect(err).ToNot(HaveOccurred())

				resp, err := client.CancelAllOrders(ctx, req)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.Cancelled).To(HaveLen(2))
			})
		})

		Context("when given invalid request", func() {
			It("should return validation error", func() {
				invalidReq := adaptor.CancelAllOrdersRequest{}
				_, err := client.CancelAllOrders(ctx, invalidReq)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("GetOrder", func() {
		var ctx context.Context

		BeforeEach(func() {
			ctx = context.Background()
			err := client.Configure(config)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when given valid order ID", func() {
			It("should retrieve the order", func() {
				mockServer.Close()
				mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					Expect(r.Method).To(Equal("GET"))
					Expect(r.URL.Path).To(ContainSubstring("/orders/order-123"))

					response := adaptor.OrderResponse{
						OrderID: "order-123",
						Status:  "OPEN",
					}
					json.NewEncoder(w).Encode(response)
				}))
				config.BaseURL = mockServer.URL
				client = adaptor.NewPolymarketClient()
				err := client.Configure(config)
				Expect(err).ToNot(HaveOccurred())

				resp, err := client.GetOrder(ctx, "order-123")
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.OrderID).To(Equal("order-123"))
			})
		})

		Context("when given empty order ID", func() {
			It("should return an error", func() {
				_, err := client.GetOrder(ctx, "")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("orderID is required"))
			})
		})
	})

	Describe("GetOpenOrders", func() {
		var ctx context.Context

		BeforeEach(func() {
			ctx = context.Background()
			err := client.Configure(config)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when given valid asset ID", func() {
			It("should retrieve open orders", func() {
				mockServer.Close()
				mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					Expect(r.Method).To(Equal("GET"))
					Expect(r.URL.Query().Get("assetID")).To(Equal("asset-123"))

					response := []adaptor.OrderResponse{
						{OrderID: "order-1", Status: "OPEN"},
						{OrderID: "order-2", Status: "OPEN"},
					}
					json.NewEncoder(w).Encode(response)
				}))
				config.BaseURL = mockServer.URL
				client = adaptor.NewPolymarketClient()
				err := client.Configure(config)
				Expect(err).ToNot(HaveOccurred())

				resp, err := client.GetOpenOrders(ctx, "asset-123")
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).To(HaveLen(2))
			})
		})
	})
})
