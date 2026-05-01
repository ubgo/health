package health

import (
	"encoding/json"
	"net/http"
	"time"
)

// Snapshot is the aggregate state of a probe at a point in time.
type Snapshot struct {
	Probe  Probe             `json:"probe"`
	Status Status            `json:"status"`
	Time   time.Time         `json:"time"`
	Checks map[string]Result `json:"checks"`
}

// SnapshotForProbe returns the registry's state aggregated for the given
// probe type:
//
//   - ProbeLiveness: always StatusUp (the process is responding).
//   - ProbeReadiness: StatusDown if any critical check is Down; StatusDegraded
//     if any degraded-severity check is Down (and no critical is Down);
//     otherwise StatusUp. SeverityInfo checks never affect the aggregate.
//   - ProbeStartup: same as readiness, but reports StatusUnknown until the
//     first time readiness has been observed Up — useful for k8s startup
//     probes that should not flap once warm.
func (r *Registry) SnapshotForProbe(probe Probe) Snapshot {
	r.mu.RLock()
	checks := make(map[string]Result, len(r.results))
	for k, v := range r.results {
		checks[k] = v
	}
	r.mu.RUnlock()

	s := Snapshot{
		Probe:  probe,
		Time:   time.Now(),
		Checks: checks,
	}

	switch probe {
	case ProbeLiveness:
		s.Status = StatusUp
	case ProbeReadiness:
		s.Status = aggregateStatus(checks)
		if s.Status == StatusUp {
			r.markStartupSeen()
		}
	case ProbeStartup:
		if !r.hasStartupSeen() {
			agg := aggregateStatus(checks)
			if agg == StatusUp {
				r.markStartupSeen()
				s.Status = StatusUp
			} else {
				s.Status = StatusUnknown
			}
		} else {
			s.Status = aggregateStatus(checks)
		}
	default:
		s.Status = StatusUnknown
	}
	return s
}

// HTTPStatus maps the snapshot to a conventional HTTP status code:
// 200 for Up or Degraded, 503 for Down or Unknown.
//
// Returns a plain int — the core does not import net/http types into its
// API, but exposes the constant for convenience to adapters.
func (s Snapshot) HTTPStatus() int {
	switch s.Status {
	case StatusUp, StatusDegraded:
		return http.StatusOK
	default:
		return http.StatusServiceUnavailable
	}
}

// JSON returns the snapshot marshalled as JSON bytes.
func (s Snapshot) JSON() ([]byte, error) {
	return json.Marshal(s)
}

// aggregateStatus reduces per-check results to a single status using the
// severity rules from the package doc.
func aggregateStatus(checks map[string]Result) Status {
	if len(checks) == 0 {
		return StatusUp
	}
	worst := StatusUp
	for _, r := range checks {
		// Skip informational checks — they never affect aggregate.
		if r.Severity == SeverityInfo {
			continue
		}
		if r.Status == StatusDown && r.Severity == SeverityCritical {
			return StatusDown
		}
		if (r.Status == StatusDown || r.Status == StatusDegraded) && r.Severity == SeverityDegraded {
			worst = StatusDegraded
		}
		if r.Status == StatusDegraded && r.Severity == SeverityCritical && worst == StatusUp {
			worst = StatusDegraded
		}
		if r.Status == StatusUnknown && r.Severity == SeverityCritical {
			return StatusUnknown
		}
	}
	return worst
}
