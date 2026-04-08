package adaptor

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deribit Adaptor Client Suite")
}

var _ = Describe("Deribit Client", func() {
	var client Client

	BeforeEach(func() {
		client = NewClient()
	})

	Describe("Configuration", func() {
		It("should not be configured initially", func() {
			Expect(client.IsConfigured()).To(BeFalse())
		})

		It("should configure successfully", func() {
			err := client.Configure("test-id", "test-secret", "https://test.deribit.com/api/v2")
			Expect(err).NotTo(HaveOccurred())
			Expect(client.IsConfigured()).To(BeTrue())
		})

		It("should require all parameters", func() {
			err := client.Configure("", "secret", "url")
			Expect(err).To(HaveOccurred())

			err = client.Configure("id", "", "url")
			Expect(err).To(HaveOccurred())

			err = client.Configure("id", "secret", "")
			Expect(err).To(HaveOccurred())
		})

		It("should prevent double configuration", func() {
			err := client.Configure("id1", "secret1", "url1")
			Expect(err).NotTo(HaveOccurred())

			err = client.Configure("id2", "secret2", "url2")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already configured"))
		})
	})

	Describe("BaseURL retrieval", func() {
		It("should return the configured base URL", func() {
			baseURL := "https://test.deribit.com/api/v2"
			err := client.Configure("id", "secret", baseURL)
			Expect(err).NotTo(HaveOccurred())

			Expect(client.GetBaseURL()).To(Equal(baseURL))
		})
	})

	Describe("API calls", func() {
		It("should reject public calls when not configured", func() {
			_, err := client.Call(context.Background(), "public/get_instruments", map[string]interface{}{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not configured"))
		})

		It("should reject private calls when not configured", func() {
			_, err := client.Call(context.Background(), "private/buy", map[string]interface{}{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not configured"))
		})
	})

})
