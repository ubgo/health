package health

import (
	"context"
	"time"
)

// Status is the outcome of a single health check.
type Status string

const (
	// StatusUp means the dependency responded successfully.
	StatusUp Status = "up"
	// StatusDown means the dependency failed or was unreachable.
	StatusDown Status = "down"
	// StatusDegraded means the dependency responded but in a reduced state
	// (slow, partial failure, fallback path active).
	StatusDegraded Status = "degraded"
	// StatusUnknown means the check has never run or its outcome is unclear.
	StatusUnknown Status = "unknown"
)

// Severity controls whether a failed check fails the readiness probe.
type Severity string

const (
	// SeverityCritical: a Down result fails the readiness probe. Default for
	// most production dependencies (DB, cache, etc.).
	SeverityCritical Severity = "critical"
	// SeverityDegraded: a Down result is reported in the snapshot but the
	// readiness probe still returns up. Use for dependencies whose absence
	// degrades but does not break the service (e.g. an email provider).
	SeverityDegraded Severity = "degraded"
	// SeverityInfo: a Down result is reported but never affects either
	// liveness or readiness. Use for purely informational checks.
	SeverityInfo Severity = "info"
)

// Probe is one of the three Kubernetes-style probe types a registry can
// snapshot for. See SnapshotForProbe.
type Probe string

const (
	// ProbeLiveness — the process is alive. Fails only if the host has
	// crashed in a way that the process itself can detect; otherwise always up.
	ProbeLiveness Probe = "liveness"
	// ProbeReadiness — the process is ready to serve traffic. Aggregates the
	// worst critical Result from the registry.
	ProbeReadiness Probe = "readiness"
	// ProbeStartup — the process has finished initialising. Mirrors readiness
	// until the first up snapshot, then mirrors readiness from then on.
	ProbeStartup Probe = "startup"
)

// Result of a single Check invocation.
type Result struct {
	Status    Status         `json:"status"`
	Severity  Severity       `json:"severity"`
	Message   string         `json:"message,omitempty"`
	Error     string         `json:"error,omitempty"`
	Latency   time.Duration  `json:"latency_ns,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// Checker performs a health check for a named component. Implementations must
// be safe for concurrent invocation by the Registry.
type Checker interface {
	// Name returns a stable, unique identifier for this check (e.g. "db", "cache").
	// The name appears as a key in the registry snapshot and in HTTP responses.
	Name() string

	// Check runs the health check honouring ctx for cancellation/timeout.
	// Implementations should never panic; on any error, return a Result with
	// Status=StatusDown and Error populated.
	Check(ctx context.Context) Result
}

// Observer receives every Result the registry produces. Used by adapters
// (e.g. OTEL spans, Prometheus metrics, alert webhooks) to react without
// polling. Observers are invoked synchronously after each check; long-running
// observers should fan out to a goroutine themselves.
type Observer func(name string, result Result)
