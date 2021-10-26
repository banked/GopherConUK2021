package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"golang.org/x/sync/errgroup"
)

var (
	ctlr     *controller.Controller
	exporter *prometheus.Exporter
	meter    metric.Meter
)

func init() {
	config := prometheus.Config{}

	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("demo5-service"),
		semconv.ServiceVersionKey.String("1.0.0"),
		semconv.DeploymentEnvironmentKey.String("production"),
		attribute.Int64("ID", 1234),
	)

	ctlr = controller.New(
		processor.NewFactory(
			selector.NewWithHistogramDistribution(
				histogram.WithExplicitBoundaries(config.DefaultHistogramBoundaries),
			),
			export.CumulativeExportKindSelector(),
			processor.WithMemory(true),
		),
		controller.WithResource(resource),
	)

	var err error

	exporter, err = prometheus.New(config, ctlr)
	if err != nil {
		log.Panicf("failed to initialize prometheus exporter %v", err)
	}

	global.SetMeterProvider(exporter.MeterProvider())

	meter = exporter.MeterProvider().Meter("github.com/banked/gopherconuk2021/demo5")
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	defer func() {
		if err := ctlr.Stop(ctx); err != nil {
			log.Println(err)
		}
	}()

	srv := &http.Server{
		Addr:    "localhost:6000",
		Handler: exporter,
	}

	var wg errgroup.Group

	wg.Go(func() error {
		return srv.ListenAndServe()
	})

	wg.Go(func() error {
		sigC := make(chan os.Signal, 1)
		signal.Notify(sigC, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigC
		log.Println(sig)

		cancel()

		return srv.Shutdown(ctx)
	})

	wg.Go(func() error {
		return makeMetrics(ctx)
	})

	if err := wg.Wait(); err != nil {
		log.Println(err)
	}
}

func makeMetrics(ctx context.Context) error {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	counter, err := meter.NewInt64Counter("demo5.counter")
	if err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
			rand.Seed(time.Now().UnixNano())
			n := rand.Int63n(10)

			counter.Add(ctx, n)
		case <-ctx.Done():
			return nil
		}
	}
}
