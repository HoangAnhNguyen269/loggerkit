package logger

// NewBuilder interface for creating loggers
type NewBuilder interface {
	NewWithOptions(opts Options) (Logger, error)
}

// Global builder instance
var globalBuilder NewBuilder

// SetBuilder sets the global logger builder
func SetBuilder(builder NewBuilder) {
	globalBuilder = builder
}

// NewDevelopment creates a development logger
func NewDevelopment(opts ...Option) (Logger, error) {
	config := DefaultDevelopmentOptions()
	for _, opt := range opts {
		opt(&config)
	}
	return globalBuilder.NewWithOptions(config)
}

// NewProduction creates a production logger
func NewProduction(opts ...Option) (Logger, error) {
	config := DefaultProductionOptions()
	for _, opt := range opts {
		opt(&config)
	}
	return globalBuilder.NewWithOptions(config)
}

// MustNew creates a logger using the legacy Config struct
func MustNew(cfg *Config) Logger {
	// Convert legacy Config to new Options
	opts := Options{
		Env:          EnvProd,
		Service:      "app",
		Level:        cfg.Level,
		TimeFormat:   "",
		EnableCaller: true,
		StacktraceAt: ErrorLevel,
	}

	if cfg.FileConfig != nil {
		opts.File = &FileSink{
			Path:       cfg.FileConfig.Filename,
			MaxSizeMB:  cfg.FileConfig.MaxSize,
			MaxBackups: cfg.FileConfig.MaxBackups,
			MaxAgeDays: cfg.FileConfig.MaxAge,
			Compress:   cfg.FileConfig.Compress,
		}
	}

	if cfg.ElasticConfig != nil {
		opts.Elastic = &ElasticSink{
			Addresses: []string{cfg.ElasticConfig.URL},
			Index:     cfg.ElasticConfig.Index,
		}
	}

	log, err := globalBuilder.NewWithOptions(opts)
	if err != nil {
		panic(err)
	}
	return log
}

// Legacy Config types for backward compatibility
type Config struct {
	Level          Level
	JSON           bool
	ConsoleEnabled bool
	FileConfig     *FileConfig
	ElasticConfig  *ElasticConfig
}

type FileConfig struct {
	Filename   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
}

type ElasticConfig struct {
	URL   string
	Index string
}

// DefaultConfig returns a default legacy config
func DefaultConfig() *Config {
	return &Config{
		Level:          InfoLevel,
		JSON:           true,
		ConsoleEnabled: true,
		FileConfig:     nil,
		ElasticConfig:  nil,
	}
}
