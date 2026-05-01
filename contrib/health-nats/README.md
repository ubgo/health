# health-nats

> **Role: Checker.** This adapter **writes** to the registry by pinging a real dependency over the network. Renderers (`health-nethttp`, `health-gin`, `health-chi`, `health-echo`, `health-fiber`) then expose the result over HTTP. See the [system diagram](https://github.com/ubgo/health#how-the-pieces-fit-together) for the full flow.

NATS adapter for [`github.com/ubgo/health`](https://github.com/ubgo/health) — implements `health.Checker` based on the connection state of a NATS client.

## Install

```sh
go get github.com/ubgo/health
go get github.com/ubgo/health/contrib/health-nats
```

## Quick start

```go
package main

import (
    "github.com/nats-io/nats.go"

    "github.com/ubgo/health"
    healthnats "github.com/ubgo/health/contrib/health-nats"
)

func main() {
    nc, _ := nats.Connect(nats.DefaultURL)
    defer nc.Close()

    reg := health.NewRegistry()
    _ = reg.Register(healthnats.New("messaging", nc))
}
```

## Status mapping

| NATS Status | health.Status |
|-------------|---------------|
| `CONNECTED` | `Up` |
| `RECONNECTING` | `Degraded` |
| `CLOSED`, `DISCONNECTED`, `DRAINING_PUBS`, `DRAINING_SUBS` | `Down` |
| anything else | `Unknown` |

`RECONNECTING` is reported as `Degraded` rather than `Down` because NATS clients buffer publishes locally while reconnecting — a service may continue to function depending on the workload.

## API

| Symbol | Purpose |
|--------|---------|
| `New(name string, c Conn) *Checker` | Construct a Checker against any `Conn`. |
| `Conn` interface | `IsConnected() bool` + `Status() nats.Status`. `*nats.Conn` satisfies this directly. |

## License

Apache-2.0 — see [`LICENSE`](../../LICENSE) at the repository root.
