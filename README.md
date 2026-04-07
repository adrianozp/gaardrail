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

Before the measured value reaches the PID, it passes through a **moving average filter** of configurable size (`filter_size`). A size of `1` disables filtering (pass-through). Larger values smooth out noisy metrics at the cost of additional lag.

| Parameter | Role |
|-----------|------|
| `Kp` | Proportional gain — immediate response to error |
| `Ki` | Integral gain — eliminates steady-state error over time |
| `Kd` | Derivative gain — dampens oscillations by reacting to error rate of change |
| `IClamp` | Anti-windup bound on the integral accumulator |
| `setpoint` | Target resource utilization (e.g. `50.0` for 50% CPU) |
| `Min / Max` | Output clamp — drain rate bounds in msgs/s |
| `filter_size` | Moving average window size (`1` = no filter) |

Parameters can be read and updated at runtime without restarting:

```bash
# read current parameters
curl http://localhost:8080/pid

# update one or more parameters (only present fields are changed)
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
  filter_size: 1   # moving average window (1 = no filter)

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

---

## Flood test

The flood test exercises the full control loop under load. It requires the infrastructure from the previous section to be running.

### 1. Start the flood stack

The flood stack extends the base infra with cAdvisor and a dedicated Grafana dashboard:

```bash
docker compose -f flood-test/docker-compose-flood.yml up -d
```

Open Grafana at [http://localhost:3000](http://localhost:3000) — the **Flood Test** dashboard is provisioned automatically.

### 2. Set up the database

Creates the `flood_data` table and seeds it with 1 million rows across 5 indexes, providing enough data volume to generate meaningful CPU pressure:

```bash
./flood-test/scripts/setup-db.sh
```

### 3. Send messages to the queue

Each message payload is a heavy SQL query (full scans, self-joins, correlated subqueries). The orchestrator will pull them from Kafka and execute them against MySQL.

```bash
# send 2000 messages (default)
./flood-test/scripts/flood.sh

# send a custom amount
./flood-test/scripts/flood.sh 5000
```

### 4. (Optional) Add an external disturbance

To observe the controller reacting to a sudden CPU spike independent of the message queue, run parallel `BENCHMARK` queries directly on MySQL:

```bash
# 3 workers for 60 seconds (defaults)
./flood-test/scripts/disturb.sh

# custom: 5 workers, lighter iterations, 30 seconds
./flood-test/scripts/disturb.sh 2 5000 30

# run indefinitely until Ctrl+C
./flood-test/scripts/disturb.sh 2 5000 0
```

### 5. Tune the controller live

PID parameters can be adjusted at any time without restarting the application. Changes take effect on the next poll tick and are visible in the **PID Params History** panel.

All fields are optional — only the ones present in the body are updated. Changing any parameter resets the integral accumulator to prevent windup artifacts from previous gains.

```bash
curl -X PATCH http://localhost:8080/pid \
  -H "Content-Type: application/json" \
  -d '{
    "setpoint": 50.0,
    "kp": 0.15,
    "ki": 0.05,
    "kd": 0.02,
    "min": 0.0,
    "max": 10.0,
    "i_clamp": 5.0,
    "filter_size": 1
  }'
```

| Field | Type | Description |
|-------|------|-------------|
| `setpoint` | float | Target CPU % |
| `kp` | float | Proportional gain |
| `ki` | float | Integral gain |
| `kd` | float | Derivative gain |
| `min` | float | Minimum drain rate (msgs/s) |
| `max` | float | Maximum drain rate (msgs/s) |
| `i_clamp` | float | Anti-windup bound on the integral accumulator |
| `filter_size` | int | Moving average window size (`1` = no filter, `>= 1`) |

