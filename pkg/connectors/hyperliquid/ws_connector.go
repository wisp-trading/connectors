package hyperliquid

import (
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/websocket"
	"github.com/backtesting-org/live-trading/pkg/connectors/types"
)

// StartWebSocket starts the WebSocket connection for real-time data
func (h *hyperliquid) StartWebSocket() error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	// Start error forwarding from realTime service
	go h.forwardWebSocketErrors()

	// Pass the WebSocket URL from config
	var wsURL *string
	if h.config.WebsocketURL != "" {
		wsURL = &h.config.WebsocketURL
	}

	return h.realTime.Connect(wsURL)
}

// forwardWebSocketErrors forwards errors from the realTime service to the connector's error channel
func (h *hyperliquid) forwardWebSocketErrors() {
	errCh := h.realTime.GetErrorChannel()
	for err := range errCh {
		select {
		case h.errorCh <- err:
		default:
			// Error channel is full, drop the error
		}
	}
}

// StopWebSocket stops the WebSocket connection
func (h *hyperliquid) StopWebSocket() error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	return h.realTime.Disconnect()
}

// IsWebSocketConnected returns whether the WebSocket is connected
func (h *hyperliquid) IsWebSocketConnected() bool {
	if !h.initialized || h.realTime == nil {
		return false
	}
	return h.realTime.IsConnected()
}

// GetOrderBookChannels returns all active orderbook channels
func (h *hyperliquid) GetOrderBookChannels() map[string]<-chan connector.OrderBook {
	h.orderBookMu.RLock()
	defer h.orderBookMu.RUnlock()

	result := make(map[string]<-chan connector.OrderBook, len(h.orderBookChannels))
	for key, ch := range h.orderBookChannels {
		result[key] = ch
	}

	return result
}

// TradeUpdates returns a channel for trade updates
func (h *hyperliquid) TradeUpdates() <-chan connector.Trade {
	return h.tradeCh
}

// PositionUpdates returns a channel for position updates
func (h *hyperliquid) PositionUpdates() <-chan connector.Position {
	return h.positionCh
}

// AccountBalanceUpdates returns a channel for account balance updates
func (h *hyperliquid) AccountBalanceUpdates() <-chan connector.AccountBalance {
	return h.balanceCh
}

// GetKlineChannels returns all active kline channels
func (h *hyperliquid) GetKlineChannels() map[string]<-chan connector.Kline {
	h.klineMu.RLock()
	defer h.klineMu.RUnlock()

	result := make(map[string]<-chan connector.Kline, len(h.klineChannels))
	for key, ch := range h.klineChannels {
		result[key] = ch
	}

	return result
}

// ErrorChannel returns a channel for WebSocket errors
func (h *hyperliquid) ErrorChannel() <-chan error {
	return h.errorCh
}

// SubscribeOrderBook subscribes to order book updates for an asset
func (h *hyperliquid) SubscribeOrderBook(asset portfolio.Asset, _ connector.Instrument) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(asset)

	// Create dedicated channel for this asset if it doesn't exist
	h.orderBookMu.Lock()
	orderBookCh, exists := h.orderBookChannels[symbol]
	if !exists {
		orderBookCh = make(chan connector.OrderBook, 100)
		h.orderBookChannels[symbol] = orderBookCh
	}
	h.orderBookMu.Unlock()

	subID, err := h.realTime.SubscribeToOrderBook(symbol, func(obMsg *websocket.OrderBookMessage) {
		bids := make([]connector.PriceLevel, len(obMsg.Bids))
		for i, bid := range obMsg.Bids {
			bids[i] = connector.PriceLevel{
				Price:    bid.Price,
				Quantity: bid.Quantity,
			}
		}

		asks := make([]connector.PriceLevel, len(obMsg.Asks))
		for i, ask := range obMsg.Asks {
			asks[i] = connector.PriceLevel{
				Price:    ask.Price,
				Quantity: ask.Quantity,
			}
		}

		orderBook := connector.OrderBook{
			Asset:     asset,
			Timestamp: obMsg.Timestamp,
			Bids:      bids,
			Asks:      asks,
		}

		select {
		case orderBookCh <- orderBook:
		default:
			// Send error to error channel if channel is full
			select {
			case h.errorCh <- fmt.Errorf("orderbook channel full for %s, dropping update", symbol):
			default:
			}
		}
	})
	if err != nil {
		return err
	}

	h.subMu.Lock()
	h.subscriptions["orderbook:"+symbol] = subID
	h.subMu.Unlock()

	return nil
}

