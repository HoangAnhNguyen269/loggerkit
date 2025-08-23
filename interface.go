package logger

import (
	"context"
	"time"
)

// Field là cặp key/value cho structured logging
type Field struct {
	Key string
	Val any
}

// Legacy field helpers for backward compatibility
func String(key, val string) Field    { return Field{key, val} }
func Int(key string, val int) Field   { return Field{key, val} }
func Bool(key string, val bool) Field { return Field{key, val} }
func Any(key string, val any) Field {
	return Field{key, val}
}
func Duration(key string, val time.Duration) Field {
	return Field{key, val}
}
func Time(key string, val time.Time) Field {
	return Field{key, val}
}

// F provides field helpers using the new structure
var F = struct {
	String   func(k, v string) Field
	Int      func(k string, v int) Field
	Bool     func(k string, v bool) Field
	Err      func(err error) Field
	Duration func(k string, v time.Duration) Field
	Any      func(k string, v any) Field
}{
	String:   func(k, v string) Field { return Field{k, v} },
	Int:      func(k string, v int) Field { return Field{k, v} },
	Bool:     func(k string, v bool) Field { return Field{k, v} },
	Err:      func(err error) Field { return Field{"error", err} },
	Duration: func(k string, v time.Duration) Field { return Field{k, v} },
	Any:      func(k string, v any) Field { return Field{k, v} },
}

// Logger interface chỉ expose những gì business code cần
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	With(fields ...Field) Logger
	WithContext(ctx context.Context) Logger
	Close(ctx context.Context) error
}
