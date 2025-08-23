package logger_test

import (
	"context"
	"testing"
	"time"

	logger "github.com/HoangAnhNguyen269/loggerkit"
	"github.com/HoangAnhNguyen269/loggerkit/contextLogger"
	_ "github.com/HoangAnhNguyen269/loggerkit/provider/zapx"
)

// TestFullIntegration tests the complete logger lifecycle and functionality
func TestFullIntegration(t *testing.T) {
	// Test development logger
	devLog, err := logger.NewDevelopment(
		logger.WithService("integration-test-dev"),
		logger.WithLevel("debug"),
	)
	if err != nil {
		t.Fatalf("Failed to create development logger: %v", err)
	}
	defer devLog.Close(context.Background())

	// Test production logger with multiple features
	prodLog, err := logger.NewProduction(
		logger.WithService("integration-test-prod"),
		logger.WithLevel("info"),
		logger.WithSampling(logger.Sampling{Initial: 5, Thereafter: 10}),
		logger.WithFile(logger.FileSink{
			Path:       "/tmp/integration_test.log",
			MaxSizeMB:  1,
			MaxBackups: 2,
			MaxAgeDays: 1,
			Compress:   true,
		}),
		logger.WithMetrics(logger.MetricsOptions{
			Enabled:      true,
			AutoRegister: false, // Don't pollute global registry in tests
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create production logger: %v", err)
	}
	defer prodLog.Close(context.Background())

	// Test field helpers
	devLog.Debug("Development debug message",
		logger.F.String("component", "integration-test"),
		logger.F.Int("test_id", 1),
		logger.F.Bool("debug_mode", true),
	)

	prodLog.Info("Production info message",
		logger.F.String("component", "integration-test"),
		logger.F.Duration("startup_time", time.Millisecond*500),
		logger.F.Any("config", map[string]interface{}{
			"sampling": true,
			"metrics":  true,
		}),
	)

	// Test context integration
	ctx := context.Background()
	ctx = context.WithValue(ctx, "request_id", "integration-test-request")
	ctx = context.WithValue(ctx, "user_id", "test-user-123")

	// Store logger in context
	ctx = contextLogger.WithLogger(ctx, prodLog)

	// Retrieve and use context logger
	ctxLog := contextLogger.FromContext(ctx)
	ctxLog.Info("Context logger test")

	// Test chaining
	serviceLog := prodLog.With(
		logger.F.String("service", "test-service"),
		logger.F.String("version", "1.0.0"),
	)

	requestLog := serviceLog.With(
		logger.F.String("request_id", "req-xyz"),
		logger.F.String("user_id", "usr-abc"),
	)

	requestLog.Info("Chained logger test")

	// Test error logging
	testErr := &testError{msg: "integration test error"}
	requestLog.Error("Test error occurred", logger.F.Err(testErr))

	// Test all log levels
	requestLog.Debug("Debug level test")
	requestLog.Info("Info level test")
	requestLog.Warn("Warn level test")
	requestLog.Error("Error level test")

	// Test legacy compatibility
	oldCfg := &logger.Config{
		Level:          logger.InfoLevel,
		JSON:           true,
		ConsoleEnabled: true,
	}
	legacyLog := logger.MustNew(oldCfg)
	defer legacyLog.Close(context.Background())

	legacyLog.Info("Legacy config test",
		logger.String("legacy", "true"),            // Old style field helper
		logger.F.String("new_style", "also works"), // New style field helper
	)

	// Verify metrics were created
	collectors := logger.MetricsCollectors()
	if len(collectors) != 5 {
		t.Errorf("Expected 5 metric collectors, got %d", len(collectors))
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// TestBackwardCompatibility ensures existing v1 code still works
func TestBackwardCompatibility(t *testing.T) {
	// This simulates existing v1 code with minimal changes
	log := logger.MustNew(logger.DefaultConfig())
	defer log.Close(context.Background()) // Only required change

	// All existing v1 APIs should still work
	log.Info("Backward compatibility test")
	log.Debug("Debug message", logger.String("key", "value"))
	log.Warn("Warning message", logger.Int("count", 42))
	log.Error("Error message", logger.Bool("flag", true))

	// With chaining
	userLog := log.With(logger.String("user_id", "123"))
	userLog.Info("User action")

	// Context usage
	ctx := context.WithValue(context.Background(), "request_id", "req-123")
	ctxLog := log.WithContext(ctx)
	ctxLog.Info("Context message")
}
