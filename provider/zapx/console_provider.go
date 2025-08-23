package zapx

import (
	logger "github.com/HoangAnhNguyen269/loggerkit"
	"go.uber.org/zap/zapcore"
	"os"
)

type consoleProvider struct{}

func (consoleProvider) Provide(cfg *logger.Config, encCfg zapcore.EncoderConfig, lvl zapcore.Level) (zapcore.Core, error) {
	if !cfg.ConsoleEnabled {
		return nil, nil
	}

	var enc zapcore.Encoder
	if cfg.JSON {
		enc = zapcore.NewJSONEncoder(encCfg)
	} else {
		enc = zapcore.NewConsoleEncoder(encCfg)
	}

	return zapcore.NewCore(enc, zapcore.Lock(os.Stdout), lvl), nil
}
