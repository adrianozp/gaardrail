# Metrics Reader Design

**Date:** 2026-04-03
**Status:** Approved

## Context

The gaardrail queue controller uses a PID controller fed by database metrics to regulate message throughput (RPM/RPS), keeping the database stable during long-running queue operations. Currently, metrics arrive only via HTTP push (`POST /metrics`). There is no mechanism for the controller to actively fetch metrics from monitoring infrastructure.

This design adds a polling-based metrics reader following the existing ports and adapters architecture, enabling gaardrail to pull metrics from exporters (e.g., `mysqld_exporter`, `pg_exporter`, `node_exporter`) without direct database access and without requiring external monitoring agents.

## Architecture

Two ingestion paths, one processing pipeline:

```
[HTTP Push]
  POST /metrics → ReceiveMetricsHandler → ProcessMetricsUseCase

[Polling Pull]
  Exporter /metrics → MetricsReader adapter → PollingHandler → ProcessMetricsUseCase
```

Both paths feed the same `ProcessMetricsUseCase`, which drives the PID controller and updates the rate limiter in the orchestrator.

The `PollingHandler` goroutine is managed by the `fx` lifecycle (same pattern as `Orchestrator`), started via `Start(ctx context.Context)`.

## New Components

### Port: `MetricsReader`

Defined in `app/repositories/metricsreader.go`.

```go
//go:generate mockery --all --output=mocks --outpkg=mocks

type MetricsReader interface {
    Read(ctx context.Context) (map[string]float64, error)
}
```

Returns a map of domain metric names (e.g., `"cpu"`, `"connections"`) to their current values. Metric name mapping from source to domain is the adapter's responsibility.

### Adapter: `PrometheusMetricsReader`

Location: `app/repositories/prometheus/`

- Issues `GET` to a configured endpoint serving Prometheus text format (`/metrics`)
- Parses the response using `github.com/prometheus/common/expfmt`
- Extracts only the metrics listed in a configured mapping: `source_metric_name → domain_name`
- Example mapping: `{"process_cpu_seconds_total": "cpu", "mysql_global_status_threads_connected": "connections"}`
- Unlisted metrics are ignored

### Adapter: `JSONMetricsReader`

Location: `app/repositories/jsonmetrics/`

- Issues `GET` to a configured endpoint returning a flat JSON object
- Extracts values via a configured mapping: `json_field → domain_name`
- Example mapping: `{"cpu_usage": "cpu", "active_connections": "connections"}`
- Intended for custom agents or sidecars that expose simple JSON payloads

### Handler: `PollingHandler`

Location: `app/handlers/pollmetrics/`

```go
//go:generate mockery --all --output=mocks --outpkg=mocks

type ProcessMetrics interface {
    Process(m entities.Metrics) error
}
```

- Holds a `MetricsReader` and a `ProcessMetrics` reference
- On each tick: calls `reader.Read(ctx)`, wraps the result in `entities.Metrics` with `time.Now()`, calls `processMetrics.Process(m)`
- On `Read()` error: logs the error, skips the tick, continues on the next interval — the PID retains its last state
- On context cancellation: exits cleanly (no goroutine leak)

## Configuration

New block added to `pkg/config/config.go`:

```go
type MetricsPoller struct {
    Enabled    bool
    IntervalMs int               // polling interval in ms; default: 5000
    Endpoint   string            // exporter URL (e.g., "http://mysqld-exporter:9104/metrics")
    Protocol   string            // "prometheus" or "json"
    Mappings   map[string]string // source field → domain name (e.g., {"process_cpu_seconds_total": "cpu"})
}
```

Added to `Config`:

```go
type Config struct {
    HTTP          HTTP
    Kafka         Kafka
    Target        Target
    MetricsPoller MetricsPoller  // new
}
```

Environment variable override follows existing Viper pattern (`APP_METRICSPOLLER_*`).

## Push-based Sources (Datadog, CloudWatch)

Datadog and CloudWatch are not handled by the `MetricsReader` interface. They push data via webhooks, so they are implemented as new HTTP handlers (same pattern as `ReceiveMetricsHandler`) that map their payload fields to `entities.Metrics` and call `ProcessMetricsUseCase`. No changes to the polling infrastructure are needed.

## Testing

### `PrometheusMetricsReader`
- Uses `httptest.NewServer` serving static Prometheus text
- Validates: correct extraction of mapped metrics, unmapped metrics ignored, HTTP errors return `error`

### `JSONMetricsReader`
- Uses `httptest.NewServer` serving static JSON
- Validates: field mapping, missing fields, HTTP errors

### `PollingHandler`
- Uses mockery-generated mocks for `MetricsReader` and `ProcessMetrics`
- Validates:
  - `Read()` + `Process()` called on each tick
  - `Read()` error does not stop the loop
  - Context cancellation exits the handler without goroutine leak

Mocks are generated via `//go:generate mockery --all --output=mocks --outpkg=mocks` on interface files. Run `go generate ./...` after defining interfaces before writing tests.

## File Map

| File | Change |
|------|--------|
| `app/repositories/metricsreader.go` | New — `MetricsReader` port interface |
| `app/repositories/prometheus/prometheus.go` | New — Prometheus text format adapter |
| `app/repositories/prometheus/prometheus_test.go` | New |
| `app/repositories/jsonmetrics/jsonmetrics.go` | New — JSON HTTP adapter |
| `app/repositories/jsonmetrics/jsonmetrics_test.go` | New |
| `app/handlers/pollmetrics/pollmetrics.go` | New — polling handler |
| `app/handlers/pollmetrics/pollmetrics_test.go` | New |
| `pkg/config/config.go` | Add `MetricsPoller` block |
| `cmd/api/modules/` | Wire `PollingHandler` into fx lifecycle |
