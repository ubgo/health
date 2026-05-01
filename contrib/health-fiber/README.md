# health-fiber

> **Role: Renderer.** This adapter **reads** the registry snapshot and exposes it over HTTP. It does **not** check any dependency itself — see the [system diagram](https://github.com/ubgo/health#how-the-pieces-fit-together) for how renderers, checkers, and observers fit together.

Fiber adapter for [`github.com/ubgo/health`](https://github.com/ubgo/health) — exposes liveness, readiness, and startup probes as `fiber.Handler`s with a `Mount` helper.

## Install

```sh
go get github.com/ubgo/health
go get github.com/ubgo/health/contrib/health-fiber
```

## Quick start

```go
package main

import (
    "github.com/gofiber/fiber/v2"

    "github.com/ubgo/health"
    healthfiber "github.com/ubgo/health/contrib/health-fiber"
)

func main() {
    reg := health.NewRegistry()
    // ... register checkers ...

    app := fiber.New()
    healthfiber.Mount(app, reg)              // GET /healthz, /readyz, /startupz
    app.Listen(":8080")
}
```

## With middleware

Middleware is `fiber.Handler`.

```go
import (
    "crypto/subtle"
    "net/http"
)

func internalKeyAuth(expected string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        if subtle.ConstantTimeCompare([]byte(c.Get("X-Internal-Key")),
            []byte(expected)) != 1 {
            return c.SendStatus(http.StatusUnauthorized)
        }
        return c.Next()
    }
}

healthfiber.Mount(app, reg,
    healthfiber.WithReadinessPath("/internal/readyz"),
    healthfiber.WithMiddleware(internalKeyAuth("secret")),
)
```

## Mounting on a route group

`Mount` accepts any `fiber.Router`, so a group works the same:

```go
api := app.Group("/api/v1", authMiddleware)
healthfiber.Mount(api, reg)            // → GET /api/v1/healthz etc.
```

## API

| Symbol | Purpose |
|--------|---------|
| `Liveness / Readiness / Startup(reg) fiber.Handler` | Handlers in isolation. |
| `Mount(r fiber.Router, reg, opts...)` | Register all three on `r` with defaults `/healthz` / `/readyz` / `/startupz`. |
| `WithLivenessPath / WithReadinessPath / WithStartupPath(p)` | Override the route. |
| `WithMiddleware(mw ...fiber.Handler)` | Apply user middleware to all three handlers. |

## License

Apache-2.0 — see [`LICENSE`](../../LICENSE) at the repository root.
