# Metrics Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the Prometheus-coupled `pkg/metrics` with an internal `Recorder` interface backed by an async channel dispatcher, so metric sends never block the caller and the backend is swappable.

**Architecture:** A `Recorder` interface in `internal/metrics` is satisfied by a Prometheus implementation and a no-op (default). An `AsyncRecorder` wraps any `Recorder` with a buffered channel and a single worker goroutine, making all sends fire-and-forget. The global `Gauge`/`Incr` functions are unchanged; only the import path of existing call sites changes.

**Tech Stack:** Go stdlib (`sync`, `sync/atomic` not needed — channel suffices), `github.com/prometheus/client_golang/prometheus`, `go.uber.org/fx` (lifecycle wiring).

---

## File Map

| Action | Path | Responsibility |
|--------|------|----------------|
| Create | `internal/metrics/noop/noop.go` | Zero-allocation no-op Recorder (default global, test default) |
| Create | `internal/metrics/metrics.go` | Recorder interface, global var, SetRecorder, Gauge, Incr |
| Create | `internal/metrics/async.go` | AsyncRecorder: buffered channel + worker goroutine |
| Create | `internal/metrics/async_test.go` | Tests for AsyncRecorder delivery and drain-on-stop |
| Create | `internal/metrics/prometheus/prometheus.go` | Prometheus implementation (logic moved from pkg/metrics) |
| Create | `cmd/api/modules/metrics.go` | fx wiring: MetricsLifecycle() |
| Modify | `cmd/api/options/options.go` | Add modules.MetricsLifecycle() |
| Modify | `internal/controller/controller.go` | Import swap: pkg/metrics → internal/metrics |
| Modify | `app/usecases/consumemessage/consumemessage.go` | Import swap |
| Modify | `app/usecases/orchestrator/orchestrator.go` | Import swap |
| Delete | `pkg/metrics/metrics.go` | Replaced by internal/metrics |

---

### Task 1: Create no-op Recorder

**Files:**
- Create: `internal/metrics/noop/noop.go`

- [ ] **Step 1: Write the file**

```go
package noop

// Recorder is a no-op metrics recorder.
// It satisfies internal/metrics.Recorder without importing it (Go duck typing).
type Recorder struct{}

func (Recorder) Gauge(map[string]float64) {}
func (Recorder) Incr([]string)            {}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /home/adrianozdp/workspace/pfc/gaardrail
go build ./internal/metrics/noop/...
```

Expected: no output, exit 0.

---

### Task 2: Create the Recorder interface and global API

**Files:**
- Create: `internal/metrics/metrics.go`

- [ ] **Step 1: Write the file**

```go
package metrics

import "github.com/adrianozp/gaardrail/internal/metrics/noop"

// Recorder is the metrics abstraction. Implementations must be safe for
// concurrent use. Gauge and Incr must never block.
type Recorder interface {
	Gauge(values map[string]float64)
	Incr(names []string)
}

// global is the active recorder. It is set once at startup via SetRecorder
// before any business goroutines start, so no mutex is needed.
var global Recorder = noop.Recorder{}

// SetRecorder replaces the global recorder. Must be called before any
// goroutines invoke Gauge or Incr.
func SetRecorder(r Recorder) { global = r }

// Gauge sets one or more named gauge metrics on the global recorder.
func Gauge(values map[string]float64) { global.Gauge(values) }

// Incr increments one or more named counter metrics by 1 on the global recorder.
func Incr(names []string) { global.Incr(names) }
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/metrics/...
```

Expected: no output, exit 0.

---

### Task 3: Create AsyncRecorder with tests

