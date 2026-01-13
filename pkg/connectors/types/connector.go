package types

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
)

const (
	Hyperliquid connector.ExchangeName = "hyperliquid"
	Paradex     connector.ExchangeName = "paradex"
	Bybit       connector.ExchangeName = "bybit"
	GateSpot    connector.ExchangeName = "gate_spot"
)

var AllConnectors = []connector.ExchangeName{
	Hyperliquid,
	Paradex,
	Bybit,
	GateSpot,
}
