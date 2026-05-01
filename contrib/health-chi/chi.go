// Package healthchi exposes a health.Registry over Chi.
package healthchi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ubgo/health"
)

// Liveness returns an http.Handler serving the liveness snapshot.
func Liveness(reg *health.Registry) http.Handler { return handlerFor(reg, health.ProbeLiveness) }

// Readiness returns an http.Handler serving the readiness snapshot.
func Readiness(reg *health.Registry) http.Handler { return handlerFor(reg, health.ProbeReadiness) }

// Startup returns an http.Handler serving the startup snapshot.
func Startup(reg *health.Registry) http.Handler { return handlerFor(reg, health.ProbeStartup) }

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

// Middleware is the chi-compatible middleware shape.
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

// WithMiddleware applies user middleware to all three handlers.
func WithMiddleware(mw ...Middleware) MountOption {
	return func(c *mountConfig) { c.middleware = append(c.middleware, mw...) }
}

// Mount registers Liveness, Readiness, Startup on r at /healthz, /readyz,
// /startupz (or paths overridden via Options).
func Mount(r chi.Router, reg *health.Registry, opts ...MountOption) {
	cfg := &mountConfig{
		livenessPath:  "/healthz",
		readinessPath: "/readyz",
		startupPath:   "/startupz",
	}
	for _, o := range opts {
		o(cfg)
	}
	r.Group(func(sub chi.Router) {
		for _, mw := range cfg.middleware {
			sub.Use(mw)
		}
		sub.Method(http.MethodGet, cfg.livenessPath, Liveness(reg))
		sub.Method(http.MethodGet, cfg.readinessPath, Readiness(reg))
		sub.Method(http.MethodGet, cfg.startupPath, Startup(reg))
	})
}
