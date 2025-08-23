package logger_test

import (
	"context"
	"github.com/HoangAnhNguyen269/loggerkit/testutil"
	"testing"
	"time"

	logger "github.com/HoangAnhNguyen269/loggerkit"
	_ "github.com/HoangAnhNguyen269/loggerkit/provider/zapx"
)

// K) Benchmarks

func BenchmarkLoggingConsole(b *testing.B) {
	log, err := logger.NewDevelopment()
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Info("Benchmark message",
				logger.F.String("key1", "value1"),
				logger.F.Int("key2", 42),
				logger.F.Bool("key3", true),
			)
		}
	})
}

func BenchmarkLoggingFile(b *testing.B) {
	tempFile, cleanup := testutil.TempFile(b, "bench-log", ".log")
	defer cleanup()

	log, err := logger.NewProduction(
		logger.WithFile(logger.FileSink{
			Path:       tempFile,
			MaxSizeMB:  100, // Large enough to avoid rotation during benchmark
			MaxBackups: 1,
		}),
		logger.WithConsoleDisabled(),
	)
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Info("Benchmark file message",
				logger.F.String("component", "benchmark"),
				logger.F.Int("iteration", 1),
				logger.F.Duration("elapsed", time.Microsecond*100),
			)
		}
	})
}

func BenchmarkLoggingESStub(b *testing.B) {
	mockES := testutil.NewElasticsearchMock()
	defer mockES.Close()

	log, err := logger.NewProduction(
		logger.WithElastic(logger.ElasticSink{
			Addresses:     []string{mockES.URL},
			FlushInterval: 100 * time.Millisecond,
		}),
		logger.WithConsoleDisabled(),
	)
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Info("Benchmark ES message",
				logger.F.String("service", "benchmark"),
				logger.F.String("environment", "test"),
				logger.F.Int("worker_id", 1),
			)
		}
	})
}

func BenchmarkFieldHelpers(b *testing.B) {
	log, err := logger.NewProduction(
		logger.WithConsoleDisabled(), // Minimize output overhead
	)
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	b.ResetTimer()

	b.Run("Legacy", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				log.Info("Legacy fields",
					logger.String("str", "value"),
					logger.Int("int", 42),
					logger.Bool("bool", true),
					logger.Duration("dur", time.Millisecond),
				)
			}
		})
	})

	b.Run("F_Helpers", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				log.Info("F helper fields",
					logger.F.String("str", "value"),
					logger.F.Int("int", 42),
					logger.F.Bool("bool", true),
					logger.F.Duration("dur", time.Millisecond),
				)
			}
		})
	})
}

func BenchmarkWithChaining(b *testing.B) {
	log, err := logger.NewDevelopment()
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	b.ResetTimer()

	b.Run("NoWith", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				log.Info("Direct message",
					logger.F.String("service", "test"),
					logger.F.String("component", "bench"),
					logger.F.String("operation", "test"),
				)
			}
		})
	})

	b.Run("WithChaining", func(b *testing.B) {
		serviceLog := log.With(logger.F.String("service", "test"))
		componentLog := serviceLog.With(logger.F.String("component", "bench"))

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				componentLog.Info("Chained message",
					logger.F.String("operation", "test"),
				)
			}
		})
	})
}

func BenchmarkContextLogger(b *testing.B) {
	log, err := logger.NewProduction()
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	ctx := context.Background()
	ctx = context.WithValue(ctx, "request_id", "req-123")
	ctx = context.WithValue(ctx, "user_id", "user-456")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		contextLog := log.WithContext(ctx) // Create once per goroutine
		for pb.Next() {
			contextLog.Info("Context message",
				logger.F.String("action", "benchmark"),
			)
		}
	})
}

func BenchmarkSampling(b *testing.B) {
	log, err := logger.NewProduction(
		logger.WithSampling(logger.Sampling{
			Initial:    1,
			Thereafter: 100, // Heavy sampling
		}),
	)
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Info("Sampled message",
				logger.F.String("type", "benchmark"),
				logger.F.Int("value", 42),
			)
		}
	})
}

func BenchmarkLogLevels(b *testing.B) {
	log, err := logger.NewProduction(
		logger.WithLevel("info"), // Debug messages will be filtered
	)
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	b.ResetTimer()

	b.Run("Debug_Filtered", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				log.Debug("Debug message", logger.F.String("level", "debug"))
			}
		})
	})

	b.Run("Info_Active", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				log.Info("Info message", logger.F.String("level", "info"))
			}
		})
	})

	b.Run("Error_Active", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				log.Error("Error message", logger.F.String("level", "error"))
			}
		})
	})
}

func BenchmarkMultipleFields(b *testing.B) {
	log, err := logger.NewProduction()
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close(context.Background())

	b.ResetTimer()

	b.Run("FewFields", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				log.Info("Few fields",
					logger.F.String("f1", "v1"),
					logger.F.Int("f2", 2),
				)
			}
		})
	})

	b.Run("ManyFields", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				log.Info("Many fields",
					logger.F.String("f1", "v1"),
					logger.F.String("f2", "v2"),
					logger.F.String("f3", "v3"),
					logger.F.String("f4", "v4"),
					logger.F.String("f5", "v5"),
					logger.F.Int("i1", 1),
					logger.F.Int("i2", 2),
					logger.F.Int("i3", 3),
					logger.F.Bool("b1", true),
					logger.F.Bool("b2", false),
					logger.F.Duration("d1", time.Millisecond),
					logger.F.Any("any1", map[string]string{"nested": "value"}),
				)
			}
		})
	})
}
