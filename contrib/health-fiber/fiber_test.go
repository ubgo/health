package healthfiber

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

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

func httpReq(t *testing.T, app *fiber.App, method, path string, headers map[string]string) (int, []byte) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body
}

func TestMount_DefaultPaths(t *testing.T) {
	reg := newReadyReg(t)
	app := fiber.New()
	Mount(app, reg)

	for _, path := range []string{"/healthz", "/readyz", "/startupz"} {
		status, _ := httpReq(t, app, http.MethodGet, path, nil)
		if status == http.StatusNotFound {
			t.Errorf("%s: 404", path)
		}
	}
}

func TestMount_WithMiddleware(t *testing.T) {
	reg := newReadyReg(t)
	auth := func(c *fiber.Ctx) error {
		if c.Get("X-Internal-Key") != "secret" {
			return c.SendStatus(http.StatusUnauthorized)
		}
		return c.Next()
	}

	app := fiber.New()
	Mount(app, reg, WithMiddleware(auth))

	statusNoKey, _ := httpReq(t, app, http.MethodGet, "/readyz", nil)
	if statusNoKey != http.StatusUnauthorized {
		t.Errorf("missing key: got %d, want 401", statusNoKey)
	}

	statusWithKey, _ := httpReq(t, app, http.MethodGet, "/readyz", map[string]string{"X-Internal-Key": "secret"})
	if statusWithKey != http.StatusOK {
		t.Errorf("with key: got %d, want 200", statusWithKey)
	}
}

func TestMount_OverridePaths(t *testing.T) {
	reg := newReadyReg(t)
	app := fiber.New()
	Mount(app, reg, WithStartupPath("/started"))
	status, _ := httpReq(t, app, http.MethodGet, "/started", nil)
	if status == http.StatusNotFound {
		t.Errorf("custom /started: 404")
	}
}
