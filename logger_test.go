package logger_test

import (
	"context"
	"sync"
	"testing"
	"time"

	logger "github.com/HoangAnhNguyen269/loggerkit"
	_ "github.com/HoangAnhNguyen269/loggerkit/provider/zapx" // Import to register the builder
)

func TestNewDevelopment(t *testing.T) {
	log, err := logger.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create development logger: %v", err)
	}
	defer log.Close(context.Background())

	// Test basic logging
	log.Info("Test message", logger.F.String("key", "value"))
	log.Debug("Debug message", logger.F.Int("count", 42))
	log.Warn("Warning message", logger.F.Bool("flag", true))
	log.Error("Error message", logger.F.Err(err))
}

func TestNewProduction(t *testing.T) {
	log, err := logger.NewProduction()
	if err != nil {
		t.Fatalf("Failed to create production logger: %v", err)
	}
	defer log.Close(context.Background())

	// Test basic logging
	log.Info("Production test", logger.F.String("service", "test"))
}

func TestSampling(t *testing.T) {
	log, err := logger.NewProduction(
		logger.WithSampling(logger.Sampling{
			Initial:    1,
			Thereafter: 10, // Sample every 10th message after the first
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create logger with sampling: %v", err)
	}
	defer log.Close(context.Background())

	// Log many messages - only some should be sampled
	for i := 0; i < 100; i++ {
		log.Info("Sampled message", logger.F.Int("iteration", i))
	}
}

func TestWithFields(t *testing.T) {
	log, err := logger.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	// Test With method
	contextLog := log.With(
		logger.F.String("service", "test"),
		logger.F.String("version", "1.0.0"),
	)

	contextLog.Info("Message with context")
}

func TestContextLogger(t *testing.T) {
	log, err := logger.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	// Create context with values
	ctx := context.Background()
	ctx = context.WithValue(ctx, "request_id", "test-request-123")
	ctx = context.WithValue(ctx, "user_id", "user-456")

	// Use context logger
	contextLog := log.WithContext(ctx)
	contextLog.Info("Message with context data")
}

func TestMetrics(t *testing.T) {
	log, err := logger.NewProduction(
		logger.WithMetrics(logger.MetricsOptions{
			Enabled:      true,
			AutoRegister: false, // Don't auto-register in tests
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create logger with metrics: %v", err)
	}
	defer log.Close(context.Background())

	// Get metrics collectors
	collectors := logger.MetricsCollectors()
	if len(collectors) != 5 {
		t.Errorf("Expected 5 metric collectors, got %d", len(collectors))
	}

	// Log some messages to generate metrics
	log.Info("Test message 1")
	log.Warn("Test warning")
	log.Error("Test error")
}

func TestConcurrentLogging(t *testing.T) {
	log, err := logger.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	// Test concurrent logging from multiple goroutines
	var wg sync.WaitGroup
	numGoroutines := 100
	messagesPerGoroutine := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				log.Info("Concurrent message",
					logger.F.Int("goroutine", id),
					logger.F.Int("message", j),
				)
			}
		}(i)
	}

	wg.Wait()
}

func TestRaceCondition(t *testing.T) {
	log, err := logger.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	// Create multiple goroutines that create child loggers and log simultaneously
	var wg sync.WaitGroup
	numGoroutines := 1000

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			// Create child logger with context
			childLog := log.With(logger.F.Int("worker", id))

			// Log with context
			ctx := context.WithValue(context.Background(), "request_id", id)
			contextLog := childLog.WithContext(ctx)

			contextLog.Info("Race test message")
		}(i)
	}

	wg.Wait()
}

func TestFileLogging(t *testing.T) {
	tempFile := "/tmp/test_logger.log"

	log, err := logger.NewProduction(
		logger.WithFile(logger.FileSink{
			Path:       tempFile,
			MaxSizeMB:  1,
			MaxBackups: 3,
			MaxAgeDays: 7,
			Compress:   true,
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create logger with file sink: %v", err)
	}
	defer log.Close(context.Background())

	// Log some messages
	log.Info("File logging test", logger.F.String("file", tempFile))
	log.Error("Test error in file", logger.F.String("reason", "testing"))
}

func TestFieldHelpers(t *testing.T) {
	log, err := logger.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	// Test all field helper functions
	log.Info("Field helpers test",
		logger.F.String("string_field", "test"),
		logger.F.Int("int_field", 42),
		logger.F.Bool("bool_field", true),
		logger.F.Duration("duration_field", time.Second*5),
		logger.F.Any("any_field", map[string]interface{}{"nested": "value"}),
	)

	// Test error field
	testErr := &customError{"test error"}
	log.Error("Error with custom error", logger.F.Err(testErr))
}

type customError struct {
	message string
}

func (e *customError) Error() string {
	return e.message
}

func TestLegacyConfig(t *testing.T) {
	cfg := &logger.Config{
		Level:          logger.InfoLevel,
		JSON:           true,
		ConsoleEnabled: true,
		FileConfig: &logger.FileConfig{
			Filename:   "/tmp/legacy_test.log",
			MaxSize:    10,
			MaxBackups: 5,
			MaxAge:     30,
			Compress:   true,
		},
	}

	log := logger.MustNew(cfg)
	defer log.Close(context.Background())

	log.Info("Legacy config test")
}

func TestClose(t *testing.T) {
	log, err := logger.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Log some messages
	log.Info("Before close")

	// Test close with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := log.Close(ctx); err != nil {
		t.Errorf("Failed to close logger: %v", err)
	}
}
