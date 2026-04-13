package hyperliquid

import (
	"fmt"

	perpws "github.com/wisp-trading/connectors/pkg/connectors/perps/hyperliquid/websocket"
	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// ─── WebSocket Lifecycle ────────────────────────────────────────────────────

// StartWebSocket implements connector.WebSocketCapable
func (h *hyperliquidSpot) StartWebSocket() error {
	if h.config == nil {
		return fmt.Errorf("connector not initialized")
	}

	// Forward errors from the realtime service to the connector's error channel
	go h.forwardWebSocketErrors()

	return h.realTime.Connect(h.config.WebsocketURL)
}

// forwardWebSocketErrors pipes errors from the WS service to the connector channel
func (h *hyperliquidSpot) forwardWebSocketErrors() {
	errCh := h.realTime.GetErrorChannel()
	for err := range errCh {
		select {
		case h.errorCh <- err:
		default:
		}
	}
}

// StopWebSocket implements connector.WebSocketCapable
func (h *hyperliquidSpot) StopWebSocket() error {
	return h.realTime.Disconnect()
}

// IsWebSocketConnected implements connector.WebSocketCapable
func (h *hyperliquidSpot) IsWebSocketConnected() bool {
	return h.realTime.IsConnected()
}

// ErrorChannel implements connector.WebSocketCapable
func (h *hyperliquidSpot) ErrorChannel() <-chan error {
	return h.errorCh
}

// ─── Subscriptions ──────────────────────────────────────────────────────────

// SubscribeOrderBook implements spot.WebSocketConnector
func (h *hyperliquidSpot) SubscribeOrderBook(pair portfolio.Pair) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(pair.Base())

	// Create dedicated channel for this pair if it doesn't exist
	h.orderBookMu.Lock()
	orderBookCh, exists := h.orderBookChannels[symbol]
	if !exists {
		orderBookCh = make(chan connector.OrderBook, 100)
		h.orderBookChannels[symbol] = orderBookCh
	}
	h.orderBookMu.Unlock()

	subID, err := h.realTime.SubscribeToOrderBook(symbol, func(obMsg *perpws.OrderBookMessage) {
		bids := make([]connector.PriceLevel, len(obMsg.Bids))
		for i, bid := range obMsg.Bids {
			bids[i] = connector.PriceLevel{Price: bid.Price, Quantity: bid.Quantity}
		}

		asks := make([]connector.PriceLevel, len(obMsg.Asks))
		for i, ask := range obMsg.Asks {
			asks[i] = connector.PriceLevel{Price: ask.Price, Quantity: ask.Quantity}
		}

		ob := connector.OrderBook{
			Pair:      pair,
			Timestamp: obMsg.Timestamp,
			Bids:      bids,
			Asks:      asks,
		}

		select {
		case orderBookCh <- ob:
		default:
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

// UnsubscribeOrderBook implements spot.WebSocketConnector
func (h *hyperliquidSpot) UnsubscribeOrderBook(pair portfolio.Pair) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(pair.Base())

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

// SubscribeTrades implements spot.WebSocketConnector
func (h *hyperliquidSpot) SubscribeTrades(pair portfolio.Pair) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(pair.Base())

	subID, err := h.realTime.SubscribeToTrades(symbol, func(trades []perpws.TradeMessage) {
		for _, trade := range trades {
			select {
			case h.tradeCh <- connector.Trade{
				Pair:      h.coinToPair(symbol),
				Exchange:  types.HyperliquidSpot,
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

// UnsubscribeTrades implements spot.WebSocketConnector
func (h *hyperliquidSpot) UnsubscribeTrades(pair portfolio.Pair) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(pair.Base())

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

// SubscribeKlines implements spot.WebSocketConnector
func (h *hyperliquidSpot) SubscribeKlines(pair portfolio.Pair, interval string) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(pair.Base())
	channelKey := fmt.Sprintf("%s:%s", symbol, interval)

	h.klineMu.Lock()
	klineCh := make(chan connector.Kline, 100)
	h.klineChannels[channelKey] = klineCh
	h.klineMu.Unlock()

	subID, err := h.realTime.SubscribeToKlines(symbol, interval, func(klineMsg *perpws.KlineMessage) {
		if klineMsg.Interval != interval {
			return
		}

		kline := connector.Kline{
			Pair:      pair,
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

// UnsubscribeKlines implements spot.WebSocketConnector
func (h *hyperliquidSpot) UnsubscribeKlines(pair portfolio.Pair, interval string) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(pair.Base())
	key := "klines:" + symbol + ":" + interval

	h.subMu.Lock()
	subID, exists := h.subscriptions[key]
	if !exists {
		h.subMu.Unlock()
		return fmt.Errorf("no active subscription for %s", key)
	}
	delete(h.subscriptions, key)
	h.subMu.Unlock()

	return h.realTime.UnsubscribeFromKlines(symbol, interval, subID)
}

// SubscribeAccountBalance implements spot.WebSocketConnector
func (h *hyperliquidSpot) SubscribeAccountBalance() error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	subID, err := h.realTime.SubscribeToAccountBalance(h.config.AccountAddress, func(balMsg *perpws.AccountBalanceMessage) {
		select {
		case h.balanceCh <- connector.AssetBalance{
			Asset:     portfolio.NewAsset("USDC"),
			Free:      numerical.Zero(),
			Locked:    numerical.Zero(),
			Total:     balMsg.TotalAccountValue,
			UpdatedAt: h.timeProvider.Now(),
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

// UnsubscribeAccountBalance implements spot.WebSocketConnector
func (h *hyperliquidSpot) UnsubscribeAccountBalance() error {
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

	return nil
}

// ─── Channel Accessors ──────────────────────────────────────────────────────

// GetOrderBookChannels implements spot.WebSocketConnector
func (h *hyperliquidSpot) GetOrderBookChannels() map[string]<-chan connector.OrderBook {
	h.orderBookMu.RLock()
	defer h.orderBookMu.RUnlock()

	result := make(map[string]<-chan connector.OrderBook, len(h.orderBookChannels))
	for k, v := range h.orderBookChannels {
		result[k] = v
	}
	return result
}

// GetKlineChannels implements spot.WebSocketConnector
func (h *hyperliquidSpot) GetKlineChannels() map[string]<-chan connector.Kline {
	h.klineMu.RLock()
	defer h.klineMu.RUnlock()

	result := make(map[string]<-chan connector.Kline, len(h.klineChannels))
	for k, v := range h.klineChannels {
		result[k] = v
	}
	return result
}

// TradeUpdates implements spot.WebSocketConnector
func (h *hyperliquidSpot) TradeUpdates() <-chan connector.Trade {
	return h.tradeCh
}

// AssetBalanceUpdates implements spot.WebSocketConnector
func (h *hyperliquidSpot) AssetBalanceUpdates() <-chan connector.AssetBalance {
	return h.balanceCh
}
