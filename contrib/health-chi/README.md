# health-chi

> **Role: Renderer.** This adapter **reads** the registry snapshot and exposes it over HTTP. It does **not** check any dependency itself — see the [system diagram](https://github.com/ubgo/health#how-the-pieces-fit-together) for how renderers, checkers, and observers fit together.

Chi adapter for [`github.com/ubgo/health`](https://github.com/ubgo/health) — exposes liveness, readiness, and startup probes as stdlib `http.Handler`s with a Chi-native `Mount` helper.

## Install

```sh
go get github.com/ubgo/health
go get github.com/ubgo/health/contrib/health-chi
```

## Quick start

```go
package main

import (
    "net/http"

    "github.com/go-chi/chi/v5"

    "github.com/ubgo/health"
    healthchi "github.com/ubgo/health/contrib/health-chi"
)

func main() {
    reg := health.NewRegistry()
    // ... register checkers ...

    r := chi.NewRouter()
    healthchi.Mount(r, reg)              // GET /healthz, /readyz, /startupz
    http.ListenAndServe(":8080", r)
}
```

## With middleware

Middleware uses the standard `func(http.Handler) http.Handler` shape (chi's native middleware shape).

```go
import "crypto/subtle"

authMW := func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Internal-Key")),
            []byte("secret")) != 1 {
            w.WriteHeader(http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}

healthchi.Mount(r, reg,
    healthchi.WithReadinessPath("/internal/readyz"),
    healthchi.WithMiddleware(authMW),
)
```

## API

| Symbol | Purpose |
|--------|---------|
| `Liveness / Readiness / Startup(reg) http.Handler` | Handlers in isolation. |
| `Mount(r chi.Router, reg, opts...)` | Register all three on `r` with defaults `/healthz` / `/readyz` / `/startupz`. |
| `WithLivenessPath / WithReadinessPath / WithStartupPath(p)` | Override the route. |
| `WithMiddleware(mw ...Middleware)` | Apply user middleware to all three handlers. |
| `Middleware = func(http.Handler) http.Handler` | The Chi-compatible middleware shape. |

## License

Apache-2.0 — see [`LICENSE`](../../LICENSE) at the repository root.
