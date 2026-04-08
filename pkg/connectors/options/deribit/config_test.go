package deribit

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deribit Config Suite")
}

var _ = Describe("Config Validation", func() {
	It("should require client_id", func() {
		cfg := &Config{
			ClientSecret: "secret",
		}

		err := cfg.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("client_id"))
	})

	It("should require client_secret", func() {
		cfg := &Config{
			ClientID: "id",
		}

		err := cfg.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("client_secret"))
	})

	It("should set default URLs for production", func() {
		cfg := &Config{
			ClientID:     "test-id",
			ClientSecret: "test-secret",
			UseTestnet:   false,
		}

		err := cfg.Validate()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.BaseURL).To(Equal("https://www.deribit.com/api/v2"))
		Expect(cfg.WebSocketURL).To(Equal("wss://www.deribit.com/ws/api/v2"))
	})

	It("should set default URLs for testnet", func() {
		cfg := &Config{
			ClientID:     "test-id",
			ClientSecret: "test-secret",
			UseTestnet:   true,
		}

		err := cfg.Validate()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.BaseURL).To(Equal("https://test.deribit.com/api/v2"))
		Expect(cfg.WebSocketURL).To(Equal("wss://test.deribit.com/ws/api/v2"))
	})

	It("should set default slippage", func() {
		cfg := &Config{
			ClientID:     "test-id",
			ClientSecret: "test-secret",
		}

		err := cfg.Validate()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.DefaultSlippage).To(Equal(0.005))
	})

	It("should reject invalid slippage", func() {
		cfg := &Config{
			ClientID:        "test-id",
			ClientSecret:    "test-secret",
			DefaultSlippage: 0.15, // 15%, too high
		}

		err := cfg.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("default_slippage"))
	})

	It("should return correct exchange name", func() {
		cfg := &Config{
			ClientID:     "test-id",
			ClientSecret: "test-secret",
		}

		Expect(cfg.ExchangeName()).To(Equal("deribit_options"))
	})
})
