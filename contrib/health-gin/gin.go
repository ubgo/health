// Package healthgin exposes a health.Registry over Gin.
package healthgin

import (
	"github.com/gin-gonic/gin"

	"github.com/ubgo/health"
)

// Liveness returns a gin.HandlerFunc serving the liveness snapshot.
func Liveness(reg *health.Registry) gin.HandlerFunc { return handlerFor(reg, health.ProbeLiveness) }

// Readiness returns a gin.HandlerFunc serving the readiness snapshot.
func Readiness(reg *health.Registry) gin.HandlerFunc { return handlerFor(reg, health.ProbeReadiness) }

// Startup returns a gin.HandlerFunc serving the startup snapshot.
func Startup(reg *health.Registry) gin.HandlerFunc { return handlerFor(reg, health.ProbeStartup) }

func handlerFor(reg *health.Registry, probe health.Probe) gin.HandlerFunc {
	return func(c *gin.Context) {
		snap := reg.SnapshotForProbe(probe)
		c.JSON(snap.HTTPStatus(), snap)
	}
}

// MountOption configures Mount.
type MountOption func(*mountConfig)

type mountConfig struct {
	livenessPath  string
	readinessPath string
	startupPath   string
	middleware    []gin.HandlerFunc
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
func WithMiddleware(mw ...gin.HandlerFunc) MountOption {
	return func(c *mountConfig) { c.middleware = append(c.middleware, mw...) }
}

// Mount registers Liveness, Readiness, Startup on r at /healthz, /readyz,
// /startupz (or paths overridden via Options), wrapping each with any
// middleware supplied via WithMiddleware.
func Mount(r gin.IRouter, reg *health.Registry, opts ...MountOption) {
	cfg := &mountConfig{
		livenessPath:  "/healthz",
		readinessPath: "/readyz",
		startupPath:   "/startupz",
	}
	for _, o := range opts {
		o(cfg)
	}
	grp := r.Group("/", cfg.middleware...)
	grp.GET(cfg.livenessPath, Liveness(reg))
	grp.GET(cfg.readinessPath, Readiness(reg))
	grp.GET(cfg.startupPath, Startup(reg))
}
