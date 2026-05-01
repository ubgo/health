package health

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestSnapshot_Liveness_AlwaysUp(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(downChecker("db", "boom"))
	reg.RunAll(context.Background())

	s := reg.SnapshotForProbe(ProbeLiveness)
	if s.Status != StatusUp {
		t.Errorf("liveness: got %q, want %q", s.Status, StatusUp)
	}
	if s.HTTPStatus() != http.StatusOK {
		t.Errorf("liveness HTTPStatus: got %d, want 200", s.HTTPStatus())
	}
}

func TestSnapshot_Readiness_DownOnCriticalDown(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(downChecker("db", "boom")) // critical (default)
	_ = reg.Register(upChecker("cache"))
	reg.RunAll(context.Background())

	s := reg.SnapshotForProbe(ProbeReadiness)
	if s.Status != StatusDown {
		t.Errorf("readiness: got %q, want %q", s.Status, StatusDown)
	}
	if s.HTTPStatus() != http.StatusServiceUnavailable {
		t.Errorf("readiness HTTPStatus: got %d, want 503", s.HTTPStatus())
	}
}

func TestSnapshot_Readiness_DegradedOnDegradedSeverityDown(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(downChecker("email", "smtp down"), WithSeverity(SeverityDegraded))
	_ = reg.Register(upChecker("db"))
	reg.RunAll(context.Background())

	s := reg.SnapshotForProbe(ProbeReadiness)
	if s.Status != StatusDegraded {
		t.Errorf("readiness: got %q, want %q", s.Status, StatusDegraded)
	}
	if s.HTTPStatus() != http.StatusOK {
		t.Errorf("degraded should still be 200 (serving traffic), got %d", s.HTTPStatus())
	}
}

func TestSnapshot_Readiness_AllUp(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(upChecker("db"))
	_ = reg.Register(upChecker("cache"))
	reg.RunAll(context.Background())

	s := reg.SnapshotForProbe(ProbeReadiness)
	if s.Status != StatusUp {
		t.Errorf("readiness: got %q, want %q", s.Status, StatusUp)
	}
}

func TestSnapshot_Startup_UnknownUntilFirstUp(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(downChecker("db", "boom"))
	reg.RunAll(context.Background())

	s := reg.SnapshotForProbe(ProbeStartup)
	if s.Status != StatusUnknown {
		t.Errorf("startup before first up: got %q, want %q", s.Status, StatusUnknown)
	}
	if s.HTTPStatus() != http.StatusServiceUnavailable {
		t.Errorf("unknown HTTPStatus: got %d, want 503", s.HTTPStatus())
	}
}

func TestSnapshot_Startup_TracksReadinessOnceUp(t *testing.T) {
	reg := NewRegistry()
	c := &fakeChecker{name: "db"}
	state := StatusDown
	c.fn = func(_ context.Context) Result { return Result{Status: state} }
	_ = reg.Register(c)

	// First run: down → startup unknown.
	reg.RunAll(context.Background())
	if reg.SnapshotForProbe(ProbeStartup).Status != StatusUnknown {
		t.Fatalf("expected unknown before first up")
	}

	// Flip to up: startup mirrors readiness.
	state = StatusUp
	reg.RunAll(context.Background())
	if reg.SnapshotForProbe(ProbeStartup).Status != StatusUp {
		t.Errorf("expected up after first up")
	}

	// Flip back to down: startup now mirrors readiness (down), no longer unknown.
	state = StatusDown
	reg.RunAll(context.Background())
	if got := reg.SnapshotForProbe(ProbeStartup).Status; got != StatusDown {
		t.Errorf("after first up, startup tracks readiness: got %q, want %q", got, StatusDown)
	}
}

func TestSnapshot_Empty_Registry_IsUp(t *testing.T) {
	reg := NewRegistry()
	if reg.SnapshotForProbe(ProbeReadiness).Status != StatusUp {
		t.Errorf("empty registry should be up")
	}
}

func TestSnapshot_InfoSeverity_NeverAffectsReadiness(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(downChecker("info", "boom"), WithSeverity(SeverityInfo))
	reg.RunAll(context.Background())

	s := reg.SnapshotForProbe(ProbeReadiness)
	if s.Status != StatusUp {
		t.Errorf("info-only down: got %q, want StatusUp", s.Status)
	}
}

func TestSnapshot_JSON_RoundTrip(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(upChecker("db"))
	reg.RunAll(context.Background())

	s := reg.SnapshotForProbe(ProbeReadiness)
	b, err := s.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}

	var got Snapshot
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v body=%s", err, b)
	}
	if got.Probe != ProbeReadiness {
		t.Errorf("probe: got %q, want %q", got.Probe, ProbeReadiness)
	}
	if got.Status != StatusUp {
		t.Errorf("status: got %q, want %q", got.Status, StatusUp)
	}
}

func TestSnapshot_UnknownProbe_ReturnsUnknown(t *testing.T) {
	reg := NewRegistry()
	s := reg.SnapshotForProbe("not-a-probe")
	if s.Status != StatusUnknown {
		t.Errorf("unknown probe: got %q, want %q", s.Status, StatusUnknown)
	}
}
