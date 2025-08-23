package corefactories

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	logger "github.com/HoangAnhNguyen269/loggerkit"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"go.uber.org/zap/zapcore"
)

// ElasticFactory creates Elasticsearch-based cores for logging output
type ElasticFactory struct{}

func init() {
	RegisterFactory(&ElasticFactory{})
}

// Name returns the unique name of this factory
func (ef *ElasticFactory) Name() string {
	return "elasticsearch"
}

// Enabled determines if Elasticsearch logging should be enabled based on options
func (ef *ElasticFactory) Enabled(opts logger.Options) bool {
	return opts.Elastic != nil
}

// Build creates an Elasticsearch core with bulk indexing and DLQ support
func (ef *ElasticFactory) Build(encCfg zapcore.EncoderConfig, lvl zapcore.Level, metrics *logger.Metrics, opts logger.Options) (zapcore.Core, func() error, error) {
	esCfg := opts.Elastic

	// Create the Elasticsearch bulk writer
	esWriter, err := newElasticsearchWriter(esCfg, opts.Service, metrics)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create elasticsearch writer: %w", err)
	}

	var ws zapcore.WriteSyncer
	if esCfg != nil && esCfg.Retry.Max > 0 {
		ws = zapcore.AddSync(newRetryableWriter(esWriter, esCfg.Retry, metrics))
	} else {
		ws = zapcore.AddSync(esWriter)
	}

	encoder := zapcore.NewJSONEncoder(encCfg)
	core := zapcore.NewCore(encoder, zapcore.Lock(ws), lvl)

	return core, esWriter.Close, nil
}

type elasticsearchWriter struct {
	client       *elasticsearch.Client
	indexer      esutil.BulkIndexer
	service      string
	indexPattern string
	dlqFile      *os.File
	dlqMutex     sync.Mutex
	metrics      *logger.Metrics
	closeOnce    sync.Once
	closed       uint32
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
		Index:         "", // set per doc
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

	if config.BulkActions > 0 && config.FlushInterval == 0 && config.BulkSizeBytes == 0 {
		// fallback an toàn (ví dụ 2s)
		bulkConfig.FlushInterval = 2 * time.Second
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
	// Guard: đã Close() thì từ chối ghi
	if atomic.LoadUint32(&w.closed) == 1 {
		w.writeToDLQ(p, "writer_closed")
		if w.metrics != nil {
			w.metrics.RecordLogDropped("elasticsearch", "writer_closed")
		}
		return 0, errors.New("elasticsearch writer is closed")
	}

	// Parse JSON gốc
	var logEntry map[string]interface{}
	if err := json.Unmarshal(p, &logEntry); err != nil {
		// lỗi dữ liệu: không retry
		w.writeToDLQ(p, "json_parse_error")
		return 0, fmt.Errorf("failed to parse log entry as JSON: %w", err)
	}

	// Tạo index name + enrich
	indexName := generateIndexName(w.indexPattern, w.service)
	logEntry["service"] = w.service

	enrichedData, err := json.Marshal(logEntry)
	if err != nil {
		// lỗi enrich: không retry (tuỳ bạn)
		w.writeToDLQ(p, "enrichment_error")
		// trả nil để không chặn luồng log; hoặc return 0, err nếu muốn cứng rắn hơn
		return len(p), nil
	}

	// Bulk item
	item := esutil.BulkIndexerItem{
		Action: "index",
		Index:  indexName,
		Body:   bytes.NewReader(enrichedData),
		OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {
			// Note: Don't record LogsWritten here - MetricsCore wrapper handles that with correct level
		},
		OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
			// Lỗi do ES trả về sau khi Add thành công → không retry được ở đây
			w.writeToDLQ(enrichedData, fmt.Sprintf("index_error_%d", res.Status))
			if w.metrics != nil {
				w.metrics.RecordLogDropped("elasticsearch", "index_failure")
			}
		},
	}

	// ✅ Điểm mấu chốt: nếu Add lỗi → TRẢ ERROR để retryableWriter xử lý
	if err := w.indexer.Add(context.Background(), item); err != nil {
		if w.metrics != nil {
			w.metrics.RecordLogDropped("elasticsearch", "indexer_add_error")
		}
		// KHÔNG DLQ ở đây — để retryableWriter DLQ nếu hết retry
		return 0, err
	}

	return len(p), nil
}

func (w *elasticsearchWriter) Sync() error {
	return nil
}

func (w *elasticsearchWriter) Close() error {
	w.closeOnce.Do(func() {
		atomic.StoreUint32(&w.closed, 1)

		// Close (flush + close) the bulk indexer
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = w.indexer.Close(ctx) // capture err if bạn muốn bubble lên

		// Close DLQ file if open
		if w.dlqFile != nil {
			w.dlqMutex.Lock()
			_ = w.dlqFile.Close()
			w.dlqMutex.Unlock()
		}
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
	w.dlqFile.Sync() // Force flush to disk
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
			time.Sleep(rw.calculateBackoff(attempt))
			if rw.metrics != nil {
				rw.metrics.RecordESBulkRetry("write_error")
			}
		}
	}
	// Hết retry → DLQ ở đây
	rw.writer.writeToDLQ(p, "retries_exhausted")
	if rw.metrics != nil {
		rw.metrics.RecordLogDropped("elasticsearch", "retries_exhausted")
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

// ScanDLQ provides a utility to scan DLQ files (for debugging/recovery)
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
