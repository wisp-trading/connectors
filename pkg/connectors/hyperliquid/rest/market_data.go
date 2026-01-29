package rest

import (
	"fmt"

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
	GetMetaAndAssetCtxs() (map[string]any, error)
	GetSpotMetaAndAssetCtxs() (map[string]any, error)
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

func (m *marketDataService) GetAllMids() (map[string]string, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}
	return info.AllMids()
}

func (m *marketDataService) GetL2Book(coin string) (*hyperliquid.L2Book, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}
	return info.L2Snapshot(coin)
}

func (m *marketDataService) GetCandles(coin, interval string, startTime, endTime int64) ([]hyperliquid.Candle, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}
	return info.CandlesSnapshot(coin, interval, startTime*millisecondsPerSecond, endTime*millisecondsPerSecond)
}

func (m *marketDataService) GetMeta() (*hyperliquid.Meta, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}
	return info.Meta()
}

func (m *marketDataService) GetSpotMeta() (*hyperliquid.SpotMeta, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}
	return info.SpotMeta()
}

func (m *marketDataService) GetMetaAndAssetCtxs() (map[string]any, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}
	return info.MetaAndAssetCtxs()
}

func (m *marketDataService) GetSpotMetaAndAssetCtxs() (map[string]any, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}
	return info.SpotMetaAndAssetCtxs()
}

func (m *marketDataService) NameToAsset(name string) int {
	info, err := m.client.GetInfo()
	if err != nil {
		return -1 // Return -1 on error since this returns int
	}
	return info.NameToAsset(name)
}
