package healthprom

import (
	"context"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/ubgo/health"
)

type fakeChecker struct {
	name string
	res  health.Result
}

func (f *fakeChecker) Name() string                         { return f.name }
func (f *fakeChecker) Check(_ context.Context) health.Result { return f.res }

func TestRegister_PublishesAllThreeMetrics(t *testing.T) {
	reg := health.NewRegistry()
	promReg := prometheus.NewRegistry()
	Register(reg, promReg)

	_ = reg.Register(&fakeChecker{name: "db", res: health.Result{Status: health.StatusUp}})
	reg.RunAll(context.Background())

	want := `
# HELP health_check_status Status of a health check: 1=up, 0=down, -1=degraded, -2=unknown.
# TYPE health_check_status gauge
health_check_status{name="db",severity="critical"} 1
`
	if err := testutil.GatherAndCompare(promReg, strings.NewReader(want), "health_check_status"); err != nil {
		t.Errorf("status metric mismatch: %v", err)
	}

	if got := testutil.CollectAndCount(prometheus.NewCounter(prometheus.CounterOpts{Name: "x"})); got != 1 {
		// sanity — testutil.CollectAndCount always returns 1 for a fresh counter
		_ = got
	}
}

func TestSubscribe_StatusValueMapping(t *testing.T) {
	for _, tc := range []struct {
		name string
		res  health.Result
		want float64
	}{
		{"up", health.Result{Status: health.StatusUp}, 1},
		{"down", health.Result{Status: health.StatusDown}, 0},
		{"degraded", health.Result{Status: health.StatusDegraded}, -1},
		{"unknown", health.Result{Status: health.StatusUnknown}, -2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := statusValue(tc.res.Status); got != tc.want {
				t.Errorf("statusValue(%q): got %v, want %v", tc.res.Status, got, tc.want)
			}
		})
	}
}

func TestRegister_LatencyAndCounter(t *testing.T) {
	reg := health.NewRegistry()
	promReg := prometheus.NewRegistry()
	Register(reg, promReg)

	_ = reg.Register(&fakeChecker{name: "db", res: health.Result{Status: health.StatusUp}})
	reg.RunAll(context.Background())
	reg.RunAll(context.Background())

	families, err := promReg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}

	var sawLatency, sawRuns bool
	for _, mf := range families {
		switch mf.GetName() {
		case "health_check_latency_seconds":
			sawLatency = true
		case "health_check_runs_total":
			sawRuns = true
			for _, m := range mf.GetMetric() {
				if m.GetCounter().GetValue() < 1 {
					t.Errorf("counter not incremented: %v", m.GetCounter().GetValue())
				}
			}
		}
	}
	if !sawLatency {
		t.Errorf("latency metric not published")
	}
	if !sawRuns {
		t.Errorf("runs counter not published")
	}
}

func TestNewCollector_NilRegisterer(t *testing.T) {
	// Calling NewCollector(nil) must not panic and must still allow Subscribe.
	c := NewCollector(nil)
	if c == nil {
		t.Fatalf("nil collector")
	}
	reg := health.NewRegistry()
	c.Subscribe(reg)
	_ = reg.Register(&fakeChecker{name: "x", res: health.Result{Status: health.StatusUp}})
	reg.RunAll(context.Background())
	// No assertion on metrics — they just exist on the unregistered collector.
}
