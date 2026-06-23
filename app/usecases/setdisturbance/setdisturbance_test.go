package setdisturbance

import (
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/disturbance"
)

type fakeComponent struct {
	query string
	rate  float64
	ttl   time.Duration
}

func (f *fakeComponent) Set(query string, rate float64, ttl time.Duration) error {
	f.query, f.rate, f.ttl = query, rate, ttl
	return nil
}

func (f *fakeComponent) Get() disturbance.State {
	return disturbance.State{Query: f.query, Rate: f.rate, Duration: f.ttl}
}

func TestSetDelegatesToComponent(t *testing.T) {
	c := &fakeComponent{}
	uc := New(c)

	if err := uc.Set("SELECT 1", 50, 2*time.Second); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.query != "SELECT 1" || c.rate != 50 || c.ttl != 2*time.Second {
		t.Fatalf("delegation mismatch: %+v", c)
	}
}

func TestGetReturnsComponentState(t *testing.T) {
	c := &fakeComponent{query: "SELECT 2", rate: 10}
	uc := New(c)

	if got := uc.Get(); got.Query != "SELECT 2" || got.Rate != 10 {
		t.Fatalf("got %+v", got)
	}
}
