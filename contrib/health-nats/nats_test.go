package healthnats

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"

	"github.com/ubgo/health"
)

type fakeConn struct {
	status nats.Status
}

func (f *fakeConn) IsConnected() bool   { return f.status == nats.CONNECTED }
func (f *fakeConn) Status() nats.Status { return f.status }

func TestCheck_Up(t *testing.T) {
	c := New("messaging", &fakeConn{status: nats.CONNECTED})
	r := c.Check(context.Background())
	if r.Status != health.StatusUp {
		t.Errorf("Status: got %q", r.Status)
	}
}

func TestCheck_Reconnecting_IsDegraded(t *testing.T) {
	c := New("messaging", &fakeConn{status: nats.RECONNECTING})
	r := c.Check(context.Background())
	if r.Status != health.StatusDegraded {
		t.Errorf("Status: got %q, want %q", r.Status, health.StatusDegraded)
	}
}

func TestCheck_Down_States(t *testing.T) {
	for _, st := range []nats.Status{nats.CLOSED, nats.DISCONNECTED, nats.DRAINING_PUBS, nats.DRAINING_SUBS} {
		c := New("messaging", &fakeConn{status: st})
		r := c.Check(context.Background())
		if r.Status != health.StatusDown {
			t.Errorf("status %v: got %q, want %q", st, r.Status, health.StatusDown)
		}
	}
}

func TestName(t *testing.T) {
	c := New("messaging", &fakeConn{})
	if c.Name() != "messaging" {
		t.Errorf("Name: got %q", c.Name())
	}
}
