# health-redis

> **Role: Checker.** This adapter **writes** to the registry by pinging a real dependency over the network. Renderers (`health-nethttp`, `health-gin`, `health-chi`, `health-echo`, `health-fiber`) then expose the result over HTTP. See the [system diagram](https://github.com/ubgo/health#how-the-pieces-fit-together) for the full flow.

Redis adapter for [`github.com/ubgo/health`](https://github.com/ubgo/health) — implements `health.Checker` by running the Redis `PING` command via go-redis.

## How it works

```
                          ┌──────────────────────────────────────┐
                          │            YOUR SERVICE              │
                          │                                      │
                          │  ┌──────────────────┐                │
   [Redis :6379] ──PING──→│  │  health-redis    │                │
   ←──────PONG────────────│  │  (CHECKER)       │                │
                          │  └────────┬─────────┘                │
                          │           │ Result{Up | Down, lat}   │
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
go get github.com/ubgo/health/contrib/health-redis
```

## Quick start

```go
package main

import (
    "github.com/redis/go-redis/v9"

    "github.com/ubgo/health"
    healthredis "github.com/ubgo/health/contrib/health-redis"
)

func main() {
    client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
    defer client.Close()

    reg := health.NewRegistry()
    _ = reg.Register(healthredis.New("cache", client))
}
```

## Cluster client

The `Pinger` interface matches both `*redis.Client` and `*redis.ClusterClient`:

```go
cluster := redis.NewClusterClient(&redis.ClusterOptions{Addrs: addrs})
_ = reg.Register(healthredis.New("cache", cluster))
```

## Behavior

- `PING` returns "PONG" → `Status: Up`.
- `PING` returns an error → `Status: Down`, `Error` set.
- Latency measured for both outcomes.

## API

| Symbol | Purpose |
|--------|---------|
| `New(name string, p Pinger) *Checker` | Construct a Checker against any `Pinger`. |
| `Pinger` interface | `Ping(ctx) *redis.StatusCmd` — what the Checker calls. |

## License

Apache-2.0 — see [`LICENSE`](../../LICENSE) at the repository root.
