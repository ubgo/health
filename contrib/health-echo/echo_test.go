package healthecho

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/ubgo/health"
)

type fakeChecker struct {
	name string
	res  health.Result
}

func (f *fakeChecker) Name() string                         { return f.name }
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
	e := echo.New()
	Mount(e, reg)

	srv := httptest.NewServer(e)
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
	auth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Header.Get("X-Internal-Key") != "secret" {
				return c.NoContent(http.StatusUnauthorized)
			}
			return next(c)
		}
	}

	e := echo.New()
	Mount(e, reg, WithMiddleware(auth))

	srv := httptest.NewServer(e)
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
	e := echo.New()
	Mount(e, reg, WithReadinessPath("/ready"))
	srv := httptest.NewServer(e)
	defer srv.Close()
	resp, _ := http.Get(srv.URL + "/ready")
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		t.Errorf("custom /ready: 404")
	}
}
