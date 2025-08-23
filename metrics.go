package logger

import (
	"github.com/prometheus/client_golang/prometheus"
	"sync"
)

// Metrics holds all the Prometheus metrics for the logger
type Metrics struct {
	LogsWritten   *prometheus.CounterVec
	LogsDropped   *prometheus.CounterVec
	ESBulkRetries *prometheus.CounterVec
	ESQueueDepth  *prometheus.GaugeVec
	ESBulkLatency *prometheus.HistogramVec
}

var (
	metricsOnce sync.Once
	metrics     *Metrics
)

// GetMetrics returns the singleton metrics instance
func GetMetrics() *Metrics {
	metricsOnce.Do(func() {
		metrics = &Metrics{
			LogsWritten: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "logs_written_total",
					Help: "Total number of log messages written",
				},
				[]string{"level", "sink"},
			),
			LogsDropped: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "logs_dropped_total",
					Help: "Total number of log messages dropped",
				},
				[]string{"sink", "reason"},
			),
			ESBulkRetries: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "es_bulk_retries_total",
					Help: "Total number of Elasticsearch bulk retries",
				},
				[]string{"reason"},
			),
			ESQueueDepth: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "es_queue_depth",
					Help: "Current depth of Elasticsearch bulk queue",
				},
				[]string{"service"},
			),
			ESBulkLatency: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "es_bulk_latency_seconds",
					Help:    "Latency of Elasticsearch bulk operations",
					Buckets: prometheus.DefBuckets,
				},
				[]string{"operation", "status"},
			),
		}
	})
	return metrics
}

// MetricsCollectors returns all metric collectors for manual registration
func MetricsCollectors() []prometheus.Collector {
	m := GetMetrics()
	return []prometheus.Collector{
		m.LogsWritten,
		m.LogsDropped,
		m.ESBulkRetries,
		m.ESQueueDepth,
		m.ESBulkLatency,
	}
}

// AutoRegisterMetrics automatically registers metrics with prometheus.DefaultRegisterer
func AutoRegisterMetrics() error {
	collectors := MetricsCollectors()
	for _, collector := range collectors {
		if err := prometheus.Register(collector); err != nil {
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				return err
			}
		}
	}
	return nil
}

// RecordLogWritten records a log message being written
func (m *Metrics) RecordLogWritten(level, sink string) {
	if m != nil && m.LogsWritten != nil {
		m.LogsWritten.WithLabelValues(level, sink).Inc()
	}
}

// RecordLogDropped records a log message being dropped
func (m *Metrics) RecordLogDropped(sink, reason string) {
	if m != nil && m.LogsDropped != nil {
		m.LogsDropped.WithLabelValues(sink, reason).Inc()
	}
}

// RecordESBulkRetry records an Elasticsearch bulk retry
func (m *Metrics) RecordESBulkRetry(reason string) {
	if m != nil && m.ESBulkRetries != nil {
		m.ESBulkRetries.WithLabelValues(reason).Inc()
	}
}

// SetESQueueDepth sets the current Elasticsearch queue depth
func (m *Metrics) SetESQueueDepth(service string, depth float64) {
	if m != nil && m.ESQueueDepth != nil {
		m.ESQueueDepth.WithLabelValues(service).Set(depth)
	}
}

// RecordESBulkLatency records the latency of an Elasticsearch bulk operation
func (m *Metrics) RecordESBulkLatency(operation, status string, latency float64) {
	if m != nil && m.ESBulkLatency != nil {
		m.ESBulkLatency.WithLabelValues(operation, status).Observe(latency)
	}
}
