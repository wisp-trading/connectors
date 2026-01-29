package hyperliquid

import (
	"fmt"
	"log"
	"strings"

	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// normaliseAssetName converts an asset symbol to the format Hyperliquid API accepts
// Examples: "btc" -> "BTC", "BTC" -> "BTC", "Btc" -> "BTC"
func (h *hyperliquid) normaliseAssetName(asset portfolio.Asset) string {
	return strings.ToUpper(asset.Symbol())
}

// extractOrderID extracts order ID from trading service response
func (h *hyperliquid) extractOrderID(result interface{}) string {
	// TODO: Implement based on actual trading service response structure
	return fmt.Sprintf("%d", h.timeProvider.Now().UnixNano())
}

// convertInterval converts standard interval format to Hyperliquid format
func convertInterval(interval string) string {
	switch interval {
	case "1m":
		return "1m"
	case "5m":
		return "5m"
	case "15m":
		return "15m"
	case "1h":
		return "1h"
	case "4h":
		return "4h"
	case "1d":
		return "1d"
	default:
		return "1h"
	}
}

// intervalToSeconds converts interval string to seconds
func intervalToSeconds(interval string) int {
	switch interval {
	case "1m":
		return 60
	case "5m":
		return 300
	case "15m":
		return 900
	case "1h":
		return 3600
	case "4h":
		return 14400
	case "1d":
		return 86400
	default:
		return 3600
	}
}

func parseDecimal(value string) numerical.Decimal {
	if value == "" {
		return numerical.Zero()
	}

	d, err := numerical.NewFromString(value)
	if err != nil {
		log.Printf("Failed to parse decimal '%s': %v", value, err)
		return numerical.Zero()
	}

	return d
}
