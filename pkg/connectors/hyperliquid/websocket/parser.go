package websocket

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/kronos/numerical"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/temporal"
	hyperliquidsdk "github.com/sonirico/go-hyperliquid"
)

// MessageParser defines the interface for parsing WebSocket messages
type MessageParser interface {
	ParseOrderBook(msg hyperliquidsdk.WSMessage) (*OrderBookMessage, error)
	ParseTrades(msg hyperliquidsdk.WSMessage) ([]TradeMessage, error)
	ParsePosition(msg hyperliquidsdk.WSMessage) (*PositionMessage, error)
	ParseAccountBalance(msg hyperliquidsdk.WSMessage) (*AccountBalanceMessage, error)
	ParseKline(msg hyperliquidsdk.WSMessage) (*KlineMessage, error)
	ParseFundingRate(msg hyperliquidsdk.WSMessage) (*FundingRateMessage, error)
}

// Parser handles parsing of WebSocket messages into typed structs
type Parser struct {
	logger       logging.ApplicationLogger
	timeProvider temporal.TimeProvider
}

// NewParser creates a new WebSocket message parser
func NewParser(logger logging.ApplicationLogger, timeProvider temporal.TimeProvider) MessageParser {
	return &Parser{
		logger:       logger,
		timeProvider: timeProvider,
	}
}

// ParseOrderBook parses a raw WebSocket message into an OrderBookMessage
func (p *Parser) ParseOrderBook(msg hyperliquidsdk.WSMessage) (*OrderBookMessage, error) {
	if msg.Channel != "l2Book" {
		return nil, fmt.Errorf("expected l2Book channel, got %s", msg.Channel)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal orderbook data: %w", err)
	}

	coin, ok := data["coin"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid coin field")
	}

	levels, ok := data["levels"].([]interface{})
	if !ok || len(levels) < 2 {
		return nil, fmt.Errorf("missing or invalid levels field")
	}

	var bids []PriceLevel
	var asks []PriceLevel

	// Parse bids
	if bidData, ok := levels[0].([]interface{}); ok {
		for _, bid := range bidData {
			if bidLevel, ok := bid.(map[string]interface{}); ok {
				priceStr, okPx := bidLevel["px"].(string)
				sizeStr, okSz := bidLevel["sz"].(string)
				if !okPx || !okSz {
					p.logger.Warn("Invalid bid level data", "coin", coin)
					continue
				}

				price, err := numerical.NewFromString(priceStr)
				if err != nil {
					p.logger.Warn("Invalid bid price", "coin", coin, "price", priceStr, "error", err)
					continue
				}

				quantity, err := numerical.NewFromString(sizeStr)
				if err != nil {
					p.logger.Warn("Invalid bid quantity", "coin", coin, "quantity", sizeStr, "error", err)
					continue
				}

				bids = append(bids, PriceLevel{
					Price:    price,
					Quantity: quantity,
				})
			}
		}
	}

	// Parse asks
	if askData, ok := levels[1].([]interface{}); ok {
		for _, ask := range askData {
			if askLevel, ok := ask.(map[string]interface{}); ok {
				priceStr, okPx := askLevel["px"].(string)
				sizeStr, okSz := askLevel["sz"].(string)
				if !okPx || !okSz {
					p.logger.Warn("Invalid ask level data", "coin", coin)
					continue
				}

				price, err := numerical.NewFromString(priceStr)
				if err != nil {
					p.logger.Warn("Invalid ask price", "coin", coin, "price", priceStr, "error", err)
					continue
				}

				quantity, err := numerical.NewFromString(sizeStr)
				if err != nil {
					p.logger.Warn("Invalid ask quantity", "coin", coin, "quantity", sizeStr, "error", err)
					continue
				}

				asks = append(asks, PriceLevel{
					Price:    price,
					Quantity: quantity,
				})
			}
		}
	}

	return &OrderBookMessage{
		Coin:      coin,
		Timestamp: p.timeProvider.Now(),
		Bids:      bids,
		Asks:      asks,
	}, nil
}

