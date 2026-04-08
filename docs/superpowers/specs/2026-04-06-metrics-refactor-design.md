# Metrics Refactor Design

**Date:** 2026-04-06
**Status:** Approved

## Problem

`pkg/metrics` is directly coupled to Prometheus. Every call to `metrics.Gauge` or `metrics.Incr` executes synchronously on the caller's goroutine, blocking hot paths like the PID controller tick and the message consume loop. Swapping the metrics backend requires touching multiple packages.

## Goal

- Introduce a `Recorder` interface so the backend can be replaced without changing call sites.
- Move metric sends off the hot path using a buffered async dispatcher.
- Keep the global function API (`metrics.Gauge`, `metrics.Incr`) unchanged — no dependency injection.

## Package Structure

```
internal/metrics/
  metrics.go        ← Recorder interface, global recorder, Gauge/Incr, SetRecorder
  async.go          ← AsyncRecorder: buffers events in a channel, drains via worker goroutine
  prometheus/
    prometheus.go   ← Prometheus implementation of Recorder (moved from pkg/metrics)
  noop/
    noop.go         ← no-op Recorder (zero allocations, used as default and in tests)

cmd/api/modules/
  metrics.go        ← wiring: creates Prometheus recorder, wraps in AsyncRecorder, calls SetRecorder

pkg/metrics/        ← DELETED
```

## Interface

```go
// internal/metrics/metrics.go

type Recorder interface {
    Gauge(values map[string]float64)
    Incr(names []string)
}

var global Recorder = noop.Recorder{}

func SetRecorder(r Recorder) { global = r }
func Gauge(values map[string]float64) { global.Gauge(values) }
func Incr(names []string)             { global.Incr(names) }
```

`SetRecorder` is called once at startup before any business goroutines start, so no mutex is needed on the global.

## AsyncRecorder

```go
// internal/metrics/async.go

type event struct {
    kind   string // "gauge" | "incr"
    gauges map[string]float64
    names  []string
}

type AsyncRecorder struct {
    inner  Recorder
    ch     chan event
    wg     sync.WaitGroup
}

func NewAsync(r Recorder, bufSize int) (*AsyncRecorder, func()) {
    a := &AsyncRecorder{inner: r, ch: make(chan event, bufSize)}
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
    case a.ch <- event{kind: "gauge", gauges: values}:
    default: // drop silently — metrics are best-effort
    }
}

func (a *AsyncRecorder) Incr(names []string) {
    select {
    case a.ch <- event{kind: "incr", names: names}:
    default:
    }
}

func (a *AsyncRecorder) worker() {
    defer a.wg.Done()
    for e := range a.ch {
        switch e.kind {
        case "gauge":
            a.inner.Gauge(e.gauges)
        case "incr":
            a.inner.Incr(e.names)
        }
    }
}
```

Buffer size: `256` events (configurable at wiring time). Drops are silent — logging drops would create noise during load spikes, which is the exact scenario where drops occur.

On shutdown, `stop()` closes the channel and waits for the worker to drain remaining events before returning.

## Prometheus Implementation

Moved from `pkg/metrics/metrics.go` to `internal/metrics/prometheus/prometheus.go`. Logic unchanged: lazy registration with `sync.Map`, `gaardrail_` prefix, `AlreadyRegisteredError` handling.

```go
type PrometheusRecorder struct{}

func (PrometheusRecorder) Gauge(values map[string]float64) { /* current logic */ }
func (PrometheusRecorder) Incr(names []string)             { /* current logic */ }
```

## No-op Implementation

```go
// internal/metrics/noop/noop.go

type Recorder struct{}

func (Recorder) Gauge(map[string]float64) {}
func (Recorder) Incr([]string)            {}
```

Used as the default global (so tests that don't call `SetRecorder` work with zero configuration) and as a compile-time check that `Recorder` is satisfied.

## Wiring

New file `cmd/api/modules/metrics.go`:

```go
func SetupMetrics() func() {
    prom := prometheus.PrometheusRecorder{}
    async, stop := metrics.NewAsync(prom, 256)
    metrics.SetRecorder(async)
    return stop
}
```

The returned `stop` function is registered in the server's graceful shutdown sequence alongside Kafka, HTTP server, etc.

## Call Site Changes

Three files change import path only — from `pkg/metrics` to `internal/metrics`. No other changes:

| File | Change |
|------|--------|
| `internal/controller/controller.go` | import swap |
| `app/usecases/consumemessage/consumemessage.go` | import swap |
| `app/usecases/orchestrator/orchestrator.go` | import swap (×2 call sites) |

## Testing

- Existing unit tests continue to work: the default global is `noop.Recorder{}`, so no `SetRecorder` call is needed in tests.
- A `recording.Recorder` (slice-based) is out of scope for now but trivially implementable behind the interface when needed.

## What Is Not Changed

- The `Gauge` / `Incr` function signatures.
- The Prometheus metric names (`gaardrail_` prefix) and registration logic.
- The HTTP `/metrics` endpoint (served by `internal/httpserver`, unaffected).
