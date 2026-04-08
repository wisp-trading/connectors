package deribit

import (
	"context"
	"fmt"
	"sync"
	"time"

	deribitWS "github.com/wisp-trading/connectors/pkg/connectors/options/deribit/websocket"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	optionsConnector "github.com/wisp-trading/sdk/pkg/types/connector/options"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// Ensure deribitOptions implements the full WebSocketConnector at compile time.
var _ optionsConnector.WebSocketConnector = (*deribitOptions)(nil)

// ============================================================================
// connector.WebSocketCapable
// ============================================================================

// StartWebSocket opens the WebSocket connection to Deribit, authenticates,
// and enables heartbeats. Must be called after Initialize.
func (d *deribitOptions) StartWebSocket() error {
	d.mu.RLock()
	initialized := d.initialized
	config := d.config
	d.mu.RUnlock()

	if !initialized {
		return fmt.Errorf("connector not initialized — call Initialize first")
	}

	wsURL := config.WebSocketURL
	if wsURL == "" {
		wsURL = defaultWSURL
	}

	return d.wsService.Connect(context.Background(), wsURL, config.ClientID, config.ClientSecret)
}

// StopWebSocket closes the WebSocket connection and all active subscriptions.
func (d *deribitOptions) StopWebSocket() error {
	return d.wsService.Disconnect()
}

// IsWebSocketConnected returns true if the WebSocket is currently connected.
func (d *deribitOptions) IsWebSocketConnected() bool {
	return d.wsService.IsConnected()
}

// ErrorChannel returns the channel that receives WebSocket errors.
func (d *deribitOptions) ErrorChannel() <-chan error {
	return d.wsService.ErrorChannel()
}

// ============================================================================
// optionsConnector.WebSocketConnector
// ============================================================================

// SubscribeExpirationUpdates subscribes to real-time ticker updates for the given contracts.
// contracts is resolved by the realtime ingestor from the store (populated by the batch
// ingestor) — no REST calls are made here.
func (d *deribitOptions) SubscribeExpirationUpdates(pair portfolio.Pair, expiration time.Time, contracts []optionsConnector.OptionContract) error {
	if !d.IsWebSocketConnected() {
		return fmt.Errorf("websocket not connected — call StartWebSocket first")
	}
	if len(contracts) == 0 {
		return fmt.Errorf("no contracts provided for %s %s", pair.Symbol(), expiration.Format("2006-01-02"))
	}

	for _, contract := range contracts {
		instrument := formatInstrumentName(contract)

		d.optionChansMu.Lock()
		if _, exists := d.optionChans[instrument]; !exists {
			d.optionChans[instrument] = make(chan optionsConnector.OptionUpdate, 256)
		}
		ch := d.optionChans[instrument]
		d.optionChansMu.Unlock()

		capturedCh := ch
		capturedContract := contract

		if err := d.wsService.SubscribeToTicker(instrument, func(data *deribitWS.TickerData) {
			update := tickerToOptionUpdate(capturedContract, data)
			safeSendOptionUpdate(capturedCh, update)
		}); err != nil {
			return fmt.Errorf("failed to subscribe to ticker for %s: %w", instrument, err)
		}
	}

	d.appLogger.Infof("Subscribed to expiration updates: %s %s (%d contracts)",
		pair.Symbol(), expiration.Format("2006-01-02"), len(contracts))
	return nil
}

// UnsubscribeExpirationUpdates removes subscriptions for the given contracts and closes
// their update channels. contracts is resolved by the realtime ingestor from the store.
func (d *deribitOptions) UnsubscribeExpirationUpdates(pair portfolio.Pair, expiration time.Time, contracts []optionsConnector.OptionContract) error {
	for _, contract := range contracts {
		instrument := formatInstrumentName(contract)

		if err := d.wsService.UnsubscribeFromTicker(instrument); err != nil {
			d.appLogger.Warnf("Failed to unsubscribe ticker for %s: %v", instrument, err)
		}

		d.optionChansMu.Lock()
		if ch, ok := d.optionChans[instrument]; ok {
			close(ch)
			delete(d.optionChans, instrument)
		}
		d.optionChansMu.Unlock()
	}

	d.appLogger.Infof("Unsubscribed from expiration updates: %s %s",
		pair.Symbol(), expiration.Format("2006-01-02"))
	return nil
}

// SubscribeOrderBook opens a real-time order book subscription for the given contract.
// Called by the realtime ingestor when the user fast-watches a specific instrument.
func (d *deribitOptions) SubscribeOrderBook(contract *optionsConnector.OptionContract) error {
	if !d.IsWebSocketConnected() {
		return fmt.Errorf("websocket not connected — call StartWebSocket first")
	}

	instrument := formatInstrumentName(*contract)

	d.obChansMu.Lock()
	if _, exists := d.obChans[instrument]; !exists {
		d.obChans[instrument] = make(chan connector.OrderBook, 64)
	}
	ch := d.obChans[instrument]
	d.obChansMu.Unlock()

	capturedCh := ch

	return d.wsService.SubscribeToOrderBook(instrument, func(data *deribitWS.OrderBookData) {
		safeSendOrderBook(capturedCh, orderBookDataToOrderBook(*contract, data))
	})
}

// UnsubscribeOrderBook closes the order book subscription and channel for the given contract.
func (d *deribitOptions) UnsubscribeOrderBook(contract *optionsConnector.OptionContract) error {
	instrument := formatInstrumentName(*contract)

	if err := d.wsService.UnsubscribeFromOrderBook(instrument); err != nil {
		d.appLogger.Warnf("Failed to unsubscribe order book for %s: %v", instrument, err)
	}

	d.obChansMu.Lock()
	if ch, ok := d.obChans[instrument]; ok {
		close(ch)
		delete(d.obChans, instrument)
	}
	d.obChansMu.Unlock()

	return nil
}

// GetOptionUpdateChannels returns a read-only view of all active option update channels,
// keyed by Deribit instrument name (e.g. "BTC-8APR26-50000-C").
func (d *deribitOptions) GetOptionUpdateChannels() map[string]<-chan optionsConnector.OptionUpdate {
	d.optionChansMu.RLock()
	defer d.optionChansMu.RUnlock()

	out := make(map[string]<-chan optionsConnector.OptionUpdate, len(d.optionChans))
	for k, ch := range d.optionChans {
		out[k] = ch
	}
	return out
}

// GetTradeChannels returns real-time trade update channels keyed by instrument name.
func (d *deribitOptions) GetTradeChannels() map[string]<-chan connector.Trade {
	d.tradeChansMu.RLock()
	defer d.tradeChansMu.RUnlock()

	out := make(map[string]<-chan connector.Trade, len(d.tradeChans))
	for k, ch := range d.tradeChans {
		out[k] = ch
	}
	return out
}

// GetOrderBookChannels returns real-time order book channels keyed by instrument name.
func (d *deribitOptions) GetOrderBookChannels() map[string]<-chan connector.OrderBook {
	d.obChansMu.RLock()
	defer d.obChansMu.RUnlock()

	out := make(map[string]<-chan connector.OrderBook, len(d.obChans))
	for k, ch := range d.obChans {
		out[k] = ch
	}
	return out
}

// ============================================================================
// Internal helpers
// ============================================================================

// tickerToOptionUpdate converts a Deribit ticker payload to the SDK OptionUpdate type.
func tickerToOptionUpdate(contract optionsConnector.OptionContract, data *deribitWS.TickerData) optionsConnector.OptionUpdate {
	return optionsConnector.OptionUpdate{
		Contract:        contract,
		MarkPrice:       data.MarkPrice,
		UnderlyingPrice: data.UnderlyingPrice,
		IV:              data.MarkIV,
		Greeks: optionsConnector.Greeks{
			Delta: data.Greeks.Delta,
			Gamma: data.Greeks.Gamma,
			Theta: data.Greeks.Theta,
			Vega:  data.Greeks.Vega,
			Rho:   data.Greeks.Rho,
		},
		Timestamp: time.UnixMilli(data.Timestamp),
	}
}

// safeSendOptionUpdate sends to ch without panicking if the channel has been closed.
// The service removes the callback from its map before the connector closes the channel,
// but there is a narrow window where an in-flight callback fires after closure.
func safeSendOptionUpdate(ch chan optionsConnector.OptionUpdate, v optionsConnector.OptionUpdate) {
	defer func() { recover() }() //nolint:errcheck
	select {
	case ch <- v:
	default:
		select {
		case <-ch:
		default:
		}
		ch <- v
	}
}

// safeSendOrderBook sends to ch without panicking if the channel has been closed.
func safeSendOrderBook(ch chan connector.OrderBook, v connector.OrderBook) {
	defer func() { recover() }() //nolint:errcheck
	select {
	case ch <- v:
	default:
		select {
		case <-ch:
		default:
		}
		ch <- v
	}
}

// orderBookDataToOrderBook converts a Deribit order book payload to the SDK OrderBook type.
// Only "new" and "change" entries are included — "delete" entries are dropped since
// the SDK OrderBook is a snapshot, not a diff.
func orderBookDataToOrderBook(contract optionsConnector.OptionContract, data *deribitWS.OrderBookData) connector.OrderBook {
	bids := make([]connector.PriceLevel, 0, len(data.Bids))
	for _, entry := range data.Bids {
		if entry.Action == deribitWS.OrderBookActionDelete {
			continue
		}
		bids = append(bids, connector.PriceLevel{
			Price:    numerical.NewFromFloat(entry.Price),
			Quantity: numerical.NewFromFloat(entry.Amount),
		})
	}

	asks := make([]connector.PriceLevel, 0, len(data.Asks))
	for _, entry := range data.Asks {
		if entry.Action == deribitWS.OrderBookActionDelete {
			continue
		}
		asks = append(asks, connector.PriceLevel{
			Price:    numerical.NewFromFloat(entry.Price),
			Quantity: numerical.NewFromFloat(entry.Amount),
		})
	}

	return connector.OrderBook{
		Pair:      contract.Pair,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.UnixMilli(data.Timestamp),
	}
}

// wsChannelState holds the per-instrument channel maps for the WebSocket connector.
// Embedded into deribitOptions.
type wsChannelState struct {
	optionChansMu sync.RWMutex
	optionChans   map[string]chan optionsConnector.OptionUpdate

	tradeChansMu sync.RWMutex
	tradeChans   map[string]chan connector.Trade

	obChansMu sync.RWMutex
	obChans   map[string]chan connector.OrderBook
}
