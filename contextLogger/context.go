package contextLogger

import (
	"context"
	"net/http"
	"sync"

	logger "github.com/HoangAnhNguyen269/loggerkit"
	"github.com/HoangAnhNguyen269/loggerkit/provider/zapx"
	"go.opentelemetry.io/otel/trace"
)

type ctxKey struct{}

// WithLogger stores a Logger in the context
func WithLogger(ctx context.Context, log logger.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

// ContextWithLogger is an alias for WithLogger for API consistency
func ContextWithLogger(ctx context.Context, log logger.Logger) context.Context {
	return WithLogger(ctx, log)
}

// FromContext retrieves Logger from context, falls back to default if none found
func FromContext(ctx context.Context) logger.Logger {
	if l, ok := ctx.Value(ctxKey{}).(logger.Logger); ok && l != nil {
		return l.WithContext(ctx)
	}
	return getFallback().WithContext(ctx)
}

// --- Fallback (zapx) as lazy singleton ---
var (
	fbOnce   sync.Once
	fbMu     sync.RWMutex
	fbLogger logger.Logger // may be set by SetFallbackLogger; else lazy zapx default
)

// Allow app/tests to override fallback (e.g., nop logger in tests)
func SetFallbackLogger(l logger.Logger) {
	fbMu.Lock()
	fbLogger = l
	fbMu.Unlock()
}

func getFallback() logger.Logger {
	fbMu.RLock()
	l := fbLogger
	fbMu.RUnlock()
	if l != nil {
		return l
	}
	fbOnce.Do(func() {
		// create exactly once
		fbMu.Lock()
		fbLogger = zapx.NewDefaultLogger()
		fbMu.Unlock()
	})
	fbMu.RLock()
	l = fbLogger
	fbMu.RUnlock()
	return l
}

// Optional: let app close fallback on shutdown
func CloseFallback(ctx context.Context) error {
	fbMu.RLock()
	l := fbLogger
	fbMu.RUnlock()
	if l != nil {
		return l.Close(ctx)
	}
	return nil
}

// HTTPMiddleware creates middleware that extracts request/user IDs from headers and traces
func HTTPMiddleware(contextKeys logger.ContextKeys) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			if contextKeys.RequestIDKey != nil && contextKeys.RequestIDHeader != "" {
				if requestID := r.Header.Get(contextKeys.RequestIDHeader); requestID != "" {
					ctx = context.WithValue(ctx, contextKeys.RequestIDKey, requestID)
				}
			}

			if contextKeys.UserIDKey != nil && contextKeys.UserIDHeader != "" {
				if userID := r.Header.Get(contextKeys.UserIDHeader); userID != "" {
					ctx = context.WithValue(ctx, contextKeys.UserIDKey, userID)
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// DefaultHTTPMiddleware creates middleware with default header names
func DefaultHTTPMiddleware() func(http.Handler) http.Handler {
	contextKeys := logger.ContextKeys{
		RequestIDKey:    "request_id",
		UserIDKey:       "user_id",
		RequestIDHeader: "X-Request-ID",
		UserIDHeader:    "X-User-ID",
	}
	return HTTPMiddleware(contextKeys)
}

// ExtractTraceFields extracts trace information from context as fields
func ExtractTraceFields(ctx context.Context) []logger.Field {
	var fields []logger.Field

	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		fields = append(fields,
			logger.F.String("trace_id", sc.TraceID().String()),
			logger.F.String("span_id", sc.SpanID().String()),
		)
	}

	return fields
}

// ExtractRequestFields extracts request/user ID fields from context
func ExtractRequestFields(ctx context.Context, contextKeys logger.ContextKeys) []logger.Field {
	var fields []logger.Field

	if contextKeys.RequestIDKey != nil {
		if rid := ctx.Value(contextKeys.RequestIDKey); rid != nil {
			fields = append(fields, logger.F.Any("request_id", rid))
		}
	}

	if contextKeys.UserIDKey != nil {
		if uid := ctx.Value(contextKeys.UserIDKey); uid != nil {
			fields = append(fields, logger.F.Any("user_id", uid))
		}
	}

	return fields
}
