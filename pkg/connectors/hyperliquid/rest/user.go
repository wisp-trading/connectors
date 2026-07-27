package rest

import (
	"context"
	"fmt"
	"time"

	"github.com/sonirico/go-hyperliquid"
)

func (m *marketDataService) GetUserState(user string) (hyperliquid.UserState, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return hyperliquid.UserState{}, fmt.Errorf("info client not configured: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Perp user state (was incorrectly calling SpotUserState before)
	state, err := info.UserState(ctx, user)
	if err != nil {
		return hyperliquid.UserState{}, err
	}
	if state == nil {
		return hyperliquid.UserState{}, fmt.Errorf("empty user state")
	}
	return *state, nil
}

func (m *marketDataService) GetOpenOrders(user string) ([]hyperliquid.OpenOrder, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return info.OpenOrders(ctx, user)
}

func (m *marketDataService) GetUserFills(user string) ([]hyperliquid.Fill, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return info.UserFills(ctx, hyperliquid.UserFillsParams{Address: user})
}
