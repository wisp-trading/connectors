package rest

import (
	"fmt"

	hyperliquid "github.com/sonirico/go-hyperliquid"
)

func (s *spotMarketDataService) FetchSpotMeta() (*hyperliquid.SpotMeta, error) {
	info, err := s.infoClient.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}

	meta, err := info.SpotMeta()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spot meta: %w", err)
	}

	return meta, nil
}

func (s *spotMarketDataService) FetchSpotMetaAndAssetCtxs() (map[string]any, error) {
	info, err := s.infoClient.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}

	data, err := info.SpotMetaAndAssetCtxs()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spot meta and asset ctxs: %w", err)
	}

	return data, nil
}

func (s *spotMarketDataService) FetchSpotUserState(address string) (*hyperliquid.UserState, error) {
	info, err := s.infoClient.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}

	state, err := info.SpotUserState(address)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spot user state: %w", err)
	}

	return state, nil
}

func (s *spotMarketDataService) FetchL2Book(coin string) (*hyperliquid.L2Book, error) {
	info, err := s.infoClient.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}

	book, err := info.L2Snapshot(coin)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch L2 book for %s: %w", coin, err)
	}

	return book, nil
}
