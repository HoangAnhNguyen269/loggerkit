package corefactories

import (
	logger "github.com/HoangAnhNguyen269/loggerkit"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// FileFactory creates file-based cores for logging output
type FileFactory struct{}

func init() {
	RegisterFactory(&FileFactory{})
}

// Name returns the unique name of this factory
func (ff *FileFactory) Name() string {
	return "file"
}

// Enabled determines if file logging should be enabled based on options
func (ff *FileFactory) Enabled(opts logger.Options) bool {
	return opts.File != nil
}

// Build creates a file core with rotation support
func (ff *FileFactory) Build(encCfg zapcore.EncoderConfig, lvl zapcore.Level, metrics *logger.Metrics, opts logger.Options) (zapcore.Core, func() error, error) {
	fileConfig := opts.File

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
	if metrics != nil {
		writer = &fileWriter{
			Logger:  lj,
			metrics: metrics,
		}
	} else {
		writer = zapcore.AddSync(lj)
	}

	encoder := zapcore.NewJSONEncoder(encCfg)
	core := zapcore.NewCore(encoder, zapcore.Lock(writer), lvl)

	// Closer function for lumberjack
	closer := func() error {
		return lj.Close()
	}

	return core, closer, nil
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
	}
	return n, err
}

func (fw *fileWriter) Sync() error {
	// lumberjack doesn't need explicit sync, but we can close and reopen if needed
	return nil
}
