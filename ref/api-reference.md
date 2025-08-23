# LoggerKit API Reference

LoggerKit is a production-ready logging library for Go that provides structured logging with multiple output sinks, metrics integration, and context correlation.

## Table of Contents

- [Core Logger Interface](#core-logger-interface)
- [Log Levels](#log-levels)
- [Structured Fields](#structured-fields)
- [Configuration](#configuration)
- [Constructor Functions](#constructor-functions)
- [Functional Options](#functional-options)
- [Sink Configuration](#sink-configuration)

## Core Logger Interface

The main `Logger` interface provides a clean, focused API:

```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    Log(level Level, msg string, fields ...Field)
    With(fields ...Field) Logger
    WithContext(ctx context.Context) Logger
    Close(ctx context.Context) error
}
```

### Methods

- **`Debug/Info/Warn/Error`**: Standard log level methods
- **`Log(level, msg, fields...)`**: Generic logging method with dynamic level
- **`With(fields...)`**: Returns a new logger with additional fields attached
- **`WithContext(ctx)`**: Returns a logger with context values (trace ID, user ID, etc.)
- **`Close(ctx)`**: Graceful shutdown with context timeout support

### Usage Examples

```go
// Basic logging
log.Info("User logged in", logger.F.String("user_id", "123"))

// With additional context
userLog := log.With(logger.F.String("user_id", "123"))
userLog.Info("Action performed")

// Context correlation
ctxLog := log.WithContext(ctx)  // Extracts trace_id, span_id automatically
ctxLog.Error("Database error", logger.F.Err(err))

// Graceful shutdown
defer log.Close(context.Background())
```

## Log Levels

```go
type Level string

const (
    DebugLevel Level = "debug"
    InfoLevel  Level = "info"
    WarnLevel  Level = "warn"
    ErrorLevel Level = "error"
)
```

### Level Methods

- **`ParseLevel(s string) (Level, error)`**: Convert string to Level
- **JSON/Text Marshaling**: Full serialization support
- **Case flexibility**: Accepts "warn"/"warning" variants

```go
level, err := logger.ParseLevel("info")
if err != nil {
    // handle error
}
log.Log(level, "Dynamic level message")
```

## Structured Fields

### Field Type

```go
type Field struct {
    Key string
    Val any
}
```

### Field Helpers

**Recommended: F Helpers**
```go
logger.F.String(key, value string) Field
logger.F.Int(key string, value int) Field
logger.F.Bool(key string, value bool) Field
logger.F.Err(err error) Field           // Key: "error"
logger.F.Duration(key string, value time.Duration) Field
logger.F.Any(key string, value any) Field
```

**Legacy Helpers** (backward compatibility)
```go
logger.String(key, value string) Field
logger.Int(key string, value int) Field
logger.Bool(key string, value bool) Field
logger.Error(err error) Field           // Key: "error"
logger.Duration(key string, value time.Duration) Field
logger.Time(key string, value time.Time) Field
logger.Any(key string, value any) Field
```

### Usage Examples

```go
// Recommended approach
log.Info("User action",
    logger.F.String("user_id", "123"),
    logger.F.Int("attempts", 3),
    logger.F.Bool("success", true),
)

// Error logging
if err != nil {
    log.Error("Operation failed", logger.F.Err(err))
}
```

## Configuration

### Options Struct

```go
type Options struct {
    Env            Env              // "dev" or "prod"
    Service        string           // Service name
    Level          Level            // Minimum log level
    TimeFormat     string           // Time format (default: ISO8601)
    EnableCaller   bool             // Include caller information
    StacktraceAt   Level            // Level to include stacktraces
    Sampling       *Sampling        // Log sampling configuration
    DisableConsole bool             // Disable console output
    File           *FileSink        // File output configuration
    Elastic        *ElasticSink     // Elasticsearch output configuration
    Context        ContextKeys      // Context extraction keys
    Metrics        MetricsOptions   // Metrics integration
}
```

### Environment Type

```go
type Env string

const (
    EnvDev  Env = "dev"   // Development: console output, human-readable
    EnvProd Env = "prod"  // Production: JSON output, structured
)
```

## Constructor Functions

### Primary Constructors

```go
// Development logging: console output, debug level, no sampling
func NewDevelopment(opts ...Option) (Logger, error)

// Production logging: JSON output, info level, sampling enabled  
func NewProduction(opts ...Option) (Logger, error)
```

### Legacy Constructor

```go
// Backward compatibility - panics on error
func MustNew(cfg *Config) Logger
```

### Required Import

**Important**: You must import the zapx provider for the logger to work:

```go
import (
    logger "github.com/HoangAnhNguyen269/loggerkit"
    _ "github.com/HoangAnhNguyen269/loggerkit/provider/zapx"
)
```

### Usage Examples

```go
// Simple development logger
log, err := logger.NewDevelopment()
if err != nil {
    panic(err)
}
defer log.Close(context.Background())

// Production logger with file output
log, err := logger.NewProduction(
    logger.WithService("my-service"),
    logger.WithLevel(logger.InfoLevel),
    logger.WithFile(logger.FileSink{
        Path:       "/var/log/app.log",
        MaxSizeMB:  100,
        MaxBackups: 3,
    }),
)
```

## Functional Options

### Core Options

```go
WithEnv(env Env) Option
WithService(service string) Option  
WithLevel(level Level) Option
WithCaller(enabled bool) Option
WithStacktrace(level Level) Option
WithTimeFormat(format string) Option
WithSampling(sampling Sampling) Option
```

### Output Control

```go
WithConsoleDisabled() Option
WithFile(sink FileSink) Option
WithElastic(sink ElasticSink) Option
```

### Advanced Features

```go
WithContext(keys ContextKeys) Option
WithMetrics(options MetricsOptions) Option
```

## Sink Configuration

### File Sink

```go
type FileSink struct {
    Path         string  // File path
    MaxSizeMB    int     // Max file size in MB (default: 100)
    MaxAge       int     // Max age in days (default: 7)
    MaxBackups   int     // Max backup files (default: 3)
    LocalTime    bool    // Use local time for backups
    Compress     bool    // Compress old files
}
```

### Elasticsearch Sink

```go
type ElasticSink struct {
    Addresses      []string      // ES cluster addresses
    Index          string        // Index pattern (supports time formatting)
    FlushInterval  time.Duration // Bulk flush interval
    BulkSizeBytes  int          // Max bulk size in bytes
    
    // Authentication (choose one)
    APIKey       string
    Username     string
    Password     string  
    ServiceToken string
    CloudID      string
    
    // TLS Configuration
    InsecureSkipVerify bool
    ClientCert         []byte
    ClientKey          []byte
    
    // Error Handling
    Retry   Retry   // Retry configuration
    DLQPath string  // Dead Letter Queue file path
}
```

### Retry Configuration

```go
type Retry struct {
    Max        int           // Max retry attempts
    BackoffMin time.Duration // Min backoff duration
    BackoffMax time.Duration // Max backoff duration
}
```

### Context Keys

```go
type ContextKeys struct {
    RequestIDKey any  // Context key for request ID
    UserIDKey    any  // Context key for user ID
}
```

### Metrics Options

```go
type MetricsOptions struct {
    Enabled      bool  // Enable Prometheus metrics
    AutoRegister bool  // Auto-register with default registry
}
```

### Sampling Configuration

```go
type Sampling struct {
    Initial    int  // Log first N entries per second
    Thereafter int  // Then log every Nth entry
}
```

## Complete Example

```go
package main

import (
    "context"
    "time"
    
    logger "github.com/HoangAnhNguyen269/loggerkit"
    _ "github.com/HoangAnhNguyen269/loggerkit/provider/zapx"
)

func main() {
    log, err := logger.NewProduction(
        logger.WithService("my-service"),
        logger.WithLevel(logger.InfoLevel),
        logger.WithCaller(true),
        logger.WithFile(logger.FileSink{
            Path:       "/var/log/app.log",
            MaxSizeMB:  100,
            MaxBackups: 3,
            Compress:   true,
        }),
        logger.WithElastic(logger.ElasticSink{
            Addresses:     []string{"http://localhost:9200"},
            Index:         "logs-%Y.%m.%d",
            FlushInterval: 5 * time.Second,
            APIKey:        "your-api-key",
        }),
        logger.WithMetrics(logger.MetricsOptions{
            Enabled:      true,
            AutoRegister: true,
        }),
    )
    if err != nil {
        panic(err)
    }
    defer log.Close(context.Background())

    // Use the logger
    log.Info("Application started",
        logger.F.String("version", "1.0.0"),
        logger.F.Int("port", 8080),
    )
    
    // Contextual logging
    userLog := log.With(logger.F.String("user_id", "123"))
    userLog.Info("User action performed")
}
```