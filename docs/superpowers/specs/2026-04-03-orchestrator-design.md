# Orchestrator Rate Limiter Design

**Date:** 2026-04-03

## Problem

The PID controller (`internal/controller/controller.go`) correctly computes a `drainRate` (messages/sec) from CPU metrics and calls `orchestrator.SetDrainRate(drainRate)`. However, the orchestrator's `run()` loop never reads `DrainRate` — it consumes exactly one message per fixed tick (hardcoded at 100ms = 10 msg/s). The PID output has zero effect on actual throughput.

Additionally, `Start(tickRate int)` spawns a goroutine with no cancellation path — the orchestrator cannot be shut down cleanly.

## Goal

Wire the PID controller's output to actual message consumption rate, and remove the tick rate concept entirely by using Go's token bucket rate limiter.

## Design

### Orchestrator (`app/usecases/orchestrator/orchestrator.go`)

Replace `DrainRate float64` and `time.Ticker` with `*rate.Limiter` from `golang.org/x/time/rate`.

- `NewOrchestrator` initializes the limiter at rate `0` (paused) with burst `1`
- `SetDrainRate(drainRate float64)` calls `limiter.SetLimit(rate.Limit(drainRate))` — goroutine-safe, no mutex needed
- `Start(ctx context.Context)` replaces `Start(tickRate int)` — the caller owns the lifecycle
- `run(ctx)` calls `limiter.Wait(ctx)` before each consume; when `ctx` is cancelled, `Wait` returns an error and the loop exits cleanly

```go
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

### Call site (`cmd/api/options/options.go`)

Pass the fx lifecycle context to `Start`:

```go
fx.Invoke(func(lc fx.Lifecycle, o *orchestrator.Orchestrator) {
    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            return o.Start(ctx)
        },
    })
}),
```

### Dependency (`go.mod`)

Add `golang.org/x/time`.

## What Does Not Change

- `internal/controller/controller.go` — PID logic is correct
- `app/usecases/processmetrics/processmetrics.go` — already calls `SetDrainRate` correctly
- `orchestrator.Consumer` interface
- `processmetrics.Orchestrator` interface (`SetDrainRate(float64) error`)

## Behavior Notes

- **Initial rate 0**: The orchestrator is paused at startup until the first PID tick fires and sets a real rate. This is safe — no messages are lost, they remain in the queue.
- **Burst = 1**: One message consumed per token. Keeps behavior consistent with the original one-at-a-time approach. Can be increased later if burst handling is desired.
- **Shutdown**: `ctx` cancellation (via fx `OnStop`) propagates through `limiter.Wait(ctx)`, giving a clean exit with no extra `OnStop` hook required.
