// Package healthpostgres implements a health.Checker for a Postgres
// connection or pool, using pgx's Ping method.
package healthpostgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ubgo/health"
)

// Pinger is the minimal pgx interface this checker depends on. Both
// *pgxpool.Pool and *pgx.Conn satisfy it.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Checker is a health.Checker that pings a Postgres pool/connection.
type Checker struct {
	name string
	p    Pinger
}

// New returns a Checker named name that pings p.
func New(name string, p Pinger) *Checker {
	return &Checker{name: name, p: p}
}

// FromPool is a convenience constructor for the common *pgxpool.Pool case.
func FromPool(name string, pool *pgxpool.Pool) *Checker {
	return New(name, pool)
}

// Name implements health.Checker.
func (c *Checker) Name() string { return c.name }

// Check implements health.Checker. It pings the underlying connection or pool
// and reports Up on success, Down on any error.
func (c *Checker) Check(ctx context.Context) health.Result {
	start := time.Now()
	err := c.p.Ping(ctx)
	latency := time.Since(start)
	r := health.Result{
		Severity:  health.SeverityCritical,
		Latency:   latency,
		Timestamp: time.Now(),
	}
	if err != nil {
		r.Status = health.StatusDown
		r.Error = err.Error()
		return r
	}
	r.Status = health.StatusUp
	return r
}
