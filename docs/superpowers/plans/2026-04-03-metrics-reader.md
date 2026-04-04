# Metrics Reader Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a polling-based metrics reader that scrapes Prometheus exporters (or JSON HTTP endpoints) and feeds the PID controller via the existing `ProcessMetricsUseCase`, following the ports-and-adapters architecture already in place.

**Architecture:** A new `MetricsReader` port (interface) lives in `app/repositories/`. Two adapters implement it: `PrometheusMetricsReader` (scrapes `/metrics` Prometheus text format) and `JSONMetricsReader` (scrapes flat JSON). A `PollingHandler` goroutine ticks on a configurable interval, calls `MetricsReader.Read()`, wraps the result in `entities.Metrics`, and calls `ProcessMetricsUseCase.Process()` — the same usecase used by the existing HTTP push path.

**Tech Stack:** Go 1.25, `go.uber.org/fx` (DI), `github.com/prometheus/common/expfmt` (Prometheus text parsing), `github.com/stretchr/testify/mock` + `mockery v2` (mocks), `net/http/httptest` (adapter tests).

---

## Task 1: Extend config with MetricsPoller and PID params

**Files:**
- Modify: `pkg/config/config.go`

- [ ] **Step 1: Add MetricsPoller and PID structs to config.go**

Replace the `Config` struct and add the two new types (preserve all existing fields):

```go
type Config struct {
	HTTP          HTTP          `mapstructure:"http"`
	Kafka         Kafka         `mapstructure:"kafka"`
	Target        Target        `mapstructure:"target"`
	MetricsPoller MetricsPoller `mapstructure:"metrics_poller"`
	PID           PID           `mapstructure:"pid"`
}

type MetricsPoller struct {
	Enabled    bool              `mapstructure:"enabled"     default:"false"`
	IntervalMs int               `mapstructure:"interval_ms" default:"5000"`
	Endpoint   string            `mapstructure:"endpoint"`
	Protocol   string            `mapstructure:"protocol"    default:"prometheus"`
	Mappings   map[string]string `mapstructure:"mappings"`
}

type PID struct {
	Kp       float64 `mapstructure:"kp"        default:"1.0"`
	Ki       float64 `mapstructure:"ki"        default:"0.1"`
	Kd       float64 `mapstructure:"kd"        default:"0.01"`
	Min      float64 `mapstructure:"min"       default:"1.0"`
	Max      float64 `mapstructure:"max"       default:"100.0"`
	IClamp   float64 `mapstructure:"i_clamp"   default:"100.0"`
	Setpoint float64 `mapstructure:"setpoint"  default:"70.0"`
}
```

- [ ] **Step 2: Add Viper BindEnv calls inside Load()**

After the existing `_ = viper.BindEnv("target.path")` line, add:

```go
_ = viper.BindEnv("metrics_poller.enabled")
_ = viper.BindEnv("metrics_poller.interval_ms")
_ = viper.BindEnv("metrics_poller.endpoint")
_ = viper.BindEnv("metrics_poller.protocol")
_ = viper.BindEnv("pid.kp")
_ = viper.BindEnv("pid.ki")
_ = viper.BindEnv("pid.kd")
_ = viper.BindEnv("pid.min")
_ = viper.BindEnv("pid.max")
_ = viper.BindEnv("pid.i_clamp")
_ = viper.BindEnv("pid.setpoint")
```

- [ ] **Step 3: Verify the project still builds**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add pkg/config/config.go
git commit -m "feat(config): add MetricsPoller and PID config blocks"
```

---

## Task 2: Define MetricsReader port and generate mock

**Files:**
- Create: `app/repositories/metricsreader.go`
- Create (generated): `app/repositories/mocks/MetricsReader.go`

- [ ] **Step 1: Create the port file**

```go
// app/repositories/metricsreader.go
package repositories

import "context"

//go:generate mockery --all --output=mocks --outpkg=mocks

