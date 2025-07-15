package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func init() {
	_ = godotenv.Load()
}

func main() {
	ctx := context.Background()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	otelShutdown, err := setupOTel(ctx)
	if err != nil {
		slog.Warn("Otel Setup Failed",
			"error", err,
		)
	}

	defer func() {
		err = errors.Join(err, otelShutdown(ctx))

		slog.Info("Application Closing", "error", err)
	}()

	//TESTING CODE
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Handling request",
			"path", r.URL.Path,
			"method", r.Method,
			"remote_addr", r.RemoteAddr,
		)
		fmt.Fprintf(w, "Hello World")
	})

	slog.Info("Starting server", "port", 8080)
	if err = http.ListenAndServe(":8080", nil); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}

	name := "orchestration-service"
	tracer := otel.Tracer(name)
	meter := otel.Meter(name)

	// Attributes represent additional key-value descriptors that can be bound
	// to a metric observer or recorder.
	commonAttrs := []attribute.KeyValue{
		attribute.String("attrA", "chocolate"),
		attribute.String("attrB", "raspberry"),
		attribute.String("attrC", "vanilla"),
	}

	runCount, err := meter.Int64Counter("run", metric.WithDescription("The number of times the iteration ran"))
	if err != nil {
		slog.Error("Error Encountered",
			"error", err)
	}

	// Work begins
	ctx, span := tracer.Start(
		ctx,
		"CollectorExporter-Example",
		trace.WithAttributes(commonAttrs...))
	defer span.End()
	for i := 0; i < 10; i++ {
		_, iSpan := tracer.Start(ctx, fmt.Sprintf("Sample-%d", i))
		runCount.Add(ctx, 1, metric.WithAttributes(commonAttrs...))
		slog.Info(fmt.Sprintf("Doing really hard work (%d / 10)\n", i+1), "count", i+1)

		<-time.After(time.Second)
		iSpan.End()
	}

	slog.Info("Done!")
}
