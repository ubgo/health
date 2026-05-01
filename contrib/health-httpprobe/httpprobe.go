// Package healthhttpprobe implements a generic outbound HTTP health.Checker.
// It issues a GET to a configured URL and considers the result Up if the
// response status is in the [200, 400) range (overridable).
package healthhttpprobe

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ubgo/health"
)

// AcceptStatus reports whether code counts as healthy. Default behaviour
// (the AcceptDefault function) accepts any 2xx or 3xx.
type AcceptStatus func(code int) bool

// AcceptDefault accepts any 2xx or 3xx status code.
func AcceptDefault(code int) bool { return code >= 200 && code < 400 }

// Option configures the Checker.
type Option func(*config)

type config struct {
	client *http.Client
	method string
	accept AcceptStatus
}

// WithClient overrides the http.Client used to issue probes. The default
// is a client with a 5s timeout.
func WithClient(c *http.Client) Option {
	return func(cfg *config) { cfg.client = c }
}

// WithMethod overrides the HTTP method used (default: GET).
func WithMethod(m string) Option {
	return func(cfg *config) { cfg.method = m }
}

// WithAccept overrides the status-code predicate.
func WithAccept(a AcceptStatus) Option {
	return func(cfg *config) { cfg.accept = a }
}

// Checker is a health.Checker that probes an HTTP endpoint.
type Checker struct {
	name string
	url  string
	cfg  *config
}

// New constructs a Checker that probes url and reports Up on a 2xx/3xx response.
func New(name, url string, opts ...Option) *Checker {
	cfg := &config{
		client: &http.Client{Timeout: 5 * time.Second},
		method: http.MethodGet,
		accept: AcceptDefault,
	}
	for _, o := range opts {
		o(cfg)
	}
	return &Checker{name: name, url: url, cfg: cfg}
}

// Name implements health.Checker.
func (c *Checker) Name() string { return c.name }

// Check implements health.Checker.
func (c *Checker) Check(ctx context.Context) health.Result {
	start := time.Now()
	r := health.Result{Severity: health.SeverityCritical, Timestamp: time.Now()}

	req, err := http.NewRequestWithContext(ctx, c.cfg.method, c.url, nil)
	if err != nil {
		r.Status = health.StatusDown
		r.Error = err.Error()
		r.Latency = time.Since(start)
		return r
	}

	resp, err := c.cfg.client.Do(req)
	r.Latency = time.Since(start)
	if err != nil {
		r.Status = health.StatusDown
		r.Error = err.Error()
		return r
	}
	defer resp.Body.Close()

	if c.cfg.accept(resp.StatusCode) {
		r.Status = health.StatusUp
		return r
	}
	r.Status = health.StatusDown
	r.Error = fmt.Sprintf("unexpected status %d", resp.StatusCode)
	return r
}
