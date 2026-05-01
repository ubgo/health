// Package healthprom exposes a health.Registry as Prometheus metrics by
// subscribing to the registry as a health.Observer.
//
// Three metrics are emitted per check:
//
//   - health_check_status{name,severity} (gauge): 1 = up, 0 = down,
//     -1 = degraded, -2 = unknown
//   - health_check_latency_seconds{name,severity} (gauge)
//   - health_check_runs_total{name,severity,status} (counter)
package healthprom

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ubgo/health"
)

// Collector groups the Prometheus metrics this adapter publishes.
type Collector struct {
	status  *prometheus.GaugeVec
	latency *prometheus.GaugeVec
	runs    *prometheus.CounterVec
}

// NewCollector constructs a Collector and registers it with reg.
func NewCollector(reg prometheus.Registerer) *Collector {
	c := &Collector{
		status: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "health_check_status",
			Help: "Status of a health check: 1=up, 0=down, -1=degraded, -2=unknown.",
		}, []string{"name", "severity"}),
		latency: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "health_check_latency_seconds",
			Help: "Last observed latency of a health check, in seconds.",
		}, []string{"name", "severity"}),
		runs: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "health_check_runs_total",
			Help: "Total number of health check runs partitioned by status.",
		}, []string{"name", "severity", "status"}),
	}
	if reg != nil {
		reg.MustRegister(c.status, c.latency, c.runs)
	}
	return c
}

// Subscribe attaches the collector to a health.Registry. Subsequent check
// runs update the metrics.
func (c *Collector) Subscribe(reg *health.Registry) {
	reg.Subscribe(func(name string, r health.Result) {
		labels := prometheus.Labels{"name": name, "severity": string(r.Severity)}

		c.status.With(labels).Set(statusValue(r.Status))
		c.latency.With(labels).Set(r.Latency.Seconds())

		runLabels := prometheus.Labels{"name": name, "severity": string(r.Severity), "status": string(r.Status)}
		c.runs.With(runLabels).Inc()
	})
}

// Register is a one-shot helper that constructs a Collector, subscribes it
// to reg, and returns the Collector for callers that want to inspect or
// reset the metrics later.
func Register(reg *health.Registry, promReg prometheus.Registerer) *Collector {
	c := NewCollector(promReg)
	c.Subscribe(reg)
	return c
}

func statusValue(s health.Status) float64 {
	switch s {
	case health.StatusUp:
		return 1
	case health.StatusDown:
		return 0
	case health.StatusDegraded:
		return -1
	default:
		return -2
	}
}
