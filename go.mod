module github.com/banked/gopherconuk21

go 1.17

require (
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.25.0
	go.opentelemetry.io/otel v1.0.1
	go.opentelemetry.io/otel/exporters/jaeger v1.0.1
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.0.1
	go.opentelemetry.io/otel/sdk v1.0.1
	go.opentelemetry.io/otel/trace v1.0.1
)

require (
	github.com/felixge/httpsnoop v1.0.2 // indirect
	go.opentelemetry.io/contrib/propagators/b3 v1.0.0 // indirect
	go.opentelemetry.io/otel/internal/metric v0.24.0 // indirect
	go.opentelemetry.io/otel/metric v0.24.0 // indirect
	golang.org/x/sys v0.0.0-20210423185535-09eb48e85fd7 // indirect
)
