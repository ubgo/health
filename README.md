# health

Thread-safe registry for runtime health and readiness checks of dependencies (databases, caches, message brokers, external HTTP services, etc.) — exposed via Kubernetes-style probes.

Zero third-party dependencies in the core. HTTP framework adapters (stdlib `net/http`, Gin, Chi, Echo, Fiber), observability adapters (OpenTelemetry, Prometheus), and concrete checkers (Postgres, Redis, NATS, generic HTTP probe, DNS) ship as separate modules under `contrib/`.

## How the pieces fit together

A real `ubgo/health` deployment has three kinds of components, each playing a different role around the central `Registry`:

```
                        ┌──────────────────────────────────────┐
                        │              YOUR SERVICE             │
                        │                                      │
   PING ─→  [Redis]  ─→ │  ┌─────────────────────────────────┐ │
                        │  │  CHECKER  (health-redis)        │ │   ← writes Result to registry
                        │  │  Result{Up, latency=2ms}        │ │
                        │  └────────────┬────────────────────┘ │
                        │               ▼                      │
                        │  ┌─────────────────────────────────┐ │
                        │  │  health.Registry                │ │   ← stores per-checker state
                        │  │  ├─ db:    Up                   │ │
                        │  │  ├─ cache: Up                   │ │
                        │  │  └─ nats:  Down                 │ │
                        │  └────┬───────────────────┬────────┘ │
                        │       │                   │          │
                        │       ▼                   ▼          │
                        │  ┌─────────────┐    ┌─────────────┐  │
                        │  │  RENDERER   │    │  OBSERVER   │  │   ← consume snapshots
                        │  │  health-gin │    │ health-otel │  │
                        │  │  /readyz    │    │ emit spans  │  │
                        │  └──────┬──────┘    └──────┬──────┘  │
                        └─────────┼──────────────────┼─────────┘
                                  ▼                  ▼
                              k8s probe        OTEL collector
                              load balancer    Prom scrape
                              curl
```