// MetricsReader is the port for pulling metrics from an external source.
// Each adapter maps source field names to domain names before returning.
type MetricsReader interface {
	Read(ctx context.Context) (map[string]float64, error)
}
```

- [ ] **Step 2: Generate the mock**

```bash
go generate ./app/repositories/
```

Expected: file `app/repositories/mocks/MetricsReader.go` is created.

- [ ] **Step 3: Commit**

```bash
git add app/repositories/metricsreader.go app/repositories/mocks/
git commit -m "feat(repositories): add MetricsReader port and generated mock"
```

---

## Task 3: Add prometheus/common dependency

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add the dependency**

```bash
go get github.com/prometheus/common@latest
```

Expected: `go.mod` and `go.sum` updated; `github.com/prometheus/common` and `github.com/prometheus/client_model` appear as dependencies.

- [ ] **Step 2: Commit**

```bash
git add go.mod go.sum
git commit -m "chore(deps): add github.com/prometheus/common for expfmt parsing"
```

---

## Task 4: PrometheusMetricsReader (TDD)

**Files:**
- Create: `app/repositories/prometheus/prometheus_test.go`
- Create: `app/repositories/prometheus/prometheus.go`

- [ ] **Step 1: Write the failing tests**

```go
// app/repositories/prometheus/prometheus_test.go
package prometheus_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adrianozp/gaardrail/app/repositories/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const prometheusBody = `# HELP process_cpu_seconds_total Total CPU time.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 0.42
# HELP mysql_global_status_threads_connected Open connections.
# TYPE mysql_global_status_threads_connected gauge
mysql_global_status_threads_connected 12
# HELP ignored_metric Not in mapping.
# TYPE ignored_metric gauge
ignored_metric 99
`

func TestPrometheusMetricsReader_Read_ExtractsMappedMetrics(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(prometheusBody))
	}))
	defer srv.Close()

	mappings := map[string]string{
		"process_cpu_seconds_total":              "cpu",
		"mysql_global_status_threads_connected":  "connections",
	}
	reader := prometheus.New(srv.URL, mappings)

	result, err := reader.Read(context.Background())

	require.NoError(t, err)
	assert.InDelta(t, 0.42, result["cpu"], 0.001)
	assert.InDelta(t, 12.0, result["connections"], 0.001)
	_, hasIgnored := result["ignored_metric"]
	assert.False(t, hasIgnored, "unmapped metric should not appear in result")
}

func TestPrometheusMetricsReader_Read_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	reader := prometheus.New(srv.URL, map[string]string{"cpu_total": "cpu"})

	_, err := reader.Read(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestPrometheusMetricsReader_Read_MissingMetricSkipped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(prometheusBody))
	}))
	defer srv.Close()

	// mapping asks for a metric that doesn't exist in the response
	mappings := map[string]string{
		"nonexistent_metric": "cpu",
	}
	reader := prometheus.New(srv.URL, mappings)

	result, err := reader.Read(context.Background())

	require.NoError(t, err)
	assert.Empty(t, result)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./app/repositories/prometheus/... -v
```

Expected: compilation error — `prometheus` package does not exist yet.

- [ ] **Step 3: Implement PrometheusMetricsReader**

```go
// app/repositories/prometheus/prometheus.go
package prometheus