// ParseTrades parses a raw WebSocket message into TradeMessages
func (p *Parser) ParseTrades(msg hyperliquidsdk.WSMessage) ([]TradeMessage, error) {
	if msg.Channel != "trades" {
		return nil, fmt.Errorf("expected trades channel, got %s", msg.Channel)
	}

	var trades []interface{}
	if err := json.Unmarshal(msg.Data, &trades); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trades data: %w", err)
	}

	result := []TradeMessage{}

	for _, tradeData := range trades {
		trade, ok := tradeData.(map[string]interface{})
		if !ok {
			continue
		}

		coin, okCoin := trade["coin"].(string)
		priceStr, okPx := trade["px"].(string)
		sizeStr, okSz := trade["sz"].(string)
		sideStr, okSide := trade["side"].(string)
		timestamp, okTime := trade["time"].(float64)

		if !okCoin || !okPx || !okSz || !okSide || !okTime {
			p.logger.Warn("Invalid trade fields",
				"hasCoin", okCoin,
				"hasPx", okPx,
				"hasSz", okSz,
				"hasSide", okSide,
				"hasTime", okTime)
			continue
		}

		price, err := numerical.NewFromString(priceStr)
		if err != nil {
			p.logger.Warn("Invalid trade price", "coin", coin, "price", priceStr, "error", err)
			continue
		}

		quantity, err := numerical.NewFromString(sizeStr)
		if err != nil {
			p.logger.Warn("Invalid trade quantity", "coin", coin, "quantity", sizeStr, "error", err)
			continue
		}

		hash, _ := trade["hash"].(string)
		tid, _ := trade["tid"].(float64)

		result = append(result, TradeMessage{
			Coin:      coin,
			Price:     price,
			Quantity:  quantity,
			Side:      sideStr,
			Timestamp: time.Unix(int64(timestamp)/1000, 0),
			Hash:      hash,
			TradeID:   int64(tid),
		})
	}

	return result, nil
}

// ParsePosition parses a raw WebSocket message into a PositionMessage
func (p *Parser) ParsePosition(msg hyperliquidsdk.WSMessage) (*PositionMessage, error) {
	if msg.Channel != "webData2" {
		return nil, fmt.Errorf("expected webData2 channel, got %s", msg.Channel)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal position data: %w", err)
	}

	clearinghouseState, ok := data["clearinghouseState"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing clearinghouseState")
	}

	assetPositions, ok := clearinghouseState["assetPositions"].([]interface{})
	if !ok || len(assetPositions) == 0 {
		return nil, fmt.Errorf("no asset positions")
	}

	// Parse first position (can be extended to handle multiple)
	firstPos, ok := assetPositions[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid position data")
	}

	position, ok := firstPos["position"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing position field")
	}

	coin, _ := position["coin"].(string)
	sziStr, _ := position["szi"].(string)
	entryPxStr, _ := position["entryPx"].(string)
	marginUsedStr, _ := position["marginUsed"].(string)
	positionValueStr, _ := position["positionValue"].(string)
	unrealizedPnlStr, _ := position["unrealizedPnl"].(string)
	returnOnEquityStr, _ := position["returnOnEquity"].(string)

	size, _ := numerical.NewFromString(sziStr)
	entryPrice, _ := numerical.NewFromString(entryPxStr)
	marginUsed, _ := numerical.NewFromString(marginUsedStr)
	positionValue, _ := numerical.NewFromString(positionValueStr)
	unrealizedPnl, _ := numerical.NewFromString(unrealizedPnlStr)
	returnOnEquity, _ := numerical.NewFromString(returnOnEquityStr)

	return &PositionMessage{
		Coin:           coin,
		Size:           size,
		EntryPrice:     entryPrice,
		MarginUsed:     marginUsed,
		PositionValue:  positionValue,
		UnrealizedPnl:  unrealizedPnl,
		ReturnOnEquity: returnOnEquity,
		Timestamp:      p.timeProvider.Now(),
	}, nil
}

// ParseAccountBalance parses a raw WebSocket message into an AccountBalanceMessage
func (p *Parser) ParseAccountBalance(msg hyperliquidsdk.WSMessage) (*AccountBalanceMessage, error) {
	if msg.Channel != "webData2" {
		return nil, fmt.Errorf("expected webData2 channel, got %s", msg.Channel)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal balance data: %w", err)
	}

	marginSummary, ok := data["marginSummary"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing marginSummary")
	}

	accountValueStr, _ := marginSummary["accountValue"].(string)
	totalMarginUsedStr, _ := marginSummary["totalMarginUsed"].(string)
	totalNtlPosStr, _ := marginSummary["totalNtlPos"].(string)
	totalRawUsdStr, _ := marginSummary["totalRawUsd"].(string)

	withdrawableStr, _ := data["withdrawable"].(string)

	accountValue, _ := numerical.NewFromString(accountValueStr)
	totalMarginUsed, _ := numerical.NewFromString(totalMarginUsedStr)
	totalNtlPos, _ := numerical.NewFromString(totalNtlPosStr)
	totalRawUsd, _ := numerical.NewFromString(totalRawUsdStr)
	withdrawable, _ := numerical.NewFromString(withdrawableStr)

	return &AccountBalanceMessage{
		TotalAccountValue: accountValue,
		TotalMarginUsed:   totalMarginUsed,
		Withdrawable:      withdrawable,
		TotalNtlPos:       totalNtlPos,
		TotalRawUsd:       totalRawUsd,
		Timestamp:         p.timeProvider.Now(),
	}, nil
}

