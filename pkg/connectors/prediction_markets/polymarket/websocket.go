package polymarket

import (
	"fmt"

	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/websocket"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
)

func (p *polymarket) StartWebSocket() error {
	if !p.initialized {
		return fmt.Errorf("connector not initialized")
	}

	if p.websocketClient == nil {
		return fmt.Errorf("websocket service not initialized")
	}

	// Pass the WebSocket URL from config
	return p.websocketClient.Connect(p.config.WebSocketURL)
}

func (p *polymarket) StopWebSocket() error {
	//TODO implement me
	panic("implement me")
}

func (p *polymarket) IsWebSocketConnected() bool {
	if p.websocketClient == nil {
		return false
	}

	return p.websocketClient.IsConnected()
}

func (p *polymarket) ErrorChannel() <-chan error {
	//TODO implement me
	panic("implement me")
}

func (p *polymarket) SubscribeMarketBook(market prediction.Market) {
	if !p.IsWebSocketConnected() {
		p.appLogger.Warn("Polymarket WebSocket not connected, cannot subscribe to market book")
		return
	}

	p.websocketClient.SubscribeToMarketBook(market.MarketId, func(msg *websocket.OrderBookMessage) {
		orderBook := convertToOrderBook(msg)

		// Send to the appropriate channel based on market ID
		p.orderBookMu.RLock()
		ch, exists := p.orderBookChannels[market.MarketId]
		p.orderBookMu.RUnlock()

		if exists {
			ch <- orderBook
		} else {
			p.appLogger.Warn("No order book channel found for market %s", market.MarketId)
		}
	})
}

func convertToOrderBook(msg *websocket.OrderBookMessage) connector.OrderBook {
	base := portfolio.NewAsset(msg.Market + ":" + msg.AssetID)
	quote := portfolio.NewAsset("USDC")

	orderbook := connector.OrderBook{
		Pair: portfolio.NewPair(base, quote),
		Bids: []connector.PriceLevel{},
		Asks: []connector.PriceLevel{},
	}

	bids, err := convertPriceLevels(msg.Bids)
	if err != nil {
		fmt.Printf("Error converting bids: %v\n", err)
		return orderbook
	}

	asks, err := convertPriceLevels(msg.Asks)

	if err != nil {
		fmt.Printf("Error converting asks: %v\n", err)
		return orderbook
	}

	orderbook.Bids = bids
	orderbook.Asks = asks

	return orderbook
}

func (p *polymarket) UnsubscribeMarket(market prediction.Market) error {
	//TODO implement me
	panic("implement me")
}

func (p *polymarket) GetMarketChannels() map[string]<-chan prediction.Market {
	//TODO implement me
	panic("implement me")
}
