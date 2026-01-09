package startup

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/runtime"
	"github.com/backtesting-org/kronos-sdk/pkg/types/strategy"
)

// Startup wraps SDK runtime with live exchange bindings
type Startup struct {
	runtime runtime.Runtime
}

func NewStartup(runtime runtime.Runtime) *Startup {
	return &Startup{runtime: runtime}
}

// Start runs a strategy in plugin mode
func (s *Startup) Start(strategyDir string, kronosPath string) error {
	// Runtime.Start handles connector init, asset registration, and booting
	// Live-trading's fx module provides the real connector implementations
	// which get registered in ConnectorRegistry before this is called
	return s.runtime.Start(strategyDir, kronosPath)
}

// StartStandalone runs a strategy in standalone mode (debuggable)
func (s *Startup) StartStandalone(strat strategy.Strategy, configPath string, kronosPath string) error {
	return s.runtime.StartStandalone(strat, configPath, kronosPath)
}

// Stop gracefully shuts down
func (s *Startup) Stop() error {
	return s.runtime.Stop()
}
