package deribit

import (
	"fmt"
	"strings"
	"time"

	optionsConnector "github.com/wisp-trading/sdk/pkg/types/connector/options"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
)

// formatInstrumentName converts an options contract to Deribit's instrument name format
// Format: "BTC-31DEC25-50000-C" (or -P for put)
// This matches Deribit's naming convention
func formatInstrumentName(contract optionsConnector.OptionContract) string {
	base := contract.Pair.Base().Symbol()
	expirationDate := contract.Expiration.Format("02JAN06") // e.g., "31DEC25"
	strike := int64(contract.Strike)
	optionType := strings.ToUpper(contract.OptionType[0:1]) // "C" or "P"

	return fmt.Sprintf("%s-%s-%d-%s", base, expirationDate, strike, optionType)
}

// parseInstrumentName extracts contract details from a Deribit instrument name
// Expects format: "BTC-31DEC25-50000-C"
func parseInstrumentName(name string, quote portfolio.Asset) (optionsConnector.OptionContract, error) {
	parts := strings.Split(name, "-")
	if len(parts) < 4 {
		return optionsConnector.OptionContract{}, fmt.Errorf("invalid instrument name format: %s", name)
	}

	base := portfolio.NewAsset(parts[0])
	pair := portfolio.NewPair(base, quote)

	// Parse expiration date
	expirationStr := parts[1]
	expiration, err := time.Parse("02JAN06", expirationStr)
	if err != nil {
		return optionsConnector.OptionContract{}, fmt.Errorf("failed to parse expiration: %w", err)
	}

	// Parse strike price
	var strike float64
	_, err = fmt.Sscanf(parts[2], "%f", &strike)
	if err != nil {
		return optionsConnector.OptionContract{}, fmt.Errorf("failed to parse strike: %w", err)
	}

	// Parse option type
	optionType := "CALL"
	if strings.ToUpper(parts[3]) == "P" {
		optionType = "PUT"
	}

	return optionsConnector.OptionContract{
		Pair:       pair,
		Strike:     strike,
		Expiration: expiration,
		OptionType: optionType,
	}, nil
}

// formatDateForDeribit formats a time.Time to Deribit's date format
// Returns format like "31DEC25"
func formatDateForDeribit(t time.Time) string {
	return t.Format("02JAN06")
}

// parseDateFromDeribit parses Deribit's date format to time.Time
// Expects format like "31DEC25"
func parseDateFromDeribit(dateStr string) (time.Time, error) {
	return time.Parse("02JAN06", dateStr)
}

