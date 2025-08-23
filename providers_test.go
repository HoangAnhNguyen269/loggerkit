package logger_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	logger "github.com/HoangAnhNguyen269/loggerkit"
	_ "github.com/HoangAnhNguyen269/loggerkit/provider/zapx"
	"github.com/HoangAnhNguyen269/loggerkit/testutil"
)

// F) Console/File Providers

func TestConsoleDefaultEnabled(t *testing.T) {
	// Test that console is enabled by default with no other sinks
	output, err := testutil.CaptureStdout(func() {
		log, err := logger.NewProduction() // No explicit sinks
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer log.Close(context.Background())

		log.Info("First message")
		log.Error("Second message")
	})
	if err != nil {
		t.Fatalf("Failed to capture stdout: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 log lines, got %d", len(lines))
	}

	// Verify both messages are JSON formatted (production mode)
	for i, line := range lines {
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			t.Errorf("Line %d not valid JSON: %v", i, err)
		}
	}
}

func TestConsoleDisabledNoSinksFails(t *testing.T) {
	// Test that disabling console with no other sinks returns an error
	_, err := logger.NewProduction(
		logger.WithConsoleDisabled(),
		// No other sinks configured
	)

	if err == nil {
		t.Error("Expected error when console disabled and no other sinks configured")
	}

	expectedMsg := "no log sinks configured"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got: %v", expectedMsg, err)
	}
}

func TestConsoleDisabledWithOtherSinks(t *testing.T) {
	tempFile, cleanup := testutil.TempFile(t, "test-log", ".log")
	defer cleanup()

	// Should work fine when console disabled but file sink enabled
	log, err := logger.NewProduction(
		logger.WithConsoleDisabled(),
		logger.WithFile(logger.FileSink{
			Path:       tempFile,
			MaxSizeMB:  1,
			MaxBackups: 1,
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create logger with console disabled but file enabled: %v", err)
	}
	defer log.Close(context.Background())

	// Capture stdout to ensure nothing goes to console
	output, err := testutil.CaptureStdout(func() {
		log.Info("This should only go to file")
	})
	if err != nil {
		t.Fatalf("Failed to capture stdout: %v", err)
	}

	// Should have no console output
	if strings.TrimSpace(output) != "" {
		t.Error("Expected no console output when console disabled")
	}

	// Give time for file write
	time.Sleep(100 * time.Millisecond)

	// Verify file has the log entry
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Expected log file to have content")
	}

	var logEntry map[string]interface{}
	if err := json.Unmarshal(content, &logEntry); err != nil {
		t.Fatalf("Failed to parse log file JSON: %v", err)
	}

	if logEntry["msg"] != "This should only go to file" {
		t.Error("Expected message in log file")
	}
}

func TestFileRotationAndWrite(t *testing.T) {
	tempDir, cleanup := testutil.TempDir(t, "log-rotation-test")
	defer cleanup()

	logFile := filepath.Join(tempDir, "test.log")

	log, err := logger.NewProduction(
		logger.WithFile(logger.FileSink{
			Path:       logFile,
			MaxSizeMB:  1, // Very small to force rotation
			MaxBackups: 2,
			MaxAgeDays: 1,
			Compress:   false, // Don't compress for easier testing
		}),
		logger.WithConsoleDisabled(),
	)
	if err != nil {
		t.Fatalf("Failed to create logger with file sink: %v", err)
	}
	defer log.Close(context.Background())

	// Write enough data to force rotation
	largeData := strings.Repeat("This is a large log message that will help fill up the log file quickly. ", 100)
	for i := 0; i < 50; i++ {
		log.Info("Large message", logger.F.String("data", largeData), logger.F.Int("iteration", i))
		time.Sleep(10 * time.Millisecond) // Small delay to ensure writes
	}

	// Force close to ensure all data is written
	log.Close(context.Background())

	// Check for rotated files
	files, err := filepath.Glob(filepath.Join(tempDir, "*.log*"))
	if err != nil {
		t.Fatalf("Failed to glob log files: %v", err)
	}

	if len(files) < 1 {
		t.Error("Expected at least one log file")
	}

	// Verify files contain JSON data
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Errorf("Failed to read file %s: %v", file, err)
			continue
		}

		if len(content) == 0 {
			continue // Empty backup files are ok
		}

		// Try to parse the first line as JSON
		lines := strings.Split(string(content), "\n")
		if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
			var logEntry map[string]interface{}
			if err := json.Unmarshal([]byte(lines[0]), &logEntry); err != nil {
				t.Errorf("First line of %s not valid JSON: %v", file, err)
			}
		}
	}
}

func TestMultipleCoresTee(t *testing.T) {
	tempFile, cleanup := testutil.TempFile(t, "multi-core-test", ".log")
	defer cleanup()

	// Enable both console and file
	output, err := testutil.CaptureStdout(func() {
		log, err := logger.NewProduction(
			logger.WithFile(logger.FileSink{
				Path:       tempFile,
				MaxSizeMB:  1,
				MaxBackups: 1,
			}),
			// Console enabled by default
		)
		if err != nil {
			t.Fatalf("Failed to create logger with multiple cores: %v", err)
		}
		defer log.Close(context.Background())

		log.Info("Message to both sinks")
	})
	if err != nil {
		t.Fatalf("Failed to capture stdout: %v", err)
	}

	// Should have console output
	if strings.TrimSpace(output) == "" {
		t.Error("Expected console output")
	}

	var consoleEntry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &consoleEntry); err != nil {
		t.Fatalf("Console output not valid JSON: %v", err)
	}

	// Give time for file write
	time.Sleep(100 * time.Millisecond)

	// Should have file output
	fileContent, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(fileContent) == 0 {
		t.Error("Expected file content")
	}

	var fileEntry map[string]interface{}
	if err := json.Unmarshal(fileContent, &fileEntry); err != nil {
		t.Fatalf("File content not valid JSON: %v", err)
	}

	// Both should have the same message
	if consoleEntry["msg"] != fileEntry["msg"] {
		t.Error("Expected same message in both console and file")
	}
	if consoleEntry["msg"] != "Message to both sinks" {
		t.Error("Expected correct message content")
	}
}
