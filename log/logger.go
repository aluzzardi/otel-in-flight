package log

import (
	"context"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/sdk/instrumentation"
)

var _ log.Logger = &logger{}

type logger struct {
	embedded.Logger

	provider             *LoggerProvider
	instrumentationScope instrumentation.Scope
}

func (l logger) Emit(ctx context.Context, r log.Record) {
	for _, proc := range l.provider.getLogProcessors() {
		proc.OnEmit(ctx, r)
	}
}