**Files:**
- Create: `internal/metrics/async.go`
- Create: `internal/metrics/async_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/metrics/async_test.go
package metrics_test

import (
	"sync"
	"testing"

	metrics "github.com/adrianozp/gaardrail/internal/metrics"
)

// recordingRecorder captures all events for test assertions.
type recordingRecorder struct {
	mu     sync.Mutex
	gauges []map[string]float64
	incrs  [][]string
}

func (r *recordingRecorder) Gauge(values map[string]float64) {
	r.mu.Lock()
	r.gauges = append(r.gauges, values)
	r.mu.Unlock()
}

func (r *recordingRecorder) Incr(names []string) {
	r.mu.Lock()
	r.incrs = append(r.incrs, names)
	r.mu.Unlock()
}

func TestAsyncRecorder_DeliverGauge(t *testing.T) {
	rec := &recordingRecorder{}
	async, stop := metrics.NewAsync(rec, 16)

	async.Gauge(map[string]float64{"cpu": 42.0})
	stop() // drain and wait

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.gauges) != 1 {
		t.Fatalf("expected 1 gauge, got %d", len(rec.gauges))
	}
	if rec.gauges[0]["cpu"] != 42.0 {
		t.Errorf("expected cpu=42.0, got %v", rec.gauges[0])
	}
}

func TestAsyncRecorder_DeliverIncr(t *testing.T) {
	rec := &recordingRecorder{}
	async, stop := metrics.NewAsync(rec, 16)

	async.Incr([]string{"messages_total"})
	stop() // drain and wait

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.incrs) != 1 {
		t.Fatalf("expected 1 incr, got %d", len(rec.incrs))
	}
	if rec.incrs[0][0] != "messages_total" {
		t.Errorf("expected messages_total, got %v", rec.incrs[0])
	}
}

func TestAsyncRecorder_StopDrainsRemaining(t *testing.T) {
	rec := &recordingRecorder{}
	async, stop := metrics.NewAsync(rec, 64)

	for i := 0; i < 10; i++ {
		async.Gauge(map[string]float64{"tick": float64(i)})
	}
	stop() // must drain all 10 before returning

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.gauges) != 10 {
		t.Errorf("expected 10 gauges after stop, got %d", len(rec.gauges))
	}
}

func TestAsyncRecorder_FullChannelDropsSilently(t *testing.T) {
	rec := &recordingRecorder{}
	// bufSize=1: second send drops
	async, stop := metrics.NewAsync(rec, 1)

	// Fill the buffer without the worker running yet.
	// Both sends must return immediately (no block).
	done := make(chan struct{})
	go func() {
		async.Gauge(map[string]float64{"a": 1})
		async.Gauge(map[string]float64{"b": 2})
		close(done)
	}()
	<-done
	stop()
	// At least one event delivered; test just verifies no deadlock/panic.
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/metrics/... -run TestAsync -v
```

Expected: `FAIL` — `metrics.NewAsync undefined`.

- [ ] **Step 3: Write the implementation**

```go
// internal/metrics/async.go
package metrics

import "sync"

type eventKind uint8

const (
	kindGauge eventKind = iota
	kindIncr
)

type event struct {
	kind   eventKind
	gauges map[string]float64
	names  []string
}

// AsyncRecorder wraps a Recorder and dispatches all sends through a buffered
// channel. Gauge and Incr never block: if the channel is full, the event is
// dropped silently (metrics are best-effort).
type AsyncRecorder struct {
	inner Recorder
	ch    chan event
	wg    sync.WaitGroup
}

// NewAsync returns an AsyncRecorder and a stop function.
// bufSize is the number of events the channel can buffer before drops occur.
// The stop function closes the channel and waits for the worker to drain all
// buffered events before returning. Call it during graceful shutdown.
func NewAsync(r Recorder, bufSize int) (*AsyncRecorder, func()) {
	a := &AsyncRecorder{
		inner: r,
		ch:    make(chan event, bufSize),
	}
	a.wg.Add(1)
	go a.worker()
	stop := func() {
		close(a.ch)
		a.wg.Wait()
	}
	return a, stop
}

func (a *AsyncRecorder) Gauge(values map[string]float64) {
	select {
	case a.ch <- event{kind: kindGauge, gauges: values}:
	default:
	}
}

func (a *AsyncRecorder) Incr(names []string) {
	select {
	case a.ch <- event{kind: kindIncr, names: names}:
	default:
	}
}

func (a *AsyncRecorder) worker() {
	defer a.wg.Done()
	for e := range a.ch {
		switch e.kind {
		case kindGauge:
			a.inner.Gauge(e.gauges)
		case kindIncr:
			a.inner.Incr(e.names)
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/metrics/... -run TestAsync -v
```

