package health

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

// ErrAlreadyRegistered is returned by Register when a checker with the given
// Name is already in the registry.
var ErrAlreadyRegistered = errors.New("health: checker already registered with that name")

// ErrNotFound is returned by Run when the requested checker name is not in
// the registry.
var ErrNotFound = errors.New("health: checker not registered with that name")

// RegisterOption configures a Register call.
type RegisterOption func(*registerConfig)

type registerConfig struct {
	severity Severity
	timeout  time.Duration
}

// WithSeverity overrides the default severity (Critical).
func WithSeverity(s Severity) RegisterOption {
	return func(c *registerConfig) { c.severity = s }
}

// WithTimeout caps how long the Check can run. Zero means no per-check
// timeout (use the parent ctx). Default: 5 seconds.
func WithTimeout(d time.Duration) RegisterOption {
	return func(c *registerConfig) { c.timeout = d }
}

type checkerEntry struct {
	checker  Checker
	severity Severity
	timeout  time.Duration
}

// Registry is the thread-safe central store of checkers and their last results.
type Registry struct {
	mu        sync.RWMutex
	checkers  map[string]checkerEntry
	results   map[string]Result
	observers []Observer

	// Background re-check loop state.
	bgMu     sync.Mutex
	bgCancel context.CancelFunc
	bgDone   chan struct{}

	// startupSeen flips true after the first all-critical-up snapshot is observed.
	// Once true, ProbeStartup mirrors ProbeReadiness.
	startupMu   sync.RWMutex
	startupSeen bool
}

// NewRegistry constructs an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		checkers: make(map[string]checkerEntry),
		results:  make(map[string]Result),
	}
}

// Register adds a checker. Returns ErrAlreadyRegistered if the name is taken.
func (r *Registry) Register(c Checker, opts ...RegisterOption) error {
	cfg := &registerConfig{
		severity: SeverityCritical,
		timeout:  5 * time.Second,
	}
	for _, o := range opts {
		o(cfg)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	name := c.Name()
	if _, exists := r.checkers[name]; exists {
		return ErrAlreadyRegistered
	}
	r.checkers[name] = checkerEntry{
		checker:  c,
		severity: cfg.severity,
		timeout:  cfg.timeout,
	}
	r.results[name] = Result{
		Status:    StatusUnknown,
		Severity:  cfg.severity,
		Timestamp: time.Now(),
	}
	return nil
}

// Unregister removes a checker. Idempotent; no-op if name not registered.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.checkers, name)
	delete(r.results, name)
}

// RunAll invokes every registered checker concurrently and stores their
// Results in the registry. Returns once all checks complete or ctx is done.
func (r *Registry) RunAll(ctx context.Context) {
	r.mu.RLock()
	entries := make(map[string]checkerEntry, len(r.checkers))
	for n, e := range r.checkers {
		entries[n] = e
	}
	r.mu.RUnlock()

	var wg sync.WaitGroup
	for name, entry := range entries {
		wg.Add(1)
		go func(name string, entry checkerEntry) {
			defer wg.Done()
			r.runOne(ctx, name, entry)
		}(name, entry)
	}
	wg.Wait()
}

// Run invokes a single checker by name and stores its Result. Returns
// ErrNotFound if the name is not registered.
func (r *Registry) Run(ctx context.Context, name string) (Result, error) {
	r.mu.RLock()
	entry, ok := r.checkers[name]
	r.mu.RUnlock()
	if !ok {
		return Result{}, ErrNotFound
	}
	return r.runOne(ctx, name, entry), nil
}

func (r *Registry) runOne(ctx context.Context, name string, entry checkerEntry) Result {
	checkCtx := ctx
	if entry.timeout > 0 {
		var cancel context.CancelFunc
		checkCtx, cancel = context.WithTimeout(ctx, entry.timeout)
		defer cancel()
	}

	start := time.Now()
	result := entry.checker.Check(checkCtx)
	if result.Latency == 0 {
		result.Latency = time.Since(start)
	}
	if result.Timestamp.IsZero() {
		result.Timestamp = time.Now()
	}
	if result.Severity == "" {
		result.Severity = entry.severity
	}

	r.mu.Lock()
	r.results[name] = result
	observers := append([]Observer(nil), r.observers...)
	r.mu.Unlock()

	for _, ob := range observers {
		ob(name, result)
	}
	return result
}

// Snapshot returns a copy of the current per-checker results. The returned
// map is owned by the caller and safe to mutate.
func (r *Registry) Snapshot() map[string]Result {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]Result, len(r.results))
	for k, v := range r.results {
		out[k] = v
	}
	return out
}

// Subscribe registers an Observer that will be invoked synchronously after
// every Check. Subscribe is safe to call concurrently with checks.
func (r *Registry) Subscribe(o Observer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observers = append(r.observers, o)
}

// Start launches a background goroutine that calls RunAll on the given
// interval. If interval is zero or negative, RunAll runs once and returns.
// Start is safe to call multiple times — a second call replaces the loop
// from the first.
func (r *Registry) Start(ctx context.Context, interval time.Duration) {
	r.bgMu.Lock()
	if r.bgCancel != nil {
		oldCancel, oldDone := r.bgCancel, r.bgDone
		r.bgMu.Unlock()
		oldCancel()
		<-oldDone
		r.bgMu.Lock()
	}
	loopCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	r.bgCancel = cancel
	r.bgDone = done
	r.bgMu.Unlock()

	go func() {
		defer close(done)
		r.RunAll(loopCtx)
		if interval <= 0 {
			return
		}
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-loopCtx.Done():
				return
			case <-t.C:
				r.RunAll(loopCtx)
			}
		}
	}()
}

// Stop terminates the background loop started by Start. Idempotent.
func (r *Registry) Stop() {
	r.bgMu.Lock()
	cancel, done := r.bgCancel, r.bgDone
	r.bgCancel = nil
	r.bgDone = nil
	r.bgMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

// names returns the registered checker names in alphabetical order so
// snapshot iteration is deterministic.
func (r *Registry) names() []string {
	r.mu.RLock()
	out := make([]string, 0, len(r.checkers))
	for n := range r.checkers {
		out = append(out, n)
	}
	r.mu.RUnlock()
	sort.Strings(out)
	return out
}

// markStartupSeen flips startupSeen to true on the first Up readiness snapshot.
func (r *Registry) markStartupSeen() {
	r.startupMu.Lock()
	r.startupSeen = true
	r.startupMu.Unlock()
}

func (r *Registry) hasStartupSeen() bool {
	r.startupMu.RLock()
	defer r.startupMu.RUnlock()
	return r.startupSeen
}
