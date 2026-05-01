// Package healthredis implements a health.Checker for a Redis client by
// running the PING command.
package healthredis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ubgo/health"
)

// Pinger is the minimal go-redis interface this checker depends on. Both
// *redis.Client and *redis.ClusterClient satisfy it.
type Pinger interface {
	Ping(ctx context.Context) *redis.StatusCmd
}

// Checker is a health.Checker that pings a Redis client.
type Checker struct {
	name string
	p    Pinger
}

// New returns a Checker named name that pings p.
func New(name string, p Pinger) *Checker {
	return &Checker{name: name, p: p}
}

// Name implements health.Checker.
func (c *Checker) Name() string { return c.name }

// Check implements health.Checker. It runs PING against Redis and returns
// Up if the response is "PONG" or empty (older clients), Down otherwise.
func (c *Checker) Check(ctx context.Context) health.Result {
	start := time.Now()
	err := c.p.Ping(ctx).Err()
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
