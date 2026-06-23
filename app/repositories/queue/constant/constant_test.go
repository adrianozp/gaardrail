package constant

import (
	"context"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/pkg/config"
)

func TestDequeueReturnsConfiguredQuery(t *testing.T) {
	q := New(config.Config{Queue: config.Queue{Query: "SELECT 1"}})

	m, err := q.Dequeue(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(m.Body) != "SELECT 1" {
		t.Fatalf("got body %q, want %q", m.Body, "SELECT 1")
	}
	if m.ID == "" {
		t.Fatal("expected a generated message ID")
	}
}

func TestDequeueGeneratesFreshIDs(t *testing.T) {
	q := New(config.Config{Queue: config.Queue{Query: "SELECT 1"}})

	a, _ := q.Dequeue(context.Background())
	b, _ := q.Dequeue(context.Background())
	if a.ID == b.ID {
		t.Fatal("expected distinct IDs across dequeues")
	}
}

func TestSetQuerySwapsPayload(t *testing.T) {
	q := New(config.Config{})
	q.SetQuery("SELECT 2")

	if got := q.Query(); got != "SELECT 2" {
		t.Fatalf("got %q, want %q", got, "SELECT 2")
	}
	m, _ := q.Dequeue(context.Background())
	if string(m.Body) != "SELECT 2" {
		t.Fatalf("got body %q, want %q", m.Body, "SELECT 2")
	}
}

func TestDequeueBlocksUntilQuerySet(t *testing.T) {
	q := New(config.Config{})

	done := make(chan string, 1)
	go func() {
		m, _ := q.Dequeue(context.Background())
		done <- string(m.Body)
	}()

	select {
	case <-done:
		t.Fatal("dequeue returned before a query was set")
	case <-time.After(20 * time.Millisecond):
	}

	q.SetQuery("SELECT 3")
	select {
	case body := <-done:
		if body != "SELECT 3" {
			t.Fatalf("got body %q, want %q", body, "SELECT 3")
		}
	case <-time.After(time.Second):
		t.Fatal("dequeue did not unblock after SetQuery")
	}
}

func TestDequeueRespectsContext(t *testing.T) {
	q := New(config.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := q.Dequeue(ctx); err == nil {
		t.Fatal("expected error from canceled context")
	}
}

func TestSizeReturnsMinusOne(t *testing.T) {
	q := New(config.Config{})
	if size, _ := q.Size(); size != -1 {
		t.Fatalf("got size %d, want -1", size)
	}
}
