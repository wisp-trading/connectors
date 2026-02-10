package prediction_markets_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPredictionMarkets(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Prediction Markets Connector Suite")
}
