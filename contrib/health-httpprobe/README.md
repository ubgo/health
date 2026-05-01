# health-httpprobe

> **Role: Checker.** This adapter **writes** to the registry by pinging a real dependency over the network. Renderers (`health-nethttp`, `health-gin`, `health-chi`, `health-echo`, `health-fiber`) then expose the result over HTTP. See the [system diagram](https://github.com/ubgo/health#how-the-pieces-fit-together) for the full flow.

Generic outbound HTTP probe for [`github.com/ubgo/health`](https://github.com/ubgo/health) — implements `health.Checker` by issuing a GET to a configured URL and accepting any 2xx/3xx response.

Zero third-party dependencies (stdlib `net/http` only).

## How it works

```
                          ┌──────────────────────────────────────┐
                          │            YOUR SERVICE              │
                          │                                      │
                          │  ┌──────────────────┐                │
   [https://api.x/health]→│  │ health-httpprobe │                │
   ←─── status code  ─────│  │  (CHECKER)       │                │
                          │  │ GET / HEAD via   │                │
                          │  │ http.Client      │                │
                          │  └────────┬─────────┘                │
                          │           │ Result{Up if 2xx/3xx,    │
                          │           │         else Down, lat}  │
                          │           ▼                          │
                          │  ┌──────────────────┐                │
                          │  │ health.Registry  │                │
                          │  └────────┬─────────┘                │
                          │           │ SnapshotForProbe         │
                          │           ▼                          │
                          │  ┌──────────────────┐                │
                          │  │  any RENDERER    │ ── /readyz ──→ │
                          │  └──────────────────┘                │
                          └──────────────────────────────────────┘
```

## Install

```sh
go get github.com/ubgo/health
go get github.com/ubgo/health/contrib/health-httpprobe
```

## Quick start

```go
package main

import (
    "github.com/ubgo/health"
    healthhttpprobe "github.com/ubgo/health/contrib/health-httpprobe"
)

func main() {
    reg := health.NewRegistry()
    _ = reg.Register(healthhttpprobe.New("payments-api", "https://api.example.com/health"))
}
```

## Custom HTTP client (timeouts, transport, etc.)

```go
import (
    "net/http"
    "time"
)

client := &http.Client{
    Timeout:   3 * time.Second,
    Transport: customTransport,
}
_ = reg.Register(healthhttpprobe.New("api", "https://api.example.com/health",
    healthhttpprobe.WithClient(client),
))
```

## Custom method or status acceptance

```go
// HEAD instead of GET
_ = reg.Register(healthhttpprobe.New("api", url, healthhttpprobe.WithMethod(http.MethodHead)))

// Treat 401 as healthy (e.g. an endpoint that requires auth but is "up" if reachable):
_ = reg.Register(healthhttpprobe.New("api", url,
    healthhttpprobe.WithAccept(func(code int) bool {
        return code == http.StatusUnauthorized
    }),
))
```

## Behavior

- `2xx` or `3xx` (default) → `Status: Up`.
- `4xx`/`5xx` → `Status: Down`, `Error` describes the unexpected status.
- Network error / context deadline → `Status: Down`, `Error` set.
- Latency measured for all outcomes.

## API

| Symbol | Purpose |
|--------|---------|
| `New(name, url string, opts...) *Checker` | Construct a Checker that probes `url`. |
| `WithClient(c *http.Client)` | Override the HTTP client (default: 5s timeout). |
| `WithMethod(m string)` | Override HTTP method (default: GET). |
| `WithAccept(a AcceptStatus)` | Override the status-code predicate (default: 2xx/3xx). |
| `AcceptDefault(code int) bool` | The default predicate, exposed for composition. |
| `AcceptStatus = func(code int) bool` | Type alias for predicates. |

## License

Apache-2.0 — see [`LICENSE`](../../LICENSE) at the repository root.
