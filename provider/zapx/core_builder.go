package zapx

import (
	"fmt"
	"os"

	logger "github.com/HoangAnhNguyen269/loggerkit"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type coreBuilder struct {
	opts    logger.Options
	encCfg  zapcore.EncoderConfig
	lvl     zapcore.Level
	metrics *logger.Metrics
}

func (cb *coreBuilder) buildCores() ([]zapcore.Core, []func() error, error) {
	var cores []zapcore.Core
	var closers []func() error

	// Determine if we should add console based on environment
	shouldAddConsole := cb.opts.Env == "dev"

	// Add console core for development or if explicitly requested
	if shouldAddConsole {
		consoleCore := cb.buildConsoleCore()
		cores = append(cores, consoleCore)
	}

	// Add file core if configured
	if cb.opts.File != nil {
		fileCore, fileCloser, err := cb.buildFileCore()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build file core: %w", err)
		}
		cores = append(cores, fileCore)
		if fileCloser != nil {
			closers = append(closers, fileCloser)
		}
	}

	// Add Elasticsearch core if configured
	if cb.opts.Elastic != nil {
		esCore, esCloser, err := cb.buildElasticsearchCore()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build elasticsearch core: %w", err)
		}
		cores = append(cores, esCore)
		if esCloser != nil {
			closers = append(closers, esCloser)
		}
	}

	// If production and no explicit sinks configured, add JSON console output
	if cb.opts.Env == "prod" && len(cores) == 0 {
		cores = append(cores, cb.buildProductionConsoleCore())
	}

	return cores, closers, nil
}

func (cb *coreBuilder) buildConsoleCore() zapcore.Core {
	encoder := zapcore.NewConsoleEncoder(cb.encCfg)
	writer := &consoleWriter{}
	return zapcore.NewCore(encoder, zapcore.Lock(zapcore.AddSync(writer)), cb.lvl)
}

func (cb *coreBuilder) buildProductionConsoleCore() zapcore.Core {
	encoder := zapcore.NewJSONEncoder(cb.encCfg)
	writer := &consoleWriter{}
	return zapcore.NewCore(encoder, zapcore.Lock(zapcore.AddSync(writer)), cb.lvl)
}

func (cb *coreBuilder) buildFileCore() (zapcore.Core, func() error, error) {
	fileConfig := cb.opts.File

	// Create lumberjack logger for rotation
	lj := &lumberjack.Logger{
		Filename:   fileConfig.Path,
		MaxSize:    fileConfig.MaxSizeMB,
		MaxBackups: fileConfig.MaxBackups,
		MaxAge:     fileConfig.MaxAgeDays,
		Compress:   fileConfig.Compress,
	}

	// Create file writer with metrics if enabled
	var writer zapcore.WriteSyncer
	if cb.metrics != nil {
		writer = &fileWriter{
			Logger:  lj,
			metrics: cb.metrics,
		}
	} else {
		writer = zapcore.AddSync(lj)
	}

	encoder := zapcore.NewJSONEncoder(cb.encCfg)
	core := zapcore.NewCore(encoder, zapcore.Lock(writer), cb.lvl)

	// Closer function for lumberjack
	closer := func() error {
		return lj.Close()
	}

	return core, closer, nil
}

func (cb *coreBuilder) buildElasticsearchCore() (zapcore.Core, func() error, error) {
	esConfig := cb.opts.Elastic

	// Create the Elasticsearch bulk writer
	esWriter, err := newElasticsearchWriter(esConfig, cb.opts.Service, cb.metrics)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create elasticsearch writer: %w", err)
	}

	encoder := zapcore.NewJSONEncoder(cb.encCfg)
	core := zapcore.NewCore(encoder, zapcore.Lock(esWriter), cb.lvl)

	// Return the core and closer
	return core, esWriter.Close, nil
}

// consoleWriter writes to stdout with metrics support
type consoleWriter struct{}

func (cw *consoleWriter) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

// fileWriter wraps lumberjack.Logger with metrics support
type fileWriter struct {
	*lumberjack.Logger
	metrics *logger.Metrics
}

func (fw *fileWriter) Write(p []byte) (int, error) {
	n, err := fw.Logger.Write(p)
	if err != nil && fw.metrics != nil {
		fw.metrics.RecordLogDropped("file", "write_error")
	} else if fw.metrics != nil {
		fw.metrics.RecordLogWritten("info", "file") // We don't have level context here
	}
	return n, err
}

func (fw *fileWriter) Sync() error {
	// lumberjack doesn't need explicit sync, but we can close and reopen if needed
	return nil
}
