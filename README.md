# gaardrail

[![CI](https://github.com/adrianozp/gaardrail/actions/workflows/ci.yml/badge.svg)](https://github.com/adrianozp/gaardrail/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/adrianozp/gaardrail)](https://github.com/adrianozp/gaardrail/releases)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL--2.0-brightgreen.svg)](LICENSE)

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

## Quickstart

gaardrail sits between a queue and a target, so you need:

- a **SQL target** it will push messages into (MySQL supported today);
- a **metrics source** exposing the resource you want to protect (any Prometheus-compatible query API);
- optionally a **Kafka** queue — the default config uses `queue.protocol: constant`, a synthetic queue that always emits the same query, so you can watch the control loop work without any broker.

Run the published image, mounting your config:

```bash
docker run -d -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config/config.yaml \
  adrianozdp/gaardrail:latest
```

Or from source:

```bash
git clone https://github.com/adrianozp/gaardrail
cd gaardrail
make run        # go run ./cmd/api — reads config/config.yaml
```

To use a real Kafka queue locally:

```bash
make kafka/up && make kafka/setup   # single-node Kafka via docker compose
# then set queue.protocol: kafka in config.yaml
```

gaardrail is headless — an example web panel (live tuning, controller switching, embedded Grafana) lives in the [gaardrail-flood-test](https://github.com/adrianozp/gaardrail-flood-test) repo under `panel/`, alongside disturbance-generation scripts for experiments.

## Orchestrator

The orchestrator drives message consumption at a controlled rate. It holds a `rate.Limiter` that gates how many messages can be consumed per second — the **drain rate**.

On each tick, each worker:
1. Waits for a token from the `rate.Limiter`
2. Calls `Consumer.Consume()` — dequeues, pushes to the target, and acks

The drain rate starts at the value defined in config and is updated in real time by the PID controller. A background poller also tracks the current queue lag (`Consumer.Size()`) and publishes it as a metric.

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

Two configurable filters shape the loop's signals. The **measurement filter** lives on the metrics chain (`metrics_poller.filter_type` / `filter_size`) and smooths the process variable before it reaches whichever controller is active. The **setpoint filter** is per controller (`setpoint_filter_type` / `setpoint_filter_size`) and turns setpoint changes into a gradual trajectory, removing startup overshoot.

Both accept the types `none`, `moving_average` and `exponential`, with sizes in samples: a moving-average setpoint filter turns a step into a linear ramp that completes in exactly N samples; the exponential type is a first-order lag with a time constant of N samples (`a = e^(−1/N)`). The legacy `setpoint_filter_tau` (seconds) is still honored and maps to the equivalent exponential.

### Controller types

The active controller is selected by `controller.type` and can be switched at runtime with `PUT /controller/type`:

| Type | Description |
|------|-------------|
| `pid` | Discrete PI (feedforward off) — the baseline. |
| `pidff` | Same PI with feedforward (`u_ff = setpoint/ff_gain`). |
| `smith` | Smith predictor (PI + internal FOPDT model) for dead-time compensation. |
| `step` | Open-loop constant output (`max`), used for identification experiments. |

## Library

The controllers are importable on their own, with no global metrics and no app config:

```go
import (
    "github.com/adrianozp/gaardrail/pkg/control"
    "github.com/adrianozp/gaardrail/pkg/control/filter"
)

pi, err := control.NewPID(control.PIDParams{
    Kp: 0.058, Ki: 0.009, Setpoint: 50,
    SetpointFilter: filter.Spec{Kind: filter.MovingAverage, Size: 4},
    Limits:         control.Limits{Min: 0, Max: 50, IClamp: 40},
})
rate, terms, err := pi.Compute(cpuPercent, time.Now())
```

Every controller implements `control.Controller`:

| Method | Description |
|--------|-------------|
| `Compute(measured, at) (output, Terms, error)` | One control step. The first sample never integrates; a sample older than the previous one returns `ErrOutdatedMeasurement`. |
| `SetSetpoint(v) error` | Moves the setpoint through the setpoint filter. |
| `Reset()` | Clears the integral, filters and history. |

| Constructor | Params |
|-------------|--------|
| `NewPID(PIDParams)` | `Kp`, `Ki`, `Kd`, `FFGain` (0 disables feedforward), `Setpoint`, `SetpointFilter`, `Limits` |
| `NewSmith(SmithParams)` | `Kp`, `Ki`, `Setpoint`, `SetpointFilter`, `Model{K, Tau, Theta}`, `SampleSeconds`, `Limits` |
| `NewStep(output)` | constant open-loop output |

`PID` and `Smith` also expose `Params()` and `SetParams(...)`; all methods are safe for concurrent use. `Terms` reports `Setpoint`, `Measured`, `Error`, `P`, `I`, `D`, `FF`, `Output`, `Feedback` and `Prediction` for your own metrics. Invalid params return `ErrInvalidLimits`, `ErrNegativeIClamp`, `ErrInvalidModel`, `ErrInvalidSetpoint`, `ErrInvalidMeasurement` or `ErrInvalidOutput`.

`filter.NewMeasurement(Spec)` and `filter.NewReference(Spec)` build the measurement and setpoint filters as `*filter.Signal` (`Filter`, `Seed`, `Reset`, `Spec`).

## Message consumption

Each `Consume()` call follows a strict sequence:

```
Dequeue  →  Push(target)  →  Ack
```

A message is only acked after the target confirms receipt. If either `Push` or `Ack` fails, the message is not acked and will be redelivered.

## Interfaces

gaardrail keeps transport and storage details outside the core logic:

```go
type Queue interface {
    Enqueue(entities.Message) (string, error)
    Dequeue(context.Context) (entities.Message, error)
    Ack(context.Context, entities.Message) error
    Size() (int64, error)
}

type Target interface {
    Push(context.Context, entities.Message) (entities.Response, error)
}

type MetricsReader interface {
    Read(ctx context.Context) (entities.Metrics, error)
}
```

Current implementations: Kafka and constant (queue), MySQL (target); Prometheus HTTP API, Prometheus exporter (text-format scrape), JSON metrics endpoint and AWS CloudWatch (metrics readers).

## Configuration

[config/config.yaml](config/config.yaml) is the reference; any field can be overridden with an `APP_*` environment variable (`pid.kp` → `APP_PID_KP`).

| Section | Field | Meaning |
|---------|-------|---------|
| `queue` | `protocol` | `constant` (synthetic queue that always emits `query`, no broker needed) or `kafka` |
| `target` | `protocol`, `driver`, `dsn` | where messages are pushed; `sql` + `mysql` today |
| `target.tls` | `enabled` | encrypt the database connection (mysql only) |
| | `skip_verify` | encrypt without verifying the server certificate (self-signed, tests) |
| | `ca_file`, `server_name` | optional PEM CA bundle and name for full verification |
| `metrics_poller` | `interval_ms` | control period T (5000 locally; 60000 for CloudWatch-class providers) |
| | `protocol`, `endpoint` | `prometheusapi`, `exporter`, `json` or `cloudwatch` |
| | `mappings` | query → process variable (`cpu`). Map the raw value, as a real provider delivers it: `irate` over a window of at least 3× the scrape interval (`[15s]` for a 5s scrape, `[3m]` for 60s). Smoothing belongs to `filter_type` / `filter_size`. |
| `controller` | `type` | `pid`, `pidff`, `smith` or `step` |
| `orchestrator` | `rate`, `workers`, `burst` | initial drain rate, consumer goroutines, limiter burst |
| `pid` | `setpoint` | target utilization (%) |
| | `kp`, `ki`, `kd` | gains; the defaults are the PI+FF baseline identified at T=10s |
| | `min`, `max` | drain-rate bounds (msgs/s); leave headroom above the steady-state drain (~22 at setpoint 50%) |
| | `i_clamp` | anti-windup bound; must reach the steady-state drain, since the PID is position-form |
| | `ff_gain` | feedforward `u_ff = setpoint / ff_gain`, the identified static gain; 0 disables |
| | `setpoint_filter_tau` | soft-start in seconds (~2T); or `setpoint_filter_type` / `setpoint_filter_size` |
| `smith` | `kp`, `ki` | keep them gentle: high gains oscillate at T=60s |
| | `model_k` | static gain of the identified FOPDT (%CPU per msg/s) |
| | `model_tau` | time constant (s) |
| | `model_theta` | effective delay: continuous θ plus about T of sampling/ZOH |
| | `sample_seconds` | sampling period T; must match `metrics_poller.interval_ms` |

## HTTP API

| Endpoint | Description |
|----------|-------------|
| `GET /ping` | Health check |
| `GET /metrics` | Prometheus metrics (drain rate, queue lag, PID terms…) |
| `POST /messages` | Enqueue a message |
| `GET /pid` | Read active controller parameters |
| `PATCH /pid` | Update parameters at runtime (only present fields change) |
| `PUT /controller/type` | Switch the active controller |

```bash
curl -X PATCH http://localhost:8080/pid \
  -H "Content-Type: application/json" \
  -d '{"kp": 0.15, "setpoint": 55.0}'
```

Changing any parameter resets the integral accumulator to prevent windup artifacts. The main endpoints are described in [openapi.yaml](openapi.yaml).

## Validation

gaardrail's control loop was validated experimentally — system identification (FOPDT), closed-loop campaigns at T=10s and T=60s, disturbance rejection, and a PI / PI+feedforward / Smith predictor comparison. The full rig (Docker Compose stack, Grafana dashboards, experiment data and analysis scripts) lives at [gaardrail-flood-test](https://github.com/adrianozp/gaardrail-flood-test).

The project started as a final thesis (PFC) in Control and Automation Engineering at UFSC (Brazil).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Security reports: [SECURITY.md](SECURITY.md).

## License

[MPL-2.0](LICENSE)

