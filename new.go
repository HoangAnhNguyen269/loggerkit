package logger

import (
	"fmt"
	"time"
)

// NewBuilder is used to create loggers without import cycles
type NewBuilder interface {
	NewWithOptions(opts Options) (Logger, error)
}

var defaultBuilder NewBuilder

// SetBuilder sets the default builder (called by provider packages)
func SetBuilder(builder NewBuilder) {
	defaultBuilder = builder
}

// New creates a logger with the provided options
func New(opts Options) (Logger, error) {
	if defaultBuilder == nil {
		return nil, fmt.Errorf("no logger builder registered")
	}
	return defaultBuilder.NewWithOptions(opts)
}

// NewDevelopment creates a logger with development defaults and applies optional overrides
func NewDevelopment(opts ...Option) (Logger, error) {
	// Start with development defaults
	options := DefaultDevelopmentOptions()

	// Apply any overrides
	for _, opt := range opts {
		opt(&options)
	}

	return New(options)
}

// NewProduction creates a logger with production defaults and applies optional overrides
func NewProduction(opts ...Option) (Logger, error) {
	// Start with production defaults
	options := DefaultProductionOptions()

	// Apply any overrides
	for _, opt := range opts {
		opt(&options)
	}

	return New(options)
}

// MustNew creates a logger and panics on error (for backward compatibility)
func MustNew(cfg *Config) Logger {
	// Convert old Config to new Options
	opts := configToOptions(cfg)

	logger, err := New(opts)
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}
	return logger
}

// MustNewDevelopment creates a development logger and panics on error
func MustNewDevelopment(opts ...Option) Logger {
	logger, err := NewDevelopment(opts...)
	if err != nil {
		panic(fmt.Sprintf("failed to create development logger: %v", err))
	}
	return logger
}

// MustNewProduction creates a production logger and panics on error
func MustNewProduction(opts ...Option) Logger {
	logger, err := NewProduction(opts...)
	if err != nil {
		panic(fmt.Errorf("failed to create production logger: %v", err))
	}
	return logger
}

// configToOptions converts the old Config to new Options (for backward compatibility)
func configToOptions(cfg *Config) Options {
	opts := Options{
		Env:          "dev", // Default to dev for backward compatibility
		Service:      "app",
		Level:        string(cfg.Level),
		TimeFormat:   time.RFC3339Nano,
		EnableCaller: true,
		StacktraceAt: "error",
		Context: ContextKeys{
			RequestIDHeader: "X-Request-ID",
			UserIDHeader:    "X-User-ID",
		},
		Metrics: MetricsOptions{
			Enabled:      false,
			AutoRegister: false,
		},
	}

	// Convert file configuration
	if cfg.FileConfig != nil {
		opts.File = &FileSink{
			Path:       cfg.FileConfig.Filename,
			MaxSizeMB:  cfg.FileConfig.MaxSize,
			MaxBackups: cfg.FileConfig.MaxBackups,
			MaxAgeDays: cfg.FileConfig.MaxAge,
			Compress:   cfg.FileConfig.Compress,
		}
	}

	// Convert Elasticsearch configuration
	if cfg.ElasticConfig != nil {
		opts.Elastic = &ElasticSink{
			Addresses:     []string{cfg.ElasticConfig.URL},
			Index:         cfg.ElasticConfig.Index,
			FlushInterval: 2 * time.Second,
			BulkActions:   5000,
			Retry:         DefaultElasticRetry(),
		}
	}

	return opts
}
