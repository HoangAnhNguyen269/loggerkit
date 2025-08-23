package logger

import (
	"context"
)

// Logger interface chỉ expose những gì business code cần
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Log(level Level, msg string, fields ...Field)
	With(fields ...Field) Logger
	WithContext(ctx context.Context) Logger
	Close(ctx context.Context) error
}
