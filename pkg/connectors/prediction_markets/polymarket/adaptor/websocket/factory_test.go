package websocket_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/websocket"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/config"
	"github.com/wisp-trading/sdk/pkg/types/logging"
)

var _ = Describe("WebSocketServiceFactory", func() {
	var (
		factory websocket.WebSocketServiceFactory
		cfg     *config.Config
		logger  logging.ApplicationLogger
	)

	BeforeEach(func() {
		logger = logging.NewNoOpLogger()
		factory = websocket.NewWebSocketServiceFactory(logger)

		cfg = &config.Config{
			APIKey:            "test-api-key",
			APISecret:         "test-api-secret",
			Passphrase:        "test-passphrase",
			PrivateKey:        "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			PolymarketAddress: "0x1234567890123456789012345678901234567890",
			WebSocketURL:      "wss://ws-test.polymarket.com/ws",
			BaseURL:           "https://clob-test.polymarket.com",
			GammaURL:          "https://gamma-test.polymarket.com",
			ChainID:           137,
			SignatureType:     2,
		}
	})

	Describe("CreateWebSocketService", func() {
		Context("with valid config", func() {
			It("should create WebSocket service successfully", func() {
				service, err := factory.CreateWebSocketService(cfg)

				Expect(err).ToNot(HaveOccurred())
				Expect(service).ToNot(BeNil())
			})
		})

		Context("with nil config", func() {
			It("should return error", func() {
				service, err := factory.CreateWebSocketService(nil)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("config cannot be nil"))
				Expect(service).To(BeNil())
			})
		})

		Context("with invalid config", func() {
			It("should return validation error", func() {
				invalidCfg := &config.Config{
					// Missing required fields
					APIKey: "",
				}

				service, err := factory.CreateWebSocketService(invalidCfg)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid config"))
				Expect(service).To(BeNil())
			})
		})

		Context("when creating multiple services", func() {
			It("should create independent service instances", func() {
				service1, err1 := factory.CreateWebSocketService(cfg)
				Expect(err1).ToNot(HaveOccurred())

				service2, err2 := factory.CreateWebSocketService(cfg)
				Expect(err2).ToNot(HaveOccurred())

				// Should be different instances
				Expect(service1).ToNot(BeIdenticalTo(service2))
			})
		})
	})

	Describe("Factory creation", func() {
		It("should create factory successfully", func() {
			Expect(factory).ToNot(BeNil())
		})

		It("should be lightweight (no dependencies on config)", func() {
			// Factory should be created without any config
			// This test passes if factory creation doesn't panic
			newFactory := websocket.NewWebSocketServiceFactory(logger)
			Expect(newFactory).ToNot(BeNil())
		})
	})
})
