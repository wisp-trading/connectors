package rest

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

func (m *marketDataService) GetUserState(user string) (hyperliquid.UserState, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return hyperliquid.UserState{}, fmt.Errorf("info client not configured: %w", err)
	}

	state, err := info.SpotUserState(user)

	if err != nil {
		return hyperliquid.UserState{}, err

	}

	return *state, nil
}

func (m *marketDataService) GetOpenOrders(user string) ([]hyperliquid.OpenOrder, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}
	return info.OpenOrders(user)
}

func (m *marketDataService) GetUserFills(user string) ([]hyperliquid.Fill, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}
	return info.UserFills(user)
}
