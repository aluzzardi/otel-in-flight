package main

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	otrace "go.opentelemetry.io/otel/trace"
)

func main() {
	ctx := context.Background()

	exp, err := stdouttrace.New()
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
	_, span = tr.Start(ctx, "hello")
	defer span.End()

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
