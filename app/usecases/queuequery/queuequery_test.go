package queuequery

import "testing"

type fakeHolder struct {
	query string
}

func (f *fakeHolder) SetQuery(q string) { f.query = q }
func (f *fakeHolder) Query() string     { return f.query }

func TestSetDelegatesToHolder(t *testing.T) {
	h := &fakeHolder{}
	uc := New(h)

	uc.Set("SELECT 1")
	if h.query != "SELECT 1" {
		t.Fatalf("got %q, want %q", h.query, "SELECT 1")
	}
}

func TestGetReturnsHolderQuery(t *testing.T) {
	h := &fakeHolder{query: "SELECT 2"}
	uc := New(h)

	if got := uc.Get(); got != "SELECT 2" {
		t.Fatalf("got %q, want %q", got, "SELECT 2")
	}
}
