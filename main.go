package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aluzzardi/otel-in-flight/log"
	"github.com/aluzzardi/otel-in-flight/log/otlploghttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	olog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	otrace "go.opentelemetry.io/otel/trace"
)

func main() {
	ctx := context.Background()

	var (
		exp trace.SpanExporter
		err error
	)

	if token := os.Getenv("DAGGER_CLOUD_TOKEN"); token != "" {
		exp, err = otlptracehttp.New(ctx,
			otlptracehttp.WithInsecure(),
			otlptracehttp.WithEndpoint("localhost:8020"),
			otlptracehttp.WithHeaders(map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", token),
			}),
			otlptracehttp.WithCompression(otlptracehttp.NoCompression), // FIXME... ? http.Client should compress anyway?
		)
	} else {
		exp, err = stdouttrace.New()
	}
	if err != nil {
		panic(err)
	}

	processor := NewBatchSpanProcessor(exp, WithBatchTimeout(1*time.Second))

	tp := trace.NewTracerProvider(
		// We're using our own custom batcher instead
		// trace.WithBatcher(exp),
		trace.WithSpanProcessor(processor),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("dagger"),
			semconv.ServiceVersionKey.String("v0.1.0"),
		)),
	)
	defer tp.Shutdown(context.Background())
	otel.SetTracerProvider(NewProxyTraceProvider(tp, func(s otrace.Span) {
		if ro, ok := s.(trace.ReadOnlySpan); ok && s.IsRecording() {
			processor.OnUpdate(ro)
		}
	}))

	tr := otel.Tracer("dagger")

	var span otrace.Span
	ctx, span = tr.Start(ctx, "hello")
	defer span.End()

	testLogs(ctx)

	for i := 0; i < 5; i++ {
		span.AddEvent("event", otrace.WithAttributes(attribute.Int("i", i)))
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(1 * time.Second)

	for i := 5; i < 10; i++ {
		span.AddEvent("event 2", otrace.WithAttributes(attribute.Int("i", i)))
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(1 * time.Second)
}

func testLogs(ctx context.Context) {
	lp := log.NewLoggerProvider(
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("dagger"),
			semconv.ServiceVersionKey.String("v0.1.0"),
		),
	)
	lr := lp.Logger("dagger")
	lexp := otlploghttp.NewClient(otlploghttp.Config{
		Endpoint: "localhost:8020",
		URLPath:  "/v1/logs",
		Insecure: true,
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", os.Getenv("DAGGER_CLOUD_TOKEN")),
		},
	})
	lproc := log.NewBatchLogProcessor(lexp, log.WithBatchTimeout(1*time.Second))
	defer lproc.Shutdown(ctx)
	lp.RegisterSpanProcessor(lproc)

	for i := 0; i < 10; i++ {
		r := olog.Record{}
		r.SetBody(olog.StringValue("hello world"))
		r.SetTimestamp(time.Now())
		r.AddAttributes(olog.Int("count", i))
		lr.Emit(ctx, r)
		time.Sleep(100 * time.Millisecond)
	}
	time.Sleep(1 * time.Second)
	r := olog.Record{}
	r.SetBody(olog.StringValue("the end"))
	r.SetTimestamp(time.Now())
	lr.Emit(ctx, r)
}
