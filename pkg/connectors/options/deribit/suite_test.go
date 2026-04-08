package deribit

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDeribitOptions(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deribit Options Suite")
}
