# health-dns

> **Role: Checker.** This adapter **writes** to the registry by pinging a real dependency over the network. Renderers (`health-nethttp`, `health-gin`, `health-chi`, `health-echo`, `health-fiber`) then expose the result over HTTP. See the [system diagram](https://github.com/ubgo/health#how-the-pieces-fit-together) for the full flow.

DNS adapter for [`github.com/ubgo/health`](https://github.com/ubgo/health) — implements `health.Checker` by resolving a host via a DNS resolver.

Zero third-party dependencies (stdlib `net` only).

## Install

```sh
go get github.com/ubgo/health
go get github.com/ubgo/health/contrib/health-dns
```

## Quick start

```go
package main

import (
    "github.com/ubgo/health"
    healthdns "github.com/ubgo/health/contrib/health-dns"
)

func main() {
    reg := health.NewRegistry()
    _ = reg.Register(healthdns.New("dns", "kubernetes.default.svc.cluster.local"))
}
```

## Why include this?

When DNS resolution breaks inside a Kubernetes cluster (CoreDNS pod restart, NodeLocalDNS misconfiguration), application-level health checks all start failing simultaneously and the noise drowns out the root cause. A dedicated DNS check surfaces the underlying problem immediately and explains why everything else is timing out.

Common targets:

- `kubernetes.default.svc.cluster.local` — exercises kube-dns end-to-end inside a cluster.
- An external host you depend on (e.g. `api.stripe.com`) — flags upstream DNS issues separately from upstream service issues.

## Custom resolver / minimum hosts

```go
import "net"

// Use a non-default resolver pointed at a specific DNS server.
r := &net.Resolver{
    PreferGo: true,
    Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
        return net.Dial(network, "8.8.8.8:53")
    },
}
_ = reg.Register(healthdns.New("dns", "example.com",
    healthdns.WithResolver(r),
    healthdns.WithMinHosts(2),  // require ≥2 A/AAAA records
))
```

## Behavior

- Resolver returns `≥ minHosts` addresses (default 1) → `Status: Up`. Metadata includes `resolved_count`.
- Resolver error → `Status: Down`, `Error` set.
- Resolver returns fewer than `minHosts` addresses → `Status: Down` with explanatory error.

## API

| Symbol | Purpose |
|--------|---------|
| `New(name, host string, opts...) *Checker` | Construct a Checker that resolves `host`. |
| `WithResolver(r Resolver)` | Override the resolver (default: `net.DefaultResolver`). |
| `WithMinHosts(n int)` | Floor on resolved address count (default: 1). |
| `Resolver` interface | `LookupHost(ctx, host) ([]string, error)`. |

## License

Apache-2.0 — see [`LICENSE`](../../LICENSE) at the repository root.