// UnsubscribeOrderBook unsubscribes from order book updates
func (h *hyperliquid) UnsubscribeOrderBook(asset portfolio.Asset, _ connector.Instrument) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(asset)

	h.subMu.Lock()
	subID, exists := h.subscriptions["orderbook:"+symbol]
	if !exists {
		h.subMu.Unlock()
		return fmt.Errorf("no active subscription for orderbook:%s", symbol)
	}
	delete(h.subscriptions, "orderbook:"+symbol)
	h.subMu.Unlock()

	return h.realTime.UnsubscribeFromOrderBook(symbol, subID)
}

// SubscribeTrades subscribes to trade updates for an asset
func (h *hyperliquid) SubscribeTrades(asset portfolio.Asset, _ connector.Instrument) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(asset)

	subID, err := h.realTime.SubscribeToTrades(symbol, func(trades []websocket.TradeMessage) {
		for _, trade := range trades {
			select {
			case h.tradeCh <- connector.Trade{
				Symbol:    trade.Coin,
				Exchange:  types.Hyperliquid,
				Price:     trade.Price,
				Quantity:  trade.Quantity,
				Side:      connector.FromString(trade.Side),
				Timestamp: trade.Timestamp,
			}:
			default:
				select {
				case h.errorCh <- fmt.Errorf("trade channel full for %s, dropping update", symbol):
				default:
				}
			}
		}
	})
	if err != nil {
		return err
	}

	h.subMu.Lock()
	h.subscriptions["trades:"+symbol] = subID
	h.subMu.Unlock()
	return nil
}

// UnsubscribeTrades unsubscribes from trade updates
func (h *hyperliquid) UnsubscribeTrades(asset portfolio.Asset, _ connector.Instrument) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(asset)

	h.subMu.Lock()
	subID, exists := h.subscriptions["trades:"+symbol]
	if !exists {
		h.subMu.Unlock()
		return fmt.Errorf("no active subscription for trades:%s", symbol)
	}
	delete(h.subscriptions, "trades:"+symbol)
	h.subMu.Unlock()

	return h.realTime.UnsubscribeFromTrades(symbol, subID)
}

// SubscribePositions subscribes to position updates
func (h *hyperliquid) SubscribePositions(asset portfolio.Asset, _ connector.Instrument) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(asset)

	subID, err := h.realTime.SubscribeToPositions(h.config.AccountAddress, func(posMsg *websocket.PositionMessage) {
		if posMsg.Coin != symbol {
			return
		}

		side := connector.OrderSideBuy
		if posMsg.Size.IsNegative() {
			side = connector.OrderSideSell
		}

		select {
		case h.positionCh <- connector.Position{
			Symbol:        asset,
			Exchange:      types.Hyperliquid,
			Side:          side,
			Size:          posMsg.Size.Abs(),
			EntryPrice:    posMsg.EntryPrice,
			MarkPrice:     posMsg.MarkPrice,
			UnrealizedPnL: posMsg.UnrealizedPnl,
			RealizedPnL:   parseDecimal("0"),
			UpdatedAt:     posMsg.Timestamp,
		}:
		default:
			select {
			case h.errorCh <- fmt.Errorf("position channel full for %s, dropping update", symbol):
			default:
			}
		}
	})
	if err != nil {
		return err
	}

	h.subMu.Lock()
	h.subscriptions["positions:"+symbol] = subID
	h.subMu.Unlock()
	return nil
}

