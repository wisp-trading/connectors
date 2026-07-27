# Wisp Connectors

Exchange connector implementations for the [Wisp](https://github.com/wisp-trading/wisp) algorithmic trading framework.

Wisp is the only Go trading framework with native Polymarket (prediction markets) support, enabling you to trade across spot, perpetual futures, options, and binary outcome markets with a unified API.

## Supported Exchanges

| Exchange | Market Type | Status | Notes |
|----------|-------------|--------|--------|
| Hyperliquid | Perpetual Futures | **Production** | Primary live venue |
| Polymarket | Prediction Markets | Alpha | First-class domain; sharp edges remain |
| Bybit | Perpetual Futures | Beta | In-tree; less battle time than HL |
| Paradex | Perpetual Futures | Beta | Known gaps (e.g. klines) |
| Gate.io | Spot | Beta | In-tree |
| Deribit | Options | Experimental | Not production-ready |

## Installation

```bash
go get github.com/wisp-trading/connectors
```

## Quick Start

Import and use a connector:

```go
import "github.com/wisp-trading/connectors/pkg/connectors"

// Example: Initialize a Binance connector
binance := connectors.NewBinanceConnector(config)
```

## Documentation

For detailed guides, examples, and API reference, visit the [Wisp documentation](https://usewisp.dev/docs).

## Contributing

We welcome contributions! Please see the main [Wisp repository](https://github.com/wisp-trading/wisp) for contribution guidelines.

## License

This project is licensed under the MIT License. See the LICENSE file in the main repository for details.
