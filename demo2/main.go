package main

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer   trace.Tracer
	provider *sdktrace.TracerProvider
)

func init() {
	// Create the exporter
	exp, err := jaeger.New(jaeger.WithAgentEndpoint())
	if err != nil {
		panic(err)
	}

	// Define resource attributes
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("demo2-service"),
		semconv.ServiceVersionKey.String("1.0.0"),
		semconv.DeploymentEnvironmentKey.String("production"),
		attribute.Int64("ID", 1234),
	)

	// Create the trace provider with the exporter and resources
	provider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp), // Always be sure to batch in production.
		sdktrace.WithResource(resource),
	)

	// Set the tracer for the package
	tracer = provider.Tracer("github.com/banked/gopherconuk21/demo1")
}

func main() {
	ctx := context.Background()

	defer func() {
		if err := provider.Shutdown(ctx); err != nil {
			log.Println(err)
		}
	}()

	// Get the tracer and start a span
	ctx, span := tracer.Start(ctx, "main")
	defer span.End()

	foo(ctx)
}

func foo(ctx context.Context) {
	_, span := tracer.Start(ctx, "foo")
	defer span.End()

	log.Println(span.SpanContext().TraceID())

	// Simulate work
	time.Sleep(time.Second)
}
