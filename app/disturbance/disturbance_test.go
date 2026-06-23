package disturbance

import (
	"context"
	"sync"
	"testing"
	"time"
)

type fakeExecutor struct {
	mu    sync.Mutex
	calls int
	last  string
}

func (f *fakeExecutor) ExecContext(_ context.Context, query string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.last = query
	return 0, nil
}

func (f *fakeExecutor) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func TestSetRunsQueryAtRate(t *testing.T) {
	exec := &fakeExecutor{}
	d := New(exec)

	if err := d.Set("SELECT 1", 200, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	d.Set("", 0, 0)

	if exec.count() == 0 {
		t.Fatal("expected the query to have run at least once")
	}
	if exec.last != "SELECT 1" {
		t.Fatalf("got last query %q, want %q", exec.last, "SELECT 1")
	}
}

func TestSetRateZeroStops(t *testing.T) {
	exec := &fakeExecutor{}
	d := New(exec)

	d.Set("SELECT 1", 200, 0)
	time.Sleep(30 * time.Millisecond)
	d.Set("", 0, 0)

	settled := exec.count()
	time.Sleep(50 * time.Millisecond)
	if exec.count() != settled {
		t.Fatal("expected no further queries after stopping")
	}
	if d.Get().Rate != 0 {
		t.Fatalf("got rate %v, want 0", d.Get().Rate)
	}
}

func TestTimedPulseAutoStops(t *testing.T) {
	exec := &fakeExecutor{}
	d := New(exec)

	d.Set("SELECT 1", 200, 40*time.Millisecond)
	time.Sleep(120 * time.Millisecond)

	if d.Get().Rate != 0 {
		t.Fatalf("got rate %v after pulse, want 0", d.Get().Rate)
	}
	settled := exec.count()
	time.Sleep(50 * time.Millisecond)
	if exec.count() != settled {
		t.Fatal("expected no further queries after pulse expired")
	}
}

func TestNilExecutorIgnored(t *testing.T) {
	d := New(nil)

	if err := d.Set("SELECT 1", 100, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Get().Rate != 0 {
		t.Fatalf("got rate %v, want 0 with nil executor", d.Get().Rate)
	}
}
