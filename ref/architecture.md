# LoggerKit Architecture Guide

This document explains the internal architecture of LoggerKit, focusing on the provider system, factory pattern, and extensibility mechanisms.

## Table of Contents

- [Overview](#overview)
- [Provider System](#provider-system)
- [Factory Architecture](#factory-architecture)
- [Import Cycle Elimination](#import-cycle-elimination)
- [Registry System](#registry-system)
- [Metrics Integration](#metrics-integration)
- [Extensibility](#extensibility)
- [Testing Architecture](#testing-architecture)

## Overview

LoggerKit uses a sophisticated factory-based architecture that provides:

- **Multiple Output Sinks**: Console, file, Elasticsearch, and extensible to custom sinks
- **Import Cycle Prevention**: Clean separation between interface and implementation
- **Plugin Architecture**: Easy addition of new output types
- **Automatic Metrics**: Transparent observability for all sinks
- **Resource Management**: Proper cleanup and graceful shutdown

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Public API    │    │   Provider       │    │   Factories     │
│                 │    │   System         │    │                 │
│  Logger         │◄───│                  │◄───│  Console        │
│  interface.go   │    │  zapx/           │    │  File           │
│                 │    │  adapter.go      │    │  Elasticsearch  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Provider System

### Core Components

1. **Logger Interface** (`interface.go`): Public API that business code uses
2. **ZapX Provider** (`provider/zapx/`): Zap-based implementation 
3. **Factory System** (`corefactories/`): Pluggable output sink creation
4. **Registry** (`registry.go`): Factory discovery and injection system

### Builder Pattern

The provider system uses a builder pattern to eliminate import cycles:

```go
// Interface definition (no provider imports)
type NewBuilder interface {
    NewWithOptions(opts Options) (Logger, error)
}

// Provider registration (self-registration)
func init() {
    logger.SetBuilder(&zapBuilder{})
}

// User code (explicit provider import)
import _ "github.com/HoangAnhNguyen269/loggerkit/provider/zapx"
```

**Benefits:**
- Main package doesn't import providers directly
- Providers self-register via `init()` functions
- Users control which providers are loaded
- No circular dependencies

## Factory Architecture

### CoreFactory Interface

All output sinks implement the same interface:

```go
type CoreFactory interface {
    Name() string                                    // "console", "file", "elasticsearch"
    Enabled(opts Options) bool                       // Configuration-based enablement
    Build(encCfg EncoderConfig, lvl Level, 
          metrics *Metrics, opts Options) 
          (Core, func() error, error)                // Core + cleanup function
}
```

### Implementation Examples

**Console Factory:**
```go
func (cf *ConsoleFactory) Enabled(opts Options) bool {
    return !opts.DisableConsole  // Default enabled unless disabled
}

func (cf *ConsoleFactory) Build(...) (Core, func() error, error) {
    writer := &consoleWriter{metrics: metrics}
    encoder := zapcore.NewConsoleEncoder(encCfg)  // Dev mode
    if opts.Env == EnvProd {
        encoder = zapcore.NewJSONEncoder(encCfg)  // Prod mode  
    }
    core := zapcore.NewCore(encoder, zapcore.AddSync(writer), lvl)
    return core, nil, nil  // No cleanup needed
}
```

**File Factory:**
```go
func (ff *FileFactory) Enabled(opts Options) bool {
    return opts.File != nil  // Enabled if File config provided
}

func (ff *FileFactory) Build(...) (Core, func() error, error) {
    rotator := &lumberjack.Logger{
        Filename:   opts.File.Path,
        MaxSize:    opts.File.MaxSizeMB,
        MaxBackups: opts.File.MaxBackups,
        MaxAge:     opts.File.MaxAge,
        Compress:   opts.File.Compress,
    }
    core := zapcore.NewCore(encoder, zapcore.AddSync(rotator), lvl)
    closer := func() error { return rotator.Close() }
    return core, closer, nil
}
```

### Factory Registration

Factories auto-register during package initialization:

```go
func init() {
    RegisterFactory(&ConsoleFactory{})
    RegisterFactory(&FileFactory{})  
    RegisterFactory(&ElasticFactory{})
}
```

## Import Cycle Elimination

### The Problem

Direct imports would create cycles:
```
main package → zapx provider → elasticsearch package → main package (for interfaces)
```

### The Solution

**Step 1: Interface Definition**
```go
// main/interface.go - No provider imports
type Logger interface {
    Info(msg string, fields ...Field)
    // ...
}

type NewBuilder interface {
    NewWithOptions(opts Options) (Logger, error)
}
```

**Step 2: Provider Implementation** 
```go
// zapx/adapter.go - Implements interfaces
type zapAdapter struct { /* ... */ }
func (z *zapAdapter) Info(msg string, fields ...Field) { /* ... */ }

type zapBuilder struct{}
func (zb *zapBuilder) NewWithOptions(opts Options) (Logger, error) {
    return NewWithOptions(opts)
}
```

**Step 3: Self-Registration**
```go
// zapx/adapter.go - Provider registers itself
func init() {
    logger.SetBuilder(&zapBuilder{})
}
```

**Step 4: User Import**
```go
// user code - Explicit provider selection
import (
    logger "github.com/HoangAnhNguyen269/loggerkit"
    _ "github.com/HoangAnhNguyen269/loggerkit/provider/zapx"
)
```

## Registry System

### Production Registry

Global factory registry for production use:

```go
var factories []CoreFactory

func RegisterFactory(factory CoreFactory) {
    mu.Lock()
    defer mu.Unlock()
    factories = append(factories, factory)
}

func Factories() []CoreFactory {
    mu.RLock()
    defer mu.RUnlock()
    result := make([]CoreFactory, len(factories))
    copy(result, factories)  // Thread-safe copy
    return result
}
```

### Test Registry

Dependency injection for testing:

```go
type Registry interface {
    All() []CoreFactory
}

// Global function pointer for registry selection
var getRegistry = func() Registry { return DefaultRegistry() }

// Test injection
func UseFactoryRegistry(r Registry) {
    getRegistry = func() Registry { return r }
}

// Usage in tests
func TestCustomFactory(t *testing.T) {
    testRegistry := &MockRegistry{factories: []CoreFactory{&MockFactory{}}}
    UseFactoryRegistry(testRegistry)
    defer UseFactoryRegistry(DefaultRegistry())
    
    // Test with isolated factories
}
```

## Metrics Integration

### Automatic Metrics Wrapping

Every core gets automatic metrics via decorator pattern:

```go
type metricsCore struct {
    inner   zapcore.Core
    sink    string        // "console", "file", "elasticsearch"
    metrics *Metrics
}

func (m *metricsCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
    err := m.inner.Write(ent, fields)
    if err == nil {
        m.metrics.RecordLogWritten(ent.Level.String(), m.sink)
    } else {
        m.metrics.RecordLogDropped(m.sink, "write_error") 
    }
    return err
}
```

### Core Builder Integration

The core builder automatically wraps every factory output:

```go
func (cb *coreBuilder) buildCores() ([]zapcore.Core, []func() error, error) {
    for _, factory := range getRegistry().All() {
        if !factory.Enabled(cb.opts) {
            continue
        }
        
        // Build the core
        core, closer, err := factory.Build(cb.encCfg, cb.lvl, cb.metrics, cb.opts)
        if err != nil {
            return nil, nil, err
        }
        
        // Automatic metrics wrapping
        core = NewMetricsCore(core, factory.Name(), cb.metrics)
        
        cores = append(cores, core)
        if closer != nil {
            closers = append(closers, closer)
        }
    }
    return cores, closers, nil
}
```

**Benefits:**
- **Zero Configuration**: Metrics happen automatically
- **Per-Sink Tracking**: Separate metrics for each output type
- **Transparent**: Factories don't need metrics awareness
- **Consistent**: Same metrics pattern across all sinks

## Extensibility  

### Adding New Factories

Create a new output type by implementing `CoreFactory`:

```go
// Custom S3 output factory
type S3Factory struct{}

func (s3 *S3Factory) Name() string {
    return "s3"
}

func (s3 *S3Factory) Enabled(opts Options) bool {
    return opts.S3 != nil  // Enable if S3 config provided
}

func (s3 *S3Factory) Build(encCfg zapcore.EncoderConfig, lvl zapcore.Level,
                          metrics *logger.Metrics, opts logger.Options) 
                          (zapcore.Core, func() error, error) {
    
    // Create S3 writer
    s3Writer := &S3Writer{
        bucket: opts.S3.Bucket,
        prefix: opts.S3.Prefix,
        client: s3.New(session.Must(session.NewSession())),
    }
    
    // Create core
    encoder := zapcore.NewJSONEncoder(encCfg)
    core := zapcore.NewCore(encoder, zapcore.AddSync(s3Writer), lvl)
    
    // Cleanup function
    closer := func() error { return s3Writer.Close() }
    
    return core, closer, nil
}

// Auto-register
func init() {
    RegisterFactory(&S3Factory{})
}
```

### Configuration Extension

Add new sink configuration to Options:

```go
type Options struct {
    // ... existing fields
    S3 *S3Sink  // New sink configuration
}

type S3Sink struct {
    Bucket      string
    Prefix      string
    Region      string
    Credentials aws.Config
}

// Functional option
func WithS3(sink S3Sink) Option {
    return func(o *Options) {
        o.S3 = &sink
    }
}
```

### Usage

```go
log, err := logger.NewProduction(
    logger.WithS3(logger.S3Sink{
        Bucket: "my-logs",
        Prefix: "app-logs/",
        Region: "us-east-1",
    }),
)
```

The new factory will:
- Auto-register during package initialization
- Be discovered by the registry system  
- Get automatic metrics wrapping
- Participate in graceful shutdown
- Work with existing configuration patterns

## Testing Architecture

### Factory Isolation

Test individual factories without affecting others:

```go
func TestS3Factory(t *testing.T) {
    // Create isolated registry
    testRegistry := &MockRegistry{
        factories: []CoreFactory{&S3Factory{}},
    }
    
    // Inject test registry
    UseFactoryRegistry(testRegistry) 
    defer UseFactoryRegistry(DefaultRegistry())
    
    // Test only S3 factory
    log, err := logger.NewProduction(
        logger.WithS3(logger.S3Sink{Bucket: "test"}),
    )
    
    // Verify S3-specific behavior
}
```

### Mock Factories

Create mock implementations for testing:

```go
type MockFactory struct {
    name         string
    enabled      bool
    buildErr     error
    writeErr     error
    writeCalls   int
}

func (mf *MockFactory) Name() string { return mf.name }
func (mf *MockFactory) Enabled(opts Options) bool { return mf.enabled }

func (mf *MockFactory) Build(...) (zapcore.Core, func() error, error) {
    if mf.buildErr != nil {
        return nil, nil, mf.buildErr
    }
    
    mockCore := &MockCore{factory: mf}
    return mockCore, nil, nil
}

type MockCore struct {
    factory *MockFactory
}

func (mc *MockCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
    mc.factory.writeCalls++
    return mc.factory.writeErr
}
```

### Integration Testing

Test complete logger behavior with multiple factories:

```go
func TestMultipleSinks(t *testing.T) {
    tempFile := "/tmp/test.log"
    
    log, err := logger.NewProduction(
        logger.WithFile(logger.FileSink{Path: tempFile}),
        logger.WithElastic(logger.ElasticSink{
            Addresses: []string{"http://localhost:9200"},
        }),
    )
    
    // Test that messages go to both sinks
    log.Info("Test message")
    
    // Verify file output
    content, _ := os.ReadFile(tempFile)
    assert.Contains(t, string(content), "Test message")
    
    // Verify Elasticsearch output (via mock)
}
```

## Summary

LoggerKit's architecture achieves several key goals:

1. **Modularity**: Clean separation between public API and implementation
2. **Extensibility**: Easy addition of new output types via factory pattern  
3. **Testability**: Dependency injection allows isolated testing
4. **Observability**: Automatic metrics collection for all sinks
5. **Resource Management**: Consistent cleanup patterns
6. **Performance**: Efficient core building and minimal overhead

The factory-based approach with automatic registration provides a plugin-like architecture that's both powerful for advanced users and simple for common use cases.