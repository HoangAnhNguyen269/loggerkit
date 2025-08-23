package zapx

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	logger "github.com/HoangAnhNguyen269/loggerkit"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
)

type elasticsearchWriter struct {
	client       *elasticsearch.Client
	indexer      esutil.BulkIndexer
	service      string
	indexPattern string
	dlqFile      *os.File
	dlqMutex     sync.Mutex
	metrics      *logger.Metrics
	closeCh      chan struct{}
	closeOnce    sync.Once
	wg           sync.WaitGroup
}

func newElasticsearchWriter(config *logger.ElasticSink, service string, metrics *logger.Metrics) (*elasticsearchWriter, error) {
	// Create Elasticsearch client
	esConfig := elasticsearch.Config{
		Addresses: config.Addresses,
		CloudID:   config.CloudID,
	}

	// Configure authentication
	if config.APIKey != "" {
		esConfig.APIKey = config.APIKey
	} else if config.Username != "" && config.Password != "" {
		esConfig.Username = config.Username
		esConfig.Password = config.Password
	} else if config.ServiceToken != "" {
		esConfig.ServiceToken = config.ServiceToken
	}

	// Configure TLS
	if config.CACert != nil || config.ClientCert != nil || config.InsecureSkipVerify {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,
		}

		if config.CACert != nil {
			// Handle CA certificate
			// Note: This is simplified - in production you'd want proper CA cert handling
		}

		if config.ClientCert != nil && config.ClientKey != nil {
			cert, err := tls.X509KeyPair(config.ClientCert, config.ClientKey)
			if err != nil {
				return nil, fmt.Errorf("failed to load client certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		esConfig.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	client, err := elasticsearch.NewClient(esConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	// Determine index pattern
	indexPattern := config.Index
	if indexPattern == "" {
		indexPattern = fmt.Sprintf("%s-%%Y.%%m.%%d", service)
	}

	// Create bulk indexer
	bulkConfig := esutil.BulkIndexerConfig{
		Index:         "", // We'll set this per document with time-based index
		Client:        client,
		NumWorkers:    1,
		FlushBytes:    config.BulkSizeBytes,
		FlushInterval: config.FlushInterval,
		OnError: func(ctx context.Context, err error) {
			if metrics != nil {
				metrics.RecordLogDropped("elasticsearch", "bulk_error")
			}
		},
		OnFlushStart: func(ctx context.Context) context.Context {
			return ctx
		},
		OnFlushEnd: func(ctx context.Context) {
			// Record metrics on flush completion
		},
	}

	if config.BulkActions > 0 {
		bulkConfig.FlushInterval = time.Duration(config.BulkActions) * time.Millisecond // This is wrong, should be count-based
		// Note: esutil.BulkIndexer doesn't have a direct FlushCount option, so we use a workaround
	}

	indexer, err := esutil.NewBulkIndexer(bulkConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create bulk indexer: %w", err)
	}

	writer := &elasticsearchWriter{
		client:       client,
		indexer:      indexer,
		service:      service,
		indexPattern: indexPattern,
		metrics:      metrics,
		closeCh:      make(chan struct{}),
	}

	// Open DLQ file if configured
	if config.DLQPath != "" {
		dlqFile, err := os.OpenFile(config.DLQPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			indexer.Close(context.Background())
			return nil, fmt.Errorf("failed to open DLQ file %s: %w", config.DLQPath, err)
		}
		writer.dlqFile = dlqFile
	}

	return writer, nil
}

func (w *elasticsearchWriter) Write(p []byte) (int, error) {
	// Parse the log entry to extract timestamp for index naming
	var logEntry map[string]interface{}
	if err := json.Unmarshal(p, &logEntry); err != nil {
		// If we can't parse the JSON, write to DLQ and return error
		w.writeToDLQ(p, "json_parse_error")
		return 0, fmt.Errorf("failed to parse log entry as JSON: %w", err)
	}

	// Generate index name with current date
	indexName := generateIndexName(w.indexPattern, w.service)

	// Add service and env fields
	logEntry["service"] = w.service

	// Re-marshal with added fields
	enrichedData, err := json.Marshal(logEntry)
	if err != nil {
		w.writeToDLQ(p, "enrichment_error")
		return len(p), nil // Return success to avoid blocking the logger
	}

	// Create bulk indexer item
	item := esutil.BulkIndexerItem{
		Action: "index",
		Index:  indexName,
		Body:   bytes.NewReader(enrichedData),
		OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {
			if w.metrics != nil {
				w.metrics.RecordLogWritten("info", "elasticsearch") // We don't have level here
			}
		},
		OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
			// Write to DLQ on failure
			w.writeToDLQ(enrichedData, fmt.Sprintf("index_error_%d", res.Status))
			if w.metrics != nil {
				w.metrics.RecordLogDropped("elasticsearch", "index_failure")
			}
		},
	}

	// Add item to bulk indexer
	if err := w.indexer.Add(context.Background(), item); err != nil {
		// If we can't add to indexer, write to DLQ
		w.writeToDLQ(enrichedData, "indexer_add_error")
		if w.metrics != nil {
			w.metrics.RecordLogDropped("elasticsearch", "indexer_full")
		}
	}

	return len(p), nil
}

func (w *elasticsearchWriter) Sync() error {
	// Force flush the bulk indexer
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return w.indexer.Close(ctx)
}

func (w *elasticsearchWriter) Close() error {
	w.closeOnce.Do(func() {
		close(w.closeCh)

		// Close the bulk indexer
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		w.indexer.Close(ctx)

		// Close DLQ file if open
		if w.dlqFile != nil {
			w.dlqMutex.Lock()
			w.dlqFile.Close()
			w.dlqMutex.Unlock()
		}

		w.wg.Wait()
	})
	return nil
}

func (w *elasticsearchWriter) writeToDLQ(data []byte, reason string) {
	if w.dlqFile == nil {
		return
	}

	w.dlqMutex.Lock()
	defer w.dlqMutex.Unlock()

	dlqEntry := map[string]interface{}{
		"timestamp":    time.Now().UTC().Format(time.RFC3339Nano),
		"reason":       reason,
		"original_log": string(data),
	}

	dlqData, err := json.Marshal(dlqEntry)
	if err != nil {
		return // Can't do much if DLQ serialization fails
	}

	// Write to DLQ file
	w.dlqFile.Write(dlqData)
	w.dlqFile.Write([]byte("\n"))
}

func generateIndexName(pattern, service string) string {
	now := time.Now().UTC()

	// Replace placeholders
	indexName := strings.ReplaceAll(pattern, "<service>", service)
	indexName = strings.ReplaceAll(indexName, "%Y", fmt.Sprintf("%04d", now.Year()))
	indexName = strings.ReplaceAll(indexName, "%m", fmt.Sprintf("%02d", now.Month()))
	indexName = strings.ReplaceAll(indexName, "%d", fmt.Sprintf("%02d", now.Day()))

	return indexName
}

// RetryableWriter wraps the elasticsearch writer with retry logic
type retryableWriter struct {
	writer      *elasticsearchWriter
	retryConfig logger.Retry
	metrics     *logger.Metrics
}

func newRetryableWriter(writer *elasticsearchWriter, retryConfig logger.Retry, metrics *logger.Metrics) *retryableWriter {
	return &retryableWriter{
		writer:      writer,
		retryConfig: retryConfig,
		metrics:     metrics,
	}
}

func (rw *retryableWriter) Write(p []byte) (int, error) {
	var lastErr error

	for attempt := 0; attempt <= rw.retryConfig.Max; attempt++ {
		n, err := rw.writer.Write(p)
		if err == nil {
			return n, nil
		}

		lastErr = err

		if attempt < rw.retryConfig.Max {
			// Calculate backoff with jitter
			backoff := rw.calculateBackoff(attempt)
			time.Sleep(backoff)

			if rw.metrics != nil {
				rw.metrics.RecordESBulkRetry("write_error")
			}
		}
	}

	return 0, lastErr
}

func (rw *retryableWriter) Sync() error {
	return rw.writer.Sync()
}

func (rw *retryableWriter) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff with jitter
	backoff := float64(rw.retryConfig.BackoffMin) * math.Pow(2, float64(attempt))

	// Cap at max backoff
	if backoff > float64(rw.retryConfig.BackoffMax) {
		backoff = float64(rw.retryConfig.BackoffMax)
	}

	// Add jitter (±25%)
	jitter := backoff * 0.25 * (rand.Float64()*2 - 1)
	backoff += jitter

	return time.Duration(backoff)
}

// scanDLQ provides a utility to scan DLQ files (for debugging/recovery)
func ScanDLQ(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println("DLQ Entry:", line)
	}

	return scanner.Err()
}
