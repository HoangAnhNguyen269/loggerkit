# LoggerKit Reference Documentation

This directory contains comprehensive reference documentation for LoggerKit, a production-ready logging library for Go.

## Documentation Index

### 📖 [API Reference](api-reference.md)
Complete public API documentation covering:
- Logger interface and methods
- Log levels and field helpers  
- Constructor functions and functional options
- Configuration options and sink types
- Usage examples and patterns

### 🏗️ [Architecture Guide](architecture.md)
Internal architecture documentation for contributors and advanced users:
- Provider system and factory pattern
- Import cycle elimination techniques
- Registry system with dependency injection
- Automatic metrics integration
- Extensibility and plugin patterns

### ⚙️ [Configuration Reference](configuration.md)
Comprehensive configuration guide for all deployment scenarios:
- Core options and functional configuration
- Output sink configuration (console, file, Elasticsearch)
- Environment-specific setups (dev/staging/production)
- JSON/YAML configuration support
- Complete configuration examples

### 📊 [Observability Guide](observability.md)
Production monitoring and observability features:
- Prometheus metrics collection and configuration
- Distributed tracing with OpenTelemetry
- Context correlation and HTTP middleware
- Production setup and monitoring dashboards
- Alerting rules and troubleshooting

## Quick Start

1. **New Users**: Start with [API Reference](api-reference.md) for basic usage
2. **Production Deployment**: See [Configuration Reference](configuration.md) + [Observability Guide](observability.md)
3. **Contributors**: Review [Architecture Guide](architecture.md) for internal implementation
4. **Troubleshooting**: Check configuration and architecture docs for debugging

## Key Features Documented

- **Multiple Output Sinks**: Console, file rotation, Elasticsearch bulk indexing
- **Production Observability**: Prometheus metrics, distributed tracing, request correlation
- **Type-Safe Configuration**: Functional options with JSON/YAML support
- **Resilience**: Retry logic, dead letter queues, graceful degradation
- **Extensibility**: Plugin architecture for custom output sinks
- **Performance**: Sampling, bulk operations, minimal overhead

## Library Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Public API    │    │   Provider       │    │   Factories     │
│                 │    │   System         │    │                 │
│  Logger         │◄───│                  │◄───│  Console        │
│  interface.go   │    │  zapx/           │    │  File           │
│                 │    │  adapter.go      │    │  Elasticsearch  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

LoggerKit uses a sophisticated factory-based architecture that provides multiple output sinks, automatic metrics collection, and plugin-style extensibility while maintaining a clean public API.

## Getting Help

- **API Questions**: See [API Reference](api-reference.md)
- **Configuration Issues**: Check [Configuration Reference](configuration.md)  
- **Production Setup**: Review [Observability Guide](observability.md)
- **Architecture Questions**: See [Architecture Guide](architecture.md)
- **Bug Reports**: Check main repository README for issue reporting