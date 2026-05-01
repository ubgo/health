// Package healthnethttp exposes a health.Registry over stdlib net/http.
package healthnethttp

import (
	"net/http"

	"github.com/ubgo/health"
)

// Liveness returns an http.Handler that responds with the registry's
// liveness snapshot as JSON.
func Liveness(reg *health.Registry) http.Handler {
	return handlerFor(reg, health.ProbeLiveness)
}

// Readiness returns an http.Handler that responds with the registry's
// readiness snapshot as JSON.
func Readiness(reg *health.Registry) http.Handler {
	return handlerFor(reg, health.ProbeReadiness)
}

// Startup returns an http.Handler that responds with the registry's
// startup snapshot as JSON.
func Startup(reg *health.Registry) http.Handler {
	return handlerFor(reg, health.ProbeStartup)
}

func handlerFor(reg *health.Registry, probe health.Probe) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		snap := reg.SnapshotForProbe(probe)
		body, err := snap.JSON()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(snap.HTTPStatus())
		_, _ = w.Write(body)
	})
}

// Middleware is the stdlib net/http middleware shape.
type Middleware = func(http.Handler) http.Handler

// MountOption configures Mount.
type MountOption func(*mountConfig)

type mountConfig struct {
	livenessPath  string
	readinessPath string
	startupPath   string
	middleware    []Middleware
}

// WithLivenessPath overrides the default `/healthz` route.
func WithLivenessPath(p string) MountOption {
	return func(c *mountConfig) { c.livenessPath = p }
}

// WithReadinessPath overrides the default `/readyz` route.
func WithReadinessPath(p string) MountOption {
	return func(c *mountConfig) { c.readinessPath = p }
}

// WithStartupPath overrides the default `/startupz` route.
func WithStartupPath(p string) MountOption {
	return func(c *mountConfig) { c.startupPath = p }
}

// WithMiddleware applies user middleware to all three handlers in declaration order.
func WithMiddleware(mw ...Middleware) MountOption {
	return func(c *mountConfig) { c.middleware = append(c.middleware, mw...) }
}

// Mount registers Liveness, Readiness, Startup on mux at /healthz, /readyz,
// /startupz (or paths overridden via Options), wrapping each with any
// middleware supplied via WithMiddleware.
func Mount(mux *http.ServeMux, reg *health.Registry, opts ...MountOption) {
	cfg := &mountConfig{
		livenessPath:  "/healthz",
		readinessPath: "/readyz",
		startupPath:   "/startupz",
	}
	for _, o := range opts {
		o(cfg)
	}
	mux.Handle(cfg.livenessPath, applyMW(Liveness(reg), cfg.middleware))
	mux.Handle(cfg.readinessPath, applyMW(Readiness(reg), cfg.middleware))
	mux.Handle(cfg.startupPath, applyMW(Startup(reg), cfg.middleware))
}

func applyMW(h http.Handler, mw []Middleware) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}