// ParseKline parses a raw WebSocket message into a KlineMessage
func (p *Parser) ParseKline(msg hyperliquidsdk.WSMessage) (*KlineMessage, error) {
	if msg.Channel != "candle" {
		return nil, fmt.Errorf("expected candle channel, got %s", msg.Channel)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kline data: %w", err)
	}

	coin, _ := data["s"].(string)
	interval, _ := data["i"].(string)
	openStr, _ := data["o"].(string)
	highStr, _ := data["h"].(string)
	lowStr, _ := data["l"].(string)
	closeStr, _ := data["c"].(string)
	volumeStr, _ := data["v"].(string)
	openTime, _ := data["t"].(float64)
	closeTime, _ := data["T"].(float64)

	open, _ := strconv.ParseFloat(openStr, 64)
	high, _ := strconv.ParseFloat(highStr, 64)
	low, _ := strconv.ParseFloat(lowStr, 64)
	closeVal, _ := strconv.ParseFloat(closeStr, 64)
	volume, _ := strconv.ParseFloat(volumeStr, 64)

	return &KlineMessage{
		Coin:      coin,
		Interval:  interval,
		OpenTime:  time.Unix(int64(openTime)/1000, 0),
		CloseTime: time.Unix(int64(closeTime)/1000, 0),
		Open:      open,
		High:      high,
		Low:       low,
		Close:     closeVal,
		Volume:    volume,
		Timestamp: p.timeProvider.Now(),
	}, nil
}

// ParseFundingRate parses a raw WebSocket message into a FundingRateMessage
// Hyperliquid activeAssetCtx message format:
// {"channel":"activeAssetCtx","data":{"coin":"ETH","ctx":{"funding":"0.00001234","markPx":"3300.5",...}}}
func (p *Parser) ParseFundingRate(msg hyperliquidsdk.WSMessage) (*FundingRateMessage, error) {
	if msg.Channel != "activeAssetCtx" {
		return nil, fmt.Errorf("expected activeAssetCtx channel, got %s", msg.Channel)
	}

	var data struct {
		Coin string `json:"coin"`
		Ctx  struct {
			Funding      string   `json:"funding"`
			OpenInterest string   `json:"openInterest"`
			PrevDayPx    string   `json:"prevDayPx"`
			DayNtlVlm    string   `json:"dayNtlVlm"`
			Premium      string   `json:"premium"`
			OraclePx     string   `json:"oraclePx"`
			MarkPx       string   `json:"markPx"`
			MidPx        string   `json:"midPx"`
			ImpactPxs    []string `json:"impactPxs"`
		} `json:"ctx"`
	}

	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal activeAssetCtx data: %w", err)
	}

	funding, err := numerical.NewFromString(data.Ctx.Funding)
	if err != nil {
		return nil, fmt.Errorf("invalid funding rate: %w", err)
	}

	markPrice, _ := numerical.NewFromString(data.Ctx.MarkPx)
	openInterest, _ := numerical.NewFromString(data.Ctx.OpenInterest)
	prevDayPx, _ := numerical.NewFromString(data.Ctx.PrevDayPx)
	dayNtlVlm, _ := numerical.NewFromString(data.Ctx.DayNtlVlm)
	premium, _ := numerical.NewFromString(data.Ctx.Premium)
	oraclePrice, _ := numerical.NewFromString(data.Ctx.OraclePx)

	return &FundingRateMessage{
		Coin:          data.Coin,
		FundingRate:   funding,
		MarkPrice:     markPrice,
		OpenInterest:  openInterest,
		PreviousDayPx: prevDayPx,
		DayNtlVlm:     dayNtlVlm,
		Premium:       premium,
		OraclePrice:   oraclePrice,
		Timestamp:     p.timeProvider.Now(),
	}, nil
}
