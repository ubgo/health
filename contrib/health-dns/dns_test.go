package healthdns

import (
	"context"
	"errors"
	"testing"

	"github.com/ubgo/health"
)

type fakeResolver struct {
	addrs []string
	err   error
}

func (f *fakeResolver) LookupHost(_ context.Context, _ string) ([]string, error) {
	return f.addrs, f.err
}

func TestCheck_Up(t *testing.T) {
	c := New("dns", "example.com", WithResolver(&fakeResolver{addrs: []string{"1.2.3.4", "5.6.7.8"}}))
	r := c.Check(context.Background())
	if r.Status != health.StatusUp {
		t.Errorf("Status: got %q", r.Status)
	}
	if r.Metadata["resolved_count"] != 2 {
		t.Errorf("resolved_count: got %v", r.Metadata["resolved_count"])
	}
}

func TestCheck_Down_OnError(t *testing.T) {
	c := New("dns", "example.com", WithResolver(&fakeResolver{err: errors.New("nxdomain")}))
	r := c.Check(context.Background())
	if r.Status != health.StatusDown {
		t.Errorf("Status: got %q", r.Status)
	}
	if r.Error != "nxdomain" {
		t.Errorf("Error: got %q", r.Error)
	}
}

func TestCheck_Down_BelowMinHosts(t *testing.T) {
	c := New("dns", "example.com",
		WithResolver(&fakeResolver{addrs: []string{"1.2.3.4"}}),
		WithMinHosts(3),
	)
	r := c.Check(context.Background())
	if r.Status != health.StatusDown {
		t.Errorf("Status: got %q", r.Status)
	}
}

func TestName(t *testing.T) {
	c := New("dns", "example.com")
	if c.Name() != "dns" {
		t.Errorf("Name: got %q", c.Name())
	}
}