Expected: all 4 `TestAsync*` tests PASS.

---

### Task 4: Create Prometheus implementation

**Files:**
- Create: `internal/metrics/prometheus/prometheus.go`

This is the existing logic from `pkg/metrics/metrics.go` moved verbatim into the new package. The only changes are the package name and the removal of the package-level function wrappers.

- [ ] **Step 1: Write the file**

```go
// internal/metrics/prometheus/prometheus.go
package prometheus

import (
	"sync"

	prom "github.com/prometheus/client_golang/prometheus"
)

var (
	gauges   sync.Map // map[string]prom.Gauge
	counters sync.Map // map[string]prom.Counter
)

// Recorder is the Prometheus implementation of internal/metrics.Recorder.
// Metrics are registered lazily with the "gaardrail_" prefix.
type Recorder struct{}

func (Recorder) Gauge(values map[string]float64) {
	for name, value := range values {
		getOrRegisterGauge(name).Set(value)
	}
}

func (Recorder) Incr(names []string) {
	for _, name := range names {
		getOrRegisterCounter(name).Inc()
	}
}

func getOrRegisterGauge(name string) prom.Gauge {
	if v, ok := gauges.Load(name); ok {
		return v.(prom.Gauge)
	}
	g := prom.NewGauge(prom.GaugeOpts{
		Name: "gaardrail_" + name,
		Help: name,
	})
	if err := prom.Register(g); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			existing := are.ExistingCollector.(prom.Gauge)
			gauges.Store(name, existing)
			return existing
		}
		panic(err)
	}
	gauges.Store(name, g)
	return g
}

func getOrRegisterCounter(name string) prom.Counter {
	if v, ok := counters.Load(name); ok {
		return v.(prom.Counter)
	}
	c := prom.NewCounter(prom.CounterOpts{
		Name: "gaardrail_" + name,
		Help: name,
	})
	if err := prom.Register(c); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			existing := are.ExistingCollector.(prom.Counter)
			counters.Store(name, existing)
			return existing
		}
		panic(err)
	}
	counters.Store(name, c)
	return c
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/metrics/...
```

Expected: no output, exit 0.

---

### Task 5: Create fx wiring module

**Files:**
- Create: `cmd/api/modules/metrics.go`

- [ ] **Step 1: Write the file**

```go
package modules

import (
	"context"

	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	promrecorder "github.com/adrianozp/gaardrail/internal/metrics/prometheus"
	"go.uber.org/fx"
)

// MetricsLifecycle wires the global metrics recorder into the fx lifecycle.
// It sets up an AsyncRecorder backed by Prometheus on startup and drains the
// channel on graceful shutdown.
func MetricsLifecycle() fx.Option {
	return fx.Invoke(func(lc fx.Lifecycle) {
		prom := promrecorder.Recorder{}
		async, stop := metrics.NewAsync(prom, 256)
		metrics.SetRecorder(async)
		lc.Append(fx.Hook{
			OnStop: func(_ context.Context) error {
				stop()
				return nil
			},
		})
	})
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./cmd/api/...
```

Expected: no output, exit 0.

---

### Task 6: Register MetricsLifecycle in options.go

**Files:**
- Modify: `cmd/api/options/options.go`

The metrics lifecycle must be registered before any module that emits metrics (orchestrator, controller). Add it as the first `fx.Invoke` in `Options()`.

- [ ] **Step 1: Add the call**

In `cmd/api/options/options.go`, add `modules.MetricsLifecycle()` to the `fx.Options(...)` block, right after the `fx.Provide` block:

```go
// Before (excerpt):
func Options() fx.Option {
    return fx.Options(
        fx.Provide(
            config.Load,
            httpserver.New,
        ),

        modules.QueueFactories(),
        // ...
    )
}

// After (add one line after the Provide block):
func Options() fx.Option {
    return fx.Options(
        fx.Provide(
            config.Load,
            httpserver.New,
        ),

        modules.MetricsLifecycle(),

        modules.QueueFactories(),
        modules.OrchestratorFactories(),
        modules.OrchestratorInjections(),
        // ... rest unchanged
    )
}
```

