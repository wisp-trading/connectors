package config_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/config"
	"github.com/wisp-trading/connectors/pkg/connectors/types"
)

var _ = Describe("Config", func() {
	var conf *config.Config

	BeforeEach(func() {
		conf = &config.Config{
			PrivateKey:        "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			PolymarketAddress: "0xabcdef1234567890abcdef1234567890abcdef12",
			PolygonRPCURL:     "https://polygon-mainnet.example.com/v2/test",
		}
	})

	Describe("ExchangeName", func() {
		Context("when called", func() {
			It("should return Polymarket exchange name", func() {
				Expect(conf.ExchangeName()).To(Equal(types.Polymarket))
			})
		})
	})

	Describe("Validate", func() {
		Context("when given valid configuration", func() {
			It("should not return an error", func() {
				err := conf.Validate()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should allow empty SignatureType (EOA default)", func() {
				conf.SignatureType = 0
				err := conf.Validate()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when PrivateKey is missing", func() {
			It("should return an error", func() {
				conf.PrivateKey = ""
				err := conf.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("private_key is required"))
			})
		})

		Context("when PolymarketAddress is missing", func() {
			It("should allow empty address (optional / auto-derived)", func() {
				conf.PolymarketAddress = ""
				err := conf.Validate()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when PrivateKey has invalid format", func() {
			It("should return an error for non-hex string", func() {
				conf.PrivateKey = "not-a-hex-string"
				err := conf.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("private_key must be a valid hex string"))
			})

			It("should return an error for short key", func() {
				conf.PrivateKey = "0x123"
				err := conf.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("private_key must be a valid hex string"))
			})
		})

		Context("when PolygonRPCURL is missing", func() {
			It("should return an error", func() {
				conf.PolygonRPCURL = ""
				err := conf.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("polygon_rpc_url is required"))
			})
		})

		Context("when PolymarketAddress has invalid format", func() {
			It("should return an error for non-hex address", func() {
				conf.PolymarketAddress = "not-an-address"
				err := conf.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("polymarket_address must be a valid Ethereum address"))
			})

			It("should return an error for wrong length", func() {
				conf.PolymarketAddress = "0x123"
				err := conf.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("polymarket_address must be a valid Ethereum address"))
			})
		})
	})
})
