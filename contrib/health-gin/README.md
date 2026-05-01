# health-gin

> **Role: Renderer.** This adapter **reads** the registry snapshot and exposes it over HTTP. It does **not** check any dependency itself — see the [system diagram](https://github.com/ubgo/health#how-the-pieces-fit-together) for how renderers, checkers, and observers fit together.

Gin adapter for [`github.com/ubgo/health`](https://github.com/ubgo/health) — exposes liveness, readiness, and startup probes as `gin.HandlerFunc`s with a `Mount` helper.

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
                       │  │  health-gin      │ ←── reads      │
                       │  │  (RENDERER)      │                │
                       │  └────────┬─────────┘                │
                       │           │ gin.HandlerFunc          │
                       │           ▼                          │
                       │  ┌──────────────────┐                │
                       │  │  gin.Engine      │                │
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
go get github.com/ubgo/health/contrib/health-gin
```

## Quick start

```go
package main

import (
    "github.com/gin-gonic/gin"

    "github.com/ubgo/health"
    healthgin "github.com/ubgo/health/contrib/health-gin"
)

func main() {
    reg := health.NewRegistry()
    // ... register checkers ...

    r := gin.Default()
    healthgin.Mount(r, reg)              // GET /healthz, /readyz, /startupz
    r.Run(":8080")
}
```

## With middleware (auth, logging, rate-limit, …)

Middleware is `gin.HandlerFunc`.

```go
import (
    "crypto/subtle"
    "net/http"
)

func internalKeyAuth(expected string) gin.HandlerFunc {
    return func(c *gin.Context) {
        if subtle.ConstantTimeCompare([]byte(c.GetHeader("X-Internal-Key")),
            []byte(expected)) != 1 {
            c.AbortWithStatus(http.StatusUnauthorized)
            return
        }
        c.Next()
    }
}

healthgin.Mount(r, reg,
    healthgin.WithReadinessPath("/internal/readyz"),
    healthgin.WithMiddleware(internalKeyAuth("secret")),
)
```

## Mounting on a route group

`Mount` accepts any `gin.IRouter`, so a group works the same:

```go
api := r.Group("/api/v1", authMiddleware())
healthgin.Mount(api, reg)            // → GET /api/v1/healthz etc., protected
```

## API

| Symbol | Purpose |
|--------|---------|
| `Liveness / Readiness / Startup(reg) gin.HandlerFunc` | Handlers in isolation. |
| `Mount(r gin.IRouter, reg, opts...)` | Register all three on `r` with defaults `/healthz` / `/readyz` / `/startupz`. |
| `WithLivenessPath / WithReadinessPath / WithStartupPath(p)` | Override the route. |
| `WithMiddleware(mw ...gin.HandlerFunc)` | Apply user middleware to all three handlers. |

## License

Apache-2.0 — see [`LICENSE`](../../LICENSE) at the repository root.
