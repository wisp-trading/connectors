package onchain_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOnchainConnector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Onchain Connector Integration Suite")
}
