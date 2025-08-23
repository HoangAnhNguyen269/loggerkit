# LoggerKit Observability Guide

This guide covers LoggerKit's production monitoring and observability features, including Prometheus metrics, distributed tracing, and context correlation.

## Table of Contents

- [Prometheus Metrics](#prometheus-metrics)
- [Metrics Configuration](#metrics-configuration)
- [Context Correlation](#context-correlation)
- [HTTP Middleware](#http-middleware)
- [Distributed Tracing](#distributed-tracing)
- [Production Setup](#production-setup)

## Prometheus Metrics

LoggerKit provides comprehensive production metrics through Prometheus integration.

### Available Metrics

**1. Log Write Tracking**
```
logs_written_total{level, sink}
```
- **Type**: Counter
- **Labels**: 
  - `level`: debug, info, warn, error
  - `sink`: console, file, elasticsearch
- **Purpose**: Track successful log writes per level and output sink

**2. Log Drop Tracking**
```
logs_dropped_total{sink, reason}
```
- **Type**: Counter  
- **Labels**:
  - `sink`: console, file, elasticsearch
  - `reason`: write_error, queue_full, etc.
- **Purpose**: Monitor log delivery failures

**3. Elasticsearch Retry Tracking**
```
es_bulk_retries_total{reason}
```
- **Type**: Counter
- **Labels**: `reason`: timeout, network_error, server_error
- **Purpose**: Track Elasticsearch bulk operation retry attempts

**4. Elasticsearch Queue Depth**
```
es_queue_depth{service}
```
- **Type**: Gauge
- **Labels**: `service`: service name from configuration
- **Purpose**: Monitor Elasticsearch bulk queue backlog

**5. Elasticsearch Latency**
```
es_bulk_latency_seconds{operation, status}
```
- **Type**: Histogram
- **Labels**:
  - `operation`: bulk_index, bulk_create
  - `status`: success, failure, retry
- **Purpose**: Track Elasticsearch operation performance

### Metrics Collection

Metrics are automatically collected through the `MetricsCore` wrapper:

```go
// Automatic collection on every log write
log.Info("User action") // Increments logs_written_total{level="info", sink="console"}

// Automatic collection on write failures  
// Network failure -> Increments logs_dropped_total{sink="elasticsearch", reason="write_error"}
```

## Metrics Configuration

### Auto Registration

Simplest approach - registers with Prometheus default registry:

```go
log, err := logger.NewProduction(
    logger.WithMetrics(logger.MetricsOptions{
        Enabled:      true,
        AutoRegister: true, // Registers with prometheus.DefaultRegisterer
    }),
)

// Metrics available at standard /metrics endpoint
http.Handle("/metrics", promhttp.Handler())
```

### Manual Registration

For custom registry management:

```go
// Create custom registry
registry := prometheus.NewRegistry()

// Configure logger without auto-registration
log, err := logger.NewProduction(
    logger.WithMetrics(logger.MetricsOptions{
        Enabled:      true,
        AutoRegister: false,
    }),
)

// Manually register collectors
collectors := logger.MetricsCollectors()
for _, collector := range collectors {
    registry.MustRegister(collector)
}

// Use custom registry
http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
```

### Metrics-Only Mode

Enable metrics without specific outputs:

```go
log, err := logger.NewProduction(
    logger.WithMetrics(logger.MetricsOptions{
        Enabled:      true,
        AutoRegister: true,
    }),
    logger.WithConsoleDisabled(), // No console output
    // Metrics still collected for all configured sinks
)
```

## Context Correlation

LoggerKit provides automatic correlation of logs with distributed tracing and request context.

### Context Keys Configuration

Define how context values are extracted:

```go
contextKeys := logger.ContextKeys{
    RequestIDKey:    "request_id",    // Context value key
    UserIDKey:       "user_id",       // Context value key  
    RequestIDHeader: "X-Request-ID",  // HTTP header name
    UserIDHeader:    "X-User-ID",     // HTTP header name
}

log, err := logger.NewProduction(
    logger.WithContext(contextKeys),
)
```

### Automatic Context Extraction

The logger automatically extracts correlation information:

```go
// Context contains request ID and user ID
ctx := context.WithValue(context.Background(), "request_id", "req-12345")
ctx = context.WithValue(ctx, "user_id", "user-67890")

// Automatic extraction when logging
ctxLog := log.WithContext(ctx)
ctxLog.Info("Processing request")

// Output includes:
// {"msg":"Processing request","request_id":"req-12345","user_id":"user-67890"}
```

### Manual Context Fields

Add context fields explicitly:

```go
// Add correlation fields manually
log.With(
    logger.F.String("request_id", requestID),
    logger.F.String("user_id", userID),
).Info("Manual correlation")

// Chain context for request-scoped logging
requestLog := log.With(
    logger.F.String("request_id", requestID),
    logger.F.String("operation", "user_authentication"),
)

requestLog.Info("Auth started")
requestLog.Error("Auth failed", logger.F.Err(err))
```

## HTTP Middleware

LoggerKit provides HTTP middleware for automatic context extraction and logger injection.

### Default Middleware

Uses standard header names:

```go
import "github.com/HoangAnhNguyen269/loggerkit/contextLogger"

// Extracts from X-Request-ID and X-User-ID headers
middleware := contextLogger.DefaultHTTPMiddleware()

// Gin integration
app.Use(middleware)

// Standard HTTP integration  
http.Handle("/api", middleware(apiHandler))
```

### Custom Middleware

Configure custom header names:

```go
contextKeys := logger.ContextKeys{
    RequestIDKey:    "req_id",
    UserIDKey:       "user_id", 
    RequestIDHeader: "Request-ID",    // Custom header
    UserIDHeader:    "User-ID",       // Custom header
}

middleware := contextLogger.HTTPMiddleware(contextKeys)
app.Use(middleware)
```

### Handler Integration

Use context-aware logging in handlers:

```go
func userHandler(w http.ResponseWriter, r *http.Request) {
    // Store logger in request context
    ctx := contextLogger.WithLogger(r.Context(), log)
    
    // Get context-aware logger
    reqLog := contextLogger.FromContext(ctx)
    
    // All logs automatically include request_id, user_id, trace_id
    reqLog.Info("Processing user request")
    
    if err := processUser(); err != nil {
        reqLog.Error("User processing failed", logger.F.Err(err))
        http.Error(w, "Internal error", 500)
        return
    }
    
    reqLog.Info("User request completed successfully")
}
```

## Distributed Tracing

### OpenTelemetry Integration

Automatic trace correlation with OpenTelemetry:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func businessLogic(ctx context.Context) {
    // Start span
    tracer := otel.Tracer("my-service")
    ctx, span := tracer.Start(ctx, "business_operation")
    defer span.End()
    
    // Logger automatically extracts trace_id and span_id
    ctxLog := log.WithContext(ctx)
    ctxLog.Info("Business operation started")
    
    // Output includes:
    // {"msg":"Business operation started","trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"00f067aa0ba902b7"}
}
```

### Trace Field Extraction

Manual trace field extraction:

```go
import "github.com/HoangAnhNguyen269/loggerkit/contextLogger"

func handleRequest(ctx context.Context) {
    // Extract trace fields manually
    traceFields := contextLogger.ExtractTraceFields(ctx)
    
    // Add to logger
    traceLog := log.With(traceFields...)
    traceLog.Info("Manual trace correlation")
}
```

### Cross-Service Correlation

Propagate trace context across services:

```go
// Service A - create outbound request
req, _ := http.NewRequestWithContext(ctx, "POST", "http://service-b/api", body)

// OpenTelemetry automatically propagates trace context in headers
resp, err := client.Do(req)

// Service B - receive request with trace context
func handler(w http.ResponseWriter, r *http.Request) {
    // Extract trace context from headers (automatic with OTel)
    ctx := r.Context()
    
    // Logger picks up trace context automatically
    log.WithContext(ctx).Info("Received cross-service request")
    // Same trace_id appears in both services
}
```

## Production Setup

### Complete Observability Stack

```go
package main

import (
    "context"
    "net/http"
    "time"
    
    logger "github.com/HoangAnhNguyen269/loggerkit"
    _ "github.com/HoangAnhNguyen269/loggerkit/provider/zapx"
    "github.com/HoangAnhNguyen269/loggerkit/contextLogger"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
    // Production logger with full observability
    log, err := logger.NewProduction(
        logger.WithService("my-service"),
        logger.WithLevel(logger.InfoLevel),
        
        // Elasticsearch for log aggregation
        logger.WithElastic(logger.ElasticSink{
            Addresses:     []string{"http://elasticsearch:9200"},
            Index:         "logs-%Y.%m.%d",
            FlushInterval: 2 * time.Second,
            BulkSizeBytes: 1024 * 1024, // 1MB batches
            APIKey:        "your-api-key",
            Retry: logger.Retry{
                Max:        3,
                BackoffMin: 100 * time.Millisecond,
                BackoffMax: 5 * time.Second,
            },
            DLQPath: "/var/log/failed-logs.log",
        }),
        
        // File backup
        logger.WithFile(logger.FileSink{
            Path:       "/var/log/app.log", 
            MaxSizeMB:  100,
            MaxBackups: 5,
            Compress:   true,
        }),
        
        // Prometheus metrics
        logger.WithMetrics(logger.MetricsOptions{
            Enabled:      true,
            AutoRegister: true,
        }),
        
        // Context correlation
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

    // HTTP middleware for automatic correlation
    middleware := contextLogger.DefaultHTTPMiddleware()
    
    // Application routes
    http.Handle("/api/", middleware(http.HandlerFunc(apiHandler)))
    
    // Metrics endpoint
    http.Handle("/metrics", promhttp.Handler())
    
    // Health check with metrics
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        log.WithContext(r.Context()).Info("Health check performed")
        w.WriteHeader(200)
        w.Write([]byte("OK"))
    })
    
    log.Info("Server starting", logger.F.Int("port", 8080))
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Error("Server failed", logger.F.Err(err))
    }
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
    // Get context-aware logger
    ctx := r.Context()
    reqLog := contextLogger.FromContext(ctx)
    
    // All logs include trace_id, span_id, request_id, user_id automatically
    reqLog.Info("API request started")
    
    // Business logic
    if err := processRequest(); err != nil {
        reqLog.Error("Request processing failed", 
            logger.F.Err(err),
            logger.F.String("endpoint", r.URL.Path),
        )
        http.Error(w, "Internal error", 500)
        return
    }
    
    reqLog.Info("API request completed successfully")
    w.WriteHeader(200)
    w.Write([]byte("Success"))
}
```

### Monitoring Dashboard

Key metrics to monitor in your dashboard:

**Log Volume and Health:**
- `rate(logs_written_total[5m])` - Log write rate per sink
- `rate(logs_dropped_total[5m])` - Log drop rate (should be near zero)

**Elasticsearch Performance:**  
- `histogram_quantile(0.95, es_bulk_latency_seconds_bucket)` - 95th percentile latency
- `rate(es_bulk_retries_total[5m])` - Retry rate (indicates ES issues)
- `es_queue_depth` - Queue backlog (should be low)

**Application Correlation:**
- Filter logs by `trace_id` for request tracing
- Group errors by `request_id` for debugging
- Aggregate metrics by `service` for multi-service monitoring

### Alerting Rules

```yaml
# Prometheus alerting rules
groups:
  - name: loggerkit
    rules:
      - alert: LogDropsHigh
        expr: rate(logs_dropped_total[5m]) > 10
        for: 2m
        annotations:
          summary: "High log drop rate detected"
          
      - alert: ElasticsearchQueueHigh  
        expr: es_queue_depth > 1000
        for: 1m
        annotations:
          summary: "Elasticsearch queue backlog is high"
          
      - alert: ElasticsearchLatencyHigh
        expr: histogram_quantile(0.95, es_bulk_latency_seconds_bucket) > 5
        for: 2m
        annotations:
          summary: "Elasticsearch bulk operations are slow"
```

This observability setup provides comprehensive monitoring, correlation, and alerting for production systems using LoggerKit.