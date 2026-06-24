package queuequery

import "testing"

type fakeHolder struct {
	query string
}

func (f *fakeHolder) SetQuery(q string) { f.query = q }
func (f *fakeHolder) Query() string     { return f.query }

type fakeStore struct {
	updates map[string]any
}

func (f *fakeStore) Set(updates map[string]any) error {
	f.updates = updates
	return nil
}

func TestSetDelegatesToHolderAndPersists(t *testing.T) {
	h := &fakeHolder{}
	s := &fakeStore{}
	uc := New(h, s)

	uc.Set("SELECT 1")
	if h.query != "SELECT 1" {
		t.Fatalf("got %q, want %q", h.query, "SELECT 1")
	}
	if s.updates["queue.query"] != "SELECT 1" {
		t.Fatalf("persisted %v, want queue.query=SELECT 1", s.updates)
	}
}

func TestGetReturnsHolderQuery(t *testing.T) {
	h := &fakeHolder{query: "SELECT 2"}
	uc := New(h, &fakeStore{})

	if got := uc.Get(); got != "SELECT 2" {
		t.Fatalf("got %q, want %q", got, "SELECT 2")
	}
}