Full updated file:

```go
package options

import (
	"context"

	"github.com/adrianozp/gaardrail/app/usecases/orchestrator"
	"github.com/adrianozp/gaardrail/cmd/api/modules"
	"github.com/adrianozp/gaardrail/internal/httpserver"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/fx"
)

func Options() fx.Option {
	return fx.Options(
		fx.Provide(
			config.Load,
			httpserver.New,
		),

		modules.MetricsLifecycle(),

		modules.QueueFactories(),

		modules.OrchestratorFactories(),
		modules.OrchestratorInjections(),

		modules.MetricsPollerFactories(),
		modules.MetricsPollerInjections(),
		modules.MetricsPollerLifecycle(),

		modules.MessageFactories(),
		modules.MessageInjections(),
		modules.MessageEndpoints(),

		modules.PIDFactories(),
		modules.PIDInjections(),
		modules.PIDEndpoints(),

		fx.Invoke(func(lc fx.Lifecycle, router *gin.Engine, cfg config.Config) {
			router.GET("/metrics", gin.WrapH(promhttp.Handler()))
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					if cfg.HTTP.TLSEnabled() {
						go router.RunTLS(cfg.HTTP.Addr, cfg.HTTP.CertFile, cfg.HTTP.KeyFile)
					} else {
						go router.Run(cfg.HTTP.Addr)
					}
					return nil
				},
			})
		}),

		fx.Invoke(func(lc fx.Lifecycle, o *orchestrator.Orchestrator) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					return o.Start(context.Background())
				},
			})
		}),
	)
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./cmd/api/...
```

Expected: no output, exit 0.

---

### Task 7: Update call sites to use internal/metrics

**Files:**
- Modify: `internal/controller/controller.go`
- Modify: `app/usecases/consumemessage/consumemessage.go`
- Modify: `app/usecases/orchestrator/orchestrator.go`

Each file only needs its import line changed. No call sites change.

- [ ] **Step 1: Update `internal/controller/controller.go`**

Change the import from:
```go
"github.com/adrianozp/gaardrail/pkg/metrics"
```
to:
```go
metrics "github.com/adrianozp/gaardrail/internal/metrics"
```

- [ ] **Step 2: Update `app/usecases/consumemessage/consumemessage.go`**

Change the import from:
```go
"github.com/adrianozp/gaardrail/pkg/metrics"
```
to:
```go
metrics "github.com/adrianozp/gaardrail/internal/metrics"
```

- [ ] **Step 3: Update `app/usecases/orchestrator/orchestrator.go`**

Change the import from:
```go
"github.com/adrianozp/gaardrail/pkg/metrics"
```
to:
```go
metrics "github.com/adrianozp/gaardrail/internal/metrics"
```

- [ ] **Step 4: Verify the whole module compiles**

```bash
go build ./...
```

Expected: no output, exit 0.

---

### Task 8: Delete pkg/metrics

**Files:**
- Delete: `pkg/metrics/metrics.go`

- [ ] **Step 1: Delete the file**

```bash
rm pkg/metrics/metrics.go
```

- [ ] **Step 2: Remove empty directory if present**

```bash
rmdir pkg/metrics 2>/dev/null || true
```

- [ ] **Step 3: Verify nothing imports it**

```bash
grep -r "pkg/metrics" . --include="*.go"
```

Expected: no output.

- [ ] **Step 4: Full build**

```bash
go build ./...
```

Expected: no output, exit 0.

---

### Task 9: Run all tests

- [ ] **Step 1: Run the full test suite**

```bash
go test ./... -v 2>&1 | tail -40
```

Expected: all existing tests PASS, 4 new `TestAsync*` tests PASS. No failures.

- [ ] **Step 2: Verify the async test package specifically**

```bash
go test ./internal/metrics/... -v
```

Expected output (exact names):
```
--- PASS: TestAsyncRecorder_DeliverGauge
--- PASS: TestAsyncRecorder_DeliverIncr
--- PASS: TestAsyncRecorder_StopDrainsRemaining
--- PASS: TestAsyncRecorder_FullChannelDropsSilently
PASS
```
