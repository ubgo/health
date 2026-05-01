package healthgin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ubgo/health"
)

func init() { gin.SetMode(gin.TestMode) }

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

func TestMount_DefaultPaths(t *testing.T) {
	reg := newReadyReg(t)
	r := gin.New()
	Mount(r, reg)

	srv := httptest.NewServer(r)
	defer srv.Close()

	for _, path := range []string{"/healthz", "/readyz", "/startupz"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		if resp.StatusCode == http.StatusNotFound {
			t.Errorf("%s: 404", path)
		}
		_ = resp.Body.Close()
	}
}

func TestMount_WithMiddleware(t *testing.T) {
	reg := newReadyReg(t)
	auth := func(c *gin.Context) {
		if c.GetHeader("X-Internal-Key") != "secret" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}

	r := gin.New()
	Mount(r, reg, WithMiddleware(auth))

	srv := httptest.NewServer(r)
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

func TestMount_OverridePaths(t *testing.T) {
	reg := newReadyReg(t)
	r := gin.New()
	Mount(r, reg, WithLivenessPath("/live"), WithReadinessPath("/ready"), WithStartupPath("/started"))

	srv := httptest.NewServer(r)
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
}
