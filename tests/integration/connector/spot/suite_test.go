package spot_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSpotConnectorIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Spot Connector Integration Suite")
}
