package healthhttpprobe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ubgo/health"
)

func TestCheck_Up_2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New("api", srv.URL)
	r := c.Check(context.Background())
	if r.Status != health.StatusUp {
		t.Errorf("Status: got %q", r.Status)
	}
}

func TestCheck_Up_3xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusMovedPermanently)
	}))
	defer srv.Close()

	// Use a client that does NOT follow redirects so we observe the 301 directly.
	noFollow := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	c := New("api", srv.URL, WithClient(noFollow))
	r := c.Check(context.Background())
	if r.Status != health.StatusUp {
		t.Errorf("Status: got %q for 301", r.Status)
	}
}

func TestCheck_Down_5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New("api", srv.URL)
	r := c.Check(context.Background())
	if r.Status != health.StatusDown {
		t.Errorf("Status: got %q", r.Status)
	}
	if !strings.Contains(r.Error, "500") {
		t.Errorf("Error: got %q", r.Error)
	}
}

func TestCheck_NetworkError(t *testing.T) {
	c := New("api", "http://localhost:1") // assume nothing listening
	r := c.Check(context.Background())
	if r.Status != health.StatusDown {
		t.Errorf("Status: got %q", r.Status)
	}
}

func TestCheck_BadURL(t *testing.T) {
	c := New("api", "://broken")
	r := c.Check(context.Background())
	if r.Status != health.StatusDown {
		t.Errorf("Status: got %q", r.Status)
	}
}

func TestWithMethod_HEAD(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("server saw method %q, want HEAD", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New("api", srv.URL, WithMethod(http.MethodHead))
	r := c.Check(context.Background())
	if r.Status != health.StatusUp {
		t.Errorf("Status: got %q", r.Status)
	}
}

func TestWithAccept_CustomPredicate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	// Treat 418 as healthy via custom predicate.
	c := New("api", srv.URL, WithAccept(func(code int) bool { return code == http.StatusTeapot }))
	r := c.Check(context.Background())
	if r.Status != health.StatusUp {
		t.Errorf("custom accept: got %q", r.Status)
	}
}
