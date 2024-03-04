package log

import (
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/sdk/instrumentation"
)

var _ log.LoggerProvider = &LoggerProvider{}

type LoggerProvider struct {
	embedded.LoggerProvider

	mu         sync.RWMutex
	isShutdown atomic.Bool
	processors []LogProcessor
}

func NewLoggerProvider() *LoggerProvider {
	return &LoggerProvider{}
}

func (p *LoggerProvider) Logger(name string, opts ...log.LoggerOption) log.Logger {
	if p.isShutdown.Load() {
		return noop.NewLoggerProvider().Logger(name, opts...)
	}

	c := log.NewLoggerConfig(opts...)
	is := instrumentation.Scope{
		Name:      name,
		Version:   c.InstrumentationVersion(),
		SchemaURL: c.SchemaURL(),
	}

	return &logger{
		provider:             p,
		instrumentationScope: is,
	}
}

func (p *LoggerProvider) RegisterSpanProcessor(lp LogProcessor) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.isShutdown.Load() {
		return
	}

	p.processors = append(p.processors, lp)
}

func (p *LoggerProvider) getLogProcessors() []LogProcessor {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.processors
}
