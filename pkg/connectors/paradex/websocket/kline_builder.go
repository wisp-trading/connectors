package websockets

import (
	"fmt"
	"sync"
	"time"

	"github.com/wisp-trading/sdk/pkg/types/temporal"
)

type KlineBuilder struct {
	activeKlines map[string]*ActiveKline
	output       chan KlineUpdate
	mu           sync.RWMutex
	timeProvider temporal.TimeProvider
}

type ActiveKline struct {
	Symbol     string
	Interval   string
	Open       float64
	High       float64
	Low        float64
	Close      float64
	Volume     float64
	OpenTime   time.Time
	CloseTime  time.Time
	TradeCount int64
	complete   bool
}

type KlineUpdate struct {
	Symbol     string
	Interval   string
	Open       float64
	High       float64
	Low        float64
	Close      float64
	Volume     float64
	OpenTime   time.Time
	CloseTime  time.Time
	TradeCount int64
}

func NewKlineBuilder(timeProvider temporal.TimeProvider) *KlineBuilder {
	return &KlineBuilder{
		activeKlines: make(map[string]*ActiveKline),
		output:       make(chan KlineUpdate, 100),
		timeProvider: timeProvider,
	}
}

func (kb *KlineBuilder) ProcessTrade(trade TradeUpdate) {
	intervals := []string{"1m", "5m", "15m", "1h"}

	for _, interval := range intervals {
		kb.updateKline(trade, interval)
	}
}

func (kb *KlineBuilder) updateKline(trade TradeUpdate, interval string) {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	key := fmt.Sprintf("%s_%s", trade.Symbol, interval)
	kline := kb.getOrCreateKline(key, trade, interval)

	// Update OHLCV data
	if kline.Open == 0 {
		kline.Open = trade.Price
	}
	kline.Close = trade.Price

	if kline.High == 0 || trade.Price > kline.High {
		kline.High = trade.Price
	}
	if kline.Low == 0 || trade.Price < kline.Low {
		kline.Low = trade.Price
	}

	kline.Volume = kline.Volume + trade.Quantity
	kline.TradeCount++

	// Check if kline period is complete
	if kb.timeProvider.Now().After(kline.CloseTime) && !kline.complete {
		kline.complete = true
		kb.emitKline(kline)
		delete(kb.activeKlines, key)
	}
}

func (kb *KlineBuilder) getOrCreateKline(key string, trade TradeUpdate, interval string) *ActiveKline {
	if kline, exists := kb.activeKlines[key]; exists {
		return kline
	}

	// Create new kline
	openTime := kb.getKlineOpenTime(trade.Timestamp, interval)
	closeTime := kb.getKlineCloseTime(openTime, interval)

	kline := &ActiveKline{
		Symbol:    trade.Symbol,
		Interval:  interval,
		OpenTime:  openTime,
		CloseTime: closeTime,
		Volume:    0,
	}

	kb.activeKlines[key] = kline
	return kline
}

func (kb *KlineBuilder) getKlineOpenTime(tradeTime time.Time, interval string) time.Time {
	switch interval {
	case "1m":
		return tradeTime.Truncate(time.Minute)
	case "5m":
		minutes := tradeTime.Minute() - (tradeTime.Minute() % 5)
		return time.Date(tradeTime.Year(), tradeTime.Month(), tradeTime.Day(),
			tradeTime.Hour(), minutes, 0, 0, tradeTime.Location())
	case "15m":
		minutes := tradeTime.Minute() - (tradeTime.Minute() % 15)
		return time.Date(tradeTime.Year(), tradeTime.Month(), tradeTime.Day(),
			tradeTime.Hour(), minutes, 0, 0, tradeTime.Location())
	case "1h":
		return tradeTime.Truncate(time.Hour)
	default:
		return tradeTime.Truncate(time.Minute)
	}
}

func (kb *KlineBuilder) getKlineCloseTime(openTime time.Time, interval string) time.Time {
	switch interval {
	case "1m":
		return openTime.Add(time.Minute)
	case "5m":
		return openTime.Add(5 * time.Minute)
	case "15m":
		return openTime.Add(15 * time.Minute)
	case "1h":
		return openTime.Add(time.Hour)
	default:
		return openTime.Add(time.Minute)
	}
}

func (kb *KlineBuilder) emitKline(kline *ActiveKline) {
	update := KlineUpdate{
		Symbol:     kline.Symbol,
		Interval:   kline.Interval,
		Open:       kline.Open,
		High:       kline.High,
		Low:        kline.Low,
		Close:      kline.Close,
		Volume:     kline.Volume,
		OpenTime:   kline.OpenTime,
		CloseTime:  kline.CloseTime,
		TradeCount: kline.TradeCount,
	}

	select {
	case kb.output <- update:
	default:
		// Channel full, drop update
	}
}

func (kb *KlineBuilder) Output() <-chan KlineUpdate {
	return kb.output
}

func (kb *KlineBuilder) Close() {
	kb.mu.Lock()
	defer kb.mu.Unlock()
	close(kb.output)
}
