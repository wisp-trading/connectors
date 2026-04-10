package order_manager_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOrderManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Polymarket Order Manager Suite")
}
