package logger

// Level định nghĩa cấp độ log
type Level string

const (
	DebugLevel Level = "debug"
	InfoLevel  Level = "info"
	WarnLevel  Level = "warn"
	ErrorLevel Level = "error"
)

// FileConfig chứa cấu hình khi ghi ra file
type FileConfig struct {
	Filename   string // đường dẫn file log
	MaxSize    int    // MB
	MaxBackups int
	MaxAge     int  // ngày
	Compress   bool // nén file cũ
}

// ElasticConfig chứa cấu hình khi đẩy log vào Elasticsearch
type ElasticConfig struct {
	URL   string // ví dụ "http://es.local:9200"
	Index string // ví dụ "app-logs"
}

// Config gom tất cả option cho các sink
type Config struct {
	Level          Level
	JSON           bool           // true: JSON encoder; false: console encoder
	ConsoleEnabled bool           // bật console sink
	FileConfig     *FileConfig    // nếu != nil thì thêm file sink
	ElasticConfig  *ElasticConfig // nếu != nil thì thêm ES sink
}

// DefaultConfig chỉ bật console+JSON ở mức Info
func DefaultConfig() *Config {
	return &Config{
		Level:          InfoLevel,
		JSON:           true,
		ConsoleEnabled: true,
		FileConfig:     nil,
		ElasticConfig:  nil,
	}
}
