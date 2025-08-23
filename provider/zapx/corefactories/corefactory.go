package corefactories

import (
	"sync"

	logger "github.com/HoangAnhNguyen269/loggerkit"
	"go.uber.org/zap/zapcore"
)

// CoreFactory interface for creating zapcore.Core instances
type CoreFactory interface {
	// Name returns the unique name of this factory
	Name() string

	// Enabled determines if this factory should create a core based on the options
	Enabled(opts logger.Options) bool

	// Build creates a zapcore.Core and returns it along with an optional closer function
	Build(encCfg zapcore.EncoderConfig, lvl zapcore.Level, metrics *logger.Metrics, opts logger.Options) (zapcore.Core, func() error, error)
}

// Global registry for CoreFactory instances
var (
	factoriesMu sync.RWMutex
	factories   []CoreFactory
)

// RegisterFactory registers a CoreFactory in the global registry
func RegisterFactory(f CoreFactory) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories = append(factories, f)
}

// Factories returns a copy of all registered factories (read-only)
func Factories() []CoreFactory {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]CoreFactory, len(factories))
	copy(result, factories)
	return result
}

// ✅ New: thin interface with concrete type, no interface{}
type Registry interface {
	All() []CoreFactory
}

type globalRegistry struct{}

func (globalRegistry) All() []CoreFactory { return Factories() }

func DefaultRegistry() Registry { return globalRegistry{} }

// For tests
func ClearFactories() {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories = nil
}
