package zapx

import (
	"fmt"
	logger "github.com/HoangAnhNguyen269/loggerkit"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
)

type fileProvider struct{}

func (fileProvider) Provide(cfg *logger.Config, encCfg zapcore.EncoderConfig, lvl zapcore.Level) (zapcore.Core, error) {
	fc := cfg.FileConfig
	if fc == nil {
		return nil, nil
	}

	if err := os.MkdirAll(filepath.Dir(fc.Filename), 0o755); err != nil {
		return nil, fmt.Errorf("cannot create log dir: %w", err)
	}

	writer := &lumberjack.Logger{
		Filename:   fc.Filename,
		MaxSize:    fc.MaxSize,
		MaxBackups: fc.MaxBackups,
		MaxAge:     fc.MaxAge,
		Compress:   fc.Compress,
	}

	var enc zapcore.Encoder
	if cfg.JSON {
		enc = zapcore.NewJSONEncoder(encCfg)
	} else {
		enc = zapcore.NewConsoleEncoder(encCfg)
	}

	return zapcore.NewCore(enc, zapcore.AddSync(writer), lvl), nil
}
