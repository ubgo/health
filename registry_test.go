package health

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeChecker is a deterministic Checker for tests.
type fakeChecker struct {
	name string
	fn   func(ctx context.Context) Result
}

func (f *fakeChecker) Name() string                       { return f.name }
func (f *fakeChecker) Check(ctx context.Context) Result   { return f.fn(ctx) }

func upChecker(name string) *fakeChecker {
	return &fakeChecker{
		name: name,
		fn:   func(_ context.Context) Result { return Result{Status: StatusUp} },
	}
}

func downChecker(name string, msg string) *fakeChecker {
	return &fakeChecker{
		name: name,
		fn: func(_ context.Context) Result {
			return Result{Status: StatusDown, Error: msg}
		},
	}
}

func TestRegistry_RegisterAndUnregister(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(upChecker("a")); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := reg.Register(upChecker("a")); !errors.Is(err, ErrAlreadyRegistered) {
		t.Errorf("duplicate Register: got %v, want ErrAlreadyRegistered", err)
	}
	reg.Unregister("a")
	if err := reg.Register(upChecker("a")); err != nil {
		t.Fatalf("Register after Unregister: %v", err)
	}
}

func TestRegistry_RunAll_StoresResults(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(upChecker("a"))
	_ = reg.Register(downChecker("b", "boom"))

	reg.RunAll(context.Background())
	snap := reg.Snapshot()

	if snap["a"].Status != StatusUp {
		t.Errorf("a: got %q, want %q", snap["a"].Status, StatusUp)
	}
	if snap["b"].Status != StatusDown {
		t.Errorf("b: got %q, want %q", snap["b"].Status, StatusDown)
	}
	if snap["b"].Error != "boom" {
		t.Errorf("b.Error: got %q, want %q", snap["b"].Error, "boom")
	}
}

func TestRegistry_Run_NotFound(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Run(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Run missing: got %v, want ErrNotFound", err)
	}
}

func TestRegistry_Run_OneByName(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(upChecker("a"))
	_ = reg.Register(downChecker("b", "boom"))

	res, err := reg.Run(context.Background(), "a")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Status != StatusUp {
		t.Errorf("a Status: got %q", res.Status)
	}
	// Only "a" was run; "b" should still be Unknown from registration.
	if reg.Snapshot()["b"].Status != StatusUnknown {
		t.Errorf("b should still be Unknown")
	}
}

func TestRegistry_RunOne_AppliesTimeout(t *testing.T) {
	slow := &fakeChecker{
		name: "slow",
		fn: func(ctx context.Context) Result {
			select {
			case <-time.After(500 * time.Millisecond):
				return Result{Status: StatusUp}
			case <-ctx.Done():
				return Result{Status: StatusDown, Error: ctx.Err().Error()}
			}
		},
	}
	reg := NewRegistry()
	_ = reg.Register(slow, WithTimeout(10*time.Millisecond))

	res, err := reg.Run(context.Background(), "slow")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Status != StatusDown {
		t.Errorf("expected Down on timeout, got %q (err=%q)", res.Status, res.Error)
	}
}

func TestRegistry_PopulatesLatencyAndTimestamp(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(upChecker("a"))
	reg.RunAll(context.Background())
	snap := reg.Snapshot()
	r := snap["a"]
	if r.Latency <= 0 {
		t.Errorf("Latency: got %v, want >0", r.Latency)
	}
	if r.Timestamp.IsZero() {
		t.Errorf("Timestamp: got zero")
	}
	if r.Severity != SeverityCritical {
		t.Errorf("default severity: got %q, want %q", r.Severity, SeverityCritical)
	}
}

func TestRegistry_WithSeverity(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(downChecker("info-only", "boom"), WithSeverity(SeverityInfo))
	reg.RunAll(context.Background())

	snap := reg.SnapshotForProbe(ProbeReadiness)
	if snap.Status != StatusUp {
		t.Errorf("readiness with only info-severity Down: got %q, want StatusUp",
			snap.Status)
	}
}

func TestRegistry_Subscribe_FiresPerCheck(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(upChecker("a"))
	_ = reg.Register(downChecker("b", "boom"))

	var calls int32
	reg.Subscribe(func(_ string, _ Result) { atomic.AddInt32(&calls, 1) })

	reg.RunAll(context.Background())

	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("observer fired %d times, want 2", got)
	}
}

func TestRegistry_StartStop_BackgroundLoop(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(upChecker("a"))

	var calls int32
	reg.Subscribe(func(_ string, _ Result) { atomic.AddInt32(&calls, 1) })

	ctx := context.Background()
	reg.Start(ctx, 25*time.Millisecond)
	time.Sleep(120 * time.Millisecond)
	reg.Stop()

	got := atomic.LoadInt32(&calls)
	if got < 3 {
		t.Errorf("expected at least 3 background runs, got %d", got)
	}
}

func TestRegistry_Stop_IsIdempotent(t *testing.T) {
	reg := NewRegistry()
	reg.Stop() // never started — must not panic
	reg.Start(context.Background(), 0)
	reg.Stop()
	reg.Stop() // double stop
}

func TestRegistry_Concurrent_RegisterRunSubscribe(t *testing.T) {
	reg := NewRegistry()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(3)
		go func(i int) {
			defer wg.Done()
			_ = reg.Register(upChecker("c-" + itoa(i)))
		}(i)
		go func() {
			defer wg.Done()
			reg.RunAll(context.Background())
		}()
		go func() {
			defer wg.Done()
			reg.Subscribe(func(_ string, _ Result) {})
		}()
	}
	wg.Wait()
}

// itoa is a tiny helper to avoid pulling fmt into the hot loop.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
