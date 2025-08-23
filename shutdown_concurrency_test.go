package logger_test

import (
	"context"
	"sync"
	"testing"
	"time"

	logger "github.com/HoangAnhNguyen269/loggerkit"
	_ "github.com/HoangAnhNguyen269/loggerkit/provider/zapx"
	"github.com/HoangAnhNguyen269/loggerkit/testutil"
)

// I) Graceful Shutdown & Resource Management

func TestLoggerCloseFlushesAll(t *testing.T) {
	tempFile, cleanup := testutil.TempFile(t, "test-close", ".log")
	defer cleanup()

	mockES := testutil.NewElasticsearchMock()
	defer mockES.Close()

	log, err := logger.NewProduction(
		logger.WithFile(logger.FileSink{
			Path:       tempFile,
			MaxSizeMB:  1,
			MaxBackups: 1,
		}),
		logger.WithElastic(logger.ElasticSink{
			Addresses:     []string{mockES.URL},
			FlushInterval: 1 * time.Second, // Long interval, rely on Close() to flush
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Log some entries
	for i := 0; i < 10; i++ {
		log.Info("Message before close", logger.F.Int("id", i))
	}

	// Close with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	startTime := time.Now()
	err = log.Close(ctx)
	closeTime := time.Since(startTime)

	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	if closeTime > 4*time.Second {
		t.Errorf("Close() took too long: %v", closeTime)
	}

	// Verify all messages were flushed to ES
	if !mockES.WaitForDocs(10, 2*time.Second) {
		t.Error("Expected all messages to be flushed to Elasticsearch on Close()")
	}

	// Note: File flushing is harder to test deterministically, but no panics is a good sign
}

func TestCloseWithTimeout(t *testing.T) {
	log, err := logger.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Log a message
	log.Info("Test message")

	// Close with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// This might timeout, but should not panic
	err = log.Close(ctx)
	// We don't check error since timeout is expected with such short duration

	// Should be able to create another logger after close
	log2, err := logger.NewDevelopment()
	if err != nil {
		t.Errorf("Failed to create second logger after close: %v", err)
	}
	if log2 != nil {
		log2.Close(context.Background())
	}
}

// J) Concurrency & Race

func TestConcurrentLoggingNoDataRace(t *testing.T) {
	log, err := logger.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	numGoroutines := 50
	messagesPerGoroutine := 20
	var wg sync.WaitGroup

	// Test concurrent logging from multiple goroutines
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < messagesPerGoroutine; j++ {
				log.Info("Concurrent message",
					logger.F.Int("goroutine", goroutineID),
					logger.F.Int("message", j),
					logger.F.String("data", "some data that might cause races"),
				)

				// Mix in some With() calls to test concurrent field modifications
				if j%3 == 0 {
					contextLog := log.With(logger.F.String("context", "test"))
					contextLog.Warn("Context message", logger.F.Int("id", j))
				}

				// Test Log() method as well
				if j%5 == 0 {
					log.Log(logger.InfoLevel, "Generic log call", logger.F.Int("via_log", j))
				}
			}
		}(i)
	}

	wg.Wait()
	// If we reach here without data races, the test passes
}

func TestConcurrentWithContext(t *testing.T) {
	log, err := logger.NewProduction()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	numGoroutines := 30
	var wg sync.WaitGroup

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			// Create context with values
			ctx := context.Background()
			ctx = context.WithValue(ctx, "request_id", goroutineID)
			ctx = context.WithValue(ctx, "user_id", goroutineID*1000)

			// Create context logger
			ctxLog := log.WithContext(ctx)

			// Log concurrently with context
			for j := 0; j < 10; j++ {
				ctxLog.Info("Message with context",
					logger.F.Int("iteration", j),
					logger.F.String("routine", "test"),
				)
			}
		}(i)
	}

	wg.Wait()
}

func TestConcurrentWithChaining(t *testing.T) {
	log, err := logger.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	numGoroutines := 25
	var wg sync.WaitGroup

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			// Test concurrent chaining of With() calls
			baseLog := log.With(logger.F.String("service", "test"))
			serviceLog := baseLog.With(logger.F.String("component", "handler"))

			for j := 0; j < 5; j++ {
				// Chain more context
				requestLog := serviceLog.With(
					logger.F.Int("request", j),
					logger.F.Int("goroutine", goroutineID),
				)

				requestLog.Info("Chained logging test")

				// Create context logger from chained logger
				ctx := context.WithValue(context.Background(), "trace_id", j*goroutineID)
				contextLog := requestLog.WithContext(ctx)
				contextLog.Debug("Deep chain test")
			}
		}(i)
	}

	wg.Wait()
}

func TestConcurrentClose(t *testing.T) {
	log, err := logger.NewProduction()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 10

	// Start goroutines that log and then try to close
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			// Log a few messages
			for j := 0; j < 3; j++ {
				log.Info("Message before concurrent close", logger.F.Int("goroutine", id))
			}

			// Try to close concurrently (only one should succeed)
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			log.Close(ctx) // Multiple closes should be safe
		}(i)
	}

	wg.Wait()
	// Should not panic or cause data races
}

// Test factory registry isolation between tests
func TestRegistryIsolation(t *testing.T) {
	// This test verifies that tests don't interfere with each other
	// The actual isolation is handled by the test setup, but we can verify
	// that logger creation still works as expected

	log1, err := logger.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create first logger: %v", err)
	}
	defer log1.Close(context.Background())

	log1.Info("First logger test")

	log2, err := logger.NewProduction()
	if err != nil {
		t.Fatalf("Failed to create second logger: %v", err)
	}
	defer log2.Close(context.Background())

	log2.Info("Second logger test")

	// Both should work without interference
}

func TestHighVolumeLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high-volume test in short mode")
	}

	log, err := logger.NewProduction(
		logger.WithSampling(logger.Sampling{Initial: 100, Thereafter: 100}), // Minimal sampling
	)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	// Log high volume of messages to test performance and race conditions
	numMessages := 10000
	var wg sync.WaitGroup
	numGoroutines := 10

	wg.Add(numGoroutines)
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < numMessages/numGoroutines; i++ {
				log.Info("High volume message",
					logger.F.Int("goroutine", goroutineID),
					logger.F.Int("message", i),
					logger.F.String("data", "test data for volume testing"),
				)

				if i%100 == 0 {
					// Occasional error messages
					log.Error("Occasional error", logger.F.Int("at", i))
				}
			}
		}(g)
	}

	start := time.Now()
	wg.Wait()
	duration := time.Since(start)

	t.Logf("Logged %d messages in %v (%.0f msg/sec)",
		numMessages, duration, float64(numMessages)/duration.Seconds())

	// Should complete without deadlocks or panics
}
