# health-otel

> **Role: Observer.** This adapter **subscribes** to the registry and reacts to every check result. It does not perform health checks itself and does not expose HTTP endpoints. See the [system diagram](https://github.com/ubgo/health#how-the-pieces-fit-together) for the full flow.

OpenTelemetry adapter for [`github.com/ubgo/health`](https://github.com/ubgo/health) — emits an OTEL span for every check the registry runs by subscribing as a `health.Observer`.

## Install

```sh
go get github.com/ubgo/health
go get github.com/ubgo/health/contrib/health-otel
```

## Quick start

```go
package main

import (
    "context"

    "github.com/ubgo/health"
    healthotel "github.com/ubgo/health/contrib/health-otel"
)

func main() {
    reg := health.NewRegistry()
    healthotel.Register(reg)              // uses the global TracerProvider

    // ... register checkers, run them ...
    reg.RunAll(context.Background())
    // Each check emits an OTEL span named "health.check" with attributes:
    //   check.name, check.status, check.severity, check.latency_ms
    // Down results set span Status=Error and record the error string.
}
```

## With a custom TracerProvider (test isolation, sdktrace)

```go
import (
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/sdk/trace/tracetest"
)

exp := tracetest.NewInMemoryExporter()
tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))

healthotel.Register(reg, healthotel.WithTracerProvider(tp))
```

## Customising span name and tracer name

```go
healthotel.Register(reg,
    healthotel.WithTracerName("myservice/health"),
    healthotel.WithSpanName("dependency.health_check"),
)
```

## What appears in your tracing UI

For each check run you get a span carrying:

| Attribute | Example |
|-----------|---------|
| `check.name` | `db` |
| `check.status` | `up` / `down` / `degraded` / `unknown` |
| `check.severity` | `critical` / `degraded` / `info` |
| `check.latency_ms` | `12` |

Span status is set to OK on Up and Error on Down (with the result error string as description).

## API

| Symbol | Purpose |
|--------|---------|
| `Register(reg, opts...)` | Subscribe to the registry; every check emits a span. |
| `WithTracerName(name)` | Override the OTEL tracer instrumentation name. |
| `WithSpanName(name)` | Override the span name. |
| `WithTracerProvider(tp)` | Use a specific `trace.TracerProvider` instead of the global one. |
| `DefaultTracerName`, `DefaultSpanName` | Constants for the defaults. |

## License

Apache-2.0 — see [`LICENSE`](../../LICENSE) at the repository root.
