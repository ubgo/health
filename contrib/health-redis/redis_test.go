package healthredis

import (
	"context"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"

	"github.com/ubgo/health"
)

type fakePinger struct{ err error }

func (f *fakePinger) Ping(_ context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(context.Background())
	if f.err != nil {
		cmd.SetErr(f.err)
	} else {
		cmd.SetVal("PONG")
	}
	return cmd
}

func TestCheck_Up(t *testing.T) {
	c := New("cache", &fakePinger{})
	r := c.Check(context.Background())
	if r.Status != health.StatusUp {
		t.Errorf("Status: got %q, want %q", r.Status, health.StatusUp)
	}
	if r.Severity != health.SeverityCritical {
		t.Errorf("Severity: got %q", r.Severity)
	}
}

func TestCheck_Down(t *testing.T) {
	c := New("cache", &fakePinger{err: errors.New("redis: nope")})
	r := c.Check(context.Background())
	if r.Status != health.StatusDown {
		t.Errorf("Status: got %q, want %q", r.Status, health.StatusDown)
	}
	if r.Error != "redis: nope" {
		t.Errorf("Error: got %q", r.Error)
	}
}

func TestName(t *testing.T) {
	c := New("cache", &fakePinger{})
	if c.Name() != "cache" {
		t.Errorf("Name: got %q", c.Name())
	}
}
