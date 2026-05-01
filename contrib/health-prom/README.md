# health-prom

Prometheus adapter for [`github.com/ubgo/health`](https://github.com/ubgo/health) — exposes health checks as Prometheus metrics by subscribing as a `health.Observer`.

## Install

```sh
go get github.com/ubgo/health
go get github.com/ubgo/health/contrib/health-prom
```

## Quick start

```go
package main

import (
    "net/http"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"

    "github.com/ubgo/health"
    healthprom "github.com/ubgo/health/contrib/health-prom"
)

func main() {
    promReg := prometheus.NewRegistry()

    reg := health.NewRegistry()
    healthprom.Register(reg, promReg)            // wire metrics to /metrics

    // ... register checkers, start re-checks ...

    http.Handle("/metrics", promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}))
    http.ListenAndServe(":8080", nil)
}
```

```sh
$ curl -s http://localhost:8080/metrics | grep health_check_
health_check_status{name="db",severity="critical"} 1
health_check_latency_seconds{name="db",severity="critical"} 0.0023
health_check_runs_total{name="db",severity="critical",status="up"} 4
```

## Metrics emitted

| Metric | Type | Labels |
|--------|------|--------|
| `health_check_status` | gauge | `name`, `severity` |
| `health_check_latency_seconds` | gauge | `name`, `severity` |
| `health_check_runs_total` | counter | `name`, `severity`, `status` |

`health_check_status` values map:

| Status | Value |
|--------|-------|
| `up` | `1` |
| `down` | `0` |
| `degraded` | `-1` |
| `unknown` | `-2` |

So a Prometheus alerting rule like `health_check_status < 1` fires on anything not Up.

## Reusing a Collector

If you want to inspect or reset metrics in tests, hold onto the returned Collector:

```go
c := healthprom.Register(reg, promReg)
// ... runs happen ...
testutil.GatherAndCompare(promReg, ...) // standard Prometheus testutil
```

You can also construct a Collector unregistered (pass `nil`) and register it later:

```go
c := healthprom.NewCollector(nil)
c.Subscribe(reg)
// ... later ...
promReg.MustRegister(...)  // attach manually
```

## API

| Symbol | Purpose |
|--------|---------|
| `Register(reg *health.Registry, promReg prometheus.Registerer) *Collector` | One-shot helper: construct a Collector, subscribe to reg, register with promReg. |
| `NewCollector(promReg prometheus.Registerer) *Collector` | Construct a Collector. Pass `nil` to skip auto-registration. |
| `(*Collector).Subscribe(reg *health.Registry)` | Attach to a registry. |

## License

Apache-2.0 — see [`LICENSE`](../../LICENSE) at the repository root.
