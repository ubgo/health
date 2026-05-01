// Package healthotel emits OpenTelemetry spans for every check the registry
// runs, by subscribing as a health.Observer.
package healthotel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ubgo/health"
)

// DefaultTracerName is the OTEL tracer instrumentation name used when no
// override is supplied via WithTracerName.
const DefaultTracerName = "github.com/ubgo/health"

// DefaultSpanName is the span name created for each health check.
const DefaultSpanName = "health.check"

// Option configures Register.
type Option func(*config)

type config struct {
	tracerName string
	spanName   string
	tp         trace.TracerProvider
}

// WithTracerName overrides the OTEL tracer instrumentation name.
func WithTracerName(name string) Option {
	return func(c *config) { c.tracerName = name }
}

// WithSpanName overrides the span name used for each emitted check span.
func WithSpanName(name string) Option {
	return func(c *config) { c.spanName = name }
}

// WithTracerProvider overrides the global OTEL tracer provider used to
// resolve the tracer. Useful for test isolation with sdktrace.
func WithTracerProvider(tp trace.TracerProvider) Option {
	return func(c *config) { c.tp = tp }
}

// Register subscribes an observer to reg that emits an OTEL span per check
// invocation. The span carries attributes for check.name, check.status,
// check.severity, and check.latency_ms; on a Down result the span status is
// set to Error and the result.Error string is recorded.
func Register(reg *health.Registry, opts ...Option) {
	cfg := &config{
		tracerName: DefaultTracerName,
		spanName:   DefaultSpanName,
		tp:         otel.GetTracerProvider(),
	}
	for _, o := range opts {
		o(cfg)
	}
	tracer := cfg.tp.Tracer(cfg.tracerName)

	reg.Subscribe(func(name string, r health.Result) {
		_, span := tracer.Start(context.Background(), cfg.spanName,
			trace.WithAttributes(
				attribute.String("check.name", name),
				attribute.String("check.status", string(r.Status)),
				attribute.String("check.severity", string(r.Severity)),
				attribute.Int64("check.latency_ms", r.Latency.Milliseconds()),
			),
		)
		if r.Status == health.StatusDown {
			span.SetStatus(codes.Error, r.Error)
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()
	})
}
