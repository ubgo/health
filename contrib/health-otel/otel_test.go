package healthotel

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/ubgo/health"
)

type fakeChecker struct {
	name string
	res  health.Result
}

func (f *fakeChecker) Name() string                          { return f.name }
func (f *fakeChecker) Check(_ context.Context) health.Result { return f.res }

func TestRegister_EmitsSpanPerCheck(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	defer func() { _ = tp.Shutdown(context.Background()) }()

	reg := health.NewRegistry()
	Register(reg, WithTracerProvider(tp))
	_ = reg.Register(&fakeChecker{name: "db", res: health.Result{Status: health.StatusUp}})
	_ = reg.Register(&fakeChecker{name: "cache", res: health.Result{Status: health.StatusDown, Error: "boom"}})

	reg.RunAll(context.Background())

	spans := exp.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("spans: got %d, want 2", len(spans))
	}
	for _, s := range spans {
		if s.Name != DefaultSpanName {
			t.Errorf("span name: got %q, want %q", s.Name, DefaultSpanName)
		}
		var hasName, hasStatus bool
		for _, a := range s.Attributes {
			if a.Key == "check.name" {
				hasName = true
			}
			if a.Key == "check.status" {
				hasStatus = true
			}
		}
		if !hasName || !hasStatus {
			t.Errorf("span missing attributes: %+v", s.Attributes)
		}
	}
}

func TestRegister_DownResultRecordsError(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	defer func() { _ = tp.Shutdown(context.Background()) }()

	reg := health.NewRegistry()
	Register(reg, WithTracerProvider(tp))
	_ = reg.Register(&fakeChecker{name: "x", res: health.Result{Status: health.StatusDown, Error: "boom"}})
	reg.RunAll(context.Background())

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("spans: got %d, want 1", len(spans))
	}
	if spans[0].Status.Description != "boom" {
		t.Errorf("status description: got %q, want %q", spans[0].Status.Description, "boom")
	}
}

func TestRegister_CustomSpanName(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	defer func() { _ = tp.Shutdown(context.Background()) }()

	reg := health.NewRegistry()
	Register(reg, WithTracerProvider(tp), WithTracerName("custom"), WithSpanName("custom.span"))
	_ = reg.Register(&fakeChecker{name: "x", res: health.Result{Status: health.StatusUp}})
	reg.RunAll(context.Background())

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("spans: got %d, want 1", len(spans))
	}
	if spans[0].Name != "custom.span" {
		t.Errorf("name override: got %q", spans[0].Name)
	}
}
