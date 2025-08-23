package corefactories

import (
	"os"

	logger "github.com/HoangAnhNguyen269/loggerkit"
	"go.uber.org/zap/zapcore"
)

// ConsoleFactory creates console-based cores for logging output
type ConsoleFactory struct{}

func init() {
	RegisterFactory(&ConsoleFactory{})
}

// Name returns the unique name of this factory
func (cf *ConsoleFactory) Name() string {
	return "console"
}

// Enabled determines if console logging should be enabled based on options
func (cf *ConsoleFactory) Enabled(opts logger.Options) bool {
	// Default ON unless explicitly disabled
	if opts.DisableConsole {
		return false
	}
	return true
}

// Build creates a console core
func (cf *ConsoleFactory) Build(encCfg zapcore.EncoderConfig, lvl zapcore.Level, metrics *logger.Metrics, opts logger.Options) (zapcore.Core, func() error, error) {
	writer := &consoleWriter{
		metrics: metrics,
	}

	var encoder zapcore.Encoder
	if opts.Env == logger.EnvDev {
		// Development: use console encoder for human-readable output
		encoder = zapcore.NewConsoleEncoder(encCfg)
	} else {
		// Production: use JSON encoder for structured output
		encoder = zapcore.NewJSONEncoder(encCfg)
	}

	core := zapcore.NewCore(encoder, zapcore.Lock(zapcore.AddSync(writer)), lvl)

	// Console doesn't need a closer
	return core, nil, nil
}

// consoleWriter writes to stdout with optional metrics support
type consoleWriter struct {
	metrics *logger.Metrics
}

func (cw *consoleWriter) Write(p []byte) (int, error) {
	n, err := os.Stdout.Write(p)
	if err != nil && cw.metrics != nil {
		cw.metrics.RecordLogDropped("console", "write_error")
	}
	return n, err
}
