# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Development Commands
```bash
# Run tests
go test ./...
go test -race ./...
go test -v ./...

# Run specific tests
go test -run TestConcurrentLogging
go test -run TestSampling

# Run benchmarks
go test -bench=. -benchmem ./...

# Build and verify
go mod tidy
go mod verify
```

### Required Import
The zapx provider must be imported for the logger to work:
```go
_ "github.com/HoangAnhNguyen269/loggerkit/provider/zapx"
```

### Console Control
Console output can now be controlled via configuration:
```go
// Disable console output (requires other sinks)
logger.NewProduction(logger.WithConsoleDisabled())
```

## Architecture Overview

### Core Components

**Logger Interface** (`interface.go`): The main public API with methods for Debug, Info, Warn, Error, Log, With, WithContext, and Close.

**Provider System** (`provider/zapx/corefactories/`): Refined CoreFactory architecture:
- CoreFactory interface with Name(), Enabled(), and Build() methods in `corefactories/` package
- Registry interface for dependency injection (global vs test registries)
- Console factory (now controllable via `DisableConsole` option)
- File factory with lumberjack rotation
- Elasticsearch factory with bulk indexing and DLQ
- MetricsCore wrapper for automatic metrics collection per sink
- Clean separation between factory definitions and registry management

**Configuration System**: Two-tier configuration approach:
- Legacy `Config` struct for backward compatibility
- New functional options pattern with `Options` struct and `WithXxx()` functions

**Context Integration** (`contextLogger/`): OpenTelemetry trace correlation and HTTP middleware for request/user ID extraction.

### Key Architecture Patterns

**Builder Pattern**: Eliminates import cycles through the `NewBuilder` interface and factory registration system.

**Factory Registry**: Refined plugin architecture with clean interfaces:
- Each output type implements CoreFactory in `corefactories/` package
- Global registry with `RegisterFactory()` and `Factories()` methods
- Registry interface allows dependency injection via `UseFactoryRegistry()` for testing
- MetricsCore automatically wraps each factory's output for per-sink metrics
- Console factory now respects `DisableConsole` option (default: enabled)

**Dual Constructor Pattern**: 
- `NewDevelopment()` - Human-readable console, debug level, no sampling
- `NewProduction()` - JSON output, info level, sampling enabled

**Field Helpers**: Two approaches for structured logging:
- Legacy helpers: `String()`, `Int()`, `Bool()`, etc.
- New F helpers: `F.String()`, `F.Int()`, `F.Err()`, etc.

### Breaking Changes in v2.0

**Required Changes**:
1. Add `Close(ctx context.Context) error` calls: `defer log.Close(context.Background())`
2. Import zapx provider: `_ "github.com/HoangAnhNguyen269/loggerkit/provider/zapx"`

**New in Current Version**:
- Console output controllable via `WithConsoleDisabled()` option
- Automatic per-sink metrics via MetricsCore wrapper
- Cleaner factory architecture with `corefactories/` package
- No fallback console core (error if no sinks enabled)

### Production Features

**Elasticsearch Integration**: Bulk indexer with exponential backoff, multiple authentication methods (API Key, Basic Auth, Service Token, Cloud ID), complete TLS support, and Dead Letter Queue for failed deliveries.

**Prometheus Metrics**: Five key production metrics with auto-registration and manual registration options:
- `logs_written_total{level,sink}` 
- `logs_dropped_total{sink,reason}`
- `es_bulk_retries_total{reason}`
- `es_queue_depth{service}`
- `es_bulk_latency_seconds{operation,status}`

**Sampling & Rate Control**: Zap-style sampling with configurable Initial/Thereafter values.

### Context & Tracing

**OpenTelemetry Integration**: Automatic trace_id/span_id extraction from spans with configurable request/user ID extraction from context values and HTTP headers.

**HTTP Middleware**: `HTTPMiddleware(ContextKeys)` and `DefaultHTTPMiddleware()` for automatic correlation.

### Reliability Features

**Graceful Shutdown**: Context-aware shutdown with timeout support and automatic sync of all sinks.

**Error Handling**: Graceful degradation for Elasticsearch unavailability, retry logic with exponential backoff + jitter, and Dead Letter Queue for undeliverable logs.

**Resource Management**: Proper closer registration and cleanup on shutdown.

### Testing Architecture

**Factory Testing**: Use `UseFactoryRegistry()` to inject custom registries:
```go
// In tests
customRegistry := &MyTestRegistry{}
zapx.UseFactoryRegistry(customRegistry)
defer zapx.UseFactoryRegistry(corefactories.DefaultRegistry())
```

**Metrics Testing**: MetricsCore automatically wraps all factory outputs for consistent per-sink metrics collection.

## Reference Documentation

Comprehensive reference documentation is available in the `/ref` directory:

### Core Documentation

- **[API Reference](ref/api-reference.md)**: Complete public API documentation including Logger interface, field helpers, configuration options, and usage examples
- **[Architecture Guide](ref/architecture.md)**: Internal architecture documentation covering the provider system, factory pattern, import cycle elimination, and extensibility mechanisms  
- **[Configuration Reference](ref/configuration.md)**: Comprehensive configuration guide for all sinks, functional options, and environment-specific setups
- **[Observability Guide](ref/observability.md)**: Production monitoring with Prometheus metrics, distributed tracing, and context correlation

### Quick References

**Getting Started**: Start with [API Reference](ref/api-reference.md) for basic usage patterns and examples.

**Production Setup**: See [Configuration Reference](ref/configuration.md) and [Observability Guide](ref/observability.md) for production deployment.

**Advanced Usage**: Review [Architecture Guide](ref/architecture.md) for extending the library or understanding internal implementation.

**Troubleshooting**: Check the architecture and configuration docs for debugging guidance and common patterns.