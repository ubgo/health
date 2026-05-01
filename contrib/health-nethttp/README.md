# health-nethttp

> **Role: Renderer.** This adapter **reads** the registry snapshot and exposes it over HTTP. It does **not** check any dependency itself — see the [system diagram](https://github.com/ubgo/health#how-the-pieces-fit-together) for how renderers, checkers, and observers fit together.

Stdlib `net/http` adapter for [`github.com/ubgo/health`](https://github.com/ubgo/health) — exposes liveness, readiness, and startup probes as `http.Handler`s with a `Mount` helper.

Zero third-party dependencies.

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
                       │  │ health-nethttp   │ ←── reads      │
                       │  │  (RENDERER)      │                │
                       │  └────────┬─────────┘                │
                       │           │ http.Handler             │
                       │           ▼                          │
                       │  ┌──────────────────┐                │
                       │  │ http.ServeMux    │                │
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
go get github.com/ubgo/health/contrib/health-nethttp
```

## Quick start

```go
package main

import (
    "net/http"

    "github.com/ubgo/health"
    healthnethttp "github.com/ubgo/health/contrib/health-nethttp"
)

func main() {
    reg := health.NewRegistry()
    // ... register checkers ...

    mux := http.NewServeMux()
    healthnethttp.Mount(mux, reg)              // GET /healthz, /readyz, /startupz
    http.ListenAndServe(":8080", mux)
}
```

## With middleware (auth, logging, rate-limit, …)

Middleware uses the standard stdlib shape `func(http.Handler) http.Handler`.

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

healthnethttp.Mount(mux, reg,
    healthnethttp.WithReadinessPath("/internal/readyz"),
    healthnethttp.WithMiddleware(authMW),
)
```

## Custom paths

```go
healthnethttp.Mount(mux, reg,
    healthnethttp.WithLivenessPath("/api/healthz"),
    healthnethttp.WithReadinessPath("/api/readyz"),
    healthnethttp.WithStartupPath("/api/startupz"),
)
```

## API

| Symbol | Purpose |
|--------|---------|
| `Liveness(reg) http.Handler` | Liveness handler in isolation. |
| `Readiness(reg) http.Handler` | Readiness handler in isolation. |
| `Startup(reg) http.Handler` | Startup handler in isolation. |
| `Mount(mux, reg, opts...)` | Register all three on `mux` with default paths `/healthz` / `/readyz` / `/startupz`. |
| `WithLivenessPath / WithReadinessPath / WithStartupPath(p)` | Override the route. |
| `WithMiddleware(mw...)` | Apply user middleware to all three handlers. |
| `Middleware = func(http.Handler) http.Handler` | The stdlib-compatible middleware shape. |

## License

Apache-2.0 — see [`LICENSE`](../../LICENSE) at the repository root.
