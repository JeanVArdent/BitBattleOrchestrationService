package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
		slog.WarnContext(ctx, "Otel Setup Failed",
			"error", err,
		)
	}

	defer func() {
		err = errors.Join(err, otelShutdown(ctx))

		slog.InfoContext(ctx, "Application Closing", "error", err)
	}()

	//TESTING CODE
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
		slog.ErrorContext(ctx, "Error Encountered",
			"error", err)
	}

	// Work begins
	tCtx, span := tracer.Start(
		ctx,
		"CollectorExporter-Example",
		trace.WithAttributes(commonAttrs...))

	for i := 0; i < 10; i++ {
		iCtx, iSpan := tracer.Start(tCtx, fmt.Sprintf("Sample-%d", i))
		runCount.Add(ctx, 1, metric.WithAttributes(commonAttrs...))
		slog.InfoContext(iCtx, fmt.Sprintf("Doing really hard work (%d / 10)\n", i+1), "count", i+1)

		<-time.After(time.Second)
		iSpan.End()
	}

	slog.InfoContext(tCtx, "Done!")

	<-time.After(time.Second)
	span.End()

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		slog.InfoContext(r.Context(), "Handling request",
			"path", r.URL.Path,
			"method", r.Method,
			"remote_addr", r.RemoteAddr,
		)
		runCount.Add(r.Context(), 1, metric.WithAttributes(commonAttrs...))
		fmt.Fprintf(w, "Hello World")
	})

	attributesFn := func(r *http.Request) []attribute.KeyValue {
		return []attribute.KeyValue{
			attribute.String("path", r.URL.Path),
			attribute.String("method", r.Method),
			attribute.String("remote_addr", r.RemoteAddr),
		}
	}

	handler := otelhttp.NewHandler(mux, "/", otelhttp.WithMetricAttributesFn(attributesFn))

	slog.InfoContext(ctx, "Starting server", "port", 8080)
	if err = http.ListenAndServe(":8080", handler); err != nil {
		slog.ErrorContext(ctx, "Server failed", "error", err)
		os.Exit(1)
	}
}
