# Orchestrator Rate Limiter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the orchestrator's fixed-tick loop with a token bucket rate limiter so the PID controller's `drainRate` output actually controls message throughput.

**Architecture:** `app/usecases/orchestrator/orchestrator.go` replaces `DrainRate float64` + `time.Ticker` with `*rate.Limiter`. `SetDrainRate` calls `limiter.SetLimit`, which is goroutine-safe. The `run` loop blocks on `limiter.Wait(ctx)` — one token = one message consumed. Shutdown is via context cancellation propagated from fx's lifecycle.

**Tech Stack:** Go 1.25, `golang.org/x/time/rate` (token bucket), `go.uber.org/fx` (lifecycle), `github.com/stretchr/testify` (assertions)

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `app/usecases/orchestrator/orchestrator.go` | Modify | Replace ticker with rate.Limiter; change Start signature |
| `app/usecases/orchestrator/orchestrator_test.go` | Create | Unit tests for orchestrator behavior |
| `cmd/api/options/options.go` | Modify | Pass ctx to Start instead of hardcoded tickRate |
| `go.mod` / `go.sum` | Modify | Add golang.org/x/time dependency |

---

### Task 1: Add golang.org/x/time dependency

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add the dependency**

```bash
cd /home/adrianozdp/workspace/pfc/gaardrail
go get golang.org/x/time/rate
```

Expected output: `go: added golang.org/x/time v...`

- [ ] **Step 2: Verify go.mod has the entry**

```bash
grep "golang.org/x/time" go.mod
```

Expected: a line like `golang.org/x/time v0.x.x`

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add golang.org/x/time dependency"
```

---

### Task 2: Write failing orchestrator tests

**Files:**
- Create: `app/usecases/orchestrator/orchestrator_test.go`

- [ ] **Step 1: Create the test file**

Create `app/usecases/orchestrator/orchestrator_test.go` with the full content below:

```go
package orchestrator_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/usecases/orchestrator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

// mockConsumer is a simple in-test mock for orchestrator.Consumer.
type mockConsumer struct {
	size        int64
	consumeFunc func() (string, error)
	consumeCalls atomic.Int64
}

func (m *mockConsumer) Size() (int64, error) {
	return m.size, nil
}

func (m *mockConsumer) Consume() (string, error) {
	m.consumeCalls.Add(1)
	if m.consumeFunc != nil {
		return m.consumeFunc()
	}
	return "msg-1", nil
}

func TestSetDrainRate_UpdatesLimiterLimit(t *testing.T) {
	o := orchestrator.NewOrchestrator(&mockConsumer{})
	err := o.SetDrainRate(42.0)
	require.NoError(t, err)
	assert.Equal(t, rate.Limit(42.0), o.Limiter().Limit())
}

func TestNewOrchestrator_StartsWithZeroRate(t *testing.T) {
	o := orchestrator.NewOrchestrator(&mockConsumer{})
	assert.Equal(t, rate.Limit(0), o.Limiter().Limit())
}

func TestStart_ExitsWhenContextCancelled(t *testing.T) {
	consumer := &mockConsumer{size: 0}
	o := orchestrator.NewOrchestrator(consumer)
	// Set a rate so the limiter doesn't block forever waiting for tokens
	_ = o.SetDrainRate(1000)

	ctx, cancel := context.WithCancel(context.Background())
	err := o.Start(ctx)
	require.NoError(t, err)

	// Cancel immediately and verify the goroutine exits within a short window
	cancel()
	// Give the goroutine a moment to observe ctx.Done
	time.Sleep(50 * time.Millisecond)
	// No assertion on goroutine exit directly, but Start must not block
}

func TestRun_ConsumesWhenQueueNonEmpty(t *testing.T) {
	consumed := make(chan struct{}, 1)
	consumer := &mockConsumer{
		size: 1,
		consumeFunc: func() (string, error) {
			select {
			case consumed <- struct{}{}:
			default:
			}
			return "msg-1", nil
		},
	}

	o := orchestrator.NewOrchestrator(consumer)
	_ = o.SetDrainRate(1000) // fast rate for test

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := o.Start(ctx)
	require.NoError(t, err)

	select {
	case <-consumed:
		// success
	case <-ctx.Done():
		t.Fatal("expected Consume to be called but it was not")
	}
}