| Role | Examples | What they do |
|------|----------|--------------|
| **Checker** *(writes to registry)* | [`health-postgres`](./contrib/health-postgres), [`health-redis`](./contrib/health-redis), [`health-nats`](./contrib/health-nats), [`health-httpprobe`](./contrib/health-httpprobe), [`health-dns`](./contrib/health-dns) | Ping a real dependency over the network. Implement `health.Checker`. |
| **Renderer** *(reads from registry)* | [`health-nethttp`](./contrib/health-nethttp), [`health-gin`](./contrib/health-gin), [`health-chi`](./contrib/health-chi), [`health-echo`](./contrib/health-echo), [`health-fiber`](./contrib/health-fiber) | Take the registry snapshot and expose `/healthz` / `/readyz` / `/startupz` as HTTP. No I/O against dependencies. |
| **Observer** *(subscribes to registry, async)* | [`health-otel`](./contrib/health-otel), [`health-prom`](./contrib/health-prom) | React to every check result — emit a span, increment a counter. No HTTP exposure of their own. |
| **Core** *(the registry itself)* | [`github.com/ubgo/health`](https://github.com/ubgo/health) | Stores results, aggregates probes, fires observers. Stdlib only, no third-party deps. |

A typical service wires **one or more checkers + one renderer + zero or more observers**:

```go
reg := health.NewRegistry()
reg.Register(healthredis.New("cache", redisClient))     // CHECKER
reg.Register(healthpostgres.New("db",  pgxPool))        // CHECKER
healthotel.Register(reg)                                // OBSERVER
healthgin.Mount(r, reg)                                 // RENDERER
```

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

## Composing checker + renderer + observer

A production wiring brings all three roles together. The example below uses Postgres + Redis as critical checkers, Slack as a degraded fallback, OTEL as the observer, and Gin as the renderer:

```go
import (
    "context"
    "time"

    "github.com/ubgo/health"
    healthgin       "github.com/ubgo/health/contrib/health-gin"
    healthhttpprobe "github.com/ubgo/health/contrib/health-httpprobe"
    healthotel      "github.com/ubgo/health/contrib/health-otel"
    healthpostgres  "github.com/ubgo/health/contrib/health-postgres"
    healthredis     "github.com/ubgo/health/contrib/health-redis"

    "github.com/gin-gonic/gin"
    "go.opentelemetry.io/otel"
)

func main() {
    reg := health.NewRegistry()

    // 1. CHECKERS — write Result to the registry on a schedule.
    reg.Register(healthpostgres.New("db", pgxPool))
    reg.Register(healthredis.New("cache", redisClient))
    reg.Register(
        healthhttpprobe.New("slack-webhook", "https://hooks.slack.com/.../health"),
        health.WithSeverity(health.SeverityDegraded), // Slack down ≠ readiness fail
    )

    // 2. OBSERVER — emit OTEL span on every check result.
    healthotel.Register(reg, healthotel.WithTracer(otel.Tracer("health")))

    // 3. RENDERER — expose /healthz, /readyz, /startupz.
    r := gin.Default()
    healthgin.Mount(r, reg)

    // Background check loop — every 30s.
    ctx := context.Background()
    reg.Start(ctx, 30*time.Second)
    defer reg.Stop()

    _ = r.Run(":8080")
}
```

Now `curl /readyz` returns 503 when the DB or cache is down (critical), but stays at 200 when only Slack is unreachable (degraded). Every result fires an OTEL span you can correlate with the application's request traces.

## Real-world: Kubernetes deployment

Map the three probes onto k8s:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-service
spec:
  template:
    spec:
      containers:
        - name: app
          ports:
            - name: http
              containerPort: 8080

          # /healthz — process is alive. Used by k8s to restart the pod.
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8080
            periodSeconds: 10
            failureThreshold: 3

          # /readyz — process is ready to serve. Used by the Service /
          # load balancer to add/remove the pod from the rotation.
          readinessProbe:
            httpGet:
              path: /readyz
              port: 8080
            periodSeconds: 2
            failureThreshold: 1

          # /startupz — process has finished initialising. Replaces the
          # liveness probe during startup so a slow boot doesn't trigger
          # a restart loop.
          startupProbe:
            httpGet:
              path: /startupz
              port: 8080
            periodSeconds: 5
            failureThreshold: 30   # 30 × 5s = 150s startup budget
```

Tune the probe periods to match your check `WithTimeout` and the registry's `Start(ctx, interval)` cadence — kubelet doesn't gain anything from polling faster than your checks update.

For the matching SIGTERM-side drain (load balancer drains pod *before* listener closes), pair this with [`ubgo/shutdown`](https://github.com/ubgo/shutdown) and use its `PhasePreShutdown` phase to flip readiness off.

## Comparison

| Feature | `alexliesenfeld/health` | `hellofresh/health-go` | `heptiolabs/healthcheck` | **`ubgo/health`** |
|---|:---:|:---:|:---:|:---:|
| Liveness / Readiness / Startup split | partial | ❌ | ❌ | **✅** |
| Per-check `Severity` (critical / degraded / info) | ❌ | ❌ | ❌ | **✅** |
| Background re-check on a schedule | ✅ | ❌ | ✅ | **✅** |
| Observer pattern (no polling) | ❌ | ❌ | ❌ | **✅** |
| OpenTelemetry adapter | ❌ | ❌ | ❌ | **✅ (contrib)** |
| Prometheus adapter | ❌ | ❌ | ❌ | **✅ (contrib)** |
| Concrete checkers (Postgres / Redis / NATS / DNS / HTTP) | ❌ | bundled in core | ❌ | **✅ (contrib, opt-in)** |
| Per-framework adapter (Gin / Chi / Echo / Fiber) | ❌ | ❌ | ❌ | **✅ (contrib)** |
| Zero-dep core | ✅ | ❌ (pulls every checker) | ✅ | **✅** |

## Adapters

Adapter modules ship as separate Go modules under `contrib/`. Import only the ones you use; each pulls in its own dependencies.

### HTTP framework renderers

| Adapter | Module path |
|---------|-------------|
| [`health-nethttp`](./contrib/health-nethttp) | `github.com/ubgo/health/contrib/health-nethttp` |
| [`health-gin`](./contrib/health-gin) | `github.com/ubgo/health/contrib/health-gin` |
| [`health-chi`](./contrib/health-chi) | `github.com/ubgo/health/contrib/health-chi` |
| [`health-echo`](./contrib/health-echo) | `github.com/ubgo/health/contrib/health-echo` |
| [`health-fiber`](./contrib/health-fiber) | `github.com/ubgo/health/contrib/health-fiber` |

Each exposes `Liveness`, `Readiness`, `Startup` handlers and a `Mount` helper with `WithMiddleware` / `WithLivenessPath` / `WithReadinessPath` / `WithStartupPath` options.

### Observability

| Adapter | Module path |
|---------|-------------|
| [`health-otel`](./contrib/health-otel) | `github.com/ubgo/health/contrib/health-otel` |
| [`health-prom`](./contrib/health-prom) | `github.com/ubgo/health/contrib/health-prom` |

### Concrete checkers

| Adapter | Module path |
|---------|-------------|
| [`health-postgres`](./contrib/health-postgres) | `github.com/ubgo/health/contrib/health-postgres` |
| [`health-redis`](./contrib/health-redis) | `github.com/ubgo/health/contrib/health-redis` |
| [`health-nats`](./contrib/health-nats) | `github.com/ubgo/health/contrib/health-nats` |
| [`health-httpprobe`](./contrib/health-httpprobe) | `github.com/ubgo/health/contrib/health-httpprobe` |
| [`health-dns`](./contrib/health-dns) | `github.com/ubgo/health/contrib/health-dns` |

Click any adapter for its dedicated README with install, quick start, middleware, and API tables. All twelve adapters ship in v0.1.0.

## Compatibility

Requires Go 1.24 or later.

## License

Apache License 2.0. See [`LICENSE`](./LICENSE) and [`NOTICE`](./NOTICE).
