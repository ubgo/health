# health-postgres

Postgres adapter for [`github.com/ubgo/health`](https://github.com/ubgo/health) — implements `health.Checker` for a Postgres connection or pool, using pgx's `Ping` method.

## Install

```sh
go get github.com/ubgo/health
go get github.com/ubgo/health/contrib/health-postgres
```

## Quick start

```go
package main

import (
    "context"

    "github.com/jackc/pgx/v5/pgxpool"

    "github.com/ubgo/health"
    healthpostgres "github.com/ubgo/health/contrib/health-postgres"
)

func main() {
    pool, _ := pgxpool.New(context.Background(), "postgres://localhost/myapp")
    defer pool.Close()

    reg := health.NewRegistry()
    _ = reg.Register(healthpostgres.New("db", pool))
    // …or use the typed helper:
    _ = reg.Register(healthpostgres.FromPool("db", pool))
}
```

## Why both `New` and `FromPool`?

`New` accepts the minimal `Pinger` interface:

```go
type Pinger interface {
    Ping(ctx context.Context) error
}
```

This matches both `*pgxpool.Pool` and `*pgx.Conn`, plus any other library or test fake that implements `Ping`. `FromPool` is just a typed convenience wrapper for the most common case.

## Behavior

- `Ping` succeeds → `Status: Up`, `Severity: Critical`.
- `Ping` returns an error → `Status: Down`, `Severity: Critical`, `Error` set to the error string.
- Latency is measured for both Up and Down results.

## API

| Symbol | Purpose |
|--------|---------|
| `New(name string, p Pinger) *Checker` | Construct a Checker against any `Pinger`. |
| `FromPool(name string, pool *pgxpool.Pool) *Checker` | Convenience constructor for `*pgxpool.Pool`. |
| `Pinger` interface | `Ping(ctx) error` — what the Checker calls. |

## License

Apache-2.0 — see [`LICENSE`](../../LICENSE) at the repository root.
