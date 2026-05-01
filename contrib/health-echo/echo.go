// Package healthecho exposes a health.Registry over Echo.
package healthecho

import (
	"github.com/labstack/echo/v4"

	"github.com/ubgo/health"
)

// Liveness returns an echo.HandlerFunc serving the liveness snapshot.
func Liveness(reg *health.Registry) echo.HandlerFunc {
	return handlerFor(reg, health.ProbeLiveness)
}

// Readiness returns an echo.HandlerFunc serving the readiness snapshot.
func Readiness(reg *health.Registry) echo.HandlerFunc {
	return handlerFor(reg, health.ProbeReadiness)
}

// Startup returns an echo.HandlerFunc serving the startup snapshot.
func Startup(reg *health.Registry) echo.HandlerFunc {
	return handlerFor(reg, health.ProbeStartup)
}

func handlerFor(reg *health.Registry, probe health.Probe) echo.HandlerFunc {
	return func(c echo.Context) error {
		snap := reg.SnapshotForProbe(probe)
		return c.JSON(snap.HTTPStatus(), snap)
	}
}

// MountOption configures Mount.
type MountOption func(*mountConfig)

type mountConfig struct {
	livenessPath  string
	readinessPath string
	startupPath   string
	middleware    []echo.MiddlewareFunc
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
func WithMiddleware(mw ...echo.MiddlewareFunc) MountOption {
	return func(c *mountConfig) { c.middleware = append(c.middleware, mw...) }
}

// Mount registers Liveness, Readiness, Startup on e at /healthz, /readyz,
// /startupz (or paths overridden via Options).
func Mount(e *echo.Echo, reg *health.Registry, opts ...MountOption) {
	cfg := &mountConfig{
		livenessPath:  "/healthz",
		readinessPath: "/readyz",
		startupPath:   "/startupz",
	}
	for _, o := range opts {
		o(cfg)
	}
	grp := e.Group("", cfg.middleware...)
	grp.GET(cfg.livenessPath, Liveness(reg))
	grp.GET(cfg.readinessPath, Readiness(reg))
	grp.GET(cfg.startupPath, Startup(reg))
}
