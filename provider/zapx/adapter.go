package zapx

import (
	"context"
	"fmt"
	logger "github.com/HoangAnhNguyen269/loggerkit"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

// Ensure zapAdapter implements Logger
var _ logger.Logger = (*zapAdapter)(nil)

// zapBuilder implements NewBuilder interface
type zapBuilder struct{}

var _ logger.NewBuilder = (*zapBuilder)(nil)

func init() {
	// Register the zapx builder as the default
	logger.SetBuilder(&zapBuilder{})
}

func (b *zapBuilder) NewWithOptions(opts logger.Options) (logger.Logger, error) {
	return NewWithOptions(opts)
}

type zapAdapter struct {
	zl             *zap.Logger
	closers        []func() error
	metrics        *logger.Metrics
	metricsEnabled bool
	contextKeys    logger.ContextKeys
	service        string
}

// NewWithOptions creates a new logger with the provided options
func NewWithOptions(opts logger.Options) (logger.Logger, error) {
	// Parse log level
	lvl, err := parseLevel(opts.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", opts.Level, err)
	}

	// Parse stacktrace level
	stackLvl, err := parseLevel(opts.StacktraceAt)
	if err != nil {
		return nil, fmt.Errorf("invalid stacktrace level %q: %w", opts.StacktraceAt, err)
	}

	// Create encoder config
	encCfg := createEncoderConfig(opts)

	// Initialize metrics if enabled
	var metrics *logger.Metrics
	if opts.Metrics.Enabled {
		metrics = logger.GetMetrics()
		if opts.Metrics.AutoRegister {
			if err := logger.AutoRegisterMetrics(); err != nil {
				return nil, fmt.Errorf("failed to auto-register metrics: %w", err)
			}
		}
	}

	// Build cores
	coreBuilder := &coreBuilder{
		opts:    opts,
		encCfg:  encCfg,
		lvl:     lvl,
		metrics: metrics,
	}

	cores, closers, err := coreBuilder.buildCores()
	if err != nil {
		return nil, fmt.Errorf("failed to build cores: %w", err)
	}

	// Create the core
	var core zapcore.Core
	if len(cores) == 0 {
		// Fallback to console if no cores configured
		core = zapcore.NewCore(
			createEncoder(encCfg, opts.Env == "prod"),
			zapcore.Lock(zapcore.AddSync(zapcore.AddSync(&consoleWriter{}))),
			lvl,
		)
	} else if len(cores) == 1 {
		core = cores[0]
	} else {
		core = zapcore.NewTee(cores...)
	}

	// Apply sampling if configured
	if opts.Sampling != nil {
		core = zapcore.NewSampler(core, time.Second, opts.Sampling.Initial, opts.Sampling.Thereafter)
	}

	// Create zap logger options
	zapOpts := []zap.Option{
		zap.AddCallerSkip(2),
	}

	if opts.EnableCaller {
		zapOpts = append(zapOpts, zap.AddCaller())
	}

	if stackLvl != zapcore.InvalidLevel {
		zapOpts = append(zapOpts, zap.AddStacktrace(stackLvl))
	}

	// Create the underlying zap logger
	zl := zap.New(core, zapOpts...)

	return &zapAdapter{
		zl:             zl,
		closers:        closers,
		metrics:        metrics,
		metricsEnabled: opts.Metrics.Enabled,
		contextKeys:    opts.Context,
		service:        opts.Service,
	}, nil
}

func createEncoderConfig(opts logger.Options) zapcore.EncoderConfig {
	timeEncoder := zapcore.ISO8601TimeEncoder
	if opts.TimeFormat != "" {
		if opts.TimeFormat == time.RFC3339Nano {
			timeEncoder = zapcore.RFC3339NanoTimeEncoder
		} else {
			timeEncoder = zapcore.TimeEncoderOfLayout(opts.TimeFormat)
		}
	}

	return zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     timeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

func createEncoder(encCfg zapcore.EncoderConfig, isProduction bool) zapcore.Encoder {
	if isProduction {
		return zapcore.NewJSONEncoder(encCfg)
	}
	return zapcore.NewConsoleEncoder(encCfg)
}

func parseLevel(level string) (zapcore.Level, error) {
	switch level {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("unknown level: %s", level)
	}
}

func (l *zapAdapter) Debug(msg string, fields ...logger.Field) {
	l.log(zapcore.DebugLevel, msg, fields...)
}

func (l *zapAdapter) Info(msg string, fields ...logger.Field) {
	l.log(zapcore.InfoLevel, msg, fields...)
}

func (l *zapAdapter) Warn(msg string, fields ...logger.Field) {
	l.log(zapcore.WarnLevel, msg, fields...)
}

func (l *zapAdapter) Error(msg string, fields ...logger.Field) {
	l.log(zapcore.ErrorLevel, msg, fields...)
}

func (l *zapAdapter) Log(level logger.Level, msg string, fields ...logger.Field) {
	l.log(toZapLevel(level), msg, fields...)
}

func (l *zapAdapter) With(fields ...logger.Field) logger.Logger {
	return &zapAdapter{
		zl:             l.zl.With(toZapFields(fields...)...),
		closers:        l.closers, // Share closers
		metrics:        l.metrics,
		metricsEnabled: l.metricsEnabled,
		contextKeys:    l.contextKeys,
		service:        l.service,
	}
}

func (l *zapAdapter) WithContext(ctx context.Context) logger.Logger {
	var fs []logger.Field

	// Extract request ID
	if l.contextKeys.RequestIDKey != nil {
		if rid := ctx.Value(l.contextKeys.RequestIDKey); rid != nil {
			fs = append(fs, logger.F.Any("request_id", rid))
		}
	}

	// Extract user ID
	if l.contextKeys.UserIDKey != nil {
		if uid := ctx.Value(l.contextKeys.UserIDKey); uid != nil {
			fs = append(fs, logger.F.Any("user_id", uid))
		}
	}

	// Extract OpenTelemetry trace information
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		fs = append(fs,
			logger.F.String("trace_id", sc.TraceID().String()),
			logger.F.String("span_id", sc.SpanID().String()),
		)
	}

	if len(fs) == 0 {
		return l
	}

	return l.With(fs...)
}

func (l *zapAdapter) Close(ctx context.Context) error {
	// First, sync the zap logger
	if err := l.zl.Sync(); err != nil {
		// Ignore known sync errors on non-seekable files
		if err.Error() != "sync /dev/stdout: invalid argument" &&
			err.Error() != "sync /dev/stderr: invalid argument" {
			return fmt.Errorf("failed to sync logger: %w", err)
		}
	}

	// Close all registered closers
	var lastErr error
	for _, closer := range l.closers {
		if err := closer(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

func (l *zapAdapter) log(level zapcore.Level, msg string, fields ...logger.Field) {
	zf := toZapFields(fields...)

	// Record metrics if enabled
	if l.metricsEnabled && l.metrics != nil {
		l.metrics.RecordLogWritten(level.String(), "zap")
	}

	switch level {
	case zapcore.DebugLevel:
		l.zl.Debug(msg, zf...)
	case zapcore.InfoLevel:
		l.zl.Info(msg, zf...)
	case zapcore.WarnLevel:
		l.zl.Warn(msg, zf...)
	case zapcore.ErrorLevel:
		l.zl.Error(msg, zf...)
	case zapcore.FatalLevel:
		l.zl.Fatal(msg, zf...)
	}
}

func toZapFields(fields ...logger.Field) []zap.Field {
	out := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		out = append(out, zap.Any(f.Key, f.Val))
	}
	return out
}

// Map logger.Level -> zapcore.Level (fallback: info)
func toZapLevel(lvl logger.Level) zapcore.Level {
	switch lvl {
	case logger.DebugLevel:
		return zapcore.DebugLevel
	case logger.InfoLevel:
		return zapcore.InfoLevel
	case logger.WarnLevel:
		return zapcore.WarnLevel
	case logger.ErrorLevel:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
