package zapx

import (
	"bytes"
	"fmt"
	logger "github.com/HoangAnhNguyen269/loggerkit"
	"github.com/elastic/go-elasticsearch/v8"
	"go.uber.org/zap/zapcore"
)

type elasticProvider struct{}

func (elasticProvider) Provide(cfg *logger.Config, encCfg zapcore.EncoderConfig, lvl zapcore.Level) (zapcore.Core, error) {
	ec := cfg.ElasticConfig
	if ec == nil {
		return nil, nil
	}

	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{ec.URL},
	})
	if err != nil {
		return nil, fmt.Errorf("init ES client: %w", err)
	}

	writer := &elasticSyncer{client: client, index: ec.Index}
	enc := zapcore.NewJSONEncoder(encCfg) // ES luôn dùng JSON
	return zapcore.NewCore(enc, zapcore.AddSync(writer), lvl), nil
}

type elasticSyncer struct {
	client *elasticsearch.Client
	index  string
}

func (e *elasticSyncer) Write(p []byte) (int, error) {
	_, err := e.client.Index(e.index, bytes.NewReader(p))
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (e *elasticSyncer) Sync() error { return nil }
