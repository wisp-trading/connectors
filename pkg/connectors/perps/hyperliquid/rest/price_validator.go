package rest

import (
	"fmt"
	"math"
	"sync"

	"github.com/sonirico/go-hyperliquid"
)

type PriceValidator interface {
	LoadAssetInfo(meta *hyperliquid.Meta) error
	RoundPrice(coin string, price float64) (float64, error)
	RoundSize(coin string, size float64) (float64, error)
	GetTickSize(coin string) (float64, error)
}

// priceValidator handles price validation and rounding based on asset tick sizes
type priceValidator struct {
	assetCache map[string]*AssetInfo
	mu         sync.RWMutex
}

// AssetInfo contains tick size and other trading rules for an asset
type AssetInfo struct {
	Name       string
	SzDecimals int
	TickSize   float64
}

// NewPriceValidator creates a new price validator
func NewPriceValidator() PriceValidator {
	return &priceValidator{
		assetCache: make(map[string]*AssetInfo),
	}
}

// LoadAssetInfo fetches and caches asset metadata
func (pv *priceValidator) LoadAssetInfo(meta *hyperliquid.Meta) error {
	pv.mu.Lock()
	defer pv.mu.Unlock()

	const MAX_DECIMALS = 6 // For perpetuals (spot uses 8)

	for _, asset := range meta.Universe {
		// Calculate tick size using Hyperliquid's formula:
		// Price decimals = MAX_DECIMALS - szDecimals
		// Tick size = 10^-priceDecimals
		priceDecimals := MAX_DECIMALS - asset.SzDecimals
		tickSize := math.Pow(10, -float64(priceDecimals))

		pv.assetCache[asset.Name] = &AssetInfo{
			Name:       asset.Name,
			SzDecimals: asset.SzDecimals,
			TickSize:   tickSize,
		}
	}

	return nil
}

// RoundPrice rounds a price to the valid tick size for the given asset
// Also enforces Hyperliquid's 5 significant figures rule
func (pv *priceValidator) RoundPrice(coin string, price float64) (float64, error) {
	pv.mu.RLock()
	assetInfo, exists := pv.assetCache[coin]
	pv.mu.RUnlock()

	if !exists {
		return 0, fmt.Errorf("asset %s not found in cache", coin)
	}

	// Round to the nearest tick
	roundedPrice := math.Round(price/assetInfo.TickSize) * assetInfo.TickSize

	// Hyperliquid enforces max 5 significant figures
	// Round to 5 significant figures if needed
	roundedPrice = roundToSignificantFigures(roundedPrice, 5)

	// Round again to tick size after sig fig rounding
	roundedPrice = math.Round(roundedPrice/assetInfo.TickSize) * assetInfo.TickSize

	// Format to proper precision to avoid floating point issues
	precision := int(math.Log10(1 / assetInfo.TickSize))
	multiplier := math.Pow(10, float64(precision))
	roundedPrice = math.Round(roundedPrice*multiplier) / multiplier

	return roundedPrice, nil
}

// roundToSignificantFigures rounds a number to n significant figures
func roundToSignificantFigures(num float64, sigFigs int) float64 {
	if num == 0 {
		return 0
	}

	// Get the order of magnitude
	magnitude := math.Floor(math.Log10(math.Abs(num)))

	// Calculate the multiplier to get n significant figures
	multiplier := math.Pow(10, float64(sigFigs-1)-magnitude)

	// Round and scale back
	return math.Round(num*multiplier) / multiplier
}

// RoundSize rounds a size to the valid size decimals for the given asset
func (pv *priceValidator) RoundSize(coin string, size float64) (float64, error) {
	pv.mu.RLock()
	assetInfo, exists := pv.assetCache[coin]
	pv.mu.RUnlock()

	if !exists {
		return 0, fmt.Errorf("asset %s not found in cache", coin)
	}

	// Round size to szDecimals precision
	multiplier := math.Pow(10, float64(assetInfo.SzDecimals))
	roundedSize := math.Round(size*multiplier) / multiplier

	return roundedSize, nil
}

// GetTickSize returns the tick size for an asset
func (pv *priceValidator) GetTickSize(coin string) (float64, error) {
	pv.mu.RLock()
	defer pv.mu.RUnlock()

	assetInfo, exists := pv.assetCache[coin]
	if !exists {
		return 0, fmt.Errorf("asset %s not found in cache", coin)
	}

	return assetInfo.TickSize, nil
}
