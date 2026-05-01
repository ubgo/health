# health

Thread-safe registry for runtime health and readiness checks of dependencies (databases, caches, message brokers, external HTTP services, etc.) — exposed via Kubernetes-style probes.

Zero third-party dependencies in the core. HTTP framework adapters (stdlib `net/http`, Gin, Chi, Echo, Fiber), observability adapters (OpenTelemetry, Prometheus), and concrete checkers (Postgres, Redis, NATS, generic HTTP probe, DNS) ship as separate modules under `contrib/`.

## Install

```sh
go get github.com/ubgo/health
```

## Quick start

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/ubgo/health"
)

type pingChecker struct{ name string }

func (p *pingChecker) Name() string { return p.name }

func (p *pingChecker) Check(ctx context.Context) health.Result {
    // Run your dependency check here. Set Status, Error, Latency, etc.
    return health.Result{Status: health.StatusUp}
}

func main() {
    reg := health.NewRegistry()
    _ = reg.Register(&pingChecker{name: "db"})
    _ = reg.Register(&pingChecker{name: "cache"})

    ctx := context.Background()
    reg.Start(ctx, 30*time.Second) // background re-checks every 30s
    defer reg.Stop()

    snap := reg.SnapshotForProbe(health.ProbeReadiness)
    log.Printf("ready=%s", snap.Status)
}
```

## Probe semantics

| Probe | When to fail it |
|-------|-----------------|
| `ProbeLiveness` | Process is responding. Always `Up`. Used by k8s to decide whether to restart the pod. |
| `ProbeReadiness` | Process is ready to serve traffic. `Down` if any **critical** check is `Down`; `Degraded` if any **degraded-severity** check is `Down`; otherwise `Up`. Informational checks never affect aggregate. |
| `ProbeStartup` | Process has finished initialising. Mirrors readiness, but reports `Unknown` until the first time readiness is `Up`. Useful for k8s startup probes that should not flap once warm. |

## Severity

A failed check's `Severity` decides whether it fails the readiness probe:

| Severity | A `Down` check ⇒ readiness is | Use for |
|----------|------------------------------|---------|
| `SeverityCritical` *(default)* | `Down` | DB, cache, primary message broker — anything you cannot serve without |
| `SeverityDegraded` | `Degraded` (still 200 OK) | Email provider, third-party APIs you can fall back from |
| `SeverityInfo` | unchanged | Purely informational — disk usage, version banner |

Set per checker at registration:

```go
reg.Register(emailChecker, health.WithSeverity(health.SeverityDegraded))
```

## Per-check timeout

```go
reg.Register(slowDB, health.WithTimeout(2*time.Second))
```

A check exceeding its timeout returns `StatusDown` with the deadline-exceeded error. Default: 5 seconds.

## Observer pattern

Observers fire after every check — used by adapters (OTEL spans, Prometheus metrics, alerting webhooks) to react without polling:

```go
reg.Subscribe(func(name string, r health.Result) {
    if r.Status == health.StatusDown {
        slack.Notify("check failed: " + name)
    }
})
```

## Adapters

Adapter modules ship as separate Go modules under `contrib/`. Import only the ones you use; each pulls in its own dependencies.

### HTTP framework renderers

| Adapter | Module path |
|---------|-------------|
| `health-nethttp` | `github.com/ubgo/health/contrib/health-nethttp` |
| `health-gin` | `github.com/ubgo/health/contrib/health-gin` |
| `health-chi` | `github.com/ubgo/health/contrib/health-chi` |
| `health-echo` | `github.com/ubgo/health/contrib/health-echo` |
| `health-fiber` | `github.com/ubgo/health/contrib/health-fiber` |

Each exposes `Liveness`, `Readiness`, `Startup` handlers and a `Mount` helper with `WithMiddleware` / `WithLivenessPath` / `WithReadinessPath` / `WithStartupPath` options.

### Observability

| Adapter | Module path |
|---------|-------------|
| `health-otel` | `github.com/ubgo/health/contrib/health-otel` |
| `health-prom` | `github.com/ubgo/health/contrib/health-prom` |

### Concrete checkers

| Adapter | Module path |
|---------|-------------|
| `health-postgres` | `github.com/ubgo/health/contrib/health-postgres` |
| `health-redis` | `github.com/ubgo/health/contrib/health-redis` |
| `health-nats` | `github.com/ubgo/health/contrib/health-nats` |
| `health-httpprobe` | `github.com/ubgo/health/contrib/health-httpprobe` |
| `health-dns` | `github.com/ubgo/health/contrib/health-dns` |

Adapters land incrementally as the core stabilises.

## Compatibility

Requires Go 1.24 or later.

## License

Apache License 2.0. See [`LICENSE`](./LICENSE) and [`NOTICE`](./NOTICE).
