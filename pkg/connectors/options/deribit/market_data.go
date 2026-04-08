package deribit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	optionsConnector "github.com/wisp-trading/sdk/pkg/types/connector/options"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
)

// fetchExpirations retrieves all unique expirations for a pair
// Calls public/get_instruments per Deribit spec
func (d *deribitOptions) fetchExpirations(ctx context.Context, pair portfolio.Pair) ([]time.Time, error) {
	result, err := d.client.Call(ctx, "public/get_instruments", map[string]interface{}{
		"currency": pair.Base().Symbol(),
		"kind":     "option",
	})
	if err != nil {
		return nil, fmt.Errorf("public/get_instruments failed: %w", err)
	}

	var instruments []struct {
		BaseCurrency  string  `json:"base_currency"`
		QuoteCurrency string  `json:"quote_currency"`
		ExpirationTs  int64   `json:"expiration_timestamp"`
		Strike        float64 `json:"strike"`
		OptionType    string  `json:"option_type"`
	}

	if err := json.Unmarshal(result, &instruments); err != nil {
		return nil, fmt.Errorf("failed to parse instruments: %w", err)
	}

	// Extract unique expirations
	expirationMap := make(map[int64]bool)
	for _, instr := range instruments {
		baseCurrency := portfolio.NewAsset(instr.BaseCurrency)
		quoteCurrency := portfolio.NewAsset(instr.QuoteCurrency)
		instrPair := portfolio.NewPair(baseCurrency, quoteCurrency)

		// Only include instruments matching the requested pair
		if instrPair.Base().Symbol() == pair.Base().Symbol() && instrPair.Quote().Symbol() == pair.Quote().Symbol() {
			expirationMap[instr.ExpirationTs] = true
		}
	}

	// Convert to sorted list
	expirations := make([]time.Time, 0, len(expirationMap))
	for ts := range expirationMap {
		expirations = append(expirations, time.UnixMilli(ts))
	}

	return expirations, nil
}

// fetchStrikes retrieves all available strikes for a pair and expiration
// Calls public/get_instruments per Deribit spec
func (d *deribitOptions) fetchStrikes(ctx context.Context, pair portfolio.Pair, expiration time.Time) ([]float64, error) {
	result, err := d.client.Call(ctx, "public/get_instruments", map[string]interface{}{
		"currency": pair.Base().Symbol(),
		"kind":     "option",
	})
	if err != nil {
		return nil, fmt.Errorf("public/get_instruments failed: %w", err)
	}

	var instruments []struct {
		BaseCurrency  string  `json:"base_currency"`
		QuoteCurrency string  `json:"quote_currency"`
		ExpirationTs  int64   `json:"expiration_timestamp"`
		Strike        float64 `json:"strike"`
	}

	if err := json.Unmarshal(result, &instruments); err != nil {
		return nil, fmt.Errorf("failed to parse instruments: %w", err)
	}

	// Extract strikes for matching pair and expiration
	strikeMap := make(map[float64]bool)
	expirationMs := expiration.UnixMilli()

	for _, instr := range instruments {
		baseCurrency := portfolio.NewAsset(instr.BaseCurrency)
		quoteCurrency := portfolio.NewAsset(instr.QuoteCurrency)
		instrPair := portfolio.NewPair(baseCurrency, quoteCurrency)

		// Check if pair and expiration match
		if instrPair.Base().Symbol() == pair.Base().Symbol() &&
			instrPair.Quote().Symbol() == pair.Quote().Symbol() &&
			instr.ExpirationTs == expirationMs {
			strikeMap[instr.Strike] = true
		}
	}

	// Convert to sorted list
	strikes := make([]float64, 0, len(strikeMap))
	for strike := range strikeMap {
		strikes = append(strikes, strike)
	}

	return strikes, nil
}

// fetchOptionData retrieves mark price and Greeks for a specific option
// Calls public/ticker per Deribit spec
func (d *deribitOptions) fetchOptionData(ctx context.Context, instrumentName string) (optionsConnector.OptionData, error) {
	result, err := d.client.Call(ctx, "public/ticker", map[string]interface{}{
		"instrument_name": instrumentName,
	})
	if err != nil {
		return optionsConnector.OptionData{}, fmt.Errorf("public/ticker failed: %w", err)
	}

	var ticker struct {
		InstrumentName  string   `json:"instrument_name"`
		Timestamp       int64    `json:"timestamp"`
		UnderlyingPrice float64  `json:"underlying_price"`
		MarkPrice       float64  `json:"mark_price"`
		MarkIV          float64  `json:"mark_iv"`
		BestBidPrice    *float64 `json:"best_bid_price"`
		BestAskPrice    *float64 `json:"best_ask_price"`
		OpenInterest    float64  `json:"open_interest"`
		Stats           struct {
			Volume float64 `json:"volume_usd"`
		} `json:"stats"`
		Greeks struct {
			Delta float64 `json:"delta"`
			Gamma float64 `json:"gamma"`
			Theta float64 `json:"theta"`
			Vega  float64 `json:"vega"`
			Rho   float64 `json:"rho"`
		} `json:"greeks"`
	}

	if err := json.Unmarshal(result, &ticker); err != nil {
		return optionsConnector.OptionData{}, fmt.Errorf("failed to parse ticker: %w", err)
	}

	// Calculate bid-ask spread
	bidAskSpread := 0.0
	if ticker.BestBidPrice != nil && ticker.BestAskPrice != nil {
		bidAskSpread = *ticker.BestAskPrice - *ticker.BestBidPrice
	}

	return optionsConnector.OptionData{
		MarkPrice:       ticker.MarkPrice,
		UnderlyingPrice: ticker.UnderlyingPrice,
		IV:              ticker.MarkIV,
		Greeks: optionsConnector.Greeks{
			Delta: ticker.Greeks.Delta,
			Gamma: ticker.Greeks.Gamma,
			Theta: ticker.Greeks.Theta,
			Vega:  ticker.Greeks.Vega,
			Rho:   ticker.Greeks.Rho,
		},
		BidAskSpread: bidAskSpread,
		Volume24h:    ticker.Stats.Volume,
		OpenInterest: ticker.OpenInterest,
		Timestamp:    time.UnixMilli(ticker.Timestamp),
	}, nil
}
