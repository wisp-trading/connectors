package deribit

import (
	"time"

	optionsConnector "github.com/wisp-trading/sdk/pkg/types/connector/options"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Instrument Name Formatting", func() {
	It("should format instrument name correctly for CALL option", func() {
		btcPair := portfolio.NewPair(portfolio.NewAsset("BTC"), portfolio.NewAsset("USDT"))
		expiration := time.Date(2025, time.December, 31, 0, 0, 0, 0, time.UTC)

		contract := optionsConnector.OptionContract{
			Pair:       btcPair,
			Strike:     50000,
			Expiration: expiration,
			OptionType: "CALL",
		}

		name := formatInstrumentName(contract)
		Expect(name).To(Equal("BTC-31DEC25-50000-C"))
	})

	It("should format instrument name correctly for PUT option", func() {
		ethPair := portfolio.NewPair(portfolio.NewAsset("ETH"), portfolio.NewAsset("USDT"))
		expiration := time.Date(2025, time.December, 31, 0, 0, 0, 0, time.UTC)

		contract := optionsConnector.OptionContract{
			Pair:       ethPair,
			Strike:     2000,
			Expiration: expiration,
			OptionType: "PUT",
		}

		name := formatInstrumentName(contract)
		Expect(name).To(Equal("ETH-31DEC25-2000-P"))
	})
})

var _ = Describe("Date Formatting", func() {
	It("should format date for Deribit correctly", func() {
		t := time.Date(2025, time.December, 31, 0, 0, 0, 0, time.UTC)
		formatted := formatDateForDeribit(t)
		Expect(formatted).To(Equal("31DEC25"))
	})

	It("should parse Deribit date format correctly", func() {
		dateStr := "31DEC25"
		t, err := parseDateFromDeribit(dateStr)
		Expect(err).NotTo(HaveOccurred())
		Expect(t.Day()).To(Equal(31))
		Expect(t.Month()).To(Equal(time.December))
		Expect(t.Year()).To(Equal(2025))
	})
})
