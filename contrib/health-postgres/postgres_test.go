package healthpostgres

import (
	"context"
	"errors"
	"testing"

	"github.com/ubgo/health"
)

type fakePinger struct{ err error }

func (f *fakePinger) Ping(_ context.Context) error { return f.err }

func TestCheck_Up(t *testing.T) {
	c := New("db", &fakePinger{err: nil})
	r := c.Check(context.Background())
	if r.Status != health.StatusUp {
		t.Errorf("Status: got %q, want %q", r.Status, health.StatusUp)
	}
	if r.Severity != health.SeverityCritical {
		t.Errorf("Severity: got %q", r.Severity)
	}
	if r.Latency <= 0 {
		t.Errorf("Latency: got %v, want >0", r.Latency)
	}
}

func TestCheck_Down(t *testing.T) {
	c := New("db", &fakePinger{err: errors.New("conn refused")})
	r := c.Check(context.Background())
	if r.Status != health.StatusDown {
		t.Errorf("Status: got %q, want %q", r.Status, health.StatusDown)
	}
	if r.Error != "conn refused" {
		t.Errorf("Error: got %q", r.Error)
	}
}

func TestName(t *testing.T) {
	c := New("primary", &fakePinger{})
	if c.Name() != "primary" {
		t.Errorf("Name: got %q", c.Name())
	}
}
