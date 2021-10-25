package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer     trace.Tracer
	provider   *sdktrace.TracerProvider
	propagator = b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader))
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
		semconv.ServiceNameKey.String("a-service"),
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
	tracer = provider.Tracer("github.com/banked/gopherconuk21/demo3/a-service")
}

func main() {
	ctx := context.Background()

	defer func() {
		if err := provider.Shutdown(ctx); err != nil {
			log.Println(err)
		}
	}()

	srv := &http.Server{
		Addr: "localhost:3000",
		Handler: otelhttp.NewHandler(
			handler(),
			"http.server",
			otelhttp.WithPropagators(propagator)),
	}

	doneC := make(chan struct{})

	go func() {
		defer close(doneC)

		if err := srv.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				log.Println(err)
			}
		}
	}()

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGINT, syscall.SIGTERM)
	log.Println(<-sigC)

	if err := srv.Shutdown(ctx); err != nil {
		log.Println(err)
	}

	<-doneC
}

func handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		ctx, span := tracer.Start(ctx, "handler")
		defer span.End()

		log.Println(span.SpanContext().TraceID())

		if err := serviceB(ctx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("a-service"))
	})
}

func client() *http.Client {
	return &http.Client{
		Transport: otelhttp.NewTransport(
			http.DefaultTransport,
			otelhttp.WithPropagators(propagator),
		),
	}
}

func serviceB(ctx context.Context) error {
	_, span := tracer.Start(ctx, "serviceB")
	defer span.End()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"http://localhost:4000",
		nil,
	)
	if err != nil {
		return err
	}

	_, err = client().Do(req)
	if err != nil {
		return err
	}

	return nil
}
