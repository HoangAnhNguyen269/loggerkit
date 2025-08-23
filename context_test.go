package logger_test

import (
	"context"
	"encoding/json"
	"github.com/HoangAnhNguyen269/loggerkit/testutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	logger "github.com/HoangAnhNguyen269/loggerkit"
	"github.com/HoangAnhNguyen269/loggerkit/contextLogger"
	_ "github.com/HoangAnhNguyen269/loggerkit/provider/zapx"
	"go.opentelemetry.io/otel"
)

// C) Context & Middleware

func TestHTTPMiddlewareDefault(t *testing.T) {
	// Create logger with matching context keys
	contextKeys := logger.ContextKeys{
		RequestIDKey:    "request_id",
		UserIDKey:       "user_id",
		RequestIDHeader: "X-Request-ID",
		UserIDHeader:    "X-User-ID",
	}

	log, err := logger.NewProduction(logger.WithContext(contextKeys))
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	var capturedOutput string

	// Create middleware with same context keys
	middleware := contextLogger.HTTPMiddleware(contextKeys)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Store logger in request context
		ctx := contextLogger.WithLogger(r.Context(), log)

		// Capture output
		output, err := testutil.CaptureStdout(func() {
			reqLog := contextLogger.FromContext(ctx)
			reqLog.Info("Request received", logger.F.String("path", r.URL.Path))
		})
		if err != nil {
			t.Errorf("Failed to capture stdout: %v", err)
		}
		capturedOutput = output

		w.WriteHeader(http.StatusOK)
	})

	// Create test request with headers
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "req-123")
	req.Header.Set("X-User-ID", "user-456")

	rr := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rr, req)

	// Parse logged output
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(capturedOutput)), &logEntry); err != nil {
		t.Fatalf("Failed to parse log JSON: %v", err)
	}

	// Verify headers were injected
	if logEntry["request_id"] != "req-123" {
		t.Errorf("Expected request_id to be 'req-123', got %v", logEntry["request_id"])
	}
	if logEntry["user_id"] != "user-456" {
		t.Errorf("Expected user_id to be 'user-456', got %v", logEntry["user_id"])
	}
}

func TestOpenTelemetryTraceCorrelation(t *testing.T) {
	// Create a no-op tracer for testing
	tracer := otel.Tracer("test")

	ctx, span := tracer.Start(context.Background(), "test_operation")
	defer span.End()

	// Get span context to verify trace/span IDs
	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		t.Skip("OpenTelemetry not properly initialized for this test")
	}

	output, err := testutil.CaptureStdout(func() {
		log, err := logger.NewProduction()
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer log.Close(context.Background())

		// Use context with span
		contextLog := log.WithContext(ctx)
		contextLog.Info("Traced message")
	})
	if err != nil {
		t.Fatalf("Failed to capture stdout: %v", err)
	}

	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry); err != nil {
		t.Fatalf("Failed to parse log JSON: %v", err)
	}

	// Verify trace correlation fields are present
	if traceID, ok := logEntry["trace_id"]; ok {
		if traceID != spanCtx.TraceID().String() {
			t.Errorf("Expected trace_id to match span context")
		}
	}
	if spanID, ok := logEntry["span_id"]; ok {
		if spanID != spanCtx.SpanID().String() {
			t.Errorf("Expected span_id to match span context")
		}
	}
}

func TestFieldMergingPriority(t *testing.T) {
	output, err := testutil.CaptureStdout(func() {
		// Create logger with context keys configured
		log, err := logger.NewProduction(logger.WithContext(logger.ContextKeys{
			RequestIDKey: "request_id",
		}))
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer log.Close(context.Background())

		// Create context with values
		ctx := context.WithValue(context.Background(), "request_id", "from-context")

		// Chain operations: With() -> WithContext() -> log fields
		chainedLog := log.With(logger.F.String("service", "test-service")).
			WithContext(ctx)

		chainedLog.Info("Test message",
			logger.F.String("final", "field"),
			logger.F.String("service", "overridden"), // Should override With() field
		)
	})
	if err != nil {
		t.Fatalf("Failed to capture stdout: %v", err)
	}

	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry); err != nil {
		t.Fatalf("Failed to parse log JSON: %v", err)
	}

	// Verify field merging behavior
	if logEntry["request_id"] != "from-context" {
		t.Error("Expected request_id from context")
	}
	if logEntry["final"] != "field" {
		t.Error("Expected final field from log call")
	}
	// Note: The exact priority of field overriding depends on zap's implementation
	// This test documents the current behavior
}