import (
	"context"
	"fmt"
	"net/http"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

// PrometheusMetricsReader scrapes a Prometheus text-format endpoint and maps
// source metric names to domain names via the configured Mappings.
// Only the first sample of each matched metric family is used (suitable for
// no-label or single-instance metrics such as those from mysqld_exporter or
// node_exporter).
type PrometheusMetricsReader struct {
	endpoint string
	mappings map[string]string
	client   *http.Client
}

func New(endpoint string, mappings map[string]string) *PrometheusMetricsReader {
	return &PrometheusMetricsReader{
		endpoint: endpoint,
		mappings: mappings,
		client:   &http.Client{},
	}
}

func (r *PrometheusMetricsReader) Read(ctx context.Context) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus: unexpected status %d", resp.StatusCode)
	}

	var parser expfmt.TextParser
	mfs, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil && len(mfs) == 0 {
		return nil, fmt.Errorf("prometheus: parse error: %w", err)
	}

	result := make(map[string]float64)
	for source, domain := range r.mappings {
		mf, ok := mfs[source]
		if !ok || len(mf.GetMetric()) == 0 {
			continue
		}
		result[domain] = sampleValue(mf.GetMetric()[0])
	}

	return result, nil
}

func sampleValue(m *dto.Metric) float64 {
	switch {
	case m.Gauge != nil:
		return m.Gauge.GetValue()
	case m.Counter != nil:
		return m.Counter.GetValue()
	case m.Untyped != nil:
		return m.Untyped.GetValue()
	default:
		return 0
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./app/repositories/prometheus/... -v
```

Expected: all 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add app/repositories/prometheus/
git commit -m "feat(repositories): add PrometheusMetricsReader adapter"
```

---

## Task 5: JSONMetricsReader (TDD)

**Files:**
- Create: `app/repositories/jsonmetrics/jsonmetrics_test.go`
- Create: `app/repositories/jsonmetrics/jsonmetrics.go`

- [ ] **Step 1: Write the failing tests**

```go
// app/repositories/jsonmetrics/jsonmetrics_test.go
package jsonmetrics_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adrianozp/gaardrail/app/repositories/jsonmetrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONMetricsReader_Read_ExtractsMappedFields(t *testing.T) {
	body := `{"cpu_usage": 0.55, "active_connections": 7, "ignored": 999}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	mappings := map[string]string{
		"cpu_usage":          "cpu",
		"active_connections": "connections",
	}
	reader := jsonmetrics.New(srv.URL, mappings)

	result, err := reader.Read(context.Background())

	require.NoError(t, err)
	assert.InDelta(t, 0.55, result["cpu"], 0.001)
	assert.InDelta(t, 7.0, result["connections"], 0.001)
	_, hasIgnored := result["ignored"]
	assert.False(t, hasIgnored, "unmapped field should not appear in result")
}

func TestJSONMetricsReader_Read_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	reader := jsonmetrics.New(srv.URL, map[string]string{"cpu_usage": "cpu"})

	_, err := reader.Read(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}

func TestJSONMetricsReader_Read_MissingFieldSkipped(t *testing.T) {
	body := `{"other_field": 1.0}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	reader := jsonmetrics.New(srv.URL, map[string]string{"cpu_usage": "cpu"})

	result, err := reader.Read(context.Background())

	require.NoError(t, err)
	assert.Empty(t, result)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./app/repositories/jsonmetrics/... -v
```

Expected: compilation error — `jsonmetrics` package does not exist yet.

- [ ] **Step 3: Implement JSONMetricsReader**

```go
// app/repositories/jsonmetrics/jsonmetrics.go
package jsonmetrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// JSONMetricsReader scrapes a flat JSON object endpoint and maps source field
// names to domain names via the configured Mappings.
// All JSON values must be numeric (float64). Non-numeric fields are ignored.
type JSONMetricsReader struct {
	endpoint string
	mappings map[string]string
	client   *http.Client
}

func New(endpoint string, mappings map[string]string) *JSONMetricsReader {
	return &JSONMetricsReader{
		endpoint: endpoint,
		mappings: mappings,
		client:   &http.Client{},
	}
}

func (r *JSONMetricsReader) Read(ctx context.Context) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jsonmetrics: unexpected status %d", resp.StatusCode)
	}

	var raw map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("jsonmetrics: decode error: %w", err)
	}

	result := make(map[string]float64)
	for source, domain := range r.mappings {
		if v, ok := raw[source]; ok {
			result[domain] = v
		}
	}

	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./app/repositories/jsonmetrics/... -v
```

Expected: all 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add app/repositories/jsonmetrics/
git commit -m "feat(repositories): add JSONMetricsReader adapter"
```

---

## Task 6: PollingHandler (TDD)

**Files:**
- Create: `app/handlers/pollmetrics/pollmetrics.go`
- Create (generated): `app/handlers/pollmetrics/mocks/ProcessMetrics.go`
- Create: `app/handlers/pollmetrics/pollmetrics_test.go`

- [ ] **Step 1: Define the handler with its interface and go:generate**

```go
// app/handlers/pollmetrics/pollmetrics.go
package pollmetrics

import (
	"context"
	"log"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/repositories"
)

//go:generate mockery --all --output=mocks --outpkg=mocks

// ProcessMetrics is the port for forwarding collected metrics into the PID pipeline.
type ProcessMetrics interface {
	Process(m entities.Metrics) error
}

// PollingHandler ticks on a configurable interval, reads metrics from a
// MetricsReader, and forwards them to ProcessMetrics. Errors from Read are
// logged and skipped — the PID controller retains its last state until the
// next successful read.
type PollingHandler struct {
	reader         repositories.MetricsReader
	processMetrics ProcessMetrics
	interval       time.Duration
	done           chan struct{}
}

func New(reader repositories.MetricsReader, pm ProcessMetrics, interval time.Duration) *PollingHandler {
	return &PollingHandler{
		reader:         reader,
		processMetrics: pm,
		interval:       interval,
		done:           make(chan struct{}),
	}
}

// Done returns a channel that is closed when the run loop exits.
// Useful in tests to assert clean shutdown.
func (h *PollingHandler) Done() <-chan struct{} {
	return h.done
}

func (h *PollingHandler) Start(ctx context.Context) error {
	go h.run(ctx)
	return nil
}

func (h *PollingHandler) run(ctx context.Context) {
	defer close(h.done)

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("pollmetrics: shutting down")
			return
		case <-ticker.C:
			metrics, err := h.reader.Read(ctx)
			if err != nil {
				log.Printf("pollmetrics: read error: %s", err)
				continue
			}

			m := entities.Metrics{
				MeasureTime: time.Now(),
				Metrics:     metrics,
			}

			if err := h.processMetrics.Process(m); err != nil {
				log.Printf("pollmetrics: process error: %s", err)
			}
		}
	}
}
```

- [ ] **Step 2: Generate mocks**

```bash
go generate ./app/handlers/pollmetrics/
```

Expected: `app/handlers/pollmetrics/mocks/ProcessMetrics.go` created.

- [ ] **Step 3: Write the tests**

```go
// app/handlers/pollmetrics/pollmetrics_test.go
package pollmetrics_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/handlers/pollmetrics"
	"github.com/adrianozp/gaardrail/app/handlers/pollmetrics/mocks"
	repomocks "github.com/adrianozp/gaardrail/app/repositories/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPollingHandler_CallsReadAndProcessOnEachTick(t *testing.T) {
	mockReader := repomocks.NewMetricsReader(t)
	mockProcess := mocks.NewProcessMetrics(t)

	ticked := make(chan struct{}, 3)

	mockReader.On("Read", mock.Anything).
		Return(map[string]float64{"cpu": 0.4}, nil)
	mockProcess.On("Process", mock.MatchedBy(func(m entities.Metrics) bool {
		return m.Metrics["cpu"] == 0.4
	})).Run(func(_ mock.Arguments) {
		ticked <- struct{}{}
	}).Return(nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := pollmetrics.New(mockReader, mockProcess, 5*time.Millisecond)
	require.NoError(t, handler.Start(ctx))

	for i := range 2 {
		select {
		case <-ticked:
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("tick %d never happened", i+1)
		}
	}
}

