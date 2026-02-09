package types

import (
	"github.com/wisp-trading/sdk/pkg/types/connector"
)

const (
	Hyperliquid connector.ExchangeName = "hyperliquid"
	Paradex     connector.ExchangeName = "paradex"
	Bybit       connector.ExchangeName = "bybit"
	GateSpot    connector.ExchangeName = "gate_spot"
	Polymarket  connector.ExchangeName = "polymarket"
)

var AllConnectors = []connector.ExchangeName{
	Hyperliquid,
	Paradex,
	Bybit,
	GateSpot,
	Polymarket,
}
