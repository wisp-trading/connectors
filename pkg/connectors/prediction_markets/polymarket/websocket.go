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

	for _, market := range p.subscribedMarkets {
		for _, outcome := range market.Outcomes {
			if err := p.UnsubscribeMarket(prediction.Market{Slug: outcome.OutcomeID.String()}); err != nil {
				p.appLogger.Warn("Failed to unsubscribe from market %s: %v", outcome.OutcomeID.String(), err)
			}
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
