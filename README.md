# Wisp Connectors

Exchange connector implementations for the [Wisp](https://github.com/wisp-trading/wisp) algorithmic trading framework.

Wisp is the only Go trading framework with native Polymarket (prediction markets) support, enabling you to trade across spot, perpetual futures, options, and binary outcome markets with a unified API.

## Supported Exchanges

| Exchange | Market Type | Status |
|----------|-------------|--------|
| Gate.io | Spot | Production |
| Hyperliquid | Perpetual Futures | Production |
| Paradex | Perpetual Futures | Production |
| Bybit | Perpetual Futures | Production |
| Deribit | Options | Production |
| Polymarket | Prediction Markets | Alpha |

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

For detailed guides, examples, and API reference, visit the [Wisp documentation](https://docs.usewisp.dev).

## Contributing

We welcome contributions! Please see the main [Wisp repository](https://github.com/wisp-trading/wisp) for contribution guidelines.

## License

This project is licensed under the MIT License. See the LICENSE file in the main repository for details.
