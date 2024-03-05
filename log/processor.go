package log

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
)

type LogData struct {
	log.Record

	Resource             *resource.Resource
	InstrumentationScope instrumentation.Scope

	TraceID trace.TraceID
	SpanID  trace.SpanID
}

type LogProcessor interface {
	OnEmit(context.Context, log.Record)
	Shutdown(context.Context) error
}

var _ LogProcessor = &simpleLogProcessor{}

type simpleLogProcessor struct {
	exporter LogExporter
}

func NewSimpleLogProcessor(exporter LogExporter) LogProcessor {
	return &simpleLogProcessor{
		exporter: exporter,
	}
}

func (p *simpleLogProcessor) OnEmit(ctx context.Context, r log.Record) {
	span := trace.SpanFromContext(ctx)

	log := &LogData{
		Record:  r,
		TraceID: span.SpanContext().TraceID(),
		SpanID:  span.SpanContext().SpanID(),
	}

	if err := p.exporter.ExportLogs(ctx, []*LogData{log}); err != nil {
		otel.Handle(err)
	}
}

func (bsp *simpleLogProcessor) Shutdown(ctx context.Context) error {
	return nil
}