// UnsubscribePositions unsubscribes from position updates
func (h *hyperliquid) UnsubscribePositions(asset portfolio.Asset, _ connector.Instrument) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(asset)

	h.subMu.Lock()
	_, exists := h.subscriptions["positions:"+symbol]
	if !exists {
		h.subMu.Unlock()
		return fmt.Errorf("no active subscription for positions:%s", symbol)
	}
	delete(h.subscriptions, "positions:"+symbol)
	h.subMu.Unlock()

	// No unsubscribe method for positions yet
	return nil
}

// SubscribeAccountBalance subscribes to account balance updates
func (h *hyperliquid) SubscribeAccountBalance() error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	subID, err := h.realTime.SubscribeToAccountBalance(h.config.AccountAddress, func(balMsg *websocket.AccountBalanceMessage) {
		select {
		case h.balanceCh <- connector.AccountBalance{
			TotalBalance:     balMsg.TotalAccountValue,
			AvailableBalance: balMsg.Withdrawable,
			UsedMargin:       balMsg.TotalMarginUsed,
			Currency:         "USD",
			UpdatedAt:        balMsg.Timestamp,
		}:
		default:
			select {
			case h.errorCh <- fmt.Errorf("balance channel full, dropping update"):
			default:
			}
		}
	})
	if err != nil {
		return err
	}

	h.subMu.Lock()
	h.subscriptions["balance"] = subID
	h.subMu.Unlock()
	return nil
}

// UnsubscribeAccountBalance unsubscribes from account balance updates
func (h *hyperliquid) UnsubscribeAccountBalance() error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	h.subMu.Lock()
	_, exists := h.subscriptions["balance"]
	if !exists {
		h.subMu.Unlock()
		return fmt.Errorf("no active subscription for balance")
	}
	delete(h.subscriptions, "balance")
	h.subMu.Unlock()

	// No unsubscribe method for account balance yet
	return nil
}

// SubscribeKlines subscribes to kline updates for an asset
func (h *hyperliquid) SubscribeKlines(asset portfolio.Asset, interval string) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(asset)
	channelKey := fmt.Sprintf("%s:%s", symbol, interval)

	// Create dedicated channel for this subscription
	h.klineMu.Lock()
	klineCh := make(chan connector.Kline, 100)
	h.klineChannels[channelKey] = klineCh
	h.klineMu.Unlock()

	subID, err := h.realTime.SubscribeToKlines(symbol, interval, func(klineMsg *websocket.KlineMessage) {
		// Only process klines matching the subscribed interval
		// Hyperliquid sends ALL intervals even if you only subscribe to one
		if klineMsg.Interval != interval {
			return
		}

		kline := connector.Kline{
			Symbol:    symbol,
			Interval:  klineMsg.Interval,
			OpenTime:  klineMsg.OpenTime,
			Open:      klineMsg.Open,
			High:      klineMsg.High,
			Low:       klineMsg.Low,
			Close:     klineMsg.Close,
			Volume:    klineMsg.Volume,
			CloseTime: klineMsg.CloseTime,
		}

		select {
		case klineCh <- kline:
		default:
			select {
			case h.errorCh <- fmt.Errorf("kline channel full for %s, dropping update", channelKey):
			default:
			}
		}
	})
	if err != nil {
		return err
	}

	h.subMu.Lock()
	h.subscriptions["klines:"+symbol+":"+interval] = subID
	h.subMu.Unlock()
	return nil
}

// UnsubscribeKlines unsubscribes from kline updates
func (h *hyperliquid) UnsubscribeKlines(asset portfolio.Asset, interval string) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(asset)

	h.subMu.Lock()
	key := "klines:" + symbol + ":" + interval
	subID, exists := h.subscriptions[key]
	if !exists {
		h.subMu.Unlock()
		return fmt.Errorf("no active subscription for %s", key)
	}
	delete(h.subscriptions, key)
	h.subMu.Unlock()

	return h.realTime.UnsubscribeFromKlines(symbol, interval, subID)
}
