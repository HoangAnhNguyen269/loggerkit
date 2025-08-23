package logger

import (
	"time"
)

// Sampling configuration for log sampling
type Sampling struct {
	Initial    int // Number of messages to log at start
	Thereafter int // Sample every Nth message after initial
}

// Retry configuration for failed operations
type Retry struct {
	Max        int           // Maximum number of retries
	BackoffMin time.Duration // Minimum backoff between retries
	BackoffMax time.Duration // Maximum backoff between retries
}

// FileSink configuration for file-based logging
type FileSink struct {
	Path       string // Path to log file
	MaxSizeMB  int    // Maximum size in MB before rotation
	MaxBackups int    // Maximum number of backup files to keep
	MaxAgeDays int    // Maximum age in days before deletion
	Compress   bool   // Compress rotated files
}

// ElasticSink configuration for Elasticsearch logging
type ElasticSink struct {
	Addresses     []string      // List of Elasticsearch addresses
	CloudID       string        // Cloud ID for Elastic Cloud
	Index         string        // Index pattern (default "<service>-%Y.%m.%d")
	FlushInterval time.Duration // How often to flush batches (default 2s)
	BulkActions   int           // DEPRECATED/IGNORED with go-elasticsearch; use FlushInterval/FlushBytes
	BulkSizeBytes int           // Size in bytes before flush (0 = disabled)
	Retry         Retry         // Retry configuration

	// Authentication
	Username     string // Basic auth username
	Password     string // Basic auth password
	APIKey       string // API Key for authentication
	ServiceToken string // Service token for authentication

	// TLS Configuration
	CACert             []byte // CA certificate
	ClientCert         []byte // Client certificate
	ClientKey          []byte // Client private key
	InsecureSkipVerify bool   // Skip TLS verification

	// Dead Letter Queue
	DLQPath string // Path for DLQ file (empty = disabled)
}

// ContextKeys configuration for extracting values from context
type ContextKeys struct {
	// Context keys for extracting values
	RequestIDKey any // Key to extract request ID from context
	UserIDKey    any // Key to extract user ID from context

	// HTTP header names for middleware extraction
	RequestIDHeader string // Header name for request ID (default "X-Request-ID")
	UserIDHeader    string // Header name for user ID (default "X-User-ID")
}

// MetricsOptions configuration for Prometheus metrics
type MetricsOptions struct {
	Enabled      bool // Enable metrics collection
	AutoRegister bool // Auto-register with prometheus.DefaultRegisterer
}

// Options represents the complete logger configuration
type Options struct {
	Env            string         // Environment: "dev" or "prod" //todo: enum
	Service        string         // Service name
	Level          string         // Log level: "debug", "info", "warn", "error" //todo:enum
	TimeFormat     string         // Time format (default RFC3339Nano)
	EnableCaller   bool           // Include caller information
	StacktraceAt   string         // Level at which to include stacktrace
	Sampling       *Sampling      // Sampling configuration
	DisableConsole bool           // default: false (console bật mặc định)
	File           *FileSink      // File sink configuration
	Elastic        *ElasticSink   // Elasticsearch sink configuration
	Context        ContextKeys    // Context extraction configuration
	Metrics        MetricsOptions // Metrics configuration

	// FactoryRegistry allows injecting custom factories for testing
	// If nil, uses the global registry from provider package
	// Using interface{} to avoid circular imports
	FactoryRegistry interface{ Factories() []interface{} }
}

// Option is a functional option for configuring the logger
type Option func(*Options)

// WithService sets the service name
func WithService(service string) Option {
	return func(o *Options) {
		o.Service = service
	}
}

// WithLevel sets the log level
func WithLevel(level string) Option {
	return func(o *Options) {
		o.Level = level
	}
}

// WithTimeFormat sets the time format
func WithTimeFormat(format string) Option {
	return func(o *Options) {
		o.TimeFormat = format
	}
}

// WithCaller enables or disables caller information
func WithCaller(enabled bool) Option {
	return func(o *Options) {
		o.EnableCaller = enabled
	}
}

// WithStacktraceAt sets the level at which stacktraces are included
func WithStacktraceAt(level string) Option {
	return func(o *Options) {
		o.StacktraceAt = level
	}
}

// WithSampling sets the sampling configuration
func WithSampling(sampling Sampling) Option {
	return func(o *Options) {
		o.Sampling = &sampling
	}
}

func WithConsoleDisabled() Option {
	return func(o *Options) { o.DisableConsole = true }
}

// WithFile sets the file sink configuration
func WithFile(file FileSink) Option {
	return func(o *Options) {
		o.File = &file
	}
}

// WithElastic sets the Elasticsearch sink configuration
func WithElastic(elastic ElasticSink) Option {
	return func(o *Options) {
		o.Elastic = &elastic
	}
}

// WithContext sets the context configuration
func WithContext(ctx ContextKeys) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}

// WithMetrics sets the metrics configuration
func WithMetrics(metrics MetricsOptions) Option {
	return func(o *Options) {
		o.Metrics = metrics
	}
}

// DefaultDevelopmentOptions returns default options for development
func DefaultDevelopmentOptions() Options {
	return Options{
		Env:          "dev",
		Service:      "app",
		Level:        "debug",
		TimeFormat:   time.RFC3339Nano,
		EnableCaller: true,
		StacktraceAt: "error",
		Sampling:     nil, // No sampling in dev
		Context: ContextKeys{
			RequestIDHeader: "X-Request-ID",
			UserIDHeader:    "X-User-ID",
		},
		Metrics: MetricsOptions{
			Enabled:      false,
			AutoRegister: false,
		},
	}
}

// DefaultProductionOptions returns default options for production
func DefaultProductionOptions() Options {
	return Options{
		Env:          "prod",
		Service:      "app",
		Level:        "info",
		TimeFormat:   time.RFC3339Nano,
		EnableCaller: true,
		StacktraceAt: "error",
		Sampling: &Sampling{
			Initial:    100,
			Thereafter: 100,
		},
		Context: ContextKeys{
			RequestIDHeader: "X-Request-ID",
			UserIDHeader:    "X-User-ID",
		},
		Metrics: MetricsOptions{
			Enabled:      false,
			AutoRegister: false,
		},
	}
}

// DefaultElasticRetry returns default retry configuration for Elasticsearch
func DefaultElasticRetry() Retry {
	return Retry{
		Max:        5,
		BackoffMin: 100 * time.Millisecond,
		BackoffMax: 5 * time.Second,
	}
}

// DefaultElasticSink returns default Elasticsearch sink configuration
func DefaultElasticSink(addresses []string, index string) ElasticSink {
	return ElasticSink{
		Addresses:     addresses,
		Index:         index,
		FlushInterval: 2 * time.Second,
		BulkActions:   5000,
		BulkSizeBytes: 0, // Disabled by default
		Retry:         DefaultElasticRetry(),
	}
}

// DefaultFileSink returns default file sink configuration
func DefaultFileSink(path string) FileSink {
	return FileSink{
		Path:       path,
		MaxSizeMB:  100,
		MaxBackups: 3,
		MaxAgeDays: 28,
		Compress:   true,
	}
}
