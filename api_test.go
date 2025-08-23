package logger_test

import (
	"context"
	"encoding/json"
	"github.com/HoangAnhNguyen269/loggerkit/testutil"
	"os"
	"strings"
	"testing"
	"time"

	logger "github.com/HoangAnhNguyen269/loggerkit"
	_ "github.com/HoangAnhNguyen269/loggerkit/provider/zapx" // Import to register the builder
)

// A) Core API & Options

func TestNewDevelopmentDefaults(t *testing.T) {
	output, err := testutil.CaptureStdout(func() {
		log, err := logger.NewDevelopment()
		if err != nil {
			t.Fatalf("Failed to create development logger: %v", err)
		}
		defer log.Close(context.Background())

		// Test that debug level works (should be visible)
		log.Debug("Debug message")
		log.Info("Info message")
	})

	if err != nil {
		t.Fatalf("Failed to capture stdout: %v", err)
	}

	// Development defaults: debug level, human-readable format, no sampling
	if !strings.Contains(output, "Debug message") {
		t.Error("Expected debug message to be logged in development mode")
	}
	if !strings.Contains(output, "Info message") {
		t.Error("Expected info message to be logged")
	}

	// Check that it's human-readable format (not JSON)
	if strings.Contains(output, `{"level":`) {
		t.Error("Expected human-readable format in development, not JSON")
	}
}

func TestNewProductionDefaults(t *testing.T) {
	output, err := testutil.CaptureStdout(func() {
		log, err := logger.NewProduction()
		if err != nil {
			t.Fatalf("Failed to create production logger: %v", err)
		}
		defer log.Close(context.Background())

		log.Debug("Debug message") // Should not appear
		log.Info("Info message")   // Should appear
	})

	if err != nil {
		t.Fatalf("Failed to capture stdout: %v", err)
	}

	// Production defaults: info level, JSON format, sampling enabled
	if strings.Contains(output, "Debug message") {
		t.Error("Expected debug message to be filtered in production mode")
	}
	if !strings.Contains(output, "Info message") {
		t.Error("Expected info message to be logged")
	}

	// Check that it's JSON format
	var logLine map[string]interface{}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) > 0 {
		if err := json.Unmarshal([]byte(lines[0]), &logLine); err != nil {
			t.Errorf("Expected JSON format in production, got error: %v", err)
		}
		if logLine["level"] != "info" {
			t.Error("Expected level field in JSON output")
		}
		if logLine["msg"] != "Info message" {
			t.Error("Expected msg field in JSON output")
		}
	}
}

func TestWithOptionsComposition(t *testing.T) {
	tempFile, cleanup := testutil.TempFile(t, "test-log", ".log")
	defer cleanup()

	log, err := logger.NewProduction(
		logger.WithService("test-service"),
		logger.WithLevel("debug"), // Override production default
		logger.WithCaller(true),
		logger.WithSampling(logger.Sampling{Initial: 1, Thereafter: 1}), // No sampling effect
		logger.WithFile(logger.FileSink{
			Path:       tempFile,
			MaxSizeMB:  1,
			MaxBackups: 1,
			MaxAgeDays: 1,
			Compress:   false,
		}),
		logger.WithMetrics(logger.MetricsOptions{
			Enabled:      true,
			AutoRegister: false, // Don't pollute global registry in tests
		}),
	)

	if err != nil {
		t.Fatalf("Failed to create logger with options: %v", err)
	}
	defer log.Close(context.Background())

	// Test that debug level works (overridden from production default)
	log.Debug("Debug with caller info")

	// Give it a moment to write to file
	time.Sleep(100 * time.Millisecond)

	// Read file content
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var logEntry map[string]interface{}
	if err := json.Unmarshal(content, &logEntry); err != nil {
		t.Fatalf("Failed to parse log JSON: %v", err)
	}

	// Verify options took effect
	if logEntry["level"] != "debug" {
		t.Error("Expected debug level to be active")
	}
	if _, hasCaller := logEntry["caller"]; !hasCaller {
		t.Error("Expected caller information to be present")
	}
}

// B) Fields & Helpers

func TestLoggerWithAndFields(t *testing.T) {
	output, err := testutil.CaptureStdout(func() {
		log, err := logger.NewProduction()
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer log.Close(context.Background())

		contextLog := log.With(logger.F.String("req_id", "r1"))
		contextLog.Info("Test message", logger.F.String("extra", "field"))
	})

	if err != nil {
		t.Fatalf("Failed to capture stdout: %v", err)
	}

	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry); err != nil {
		t.Fatalf("Failed to parse log JSON: %v", err)
	}

	if logEntry["req_id"] != "r1" {
		t.Error("Expected req_id field from With()")
	}
	if logEntry["extra"] != "field" {
		t.Error("Expected extra field from log call")
	}
}

