package polymarket

import (
	"fmt"

	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
)

func (p *polymarket) StartWebSocket() error {
	if !p.initialized {
		return fmt.Errorf("connector not initialized")
	}

	if p.clobWebsocket == nil {
		return fmt.Errorf("websocket client not configured")
	}

	return nil

}

func (p *polymarket) StopWebSocket() error {
	if p.clobWebsocket == nil {
		return fmt.Errorf("websocket service not initialized")
	}

	for marketID := range p.orderBookChannels {
		if err := p.UnsubscribeMarket(prediction.Market{Slug: marketID}); err != nil {
			p.appLogger.Warn("Failed to unsubscribe from market %s: %v", marketID, err)
		}
	}

	p.clobWebsocket.Close()
	p.appLogger.Info("WebSocket connection closed successfully")
	return nil
}

func (p *polymarket) IsWebSocketConnected() bool {
	if !p.initialized {
		return false
	}

	return p.clobWebsocket != nil
}
