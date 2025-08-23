package zapx

import (
	logger "github.com/HoangAnhNguyen269/loggerkit"
	"go.uber.org/zap/zapcore"
)

type metricsCore struct {
	inner   zapcore.Core
	sink    string
	metrics *logger.Metrics
}

func NewMetricsCore(inner zapcore.Core, sink string, m *logger.Metrics) zapcore.Core {
	if m == nil {
		return inner
	}
	return &metricsCore{inner: inner, sink: sink, metrics: m}
}

func (m *metricsCore) Enabled(l zapcore.Level) bool { return m.inner.Enabled(l) }

func (m *metricsCore) With(fields []zapcore.Field) zapcore.Core {
	return &metricsCore{
		inner:   m.inner.With(fields),
		sink:    m.sink,
		metrics: m.metrics,
	}
}

func (m *metricsCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	return m.inner.Check(ent, ce)
}

func (m *metricsCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	err := m.inner.Write(ent, fields)
	if m.metrics == nil {
		return err
	}
	if err == nil {
		m.metrics.RecordLogWritten(ent.Level.String(), m.sink)
	} else {
		m.metrics.RecordLogDropped(m.sink, "write_error")
	}
	return err
}

func (m *metricsCore) Sync() error { return m.inner.Sync() }
