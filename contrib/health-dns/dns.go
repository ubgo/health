// Package healthdns implements a health.Checker that resolves a DNS host.
//
// Useful for verifying DNS resolution paths inside k8s clusters where a
// CoreDNS regression manifests before any application-level dependency does.
package healthdns

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ubgo/health"
)

// Resolver is the minimal interface this checker depends on. The default
// is net.DefaultResolver.
type Resolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
}

// Option configures the Checker.
type Option func(*config)

type config struct {
	resolver Resolver
	minHosts int
}

// WithResolver overrides the resolver. Useful for tests or custom DNS servers.
func WithResolver(r Resolver) Option {
	return func(c *config) { c.resolver = r }
}

// WithMinHosts sets a floor on how many resolved addresses are considered
// healthy. Default: 1.
func WithMinHosts(n int) Option {
	return func(c *config) { c.minHosts = n }
}

// Checker is a health.Checker that resolves a DNS host.
type Checker struct {
	name string
	host string
	cfg  *config
}

// New constructs a Checker that resolves host via the configured resolver.
func New(name, host string, opts ...Option) *Checker {
	cfg := &config{resolver: net.DefaultResolver, minHosts: 1}
	for _, o := range opts {
		o(cfg)
	}
	return &Checker{name: name, host: host, cfg: cfg}
}

// Name implements health.Checker.
func (c *Checker) Name() string { return c.name }

// Check implements health.Checker. Reports Up when the resolver returns at
// least minHosts addresses, Down on resolver error or below the threshold.
func (c *Checker) Check(ctx context.Context) health.Result {
	start := time.Now()
	r := health.Result{Severity: health.SeverityCritical, Timestamp: time.Now()}

	addrs, err := c.cfg.resolver.LookupHost(ctx, c.host)
	r.Latency = time.Since(start)
	if err != nil {
		r.Status = health.StatusDown
		r.Error = err.Error()
		return r
	}
	if len(addrs) < c.cfg.minHosts {
		r.Status = health.StatusDown
		r.Error = fmt.Sprintf("resolved %d addresses, want >=%d", len(addrs), c.cfg.minHosts)
		return r
	}
	r.Status = health.StatusUp
	r.Metadata = map[string]any{"resolved_count": len(addrs)}
	return r
}
