package connector_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConnectorIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Connector Integration Suite")
}
