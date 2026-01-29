package bybit

import (
	"fmt"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
)

// StartWebSocket starts the WebSocket connection for real-time data
func (b *bybit) StartWebSocket() error {
	if !b.initialized {
		return fmt.Errorf("connector not initialized")
	}
	return b.realTime.Connect()
}

// StopWebSocket stops the WebSocket connection
func (b *bybit) StopWebSocket() error {
	return b.realTime.Disconnect()
}

func (b *bybit) Connect() error {
	if !b.initialized {
		return fmt.Errorf("connector not initialized")
	}
	return b.realTime.Connect()
}

func (b *bybit) Disconnect() error {
	return b.realTime.Disconnect()
}

func (b *bybit) SubscribeOrderBook(asset portfolio.Asset, instrument connector.Instrument) error {
	return b.realTime.SubscribeOrderBook(asset, instrument)
}

func (b *bybit) UnsubscribeOrderBook(asset portfolio.Asset, instrument connector.Instrument) error {
	return b.realTime.UnsubscribeOrderBook(asset, instrument)
}

func (b *bybit) SubscribeTrades(asset portfolio.Asset, instrument connector.Instrument) error {
	return b.realTime.SubscribeTrades(asset, instrument)
}

func (b *bybit) UnsubscribeTrades(asset portfolio.Asset, instrument connector.Instrument) error {
	return b.realTime.UnsubscribeTrades(asset, instrument)
}

func (b *bybit) SubscribePositions(asset portfolio.Asset, instrument connector.Instrument) error {
	return b.realTime.SubscribePositions(asset, instrument)
}

func (b *bybit) SubscribeAccountBalance() error {
	return b.realTime.SubscribeAccountBalance()
}

func (b *bybit) UnsubscribeAccountBalance() error {
	return b.realTime.UnsubscribeAccountBalance()
}

func (b *bybit) UnsubscribePositions(asset portfolio.Asset, instrument connector.Instrument) error {
	return b.realTime.UnsubscribePositions(asset, instrument)
}

func (b *bybit) SubscribeKlines(asset portfolio.Asset, interval string) error {
	return b.realTime.SubscribeKlines(asset, interval)
}

func (b *bybit) UnsubscribeKlines(asset portfolio.Asset, interval string) error {
	return b.realTime.UnsubscribeKlines(asset, interval)
}

func (b *bybit) GetErrorChannel() <-chan error {
	return b.errorCh
}
