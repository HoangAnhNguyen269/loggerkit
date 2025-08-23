package corefactories

import (
	"testing"

	logger "github.com/HoangAnhNguyen269/loggerkit"
	"go.uber.org/zap/zapcore"
)

// MockFactory for testing
type MockFactory struct {
	name    string
	enabled bool
}

func (mf *MockFactory) Name() string {
	return mf.name
}

func (mf *MockFactory) Enabled(opts logger.Options) bool {
	return mf.enabled
}

func (mf *MockFactory) Build(encCfg zapcore.EncoderConfig, lvl zapcore.Level, metrics *logger.Metrics, opts logger.Options) (zapcore.Core, func() error, error) {
	// Return a no-op core for testing
	return zapcore.NewNopCore(), nil, nil
}

func TestFactoryRegistry(t *testing.T) {
	// Clear existing factories
	ClearFactories()

	// Register test factories
	factory1 := &MockFactory{name: "test1", enabled: true}
	factory2 := &MockFactory{name: "test2", enabled: false}

	RegisterFactory(factory1)
	RegisterFactory(factory2)

	factories := Factories()

	if len(factories) != 2 {
		t.Errorf("Expected 2 factories, got %d", len(factories))
	}

	if factories[0].Name() != "test1" {
		t.Errorf("Expected first factory name to be 'test1', got '%s'", factories[0].Name())
	}

	if factories[1].Name() != "test2" {
		t.Errorf("Expected second factory name to be 'test2', got '%s'", factories[1].Name())
	}
}

func TestGlobalRegistryInterface(t *testing.T) {
	// Clear existing factories
	ClearFactories()

	// Register a test factory
	factory := &MockFactory{name: "test", enabled: true}
	RegisterFactory(factory)

	// Test the global registry interface
	registry := DefaultRegistry()
	factories := registry.All()

	if len(factories) != 1 {
		t.Errorf("Expected 1 factory, got %d", len(factories))
	}

	// Verify factory
	if factories[0].Name() != "test" {
		t.Errorf("Expected factory name to be 'test', got '%s'", factories[0].Name())
	}
}
