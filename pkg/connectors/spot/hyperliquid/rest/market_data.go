package rest

import (
	"context"
	"fmt"

	hyperliquid "github.com/sonirico/go-hyperliquid"
)

func (s *spotMarketDataService) FetchSpotMeta() (*hyperliquid.SpotMeta, error) {
	info, err := s.infoClient.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}

	meta, err := info.SpotMeta(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spot meta: %w", err)
	}

	return meta, nil
}

func (s *spotMarketDataService) FetchSpotMetaAndAssetCtxs() (*hyperliquid.SpotMetaAndAssetCtxs, error) {
	info, err := s.infoClient.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}

	data, err := info.SpotMetaAndAssetCtxs(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spot meta and asset ctxs: %w", err)
	}

	return data, nil
}

func (s *spotMarketDataService) FetchSpotUserState(address string) (*hyperliquid.SpotUserState, error) {
	info, err := s.infoClient.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}

	state, err := info.SpotUserState(context.Background(), address)
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

	book, err := info.L2Snapshot(context.Background(), coin)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch L2 book for %s: %w", coin, err)
	}

	return book, nil
}

const millisecondsPerSecond = int64(1000)

func (s *spotMarketDataService) GetCandles(coin, interval string, startTime, endTime int64) ([]hyperliquid.Candle, error) {
	info, err := s.infoClient.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}

	candles, err := info.CandlesSnapshot(context.Background(), coin, interval, startTime*millisecondsPerSecond, endTime*millisecondsPerSecond)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch candles for %s: %w", coin, err)
	}

	return candles, nil
}

func (s *spotMarketDataService) GetOpenOrders(user string) ([]hyperliquid.OpenOrder, error) {
	info, err := s.infoClient.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}

	orders, err := info.OpenOrders(context.Background(), user)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch open orders: %w", err)
	}

	return orders, nil
}

func (s *spotMarketDataService) GetUserFills(user string) ([]hyperliquid.Fill, error) {
	info, err := s.infoClient.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}

	fills, err := info.UserFills(context.Background(), hyperliquid.UserFillsParams{Address: user})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user fills: %w", err)
	}

	return fills, nil
}

func (s *spotMarketDataService) GetOrderByOid(user string, oid int64) (*hyperliquid.OrderQueryResult, error) {
	info, err := s.infoClient.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}

	result, err := info.QueryOrderByOid(context.Background(), user, oid)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order %d: %w", oid, err)
	}

	return result, nil
}
