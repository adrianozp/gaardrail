# gaardrail

**gaardrail** is a back-pressure controller for message queue consumers. It uses a discrete PID controller to regulate how fast messages are drained from a queue, targeting a configurable resource utilization setpoint (e.g. database CPU at 50%).

## How it works

```
  Producer
     │
     ▼
 ┌────────┐   enqueue    ┌──────────────────────────────────────────┐
 │  Queue │ ──────────►  │              Orchestrator                │
 │(Kafka) │              │                                          │
 └────────┘              │  rate.Limiter(drainRate msgs/s)          │
                         │       │                                  │
                         │       ▼  (workers)                       │
                         │  ConsumeMessage                          │
                         │    Dequeue → Push(target) → Ack          │
                         └──────────────────────────────────────────┘
                                                  ▲
                                                  │ SetDrainRate(output)
                                    ┌─────────────┴──────────────┐
                                    │       PID Controller        │
                                    │                             │
                                    │  error = setpoint - CPU%    │
                                    │  output = P + I + D         │
                                    └─────────────────────────────┘
                                                  ▲
                                                  │ measured CPU%
                                    ┌─────────────┴──────────────┐
                                    │      Metrics Poller         │
                                    │  (polls Prometheus API)     │
                                    └─────────────────────────────┘
```

---

## Orchestrator

The orchestrator drives message consumption at a controlled rate. It holds a `rate.Limiter` that gates how many messages can be consumed per second — the **drain rate**.

On each tick, each worker:
1. Waits for a token from the `rate.Limiter`
2. Calls `Consumer.Consume()` — dequeues, pushes to the target, and acks

The drain rate starts at the value defined in config and is updated in real time by the PID controller. A background poller also tracks the current queue lag (`Consumer.Size()`) and publishes it as a metric.

```go
type Consumer interface {
    Consume() (string, error)
    Size()    (int64, error)
}
```

---

## PID Controller

The controller is a **discrete position-form PID** that runs on each metrics poll tick:

```
e(t)      = setpoint − measured
P         = Kp × e(t)
I(t)      = clamp(I(t-1) + Ki × e(t) × dt,  −IClamp, +IClamp)
D         = Kd × (e(t) − e(t-1)) / dt
output(t) = clamp(P + I + D,  Min,  Max)
```

The output is the new **drain rate** in messages/second, clamped to `[Min, Max]`.

| Parameter | Role |
|-----------|------|
| `Kp` | Proportional gain — immediate response to error |
| `Ki` | Integral gain — eliminates steady-state error over time |
| `Kd` | Derivative gain — dampens oscillations by reacting to error rate of change |
| `IClamp` | Anti-windup bound on the integral accumulator |
| `setpoint` | Target resource utilization (e.g. `50.0` for 50% CPU) |
| `Min / Max` | Output clamp — drain rate bounds in msgs/s |

Parameters can be updated at runtime without restarting:

```bash
curl -X PATCH http://localhost:8080/pid \
  -H "Content-Type: application/json" \
  -d '{"kp": 0.15, "setpoint": 55.0}'
```

Changing any parameter resets the integral accumulator to prevent windup artifacts from the previous gains.

---

## Message consumption

Each `Consume()` call follows a strict sequence:

```
Dequeue  →  Push(target)  →  Ack
```

- **Dequeue**: pulls one message from the queue
- **Push**: forwards it to the configured target (e.g. a SQL database)
- **Ack**: commits the offset back to the queue

A message is only acked after the target confirms receipt. If either `Push` or `Ack` fails, the message is not acked and will be redelivered.

---

## Interfaces

gaardrail is built around two pairs of interfaces, keeping transport and storage details outside the core logic.

**Queue** — message source

```go
type Queue interface {
    Dequeue() (entities.Message, error)
    Ack(entities.Message) error
    Size() (int64, error)
}
```

**Target** — message sink

```go
type Target interface {
    Push(entities.Message) error
}
```

**MetricsReader** — process variable source for the PID controller

```go
type MetricsReader interface {
    Read(ctx context.Context) (entities.Metrics, error)
}
```

Current implementations: Kafka (queue), MySQL (target), Prometheus HTTP API (metrics reader).

---

## Configuration

`config.yaml` (all fields also settable via `APP_*` environment variables):

```yaml
pid:
  setpoint: 50.0   # target CPU %
  kp: 0.1
  ki: 0.1
  kd: 0.05
  min: 0.0         # minimum drain rate (msgs/s)
  max: 10.0        # maximum drain rate (msgs/s)
  i_clamp: 5.0     # integral anti-windup bound

orchestrator:
  workers: 1       # parallel consumer goroutines
  burst: 5         # token bucket burst size

metrics_poller:
  enabled: true
  interval_ms: 1000
  protocol: prometheusapi
  endpoint: "http://localhost:9090/api/v1/query"
  mappings:
    'irate(container_cpu_usage_seconds_total{...}[1m])*100': cpu
```

---

## Running locally

```bash
# start infrastructure (MySQL, Prometheus, Grafana)
make infra/up
make kafka/up
make kafka/setup

# run the application
make run
```

