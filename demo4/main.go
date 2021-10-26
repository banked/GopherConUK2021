package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
	log.Logger = zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	// Create the exporter
	exp, err := jaeger.New(jaeger.WithAgentEndpoint())
	if err != nil {
		panic(err)
	}

	// Define resource attributes
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("demo4-service"),
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
	tracer = provider.Tracer("github.com/banked/gopherconuk2021/demo4")
}

func main() {
	ctx := context.Background()

	defer func() {
		if err := provider.Shutdown(ctx); err != nil {
			log.Error().Err(err).Send()
		}
	}()

	srv := &http.Server{
		Addr: "localhost:4000",
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
				log.Error().Err(err).Send()
			}
		}
	}()

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigC
	log.Debug().Stringer("signal", sig).Send()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Send()
	}

	<-doneC
}

func handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		ctx, span := tracer.Start(ctx, "handler")
		defer span.End()

		foo(ctx)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("demo4-service"))
	})
}

func foo(ctx context.Context) {
	_, span := tracer.Start(ctx, "foo")
	defer span.End()

	l := log.Hook(SpanLogHook(span))
	l.Debug().Send()

	// Simulate work
	time.Sleep(time.Second)
}

// SpanLogHook adds a gook into a zerolog.Logger if the span is
// recording and has a valid TraceID and SpanID.
func SpanLogHook(span trace.Span) zerolog.Hook {
	return zerolog.HookFunc(func(event *zerolog.Event, _ zerolog.Level, _ string) {
		if span.IsRecording() {
			ctx := span.SpanContext()

			if ctx.HasTraceID() {
				event.Str("traceID", ctx.TraceID().String())
			}

			if ctx.HasSpanID() {
				event.Str("spanID", ctx.SpanID().String())
			}
		}
	})
}
