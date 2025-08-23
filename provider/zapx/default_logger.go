package zapx

import (
	"fmt"
	logger "github.com/HoangAnhNguyen269/loggerkit"
)

// DefaultConfig chỉ bật console+JSON ở mức Info
func DefaultConfig() *logger.Config {
	return &logger.Config{
		Level:          logger.InfoLevel,
		JSON:           true,
		ConsoleEnabled: true,
		FileConfig:     nil,
		ElasticConfig:  nil,
	}
}

// NewDefaultLogger khởi một Logger dùng DefaultConfig và panic nếu lỗi
func NewDefaultLogger() logger.Logger {
	// Use development defaults for the default logger
	opts := logger.DefaultDevelopmentOptions()
	log, err := NewWithOptions(opts)
	if err != nil {
		panic(fmt.Sprintf("failed to create default logger: %v", err))
	}
	return log
}
