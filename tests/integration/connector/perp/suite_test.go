//go:build integration

package perp_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPerpConnectorIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Perp Connector Integration Suite")
}
