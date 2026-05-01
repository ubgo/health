// Package healthnats implements a health.Checker for a NATS connection.
package healthnats

import (
	"context"
	"errors"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/ubgo/health"
)

// Conn is the minimal nats.go interface this checker depends on. *nats.Conn
// satisfies it directly.
type Conn interface {
	IsConnected() bool
	Status() nats.Status
}

// Checker is a health.Checker that reports the NATS connection status.
type Checker struct {
	name string
	c    Conn
}

// New returns a Checker named name that observes c.
func New(name string, c Conn) *Checker {
	return &Checker{name: name, c: c}
}

// Name implements health.Checker.
func (c *Checker) Name() string { return c.name }

// Check implements health.Checker. Returns Up when IsConnected, Degraded
// when reconnecting, Down when closed/disconnected/draining.
func (c *Checker) Check(_ context.Context) health.Result {
	start := time.Now()
	r := health.Result{
		Severity:  health.SeverityCritical,
		Timestamp: time.Now(),
	}

	switch c.c.Status() {
	case nats.CONNECTED:
		r.Status = health.StatusUp
	case nats.RECONNECTING:
		r.Status = health.StatusDegraded
		r.Error = errors.New("nats: reconnecting").Error()
	case nats.CLOSED, nats.DISCONNECTED, nats.DRAINING_PUBS, nats.DRAINING_SUBS:
		r.Status = health.StatusDown
		r.Error = errors.New("nats: " + c.c.Status().String()).Error()
	default:
		r.Status = health.StatusUnknown
	}
	r.Latency = time.Since(start)
	return r
}
