package hyperliquid

import (
	"log"
	"strings"

	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// normaliseAssetName converts an asset symbol to the format Hyperliquid API accepts
func (h *hyperliquidSpot) normaliseAssetName(asset portfolio.Asset) string {
	return strings.ToUpper(asset.Symbol())
}

func (h *hyperliquidSpot) coinToPair(coin string) portfolio.Pair {
	return portfolio.NewPair(
		portfolio.NewAsset(strings.ToUpper(coin)),
		portfolio.NewAsset("USDC"),
	)
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

// convertInterval converts standard interval format to Hyperliquid format
func convertInterval(interval string) string {
	switch interval {
	case "1m", "5m", "15m", "1h", "4h", "1d":
		return interval
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
