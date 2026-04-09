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

			It("should set default SignatureType to 2", func() {
				conf.SignatureType = 0
				err := conf.Validate()
				Expect(err).ToNot(HaveOccurred())
				Expect(conf.SignatureType).To(Equal(2))
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
			It("should return an error", func() {
				conf.PolymarketAddress = ""
				err := conf.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("funder_address is required"))
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

		Context("when PolymarketAddress has invalid format", func() {
			It("should return an error for non-hex address", func() {
				conf.PolymarketAddress = "not-an-address"
				err := conf.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("funder_address must be a valid Ethereum address"))
			})

			It("should return an error for wrong length", func() {
				conf.PolymarketAddress = "0x123"
				err := conf.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("funder_address must be a valid Ethereum address"))
			})
		})
	})
})
