module github.com/HoangAnhNguyen269/loggerkit

go 1.23.0

toolchain go1.24.2

require (
	github.com/elastic/go-elasticsearch/v8 v8.19.0
	github.com/prometheus/client_golang v1.23.0
	go.opentelemetry.io/otel v1.28.0
	go.opentelemetry.io/otel/trace v1.28.0
	go.uber.org/zap v1.27.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/elastic/elastic-transport-go/v8 v8.7.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.65.0 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	go.opentelemetry.io/otel/metric v1.28.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

retract (
	v1.0.1 // invalid release, use v0.0.1
	v1.0.0 // invalid release, use v0.0.1
)
