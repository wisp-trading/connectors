package polymarket_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket"
	"github.com/wisp-trading/connectors/pkg/connectors/types"
)

var _ = Describe("Config", func() {
	var config *polymarket.Config

	BeforeEach(func() {
		config = &polymarket.Config{
			APIKey:        "test-api-key",
			APISecret:     "test-api-secret",
			Passphrase:    "test-passphrase",
			PrivateKey:    "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			FunderAddress: "0xabcdef1234567890abcdef1234567890abcdef12",
		}
	})

	Describe("ExchangeName", func() {
		Context("when called", func() {
			It("should return Polymarket exchange name", func() {
				Expect(config.ExchangeName()).To(Equal(types.Polymarket))
			})
		})
	})

	Describe("Validate", func() {
		Context("when given valid configuration", func() {
			It("should not return an error", func() {
				err := config.Validate()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should set default BaseURL", func() {
				config.BaseURL = ""
				err := config.Validate()
				Expect(err).ToNot(HaveOccurred())
				Expect(config.BaseURL).To(Equal("https://clob.polymarket.com"))
			})

			It("should set default GammaURL", func() {
				config.GammaURL = ""
				err := config.Validate()
				Expect(err).ToNot(HaveOccurred())
				Expect(config.GammaURL).To(Equal("https://gamma-api.polymarket.com"))
			})

			It("should set default WebSocketURL", func() {
				config.WebSocketURL = ""
				err := config.Validate()
				Expect(err).ToNot(HaveOccurred())
				Expect(config.WebSocketURL).To(Equal("wss://ws-subscriptions-clob.polymarket.com/ws/market"))
			})

			It("should set default ChainID to 137 (Polygon)", func() {
				config.ChainID = 0
				err := config.Validate()
				Expect(err).ToNot(HaveOccurred())
				Expect(config.ChainID).To(Equal(137))
			})

			It("should set default SignatureType to 2", func() {
				config.SignatureType = 0
				err := config.Validate()
				Expect(err).ToNot(HaveOccurred())
				Expect(config.SignatureType).To(Equal(2))
			})
		})

		Context("when APIKey is missing", func() {
			It("should return an error", func() {
				config.APIKey = ""
				err := config.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("api_key is required"))
			})
		})

		Context("when APISecret is missing", func() {
			It("should return an error", func() {
				config.APISecret = ""
				err := config.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("api_secret is required"))
			})
		})

		Context("when Passphrase is missing", func() {
			It("should return an error", func() {
				config.Passphrase = ""
				err := config.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("passphrase is required"))
			})
		})

		Context("when PrivateKey is missing", func() {
			It("should return an error", func() {
				config.PrivateKey = ""
				err := config.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("private_key is required"))
			})
		})

		Context("when FunderAddress is missing", func() {
			It("should return an error", func() {
				config.FunderAddress = ""
				err := config.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("funder_address is required"))
			})
		})

		Context("when PrivateKey has invalid format", func() {
			It("should return an error for non-hex string", func() {
				config.PrivateKey = "not-a-hex-string"
				err := config.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("private_key must be a valid hex string"))
			})

			It("should return an error for short key", func() {
				config.PrivateKey = "0x123"
				err := config.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("private_key must be a valid hex string"))
			})
		})

		Context("when FunderAddress has invalid format", func() {
			It("should return an error for non-hex address", func() {
				config.FunderAddress = "not-an-address"
				err := config.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("funder_address must be a valid Ethereum address"))
			})

			It("should return an error for wrong length", func() {
				config.FunderAddress = "0x123"
				err := config.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("funder_address must be a valid Ethereum address"))
			})
		})

		Context("edge case: custom URLs provided", func() {
			It("should preserve custom BaseURL", func() {
				config.BaseURL = "https://custom-clob.example.com"
				err := config.Validate()
				Expect(err).ToNot(HaveOccurred())
				Expect(config.BaseURL).To(Equal("https://custom-clob.example.com"))
			})

			It("should preserve custom GammaURL", func() {
				config.GammaURL = "https://custom-gamma.example.com"
				err := config.Validate()
				Expect(err).ToNot(HaveOccurred())
				Expect(config.GammaURL).To(Equal("https://custom-gamma.example.com"))
			})

			It("should preserve custom WebSocketURL", func() {
				config.WebSocketURL = "wss://custom-ws.example.com/ws"
				err := config.Validate()
				Expect(err).ToNot(HaveOccurred())
				Expect(config.WebSocketURL).To(Equal("wss://custom-ws.example.com/ws"))
			})
		})

		Context("edge case: non-default ChainID", func() {
			It("should preserve custom ChainID", func() {
				config.ChainID = 1 // Ethereum mainnet
				err := config.Validate()
				Expect(err).ToNot(HaveOccurred())
				Expect(config.ChainID).To(Equal(1))
			})
		})
	})
})