func TestPollingHandler_ReadErrorDoesNotStopLoop(t *testing.T) {
	mockReader := repomocks.NewMetricsReader(t)
	mockProcess := mocks.NewProcessMetrics(t)

	processed := make(chan struct{}, 1)

	mockReader.On("Read", mock.Anything).
		Return(map[string]float64(nil), errors.New("read failed")).Once()
	mockReader.On("Read", mock.Anything).
		Return(map[string]float64{"cpu": 0.5}, nil)
	mockProcess.On("Process", mock.MatchedBy(func(m entities.Metrics) bool {
		return m.Metrics["cpu"] == 0.5
	})).Run(func(_ mock.Arguments) {
		processed <- struct{}{}
	}).Return(nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := pollmetrics.New(mockReader, mockProcess, 5*time.Millisecond)
	require.NoError(t, handler.Start(ctx))

	select {
	case <-processed:
		// success: loop continued past the error
	case <-time.After(500 * time.Millisecond):
		t.Fatal("process was never called after read error — loop may have stopped")
	}
}

func TestPollingHandler_ContextCancellationExitsCleanly(t *testing.T) {
	mockReader := repomocks.NewMetricsReader(t)
	mockProcess := mocks.NewProcessMetrics(t)

	ctx, cancel := context.WithCancel(context.Background())

	// long interval — we cancel before the first tick
	handler := pollmetrics.New(mockReader, mockProcess, 1*time.Hour)
	require.NoError(t, handler.Start(ctx))

	cancel()

	select {
	case <-handler.Done():
		// success
	case <-time.After(200 * time.Millisecond):
		t.Fatal("goroutine did not exit after context cancellation")
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./app/handlers/pollmetrics/... -v
```

Expected: all 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add app/handlers/pollmetrics/
git commit -m "feat(handlers): add PollingHandler with ProcessMetrics interface and tests"
```

---

## Task 7: Wire everything into fx

**Files:**
- Create: `cmd/api/modules/metricspoller.go`
- Modify: `cmd/api/options/options.go`

- [ ] **Step 1: Create the metricspoller module**

```go
// cmd/api/modules/metricspoller.go
package modules

import (
	"context"
	"fmt"
	"time"

	"github.com/adrianozp/gaardrail/app/handlers/pollmetrics"
	"github.com/adrianozp/gaardrail/app/repositories"
	jsonmetricsrepo "github.com/adrianozp/gaardrail/app/repositories/jsonmetrics"
	prometheusrepo "github.com/adrianozp/gaardrail/app/repositories/prometheus"
	"github.com/adrianozp/gaardrail/app/usecases/processmetrics"
	"github.com/adrianozp/gaardrail/internal/controller"
	"github.com/adrianozp/gaardrail/pkg/config"
	"go.uber.org/fx"
)

func MetricsPollerFactories() fx.Option {
	return fx.Provide(
		newPIDController,
		processmetrics.NewProcessMetricsUseCase,
		newMetricsReader,
		newPollingHandler,
	)
}

func MetricsPollerInjections() fx.Option {
	return fx.Provide(
		func(c *controller.Controller) processmetrics.Controller { return c },
		func(uc processmetrics.ProcessMetricsUseCase) pollmetrics.ProcessMetrics { return uc },
	)
}

func MetricsPollerLifecycle() fx.Option {
	return fx.Invoke(func(lc fx.Lifecycle, ph *pollmetrics.PollingHandler, cfg config.Config) {
		if !cfg.MetricsPoller.Enabled {
			return
		}
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				return ph.Start(ctx)
			},
		})
	})
}

func newPIDController(cfg config.Config) *controller.Controller {
	return controller.New(controller.ControllerParams{
		Kp:       cfg.PID.Kp,
		Ki:       cfg.PID.Ki,
		Kd:       cfg.PID.Kd,
		Min:      cfg.PID.Min,
		Max:      cfg.PID.Max,
		IClamp:   cfg.PID.IClamp,
		Setpoint: cfg.PID.Setpoint,
	})
}

func newMetricsReader(cfg config.Config) (repositories.MetricsReader, error) {
	if !cfg.MetricsPoller.Enabled {
		return &noopMetricsReader{}, nil
	}
	switch cfg.MetricsPoller.Protocol {
	case "prometheus":
		return prometheusrepo.New(cfg.MetricsPoller.Endpoint, cfg.MetricsPoller.Mappings), nil
	case "json":
		return jsonmetricsrepo.New(cfg.MetricsPoller.Endpoint, cfg.MetricsPoller.Mappings), nil
	default:
		return nil, fmt.Errorf("metricspoller: unknown protocol %q", cfg.MetricsPoller.Protocol)
	}
}

func newPollingHandler(reader repositories.MetricsReader, pm pollmetrics.ProcessMetrics, cfg config.Config) *pollmetrics.PollingHandler {
	interval := time.Duration(cfg.MetricsPoller.IntervalMs) * time.Millisecond
	return pollmetrics.New(reader, pm, interval)
}

// noopMetricsReader is used when MetricsPoller.Enabled is false.
// It allows fx to satisfy the repositories.MetricsReader dependency without
// starting any real scraping.
type noopMetricsReader struct{}

func (n *noopMetricsReader) Read(_ context.Context) (map[string]float64, error) {
	return nil, nil
}
```

- [ ] **Step 2: Register the metrics poller modules in options.go**

Add three lines inside `Options()` in `cmd/api/options/options.go`, after `modules.OrchestratorInjections()`:

```go
modules.MetricsPollerFactories(),
modules.MetricsPollerInjections(),
modules.MetricsPollerLifecycle(),
```

The full `Options()` function should look like:

```go
func Options() fx.Option {
	return fx.Options(
		fx.Provide(
			config.Load,
			httpserver.New,
		),

		modules.KafkaFactories(),
		modules.HTTPFactories(),

		modules.OrchestratorFactories(),
		modules.OrchestratorInjections(),

		modules.MetricsPollerFactories(),
		modules.MetricsPollerInjections(),
		modules.MetricsPollerLifecycle(),

		modules.MessageFactories(),
		modules.MessageInjections(),
		modules.MessageEndpoints(),

		fx.Invoke(func(lc fx.Lifecycle, router *gin.Engine, cfg config.Config) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go router.Run(cfg.HTTP.Addr)
					return nil
				},
			})
		}),

		fx.Invoke(func(lc fx.Lifecycle, o *orchestrator.Orchestrator) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					return o.Start(ctx)
				},
			})
		}),
	)
}
```

- [ ] **Step 3: Build to verify no wiring errors**

```bash
go build ./...
```

Expected: compiles without errors.

- [ ] **Step 4: Run the full test suite**

```bash
go test ./...
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/api/modules/metricspoller.go cmd/api/options/options.go
git commit -m "feat(modules): wire MetricsPoller, PID controller, and PollingHandler into fx"
```

---

## Verification

To verify end-to-end with a real Prometheus exporter:

1. Start a `node_exporter` locally (or any Prometheus-format exporter).
2. Set config (or env vars):
   ```yaml
   metrics_poller:
     enabled: true
     interval_ms: 3000
     endpoint: "http://localhost:9100/metrics"
     protocol: "prometheus"
     mappings:
       node_load1: "cpu"
   pid:
     setpoint: 1.0
     kp: 1.0
     ki: 0.1
     kd: 0.01
     min: 1.0
     max: 50.0
     i_clamp: 50.0
   ```
3. Start gaardrail: `go run ./cmd/api/`
4. Observe logs: every 3s you should see the orchestrator's drain rate being adjusted by the PID output.
5. To test JSON path: stand up a simple HTTP server returning `{"cpu": 0.7}` and set `protocol: "json"` with `mappings: {cpu: "cpu"}`.
