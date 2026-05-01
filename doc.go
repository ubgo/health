// Package health is a thread-safe registry for runtime health and readiness
// checks of dependencies (databases, caches, message brokers, external HTTP
// services, etc.) exposed via Kubernetes-style probes.
//
// The package has zero third-party dependencies and contains no HTTP types.
// All rendering — net/http, Gin, Chi, Echo, Fiber, OpenTelemetry, Prometheus,
// gRPC — lives in adapter modules under contrib/.
//
// Typical use:
//
//	reg := health.NewRegistry()
//	reg.Register(myDBChecker)
//	reg.Register(myCacheChecker)
//	reg.Start(ctx, 30*time.Second) // background re-checks
//
//	snap := reg.SnapshotForProbe(health.ProbeReadiness)
//	if snap.Status != health.StatusUp { /* not ready */ }
//
// Plug an HTTP framework adapter for endpoint exposure:
//
//	import healthnethttp "github.com/ubgo/health/contrib/health-nethttp"
//
//	mux := http.NewServeMux()
//	healthnethttp.Mount(mux, reg)
//
// The Checker interface is the integration point for downstream plugins:
//
//	type Checker interface {
//	    Name() string
//	    Check(ctx context.Context) Result
//	}
package health
