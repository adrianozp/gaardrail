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
| `autopid` | **Self-tuning**: runs an open-loop step, identifies a FOPDT model (two-point Smith method), computes the gains automatically (AMIGO or SIMC) and closes the loop. |

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

`config/config.yaml` is the fully annotated reference; fields can be overridden with `APP_*` environment variables. The core sections:

```yaml
queue:
  protocol: constant          # constant | kafka
target:
  protocol: sql
  driver: mysql
  dsn: "root:root@tcp(localhost:3306)/gaardrail"
metrics_poller:
  interval_ms: 5000           # control period T
  protocol: prometheusapi
  endpoint: "http://localhost:9090/api/v1/query"
  mappings:
    '<promql query>': cpu     # maps a query to the process variable
controller:
  type: "pidff"               # pid | pidff | smith | step | autopid
pid:
  setpoint: 50                # target resource utilization (%)
  kp: 0.058
  ki: 0.009
  min: 0                      # drain rate bounds (msgs/s)
  max: 50
  i_clamp: 40                 # anti-windup bound
```

`smith` and `autopid` have their own sections (internal FOPDT model and identification/tuning settings) — see the comments in [config/config.yaml](config/config.yaml).

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

