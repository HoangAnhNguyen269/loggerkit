package zapx

import (
	logger "github.com/HoangAnhNguyen269/loggerkit"
	"go.uber.org/zap/zapcore"
)

// Provider là strategy để build một zapcore.Core từ Config
type Provider interface {
	Provide(cfg *logger.Config, encCfg zapcore.EncoderConfig, lvl zapcore.Level) (zapcore.Core, error)
}

// ProviderRegistry quản lý danh sách providers
type ProviderRegistry struct {
	providers []Provider
}

// NewProviderRegistry tạo registry mới
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make([]Provider, 0),
	}
}

// Register đăng ký một Provider mới
func (r *ProviderRegistry) Register(p Provider) {
	r.providers = append(r.providers, p)
}

// GetProviders trả về danh sách providers
func (r *ProviderRegistry) GetProviders() []Provider {
	return r.providers
}

// BuildCores build tất cả cores từ các providers
func (r *ProviderRegistry) BuildCores(cfg *logger.Config, encCfg zapcore.EncoderConfig, lvl zapcore.Level) ([]zapcore.Core, error) {
	var cores []zapcore.Core

	for _, p := range r.providers {
		core, err := p.Provide(cfg, encCfg, lvl)
		if err != nil {
			return nil, err
		}
		if core != nil {
			cores = append(cores, core)
		}
	}

	return cores, nil
}

// DefaultProviderRegistry tạo registry với built-in providers
func DefaultProviderRegistry() *ProviderRegistry {
	registry := NewProviderRegistry()

	// Đăng ký built-in providers
	registry.Register(&consoleProvider{})
	registry.Register(&fileProvider{})
	registry.Register(&elasticProvider{})

	return registry
}
