// Package healthfiber exposes a health.Registry over Fiber.
package healthfiber

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ubgo/health"
)

// Liveness returns a fiber.Handler serving the liveness snapshot.
func Liveness(reg *health.Registry) fiber.Handler { return handlerFor(reg, health.ProbeLiveness) }

// Readiness returns a fiber.Handler serving the readiness snapshot.
func Readiness(reg *health.Registry) fiber.Handler { return handlerFor(reg, health.ProbeReadiness) }

// Startup returns a fiber.Handler serving the startup snapshot.
func Startup(reg *health.Registry) fiber.Handler { return handlerFor(reg, health.ProbeStartup) }

func handlerFor(reg *health.Registry, probe health.Probe) fiber.Handler {
	return func(c *fiber.Ctx) error {
		snap := reg.SnapshotForProbe(probe)
		return c.Status(snap.HTTPStatus()).JSON(snap)
	}
}

// MountOption configures Mount.
type MountOption func(*mountConfig)

type mountConfig struct {
	livenessPath  string
	readinessPath string
	startupPath   string
	middleware    []fiber.Handler
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
func WithMiddleware(mw ...fiber.Handler) MountOption {
	return func(c *mountConfig) { c.middleware = append(c.middleware, mw...) }
}

// Mount registers Liveness, Readiness, Startup on r at /healthz, /readyz,
// /startupz (or paths overridden via Options).
func Mount(r fiber.Router, reg *health.Registry, opts ...MountOption) {
	cfg := &mountConfig{
		livenessPath:  "/healthz",
		readinessPath: "/readyz",
		startupPath:   "/startupz",
	}
	for _, o := range opts {
		o(cfg)
	}
	chain := func(h fiber.Handler) []fiber.Handler {
		out := append([]fiber.Handler{}, cfg.middleware...)
		return append(out, h)
	}
	r.Get(cfg.livenessPath, chain(Liveness(reg))...)
	r.Get(cfg.readinessPath, chain(Readiness(reg))...)
	r.Get(cfg.startupPath, chain(Startup(reg))...)
}
