package zapx

import (
	"fmt"
	logger "github.com/HoangAnhNguyen269/loggerkit"
	"go.uber.org/zap/zapcore"
)

type coreBuilder struct {
	opts    logger.Options
	encCfg  zapcore.EncoderConfig
	lvl     zapcore.Level
	metrics *logger.Metrics
}

// provider/zapx/core_builder.go
func (cb *coreBuilder) buildCores() ([]zapcore.Core, []func() error, error) {
	var cores []zapcore.Core
	var closers []func() error

	reg := getRegistry() // default or injected by tests
	for _, factory := range reg.All() {
		if !factory.Enabled(cb.opts) {
			continue
		}
		core, closer, err := factory.Build(cb.encCfg, cb.lvl, cb.metrics, cb.opts)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build %s core: %w", factory.Name(), err)
		}
		if core != nil {
			core = NewMetricsCore(core, factory.Name(), cb.metrics)
			cores = append(cores, core)
		}
		if closer != nil {
			closers = append(closers, closer)
		}
	}

	return cores, closers, nil
}
