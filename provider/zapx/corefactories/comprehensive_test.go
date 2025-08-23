package corefactories_test

import (
	"os"
	"testing"

	logger "github.com/HoangAnhNguyen269/loggerkit"
	"github.com/HoangAnhNguyen269/loggerkit/provider/zapx/corefactories"
	"go.uber.org/zap/zapcore"
)

func TestConsoleFactoryEnabled(t *testing.T) {
	factory := &corefactories.ConsoleFactory{}

	testCases := []struct {
		name     string
		opts     logger.Options
		expected bool
	}{
		{
			name: "Default enabled",
			opts: logger.Options{
				DisableConsole: false,
			},
			expected: true,
		},
		{
			name: "Explicitly disabled",
			opts: logger.Options{
				DisableConsole: true,
			},
			expected: false,
		},
		{
			name: "Dev environment default",
			opts: logger.Options{
				Env: logger.EnvDev,
			},
			expected: true,
		},
		{
			name: "Prod environment default",
			opts: logger.Options{
				Env: "prod",
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enabled := factory.Enabled(tc.opts)
			if enabled != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, enabled)
			}
		})
	}
}

func TestConsoleFactoryBuild(t *testing.T) {
	factory := &corefactories.ConsoleFactory{}

	encCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		MessageKey:     "msg",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}

	testCases := []struct {
		name string
		opts logger.Options
	}{
		{
			name: "Development mode",
			opts: logger.Options{Env: logger.EnvDev},
		},
		{
			name: "Production mode",
			opts: logger.Options{Env: logger.EnvProd},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			core, closer, err := factory.Build(encCfg, zapcore.InfoLevel, nil, tc.opts)

			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}

			if core == nil {
				t.Fatal("Expected core to be non-nil")
			}

			if closer != nil {
				t.Error("Console factory should not return a closer")
			}

			// Verify core is enabled at info level
			if !core.Enabled(zapcore.InfoLevel) {
				t.Error("Core should be enabled at info level")
			}
		})
	}
}

func TestFileFactoryEnabled(t *testing.T) {
	factory := &corefactories.FileFactory{}

	testCases := []struct {
		name     string
		opts     logger.Options
		expected bool
	}{
		{
			name: "No file config",
			opts: logger.Options{
				File: nil,
			},
			expected: false,
		},
		{
			name: "With file config",
			opts: logger.Options{
				File: &logger.FileSink{
					Path: "/tmp/test.log",
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enabled := factory.Enabled(tc.opts)
			if enabled != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, enabled)
			}
		})
	}
}

func TestFileFactoryBuild(t *testing.T) {
	factory := &corefactories.FileFactory{}

	tempFile, err := os.CreateTemp("", "test-log-*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	opts := logger.Options{
		File: &logger.FileSink{
			Path:       tempFile.Name(),
			MaxSizeMB:  1,
			MaxBackups: 1,
			MaxAgeDays: 1,
			Compress:   false,
		},
	}

	encCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		MessageKey:     "msg",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}

	core, closer, err := factory.Build(encCfg, zapcore.InfoLevel, nil, opts)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if core == nil {
		t.Fatal("Expected core to be non-nil")
	}

	if closer == nil {
		t.Fatal("File factory should return a closer")
	}

	// Test closer
	if err := closer(); err != nil {
		t.Errorf("Closer failed: %v", err)
	}
}

func TestElasticFactoryEnabled(t *testing.T) {
	factory := &corefactories.ElasticFactory{}

	testCases := []struct {
		name     string
		opts     logger.Options
		expected bool
	}{
		{
			name: "No elastic config",
			opts: logger.Options{
				Elastic: nil,
			},
			expected: false,
		},
		{
			name: "With elastic config",
			opts: logger.Options{
				Elastic: &logger.ElasticSink{
					Addresses: []string{"http://localhost:9200"},
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enabled := factory.Enabled(tc.opts)
			if enabled != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, enabled)
			}
		})
	}
}

func TestFactoryRegistration(t *testing.T) {
	// Clear factories first
	corefactories.ClearFactories()

	// Verify no factories registered
	factories := corefactories.Factories()
	if len(factories) != 0 {
		t.Errorf("Expected 0 factories after clear, got %d", len(factories))
	}

	// Register test factories
	console := &corefactories.ConsoleFactory{}
	file := &corefactories.FileFactory{}

	corefactories.RegisterFactory(console)
	corefactories.RegisterFactory(file)

	// Verify registration
	factories = corefactories.Factories()
	if len(factories) != 2 {
		t.Errorf("Expected 2 factories, got %d", len(factories))
	}

	// Check factory names
	names := make(map[string]bool)
	for _, f := range factories {
		names[f.Name()] = true
	}

	if !names["console"] {
		t.Error("Expected console factory to be registered")
	}
	if !names["file"] {
		t.Error("Expected file factory to be registered")
	}
}

func TestDefaultRegistry(t *testing.T) {
	// Clear and re-register default factories
	corefactories.ClearFactories()
	corefactories.RegisterFactory(&corefactories.ConsoleFactory{})
	corefactories.RegisterFactory(&corefactories.FileFactory{})
	corefactories.RegisterFactory(&corefactories.ElasticFactory{})

	registry := corefactories.DefaultRegistry()
	factories := registry.All()

	if len(factories) != 3 {
		t.Errorf("Expected 3 default factories, got %d", len(factories))
	}

	// Verify all expected factories are present
	names := make(map[string]bool)
	for _, f := range factories {
		names[f.Name()] = true
	}

	expected := []string{"console", "file", "elasticsearch"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("Expected %s factory in default registry", name)
		}
	}
}

// Test factory name uniqueness
func TestFactoryNames(t *testing.T) {
	factories := []corefactories.CoreFactory{
		&corefactories.ConsoleFactory{},
		&corefactories.FileFactory{},
		&corefactories.ElasticFactory{},
	}

	names := make(map[string]bool)
	for _, f := range factories {
		name := f.Name()
		if names[name] {
			t.Errorf("Duplicate factory name: %s", name)
		}
		names[name] = true

		// Verify name is not empty
		if name == "" {
			t.Error("Factory name should not be empty")
		}
	}
}

// Test concurrent factory registration
func TestConcurrentRegistration(t *testing.T) {
	corefactories.ClearFactories()

	// Register factories concurrently
	done := make(chan bool, 3)

	go func() {
		corefactories.RegisterFactory(&corefactories.ConsoleFactory{})
		done <- true
	}()

	go func() {
		corefactories.RegisterFactory(&corefactories.FileFactory{})
		done <- true
	}()

	go func() {
		corefactories.RegisterFactory(&corefactories.ElasticFactory{})
		done <- true
	}()

	// Wait for all registrations
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify all were registered
	factories := corefactories.Factories()
	if len(factories) != 3 {
		t.Errorf("Expected 3 factories after concurrent registration, got %d", len(factories))
	}
}

// Test that Factories() returns a copy
func TestFactoriesCopy(t *testing.T) {
	corefactories.ClearFactories()
	corefactories.RegisterFactory(&corefactories.ConsoleFactory{})

	factories1 := corefactories.Factories()
	factories2 := corefactories.Factories()

	// Should be different slice instances
	if &factories1[0] == &factories2[0] {
		t.Error("Factories() should return a copy, not the same slice")
	}

	// But same content
	if factories1[0].Name() != factories2[0].Name() {
		t.Error("Factory copies should have same content")
	}
}
