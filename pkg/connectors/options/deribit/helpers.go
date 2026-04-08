package deribit

import (
	"fmt"
	"strings"
	"time"

	optionsConnector "github.com/wisp-trading/sdk/pkg/types/connector/options"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
)

// formatInstrumentName converts an options contract to Deribit's instrument name format
// Format: "BTC-8APR26-59000-C" (or -P for put)
// This matches Deribit's naming convention: SYMBOL-DDMMMYY-STRIKE-TYPE
// NOTE: Uses UTC time to match Deribit's instrument naming (which is UTC-based)
func formatInstrumentName(contract optionsConnector.OptionContract) string {
	base := contract.Pair.Base().Symbol()

	// Convert to UTC to match Deribit's instrument naming convention
	expirationUTC := contract.Expiration.UTC()

	// Format expiration as DDMMMYY (e.g., "08APR26") in UTC
	day := expirationUTC.Day()
	monthName := strings.ToUpper(expirationUTC.Format("Jan")[:3])
	year := expirationUTC.Year() % 100 // Last 2 digits
	expirationDate := fmt.Sprintf("%d%s%02d", day, monthName, year)

	strike := int64(contract.Strike)
	optionType := strings.ToUpper(contract.OptionType[0:1]) // "C" or "P"

	return fmt.Sprintf("%s-%s-%d-%s", base, expirationDate, strike, optionType)
}

// parseInstrumentName extracts contract details from a Deribit instrument name
// Expects format: "BTC-8APR26-59000-C"
func parseInstrumentName(name string, quote portfolio.Asset) (optionsConnector.OptionContract, error) {
	parts := strings.Split(name, "-")
	if len(parts) < 4 {
		return optionsConnector.OptionContract{}, fmt.Errorf("invalid instrument name format: %s", name)
	}

	base := portfolio.NewAsset(parts[0])
	pair := portfolio.NewPair(base, quote)

	// Parse expiration date from format DDMMMYY (e.g., "08APR26")
	expirationStr := parts[1]
	expiration, err := parseDateFromDeribit(expirationStr)
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
// Returns format like "08APR26" (DDMMMYY)
func formatDateForDeribit(t time.Time) string {
	day := t.Day()
	monthName := strings.ToUpper(t.Format("Jan")[:3])
	year := t.Year() % 100
	return fmt.Sprintf("%02d%s%02d", day, monthName, year)
}

// parseDateFromDeribit parses Deribit's date format to time.Time
// Expects format like "08APR26" (DDMMMYY)
func parseDateFromDeribit(dateStr string) (time.Time, error) {
	if len(dateStr) != 7 {
		return time.Time{}, fmt.Errorf("invalid date format: expected DDMMMYY, got %s", dateStr)
	}

	day := dateStr[0:2]
	month := dateStr[2:5]
	year := dateStr[5:7]

	// Parse with Go's reference format (needs lowercase month)
	dateFormatStr := fmt.Sprintf("%s%s%s", day, strings.ToLower(month), year)
	return time.Parse("02Jan06", dateFormatStr)
}

