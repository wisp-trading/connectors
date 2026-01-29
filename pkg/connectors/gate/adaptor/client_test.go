package adaptor_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wisp-trading/connectors/pkg/connectors/gate/adaptor"
)

func TestAdaptor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gate Adaptor Suite")
}

var _ = Describe("SpotClient", func() {
	var client adaptor.SpotClient

	BeforeEach(func() {
		client = adaptor.NewSpotClient()
	})

	Describe("Configuration", func() {
		Context("when newly created", func() {
			It("should not be configured", func() {
				Expect(client.IsConfigured()).To(BeFalse())
			})
		})

		Context("when configured with valid credentials", func() {
			It("should be configured successfully", func() {
				err := client.Configure("test-key", "test-secret", "https://api.gateio.ws/api/v4")
				Expect(err).ToNot(HaveOccurred())
				Expect(client.IsConfigured()).To(BeTrue())
			})
		})

		Context("when configured twice", func() {
			It("should return an error", func() {
				err := client.Configure("test-key", "test-secret", "https://api.gateio.ws/api/v4")
				Expect(err).ToNot(HaveOccurred())

				err = client.Configure("test-key", "test-secret", "https://api.gateio.ws/api/v4")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("already configured"))
			})
		})

		Context("when getting API before configuration", func() {
			It("should return an error", func() {
				api, err := client.GetSpotApi()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not configured"))
				Expect(api).To(BeNil())
			})
		})

		Context("when getting API after configuration", func() {
			It("should return the API client", func() {
				err := client.Configure("test-key", "test-secret", "https://api.gateio.ws/api/v4")
				Expect(err).ToNot(HaveOccurred())

				api, err := client.GetSpotApi()
				Expect(err).ToNot(HaveOccurred())
				Expect(api).ToNot(BeNil())
			})
		})
	})
})

var _ = Describe("FuturesClient", func() {
	var client adaptor.FuturesClient

	BeforeEach(func() {
		client = adaptor.NewFuturesClient()
	})

	Describe("Configuration", func() {
		Context("when newly created", func() {
			It("should not be configured", func() {
				Expect(client.IsConfigured()).To(BeFalse())
			})
		})

		Context("when configured with valid credentials", func() {
			It("should be configured successfully", func() {
				err := client.Configure("test-key", "test-secret", "https://api.gateio.ws/api/v4", "usdt")
				Expect(err).ToNot(HaveOccurred())
				Expect(client.IsConfigured()).To(BeTrue())
				Expect(client.GetSettle()).To(Equal("usdt"))
			})
		})

		Context("when configured with empty settle", func() {
			It("should default to usdt", func() {
				err := client.Configure("test-key", "test-secret", "https://api.gateio.ws/api/v4", "")
				Expect(err).ToNot(HaveOccurred())
				Expect(client.GetSettle()).To(Equal("usdt"))
			})
		})

		Context("when configured twice", func() {
			It("should return an error", func() {
				err := client.Configure("test-key", "test-secret", "https://api.gateio.ws/api/v4", "usdt")
				Expect(err).ToNot(HaveOccurred())

				err = client.Configure("test-key", "test-secret", "https://api.gateio.ws/api/v4", "usdt")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("already configured"))
			})
		})

		Context("when getting API before configuration", func() {
			It("should return an error", func() {
				api, err := client.GetFuturesApi()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not configured"))
				Expect(api).To(BeNil())
			})
		})

		Context("when getting API after configuration", func() {
			It("should return the API client", func() {
				err := client.Configure("test-key", "test-secret", "https://api.gateio.ws/api/v4", "usdt")
				Expect(err).ToNot(HaveOccurred())

				api, err := client.GetFuturesApi()
				Expect(err).ToNot(HaveOccurred())
				Expect(api).ToNot(BeNil())
			})
		})
	})
})
