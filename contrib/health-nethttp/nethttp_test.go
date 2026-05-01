package healthnethttp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ubgo/health"
)

type fakeChecker struct {
	name string
	res  health.Result
}

func (f *fakeChecker) Name() string                          { return f.name }
func (f *fakeChecker) Check(_ context.Context) health.Result { return f.res }

func newReadyReg(t *testing.T) *health.Registry {
	t.Helper()
	reg := health.NewRegistry()
	if err := reg.Register(&fakeChecker{name: "db", res: health.Result{Status: health.StatusUp}}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	reg.RunAll(context.Background())
	return reg
}

func TestLiveness_AlwaysOK(t *testing.T) {
	reg := health.NewRegistry()
	srv := httptest.NewServer(Liveness(reg))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
}

func TestReadiness_DownOnCriticalDown(t *testing.T) {
	reg := health.NewRegistry()
	_ = reg.Register(&fakeChecker{name: "db", res: health.Result{Status: health.StatusDown}})
	reg.RunAll(context.Background())

	srv := httptest.NewServer(Readiness(reg))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want 503", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var snap health.Snapshot
	if err := json.Unmarshal(body, &snap); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, body)
	}
	if snap.Probe != health.ProbeReadiness || snap.Status != health.StatusDown {
		t.Errorf("snap: got %+v", snap)
	}
}

func TestMount_DefaultPaths(t *testing.T) {
	reg := newReadyReg(t)
	mux := http.NewServeMux()
	Mount(mux, reg)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, path := range []string{"/healthz", "/readyz", "/startupz"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		_ = resp.Body.Close()
	}
}

func TestMount_WithPathOverrides(t *testing.T) {
	reg := newReadyReg(t)
	mux := http.NewServeMux()
	Mount(mux, reg,
		WithLivenessPath("/live"),
		WithReadinessPath("/ready"),
		WithStartupPath("/started"),
	)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, path := range []string{"/live", "/ready", "/started"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		if resp.StatusCode == http.StatusNotFound {
			t.Errorf("%s: 404", path)
		}
		_ = resp.Body.Close()
	}
	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("/healthz should be 404 after override, got %d", resp.StatusCode)
	}
}

func TestMount_WithMiddleware(t *testing.T) {
	reg := newReadyReg(t)
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Internal-Key") != "secret" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	mux := http.NewServeMux()
	Mount(mux, reg, WithMiddleware(auth))

	srv := httptest.NewServer(mux)
	defer srv.Close()

	respNoKey, err := http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer respNoKey.Body.Close()
	if respNoKey.StatusCode != http.StatusUnauthorized {
		t.Errorf("missing key: got %d, want 401", respNoKey.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/readyz", nil)
	req.Header.Set("X-Internal-Key", "secret")
	respWithKey, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer respWithKey.Body.Close()
	if respWithKey.StatusCode != http.StatusOK {
		t.Errorf("with key: got %d, want 200", respWithKey.StatusCode)
	}
}