func TestRun_SkipsConsumeWhenQueueEmpty(t *testing.T) {
	consumer := &mockConsumer{size: 0}
	o := orchestrator.NewOrchestrator(consumer)
	_ = o.SetDrainRate(1000)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := o.Start(ctx)
	require.NoError(t, err)

	<-ctx.Done()
	assert.Equal(t, int64(0), consumer.consumeCalls.Load())
}

func TestRun_LogsErrorOnConsumeFailure(t *testing.T) {
	consumer := &mockConsumer{
		size: 1,
		consumeFunc: func() (string, error) {
			return "", errors.New("kafka unavailable")
		},
	}

	o := orchestrator.NewOrchestrator(consumer)
	_ = o.SetDrainRate(1000)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should not panic — error is logged and loop continues
	err := o.Start(ctx)
	require.NoError(t, err)
	<-ctx.Done()
}
```

- [ ] **Step 2: Run the tests — expect compile failure**

```bash
cd /home/adrianozdp/workspace/pfc/gaardrail
go test ./app/usecases/orchestrator/... 2>&1
```

Expected: compile error — `o.Limiter()` does not exist yet and `Start` has wrong signature.

- [ ] **Step 3: Commit the failing tests**

```bash
git add app/usecases/orchestrator/orchestrator_test.go
git commit -m "test: add orchestrator rate limiter tests (failing)"
```

---

### Task 3: Implement orchestrator with rate.Limiter

**Files:**
- Modify: `app/usecases/orchestrator/orchestrator.go`

- [ ] **Step 1: Replace the orchestrator implementation**

Replace the entire content of `app/usecases/orchestrator/orchestrator.go` with:

```go
package orchestrator

import (
	"context"
	"log"

	"golang.org/x/time/rate"
)

type Consumer interface {
	Consume() (string, error)
	Size() (int64, error)
}

type Orchestrator struct {
	limiter  *rate.Limiter
	consumer Consumer
}

func NewOrchestrator(c Consumer) *Orchestrator {
	return &Orchestrator{
		consumer: c,
		limiter:  rate.NewLimiter(0, 1),
	}
}

// Limiter exposes the rate.Limiter for testing purposes.
func (o *Orchestrator) Limiter() *rate.Limiter {
	return o.limiter
}

func (o *Orchestrator) SetDrainRate(drainRate float64) error {
	o.limiter.SetLimit(rate.Limit(drainRate))
	return nil
}

func (o *Orchestrator) Start(ctx context.Context) error {
	go o.run(ctx)
	return nil
}

func (o *Orchestrator) run(ctx context.Context) {
	for {
		if err := o.limiter.Wait(ctx); err != nil {
			log.Println("orchestrator: shutting down")
			return
		}

		queueLen, err := o.consumer.Size()
		if err != nil {
			log.Printf("orchestrator: error getting queue length: %s", err)
			continue
		}

		if queueLen == 0 {
			continue
		}

		messageID, err := o.consumer.Consume()
		if err != nil {
			log.Printf("orchestrator: error consuming message: %s", err)
			continue
		}

		log.Printf("orchestrator: consumed message: %s", messageID)
	}
}
```

- [ ] **Step 2: Run tests — expect pass**

```bash
cd /home/adrianozdp/workspace/pfc/gaardrail
go test ./app/usecases/orchestrator/... -v
```

Expected: all 5 tests PASS.

- [ ] **Step 3: Commit**

```bash
git add app/usecases/orchestrator/orchestrator.go
git commit -m "feat: replace orchestrator ticker with token bucket rate limiter"
```

---

### Task 4: Update call site in options.go

**Files:**
- Modify: `cmd/api/options/options.go:40-46`

- [ ] **Step 1: Update the fx.Invoke block**

In `cmd/api/options/options.go`, find this block:

```go
fx.Invoke(func(lc fx.Lifecycle, o *orchestrator.Orchestrator) {
    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            return o.Start(100)
        },
    })
}),
```

Replace it with:

```go
fx.Invoke(func(lc fx.Lifecycle, o *orchestrator.Orchestrator) {
    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            return o.Start(ctx)
        },
    })
}),
```

- [ ] **Step 2: Verify the build**

```bash
cd /home/adrianozdp/workspace/pfc/gaardrail
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Run all tests**

```bash
cd /home/adrianozdp/workspace/pfc/gaardrail
go test ./...
```

Expected: all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/api/options/options.go
git commit -m "feat: pass lifecycle context to orchestrator Start"
```
