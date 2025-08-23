package zapx

import (
	"fmt"
	logger "github.com/HoangAnhNguyen269/loggerkit"
	"go.uber.org/zap/zapcore"
)

func ToZapLevel(l logger.Level) (zapcore.Level, error) {
	switch l {
	case logger.DebugLevel:
		return zapcore.DebugLevel, nil
	case logger.InfoLevel:
		return zapcore.InfoLevel, nil
	case logger.WarnLevel:
		return zapcore.WarnLevel, nil
	case logger.ErrorLevel:
		return zapcore.ErrorLevel, nil
	case "":
		return zapcore.InvalidLevel, nil
	default:
		return zapcore.InvalidLevel, fmt.Errorf("invalid level %q", l)
	}
}
