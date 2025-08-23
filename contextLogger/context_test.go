package contextLogger_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	logger "github.com/HoangAnhNguyen269/loggerkit"
	"github.com/HoangAnhNguyen269/loggerkit/contextLogger"
	_ "github.com/HoangAnhNguyen269/loggerkit/provider/zapx" // Import to register the builder
)

func TestWithLogger(t *testing.T) {
	log, err := logger.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	ctx := contextLogger.WithLogger(context.Background(), log)
	retrievedLog := contextLogger.FromContext(ctx)

	if retrievedLog == nil {
		t.Error("Expected logger from context, got nil")
	}

	// Test logging with retrieved logger
	retrievedLog.Info("Test message from context")
}

func TestContextWithLogger(t *testing.T) {
	log, err := logger.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	// Test alias function
	ctx := contextLogger.ContextWithLogger(context.Background(), log)
	retrievedLog := contextLogger.FromContext(ctx)

	if retrievedLog == nil {
		t.Error("Expected logger from context, got nil")
	}
}

func TestFromContextFallback(t *testing.T) {
	// Test fallback behavior when no logger in context
	ctx := context.Background()
	retrievedLog := contextLogger.FromContext(ctx)

	if retrievedLog == nil {
		t.Error("Expected fallback logger, got nil")
	}

	// Should not panic
	retrievedLog.Info("Fallback logger test")
}

func TestHTTPMiddleware(t *testing.T) {
	log, err := logger.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	contextKeys := logger.ContextKeys{
		RequestIDKey:    "request_id",
		UserIDKey:       "user_id",
		RequestIDHeader: "X-Request-ID",
		UserIDHeader:    "X-User-ID",
	}

	middleware := contextLogger.HTTPMiddleware(contextKeys)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Store logger in context for this request
		ctx := contextLogger.WithLogger(r.Context(), log)
		r = r.WithContext(ctx)

		// Get logger from context - should have request/user IDs
		contextLog := contextLogger.FromContext(r.Context())
		contextLog.Info("Request processed")

		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "test-request-123")
	req.Header.Set("X-User-ID", "user-456")

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}
}

func TestDefaultHTTPMiddleware(t *testing.T) {
	log, err := logger.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	middleware := contextLogger.DefaultHTTPMiddleware()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := contextLogger.WithLogger(r.Context(), log)
		r = r.WithContext(ctx)

		contextLog := contextLogger.FromContext(r.Context())
		contextLog.Info("Default middleware test")

		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "default-request-123")
	req.Header.Set("X-User-ID", "default-user-456")

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}
}

func TestExtractTraceFields(t *testing.T) {
	ctx := context.Background()

	// Test with empty context (no trace)
	fields := contextLogger.ExtractTraceFields(ctx)
	if len(fields) != 0 {
		t.Errorf("Expected no fields for empty context, got %d", len(fields))
	}

	// Note: Testing with actual OpenTelemetry traces would require more setup
	// This test verifies the function doesn't panic with empty context
}

func TestExtractRequestFields(t *testing.T) {
	contextKeys := logger.ContextKeys{
		RequestIDKey: "request_id",
		UserIDKey:    "user_id",
	}

	// Test with empty context
	ctx := context.Background()
	fields := contextLogger.ExtractRequestFields(ctx, contextKeys)
	if len(fields) != 0 {
		t.Errorf("Expected no fields for empty context, got %d", len(fields))
	}

	// Test with context values
	ctx = context.WithValue(ctx, "request_id", "test-request-789")
	ctx = context.WithValue(ctx, "user_id", "user-999")

	fields = contextLogger.ExtractRequestFields(ctx, contextKeys)
	if len(fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(fields))
	}
}
