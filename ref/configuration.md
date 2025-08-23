# LoggerKit Configuration Reference

This document provides comprehensive configuration options for LoggerKit, including all sink types, functional options, and configuration patterns.

## Table of Contents

- [Configuration Overview](#configuration-overview)
- [Core Options](#core-options)
- [Output Sinks](#output-sinks)
- [Advanced Features](#advanced-features)
- [Environment-Specific Configs](#environment-specific-configs)
- [JSON/YAML Configuration](#jsonyaml-configuration)

## Configuration Overview

LoggerKit supports two configuration approaches:

1. **Functional Options** (Recommended): Modern, flexible approach
2. **Config Struct** (Legacy): Backward compatibility support

### Functional Options Pattern

```go
log, err := logger.NewProduction(
    logger.WithService("my-service"),
    logger.WithLevel(logger.InfoLevel),
    logger.WithFile(logger.FileSink{Path: "/var/log/app.log"}),
)
```

### Config Struct Pattern

```go
config := &logger.Config{
    Service: "my-service",
    Level:   "info",
    File: &logger.FileSink{
        Path: "/var/log/app.log",
    },
}
log := logger.MustNew(config) // Panics on error
```

## Core Options

### Basic Configuration

```go
// Service identification
WithService(service string) Option

// Environment mode: "dev" or "prod"  
WithEnv(env Env) Option

// Minimum log level: debug, info, warn, error
WithLevel(level Level) Option

// Time format (default: ISO8601)
WithTimeFormat(format string) Option

// Enable caller information (file:line)
WithCaller(enabled bool) Option

// Enable stacktraces at specified level
WithStacktrace(level Level) Option
```

### Example: Basic Production Config

```go
log, err := logger.NewProduction(
    logger.WithService("payment-service"),
    logger.WithEnv(logger.EnvProd),
    logger.WithLevel(logger.InfoLevel),
    logger.WithCaller(true),
    logger.WithStacktrace(logger.ErrorLevel),
    logger.WithTimeFormat(time.RFC3339Nano),
)
```

### Sampling Configuration

Control log volume with sampling:

```go
type Sampling struct {
    Initial    int  // Log first N entries per second
    Thereafter int  // Then log every Nth entry  
}

// Enable sampling
WithSampling(sampling Sampling) Option
```

Example:
```go
log, err := logger.NewProduction(
    logger.WithSampling(logger.Sampling{
        Initial:    100,  // First 100 logs per second
        Thereafter: 100,  // Then every 100th log
    }),
)
```

## Output Sinks

### Console Output

Console output is **enabled by default**. Disable it explicitly:

```go
// Disable console output (requires other sinks)
WithConsoleDisabled() Option
```

Console behavior:
- **Development**: Human-readable console format
- **Production**: JSON format for structured parsing

```go
// Development with console only
log, err := logger.NewDevelopment()

// Production without console 
log, err := logger.NewProduction(
    logger.WithConsoleDisabled(),
    logger.WithFile(logger.FileSink{Path: "/var/log/app.log"}),
)
```

### File Output

File output with automatic rotation:

```go
type FileSink struct {
    Path         string  // File path (required)
    MaxSizeMB    int     // Max file size in MB (default: 100)
    MaxAge       int     // Max age in days (default: 7) 
    MaxBackups   int     // Max backup files (default: 3)
    LocalTime    bool    // Use local time for backups (default: false)
    Compress     bool    // Compress old files (default: false)
}

WithFile(sink FileSink) Option
```

#### File Configuration Examples

**Basic file logging:**
```go
log, err := logger.NewProduction(
    logger.WithFile(logger.FileSink{
        Path: "/var/log/app.log",
    }),
)
```

**Advanced file rotation:**
```go
log, err := logger.NewProduction(
    logger.WithFile(logger.FileSink{
        Path:         "/var/log/app.log",
        MaxSizeMB:    50,    // Rotate at 50MB
        MaxAge:       30,    // Keep for 30 days
        MaxBackups:   10,    // Keep 10 backup files
        LocalTime:    true,  // Use local timezone
        Compress:     true,  // Compress rotated files
    }),
)
```

### Elasticsearch Output

Elasticsearch sink with bulk indexing and resilience features:

```go
type ElasticSink struct {
    // Connection
    Addresses      []string      // ES cluster addresses (required)
    Index          string        // Index pattern (required)
    
    // Bulk Configuration
    FlushInterval  time.Duration // Bulk flush interval (default: 1s)
    BulkSizeBytes  int          // Max bulk size in bytes (default: 5MB)
    
    // Authentication (choose one)
    APIKey       string        // Elasticsearch API Key
    Username     string        // Basic auth username
    Password     string        // Basic auth password
    ServiceToken string        // Service token
    CloudID      string        // Elastic Cloud ID
    
    // TLS Configuration  
    InsecureSkipVerify bool     // Skip TLS verification
    ClientCert         []byte   // Client certificate
    ClientKey          []byte   // Client private key
    
    // Resilience
    Retry   Retry   // Retry configuration
    DLQPath string  // Dead Letter Queue file path
}

WithElastic(sink ElasticSink) Option
```

#### Elasticsearch Authentication Examples

**API Key Authentication:**
```go
log, err := logger.NewProduction(
    logger.WithElastic(logger.ElasticSink{
        Addresses:     []string{"https://localhost:9200"},
        Index:         "logs-%Y.%m.%d",
        APIKey:        "base64-encoded-api-key",
        FlushInterval: 5 * time.Second,
    }),
)
```

**Basic Authentication:**
```go
log, err := logger.NewProduction(
    logger.WithElastic(logger.ElasticSink{
        Addresses: []string{"https://elasticsearch:9200"},
        Index:     "app-logs-%Y.%m.%d",
        Username:  "elastic",
        Password:  "password",
    }),
)
```

**Elastic Cloud:**
```go
log, err := logger.NewProduction(
    logger.WithElastic(logger.ElasticSink{
        CloudID: "my-cluster:dGVzdC5jbG91ZC5lcy5pbzo5MjQz",
        APIKey:  "cloud-api-key",
        Index:   "logs-%Y.%m.%d",
    }),
)
```

#### Elasticsearch Resilience Configuration

```go
type Retry struct {
    Max        int           // Maximum retry attempts
    BackoffMin time.Duration // Minimum backoff duration
    BackoffMax time.Duration // Maximum backoff duration  
}
```

**Production resilience setup:**
```go
log, err := logger.NewProduction(
    logger.WithElastic(logger.ElasticSink{
        Addresses: []string{
            "https://es-node1:9200", 
            "https://es-node2:9200",
        },
        Index:         "logs-%Y.%m.%d",
        FlushInterval: 2 * time.Second,
        BulkSizeBytes: 2 * 1024 * 1024, // 2MB batches
        APIKey:        "your-api-key",
        
        // Retry configuration
        Retry: logger.Retry{
            Max:        5,
            BackoffMin: 100 * time.Millisecond,
            BackoffMax: 30 * time.Second,
        },
        
        // Dead Letter Queue for failed logs
        DLQPath: "/var/log/failed-logs.dlq",
    }),
)
```

## Advanced Features

### Context Correlation

Configure automatic context value extraction:

```go
type ContextKeys struct {
    RequestIDKey    any    // Context value key for request ID
    UserIDKey       any    // Context value key for user ID
    RequestIDHeader string // HTTP header name for request ID  
    UserIDHeader    string // HTTP header name for user ID
}

WithContext(keys ContextKeys) Option
```

**Context correlation setup:**
```go
log, err := logger.NewProduction(
    logger.WithContext(logger.ContextKeys{
        RequestIDKey:    "request_id",
        UserIDKey:       "user_id", 
        RequestIDHeader: "X-Request-ID",
        UserIDHeader:    "X-User-ID",
    }),
)
```

### Prometheus Metrics

Enable production metrics collection:

```go
type MetricsOptions struct {
    Enabled      bool  // Enable metrics collection
    AutoRegister bool  // Auto-register with default registry
}

WithMetrics(options MetricsOptions) Option
```

**Metrics configuration examples:**

**Auto-registration (simple):**
```go
log, err := logger.NewProduction(
    logger.WithMetrics(logger.MetricsOptions{
        Enabled:      true,
        AutoRegister: true,
    }),
)
```

**Manual registration (custom registry):**
```go
log, err := logger.NewProduction(
    logger.WithMetrics(logger.MetricsOptions{
        Enabled:      true,
        AutoRegister: false,
    }),
)

// Manual registration with custom registry
collectors := logger.MetricsCollectors()
for _, collector := range collectors {
    customRegistry.MustRegister(collector)
}
```

## Environment-Specific Configs

### Development Configuration

Optimized for local development:

```go
log, err := logger.NewDevelopment(
    logger.WithService("my-app-dev"),
    logger.WithLevel(logger.DebugLevel),
    logger.WithCaller(true),
    // Console enabled by default with human-readable format
)
```

Features:
- Human-readable console output
- Debug level logging
- No sampling
- Caller information enabled

### Production Configuration  

Optimized for production deployment:

```go
log, err := logger.NewProduction(
    logger.WithService("my-app"),
    logger.WithLevel(logger.InfoLevel),
    
    // Multiple outputs
    logger.WithFile(logger.FileSink{
        Path:       "/var/log/app.log",
        MaxSizeMB:  100,
        MaxBackups: 5,
        Compress:   true,
    }),
    logger.WithElastic(logger.ElasticSink{
        Addresses:     []string{"https://elasticsearch:9200"},
        Index:         "logs-%Y.%m.%d",
        FlushInterval: 2 * time.Second,
        APIKey:        "production-api-key",
        Retry: logger.Retry{
            Max:        3,
            BackoffMin: 100 * time.Millisecond,
            BackoffMax: 5 * time.Second,
        },
        DLQPath: "/var/log/failed-logs.dlq",
    }),
    
    // Sampling to control volume
    logger.WithSampling(logger.Sampling{
        Initial:    100,
        Thereafter: 100,
    }),
    
    // Observability
    logger.WithMetrics(logger.MetricsOptions{
        Enabled:      true,
        AutoRegister: true,
    }),
    logger.WithContext(logger.ContextKeys{
        RequestIDKey:    "request_id",
        UserIDKey:       "user_id",
        RequestIDHeader: "X-Request-ID", 
        UserIDHeader:    "X-User-ID",
    }),
    
    // Disable console in production
    logger.WithConsoleDisabled(),
)
```

### Staging Configuration

Balanced for testing with production-like behavior:

```go
log, err := logger.NewProduction(
    logger.WithService("my-app-staging"),
    logger.WithLevel(logger.DebugLevel), // More verbose for testing
    
    // File for local analysis
    logger.WithFile(logger.FileSink{
        Path: "/var/log/staging.log",
    }),
    
    // Elasticsearch for integration testing
    logger.WithElastic(logger.ElasticSink{
        Addresses: []string{"https://staging-es:9200"},
        Index:     "staging-logs-%Y.%m.%d", 
        APIKey:    "staging-api-key",
    }),
    
    // Metrics enabled but not sampled
    logger.WithMetrics(logger.MetricsOptions{
        Enabled:      true,
        AutoRegister: true,
    }),
    
    // Keep console for immediate feedback
    // logger.WithConsoleDisabled() - commented out
)
```

## JSON/YAML Configuration

### JSON Configuration Support

LoggerKit types support JSON marshaling/unmarshaling:

```json
{
  "service": "my-service",
  "env": "prod",
  "level": "info", 
  "enableCaller": true,
  "stacktraceAt": "error",
  "timeFormat": "2006-01-02T15:04:05.000Z",
  "sampling": {
    "initial": 100,
    "thereafter": 100
  },
  "disableConsole": true,
  "file": {
    "path": "/var/log/app.log",
    "maxSizeMB": 100,
    "maxBackups": 5,
    "compress": true
  },
  "elastic": {
    "addresses": ["https://elasticsearch:9200"],
    "index": "logs-%Y.%m.%d",
    "flushInterval": "2s",
    "apiKey": "your-api-key"
  },
  "metrics": {
    "enabled": true,
    "autoRegister": true
  }
}
```

### YAML Configuration Support

```yaml
service: my-service
env: prod
level: info
enableCaller: true
stacktraceAt: error
timeFormat: "2006-01-02T15:04:05.000Z"

sampling:
  initial: 100
  thereafter: 100

disableConsole: true

file:
  path: /var/log/app.log
  maxSizeMB: 100
  maxBackups: 5
  compress: true

elastic:
  addresses:
    - https://elasticsearch:9200
  index: "logs-%Y.%m.%d"  
  flushInterval: 2s
  apiKey: your-api-key

metrics:
  enabled: true
  autoRegister: true
```

### Loading Configuration

```go
// Load from JSON file
func loadConfig(path string) (*logger.Options, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    var opts logger.Options
    if err := json.Unmarshal(data, &opts); err != nil {
        return nil, err
    }
    
    return &opts, nil
}

// Use loaded configuration
opts, err := loadConfig("config.json")
if err != nil {
    return err
}

log, err := logger.NewWithOptions(*opts)
```

### Environment Variable Override

Combine configuration files with environment variables:

```go
func loadConfigWithOverrides() (*logger.Options, error) {
    // Load base configuration
    opts, err := loadConfig("config.json")
    if err != nil {
        return nil, err
    }
    
    // Environment-specific overrides
    if level := os.Getenv("LOG_LEVEL"); level != "" {
        if parsedLevel, err := logger.ParseLevel(level); err == nil {
            opts.Level = parsedLevel
        }
    }
    
    if service := os.Getenv("SERVICE_NAME"); service != "" {
        opts.Service = service
    }
    
    if esURL := os.Getenv("ELASTICSEARCH_URL"); esURL != "" && opts.Elastic != nil {
        opts.Elastic.Addresses = []string{esURL}
    }
    
    return opts, nil
}
```

## Complete Configuration Example

```go
package main

import (
    "context"
    "time"
    
    logger "github.com/HoangAnhNguyen269/loggerkit"
    _ "github.com/HoangAnhNguyen269/loggerkit/provider/zapx"
)

func main() {
    // Comprehensive production configuration
    log, err := logger.NewProduction(
        // Core settings
        logger.WithService("payment-processor"),
        logger.WithEnv(logger.EnvProd),
        logger.WithLevel(logger.InfoLevel),
        logger.WithCaller(true),
        logger.WithStacktrace(logger.ErrorLevel),
        logger.WithTimeFormat(time.RFC3339Nano),
        
        // Sampling for high-volume services
        logger.WithSampling(logger.Sampling{
            Initial:    200,  // First 200 logs per second
            Thereafter: 50,   // Then every 50th log
        }),
        
        // Multiple output sinks
        logger.WithConsoleDisabled(), // No console in production
        
        logger.WithFile(logger.FileSink{
            Path:         "/var/log/payments.log", 
            MaxSizeMB:    200,
            MaxAge:       14,  // 2 weeks retention
            MaxBackups:   20,
            LocalTime:    true,
            Compress:     true,
        }),
        
        logger.WithElastic(logger.ElasticSink{
            Addresses: []string{
                "https://es-1.company.com:9200",
                "https://es-2.company.com:9200",
                "https://es-3.company.com:9200",
            },
            Index:         "payment-logs-%Y.%m.%d",
            FlushInterval: 1 * time.Second,
            BulkSizeBytes: 5 * 1024 * 1024, // 5MB
            APIKey:        "production-es-key",
            
            Retry: logger.Retry{
                Max:        5,
                BackoffMin: 50 * time.Millisecond,
                BackoffMax: 10 * time.Second,
            },
            DLQPath: "/var/log/payments-failed.dlq",
        }),
        
        // Observability  
        logger.WithMetrics(logger.MetricsOptions{
            Enabled:      true,
            AutoRegister: true,
        }),
        
        // Request correlation
        logger.WithContext(logger.ContextKeys{
            RequestIDKey:    "request_id",
            UserIDKey:       "user_id",
            RequestIDHeader: "X-Request-ID",
            UserIDHeader:    "X-User-ID",
        }),
    )
    if err != nil {
        panic(err)
    }
    defer log.Close(context.Background())
    
    // Use the logger
    log.Info("Payment processor started",
        logger.F.String("version", "2.1.0"),
        logger.F.Int("workers", 10),
    )
}
```

This configuration provides:
- Structured JSON logs to Elasticsearch
- Local file backup with rotation
- Prometheus metrics collection  
- Request/trace correlation
- High availability with retry logic
- Dead letter queue for failed logs
- Sampling to control log volume
- Proper resource cleanup