func TestLegacyVsNewFieldHelpers(t *testing.T) {
	output1, err := testutil.CaptureStdout(func() {
		log, err := logger.NewProduction()
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer log.Close(context.Background())

		// Legacy helpers
		log.Info("Legacy",
			logger.String("str", "value"),
			logger.Int("num", 42),
			logger.Bool("flag", true),
		)
	})
	if err != nil {
		t.Fatalf("Failed to capture stdout: %v", err)
	}

	output2, err := testutil.CaptureStdout(func() {
		log, err := logger.NewProduction()
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer log.Close(context.Background())

		// New F helpers
		log.Info("New",
			logger.F.String("str", "value"),
			logger.F.Int("num", 42),
			logger.F.Bool("flag", true),
		)
	})
	if err != nil {
		t.Fatalf("Failed to capture stdout: %v", err)
	}

	var legacy, newStyle map[string]interface{}
	json.Unmarshal([]byte(strings.TrimSpace(output1)), &legacy)
	json.Unmarshal([]byte(strings.TrimSpace(output2)), &newStyle)

	// Should produce same field values
	if legacy["str"] != newStyle["str"] ||
		legacy["num"] != newStyle["num"] ||
		legacy["flag"] != newStyle["flag"] {
		t.Error("Legacy and new field helpers should produce identical output")
	}
}

// D) Sampling & Levels

func TestSamplingZapSemantics(t *testing.T) {
	messageCount := 0
	output, err := testutil.CaptureStdout(func() {
		log, err := logger.NewProduction(
			logger.WithSampling(logger.Sampling{
				Initial:    3, // First 3 messages
				Thereafter: 5, // Then every 5th message
			}),
		)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer log.Close(context.Background())

		// Send 20 identical messages
		for i := 0; i < 20; i++ {
			log.Info("Sampled message", logger.F.Int("i", i))
		}
	})
	if err != nil {
		t.Fatalf("Failed to capture stdout: %v", err)
	}

	// Count actual logged messages
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			messageCount++
		}
	}

	// With Initial=3, Thereafter=5: expect 3 + 3 + 2 = 8 messages out of 20
	// (messages 0,1,2 then 7,12,17)
	expectedRange := []int{6, 10} // Allow some variance
	if messageCount < expectedRange[0] || messageCount > expectedRange[1] {
		t.Errorf("Expected %d-%d sampled messages, got %d", expectedRange[0], expectedRange[1], messageCount)
	}
}

func TestStacktraceAt(t *testing.T) {
	output, err := testutil.CaptureStdout(func() {
		log, err := logger.NewProduction(
			logger.WithLevel("debug"),
			logger.WithStacktraceAt("error"), // Only errors and above get stacktraces
		)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer log.Close(context.Background())

		log.Warn("Warning message")
		log.Error("Error message")
	})
	if err != nil {
		t.Fatalf("Failed to capture stdout: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")

	var warnEntry, errorEntry map[string]interface{}
	json.Unmarshal([]byte(lines[0]), &warnEntry)
	json.Unmarshal([]byte(lines[1]), &errorEntry)

	// Warn should not have stacktrace
	if _, hasStacktrace := warnEntry["stacktrace"]; hasStacktrace {
		t.Error("Expected no stacktrace for warn level")
	}

	// Error should have stacktrace
	if _, hasStacktrace := errorEntry["stacktrace"]; !hasStacktrace {
		t.Error("Expected stacktrace for error level")
	}
}

func TestEnableCaller(t *testing.T) {
	// With caller enabled
	output1, err := testutil.CaptureStdout(func() {
		log, err := logger.NewProduction(logger.WithCaller(true))
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer log.Close(context.Background())

		log.Info("With caller")
	})
	if err != nil {
		t.Fatalf("Failed to capture stdout: %v", err)
	}

	// With caller disabled
	output2, err := testutil.CaptureStdout(func() {
		log, err := logger.NewProduction(logger.WithCaller(false))
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer log.Close(context.Background())

		log.Info("Without caller")
	})
	if err != nil {
		t.Fatalf("Failed to capture stdout: %v", err)
	}

	var withCaller, withoutCaller map[string]interface{}
	json.Unmarshal([]byte(strings.TrimSpace(output1)), &withCaller)
	json.Unmarshal([]byte(strings.TrimSpace(output2)), &withoutCaller)

	// Check caller presence
	if _, hasCaller := withCaller["caller"]; !hasCaller {
		t.Error("Expected caller field when enabled")
	}
	if _, hasCaller := withoutCaller["caller"]; hasCaller {
		t.Error("Expected no caller field when disabled")
	}
}
