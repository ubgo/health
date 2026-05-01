# health-echo

> **Role: Renderer.** This adapter **reads** the registry snapshot and exposes it over HTTP. It does **not** check any dependency itself — see the [system diagram](https://github.com/ubgo/health#how-the-pieces-fit-together) for how renderers, checkers, and observers fit together.

Echo adapter for [`github.com/ubgo/health`](https://github.com/ubgo/health) — exposes liveness, readiness, and startup probes as `echo.HandlerFunc`s with a `Mount` helper.

## How it works

```
                       ┌──────────────────────────────────────┐
                       │            YOUR SERVICE              │
                       │                                      │
   CHECKERS ─────────→ │  ┌──────────────────┐                │
   (postgres, redis,   │  │ health.Registry  │                │
    nats, dns, …)      │  └────────┬─────────┘                │
                       │           │ SnapshotForProbe(probe)  │
                       │           ▼                          │
                       │  ┌──────────────────┐                │
                       │  │  health-echo     │ ←── reads      │
                       │  │  (RENDERER)      │                │
                       │  └────────┬─────────┘                │
                       │           │ echo.HandlerFunc         │
                       │           ▼                          │
                       │  ┌──────────────────┐                │
                       │  │  echo.Echo       │                │
                       │  │   /healthz       │                │
                       │  │   /readyz        │                │
                       │  │   /startupz      │                │
                       │  └────────┬─────────┘                │
                       └───────────┼──────────────────────────┘
                                   ▼
                              k8s probe / load balancer / curl
```

## Install

```sh
go get github.com/ubgo/health
go get github.com/ubgo/health/contrib/health-echo
```

## Quick start

```go
package main

import (
    "github.com/labstack/echo/v4"

    "github.com/ubgo/health"
    healthecho "github.com/ubgo/health/contrib/health-echo"
)

func main() {
    reg := health.NewRegistry()
    // ... register checkers ...

    e := echo.New()
    healthecho.Mount(e, reg)              // GET /healthz, /readyz, /startupz
    e.Logger.Fatal(e.Start(":8080"))
}
```

## With middleware

Middleware is `echo.MiddlewareFunc`.

```go
import (
    "crypto/subtle"
    "net/http"
)

func internalKeyAuth(expected string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            if subtle.ConstantTimeCompare([]byte(c.Request().Header.Get("X-Internal-Key")),
                []byte(expected)) != 1 {
                return c.NoContent(http.StatusUnauthorized)
            }
            return next(c)
        }
    }
}

healthecho.Mount(e, reg,
    healthecho.WithReadinessPath("/internal/readyz"),
    healthecho.WithMiddleware(internalKeyAuth("secret")),
)
```

## API

| Symbol | Purpose |
|--------|---------|
| `Liveness / Readiness / Startup(reg) echo.HandlerFunc` | Handlers in isolation. |
| `Mount(e *echo.Echo, reg, opts...)` | Register all three on `e` with defaults `/healthz` / `/readyz` / `/startupz`. |
| `WithLivenessPath / WithReadinessPath / WithStartupPath(p)` | Override the route. |
| `WithMiddleware(mw ...echo.MiddlewareFunc)` | Apply user middleware to all three handlers. |

## License

Apache-2.0 — see [`LICENSE`](../../LICENSE) at the repository root.
