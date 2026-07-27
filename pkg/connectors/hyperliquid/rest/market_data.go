package rest

import (
	"context"
	"fmt"
	"time"

	"github.com/sonirico/go-hyperliquid"
	"github.com/wisp-trading/connectors/pkg/connectors/hyperliquid/adaptors"
)

// MarketDataService interface for market data operations
type MarketDataService interface {
	GetAllMids() (map[string]string, error)
	GetL2Book(coin string) (*hyperliquid.L2Book, error)
	GetCandles(coin, interval string, startTime, endTime int64) ([]hyperliquid.Candle, error)
	GetMeta() (*hyperliquid.Meta, error)
	GetSpotMeta() (*hyperliquid.SpotMeta, error)
	GetMetaAndAssetCtxs() (*hyperliquid.MetaAndAssetCtxs, error)
	GetSpotMetaAndAssetCtxs() (*hyperliquid.SpotMetaAndAssetCtxs, error)
	NameToAsset(name string) int

	// User data methods
	GetUserState(user string) (hyperliquid.UserState, error)
	GetOpenOrders(user string) ([]hyperliquid.OpenOrder, error)
	GetUserFills(user string) ([]hyperliquid.Fill, error)

	// Funding rate methods - historical only
	GetAssetContext(coin string) (*AssetContext, error)
	GetAllAssetContexts() ([]AssetContext, error)
	GetHistoricalFundingRates(coin string, startTime, endTime int64) ([]hyperliquid.FundingHistory, error)
}

// marketDataService implementation
type marketDataService struct {
	client adaptors.InfoClient
}

var millisecondsPerSecond = int64(1000)

func NewMarketDataService(client adaptors.InfoClient) MarketDataService {
	return &marketDataService{client: client}
}

func (m *marketDataService) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

func (m *marketDataService) GetAllMids() (map[string]string, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}
	ctx, cancel := m.ctx()
	defer cancel()
	return info.AllMids(ctx)
}

func (m *marketDataService) GetL2Book(coin string) (*hyperliquid.L2Book, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}
	ctx, cancel := m.ctx()
	defer cancel()
	return info.L2Snapshot(ctx, coin)
}

func (m *marketDataService) GetCandles(coin, interval string, startTime, endTime int64) ([]hyperliquid.Candle, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}
	ctx, cancel := m.ctx()
	defer cancel()
	return info.CandlesSnapshot(ctx, coin, interval, startTime*millisecondsPerSecond, endTime*millisecondsPerSecond)
}

func (m *marketDataService) GetMeta() (*hyperliquid.Meta, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}
	ctx, cancel := m.ctx()
	defer cancel()
	return info.Meta(ctx)
}

func (m *marketDataService) GetSpotMeta() (*hyperliquid.SpotMeta, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}
	ctx, cancel := m.ctx()
	defer cancel()
	return info.SpotMeta(ctx)
}

func (m *marketDataService) GetMetaAndAssetCtxs() (*hyperliquid.MetaAndAssetCtxs, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}
	ctx, cancel := m.ctx()
	defer cancel()
	return info.MetaAndAssetCtxs(ctx, hyperliquid.MetaAndAssetCtxsParams{})
}

func (m *marketDataService) GetSpotMetaAndAssetCtxs() (*hyperliquid.SpotMetaAndAssetCtxs, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}
	ctx, cancel := m.ctx()
	defer cancel()
	return info.SpotMetaAndAssetCtxs(ctx)
}

func (m *marketDataService) NameToAsset(name string) int {
	info, err := m.client.GetInfo()
	if err != nil {
		return -1
	}
	// CoinToAsset replaced NameToAsset in go-hyperliquid v0.35
	id, ok := info.CoinToAsset(name)
	if !ok {
		return -1
	}
	return id
}
