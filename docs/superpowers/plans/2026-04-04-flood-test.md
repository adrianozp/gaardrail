# Flood Test Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Validate the PID controller end-to-end: flood heavy SQL queries via `POST /messages`, observe MySQL CPU rising, confirm the controller stabilizes drain rate at setpoint.

**Architecture:** Messages with SQL query payloads enter via HTTP API → Kafka → orchestrator (rate-limited by PID) → SQLRepository → MySQL. A new `prometheusapi` MetricsReader queries the Prometheus HTTP API with PromQL to compute real CPU% (`rate(process_cpu_seconds_total[15s])*100`), feeding the PID loop. All infra runs in `docker-compose-flood.yml`; gaardrail runs locally.

**Tech Stack:** Go, MySQL 8.0, prom/mysqld-exporter, prom/prometheus, grafana/grafana, apache/kafka (KRaft), `go-sql-driver/mysql`, `lib/pq`

---

## Task 1: sqlclient MySQL driver support

**Files:**
- Modify: `internal/sqlclient/sqlclient.go`
- Add dependency: `go.mod` / `go.sum`

- [ ] **Step 1: Add go-sql-driver/mysql**

```bash
cd /home/adrianozdp/workspace/pfc/gaardrail
go get github.com/go-sql-driver/mysql
```

Expected: `go: added github.com/go-sql-driver/mysql vX.X.X`

- [ ] **Step 2: Rewrite sqlclient.go with Driver field**

Replace the entire file `internal/sqlclient/sqlclient.go`:

```go
package sqlclient

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type Config struct {
	DSN    string
	Driver string // "mysql" or "postgres"; defaults to "mysql" when empty
}

type Client struct {
	db *sql.DB
}

func New(cfg Config) (*Client, error) {
	driver := cfg.Driver
	if driver == "" {
		driver = "mysql"
	}
	db, err := sql.Open(driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("sqlclient: open: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlclient: ping: %w", err)
	}
	return &Client{db: db}, nil
}

// NewFromDB creates a Client from an existing *sql.DB. Useful for testing.
func NewFromDB(db *sql.DB) *Client {
	return &Client{db: db}
}

func (c *Client) Exec(query string) error {
	_, err := c.db.Exec(query)
	return err
}

func (c *Client) Close() error {
	return c.db.Close()
}
```

- [ ] **Step 3: Verify build**

```bash
go build ./internal/sqlclient/...
```

Expected: no output (clean build)

- [ ] **Step 4: Verify existing tests still pass**

```bash
go test ./app/repositories/targets/sql/...
```

Expected: `PASS` (tests use `NewFromDB` / sqlmock, unaffected by driver change)

- [ ] **Step 5: Commit**

```bash
git add internal/sqlclient/sqlclient.go go.mod go.sum
git commit -m "feat(sqlclient): add configurable driver, add go-sql-driver/mysql"
```

---

## Task 2: Driver field in Target config

**Files:**
- Modify: `pkg/config/config.go`
- Modify: `app/repositories/targets/targets.go`

- [ ] **Step 1: Add Driver to Target struct in config.go**

In `pkg/config/config.go`, replace the `Target` struct:

```go
type Target struct {
	Protocol string `mapstructure:"protocol" default:"http" validate:"required"`
	BaseURL  string `mapstructure:"base_url" default:"http://localhost:9090"`
	Path     string `mapstructure:"path"     default:"/events"`
	DSN      string `mapstructure:"dsn"`
	Driver   string `mapstructure:"driver"   default:"mysql"`
}
```

- [ ] **Step 2: Add env binding for target.driver in config.go**

In the `Load()` function, after the existing `_ = viper.BindEnv("target.dsn")` line, add:

```go
_ = viper.BindEnv("target.driver")
```

- [ ] **Step 3: Pass Driver in targets.go**

In `app/repositories/targets/targets.go`, replace the `"sql"` case:

```go
case "sql":
    client, err := sqlclient.New(sqlclient.Config{
        DSN:    cfg.Target.DSN,
        Driver: cfg.Target.Driver,
    })
    if err != nil {
        return nil, fmt.Errorf("target: sql: %w", err)
    }
    return sqlrepo.NewSQLRepository(client), nil
```

- [ ] **Step 4: Build**

```bash
go build ./...
```

Expected: no output

- [ ] **Step 5: Commit**

```bash
git add pkg/config/config.go app/repositories/targets/targets.go
git commit -m "feat(config): add Target.Driver for configurable sql driver"
```

---

## Task 3: prometheusapi MetricsReader — RED (failing test)

**Files:**
- Create: `app/repositories/readers/prometheusapi/prometheusapi_test.go`

- [ ] **Step 1: Write the failing test**

Create `app/repositories/readers/prometheusapi/prometheusapi_test.go`:

```go
package prometheusapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/repositories/readers/prometheusapi"
	"github.com/adrianozp/gaardrail/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const successBody = `{
	"status": "success",
	"data": {
		"resultType": "vector",
		"result": [{"metric": {}, "value": [1234567890, "47.5"]}]
	}
}`

func TestPrometheusAPIReader_Read_ReturnsMappedValue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "rate(process_cpu_seconds_total[15s])*100", r.URL.Query().Get("query"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(successBody))
	}))
	defer srv.Close()

	clock.WithTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	reader := prometheusapi.New(srv.URL, map[string]string{
		"rate(process_cpu_seconds_total[15s])*100": "cpu",
	})

	result, err := reader.Read(context.Background())
	require.NoError(t, err)

	expected := entities.Metrics{
		MeasureTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Metrics:     map[string]float64{"cpu": 47.5},
	}
	require.Equal(t, expected, result)
}

func TestPrometheusAPIReader_Read_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	reader := prometheusapi.New(srv.URL, map[string]string{"rate(cpu[15s])": "cpu"})

	_, err := reader.Read(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestPrometheusAPIReader_Read_EmptyResult_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","data":{"result":[]}}`))
	}))
	defer srv.Close()

	reader := prometheusapi.New(srv.URL, map[string]string{"nonexistent": "cpu"})

	_, err := reader.Read(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no results")
}
```

- [ ] **Step 2: Run to verify it fails (compile error)**

```bash
go test ./app/repositories/readers/prometheusapi/...
```

Expected: `cannot find package "...prometheusapi"` or similar compile error

---

## Task 4: prometheusapi MetricsReader — GREEN (implementation)

**Files:**
- Create: `app/repositories/readers/prometheusapi/prometheusapi.go`

- [ ] **Step 1: Write the implementation**

Create `app/repositories/readers/prometheusapi/prometheusapi.go`:

```go
package prometheusapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/pkg/clock"
)

type Reader struct {
	endpoint string
	mappings map[string]string
	http     *http.Client
}

func New(endpoint string, mappings map[string]string) *Reader {
	return &Reader{
		endpoint: endpoint,
		mappings: mappings,
		http:     &http.Client{Timeout: 5 * time.Second},
	}
}

type apiResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result []struct {
			Value [2]json.RawMessage `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func (r *Reader) Read(ctx context.Context) (entities.Metrics, error) {
	metrics := make(map[string]float64)
	for expr, name := range r.mappings {
		val, err := r.queryValue(ctx, expr)
		if err != nil {
			return entities.Metrics{}, fmt.Errorf("prometheusapi: %s: %w", name, err)
		}
		metrics[name] = val
	}
	return entities.Metrics{
		MeasureTime: clock.Now(),
		Metrics:     metrics,
	}, nil
}

func (r *Reader) queryValue(ctx context.Context, expr string) (float64, error) {
	reqURL := r.endpoint + "?query=" + url.QueryEscape(expr)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, err
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return 0, err
	}

	if len(apiResp.Data.Result) == 0 {
		return 0, fmt.Errorf("no results for query %q", expr)
	}

	var valStr string
	if err := json.Unmarshal(apiResp.Data.Result[0].Value[1], &valStr); err != nil {
		return 0, err
	}

	return strconv.ParseFloat(valStr, 64)
}
```

- [ ] **Step 2: Run tests to verify they pass**

```bash
go test ./app/repositories/readers/prometheusapi/... -v
```

Expected:
```
--- PASS: TestPrometheusAPIReader_Read_ReturnsMappedValue
--- PASS: TestPrometheusAPIReader_Read_HTTPError
--- PASS: TestPrometheusAPIReader_Read_EmptyResult_ReturnsError
PASS
```

- [ ] **Step 3: Commit**

```bash
git add app/repositories/readers/prometheusapi/
git commit -m "feat(readers): add prometheusapi MetricsReader for PromQL-computed metrics"
```

---

## Task 5: Register prometheusapi in readers switch

**Files:**
- Modify: `app/repositories/readers/repositories.go`

- [ ] **Step 1: Add import and case**

In `app/repositories/readers/repositories.go`, add the import:

```go
prometheusapirepo "github.com/adrianozp/gaardrail/app/repositories/readers/prometheusapi"
```

In the `NewMetricsReader` switch, add before `default`:

```go
case "prometheusapi":
    return prometheusapirepo.New(cfg.MetricsPoller.Endpoint, cfg.MetricsPoller.Mappings), nil
```

The full updated `NewMetricsReader`:

```go
func NewMetricsReader(cfg config.Config) (MetricsReader, error) {
	if !cfg.MetricsPoller.Enabled {
		return &noopMetricsReader{}, nil
	}
	switch cfg.MetricsPoller.Protocol {
	case "prometheus":
		return prometheusrepo.New(cfg.MetricsPoller.Endpoint, cfg.MetricsPoller.Mappings), nil
	case "json":
		return jsonmetricsrepo.New(cfg.MetricsPoller.Endpoint, cfg.MetricsPoller.Mappings), nil
	case "prometheusapi":
		return prometheusapirepo.New(cfg.MetricsPoller.Endpoint, cfg.MetricsPoller.Mappings), nil
	default:
		return nil, fmt.Errorf("metricspoller: unknown protocol %q", cfg.MetricsPoller.Protocol)
	}
}
```

- [ ] **Step 2: Build and test all**

```bash
go build ./... && go test ./...
```

Expected: all packages build and pass (orchestrator tests pass too)

- [ ] **Step 3: Commit**

```bash
git add app/repositories/readers/repositories.go
git commit -m "feat(readers): register prometheusapi in MetricsReader factory"
```

---

## Task 6: flood-test infrastructure files

**Files:**
- Create: `flood-test/docker-compose-flood.yml`
- Create: `flood-test/prometheus.yml`
- Create: `flood-test/grafana/provisioning/datasources/prometheus.yml`
- Create: `flood-test/grafana/provisioning/dashboards/flood.yml`
- Create: `flood-test/grafana/dashboards/flood.json`

- [ ] **Step 1: Create flood-test/docker-compose-flood.yml**

```yaml
services:
  mysql:
    image: mysql:8.0
    ports:
      - "3307:3306"
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: gaardrail
    command: >
      --innodb-buffer-pool-size=128M
      --max-connections=50
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-uroot", "-proot"]
      interval: 5s
      timeout: 5s
      retries: 10

  mysqld-exporter:
    image: prom/mysqld-exporter:latest
    ports:
      - "9105:9104"
    environment:
      DATA_SOURCE_NAME: "root:root@(mysql:3306)/"
    depends_on:
      mysql:
        condition: service_healthy

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
    depends_on:
      - mysqld-exporter

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      GF_AUTH_ANONYMOUS_ENABLED: "true"
      GF_AUTH_ANONYMOUS_ORG_ROLE: "Admin"
    volumes:
      - ./grafana/provisioning:/etc/grafana/provisioning:ro
      - ./grafana/dashboards:/var/lib/grafana/dashboards:ro
    depends_on:
      - prometheus

  kafka:
    image: apache/kafka:latest
    ports:
      - "9093:9092"
    environment:
      KAFKA_NODE_ID: 1
      KAFKA_PROCESS_ROLES: broker,controller
      KAFKA_CONTROLLER_QUORUM_VOTERS: 1@kafka:9094
      KAFKA_LISTENERS: PLAINTEXT://:9092,CONTROLLER://:9094
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9093
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_AUTO_CREATE_TOPICS_ENABLE: "true"
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: 1
      KAFKA_TRANSACTION_STATE_LOG_MIN_ISR: 1
      CLUSTER_ID: 5L6g3nShT-eMCtK--X86sw
```

- [ ] **Step 2: Create flood-test/prometheus.yml**

```yaml
global:
  scrape_interval: 5s

scrape_configs:
  - job_name: mysqld
    static_configs:
      - targets: ["mysqld-exporter:9104"]
```

- [ ] **Step 3: Create Grafana datasource provisioning**

Create `flood-test/grafana/provisioning/datasources/prometheus.yml`:

```yaml
apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    uid: prometheus
    url: http://prometheus:9090
    isDefault: true
    access: proxy
```

- [ ] **Step 4: Create Grafana dashboard provisioning config**

Create `flood-test/grafana/provisioning/dashboards/flood.yml`:

```yaml
apiVersion: 1
providers:
  - name: flood
    folder: Flood Test
    type: file
    options:
      path: /var/lib/grafana/dashboards
```

- [ ] **Step 5: Create Grafana dashboard JSON**

Create `flood-test/grafana/dashboards/flood.json`:

```json
{
  "annotations": {"list": []},
  "editable": true,
  "graphTooltip": 0,
  "id": null,
  "links": [],
  "panels": [
    {
      "datasource": {"type": "prometheus", "uid": "prometheus"},
      "fieldConfig": {
        "defaults": {
          "color": {"mode": "palette-classic"},
          "custom": {
            "drawStyle": "line",
            "fillOpacity": 10,
            "lineWidth": 2,
            "showPoints": "auto",
            "thresholdsStyle": {"mode": "line"}
          },
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {"color": "green", "value": null},
              {"color": "red", "value": 60}
            ]
          },
          "unit": "percent"
        },
        "overrides": []
      },
      "gridPos": {"h": 14, "w": 24, "x": 0, "y": 0},
      "id": 1,
      "options": {
        "legend": {
          "calcs": ["mean", "max"],
          "displayMode": "table",
          "placement": "bottom",
          "showLegend": true
        },
        "tooltip": {"mode": "single", "sort": "none"}
      },
      "targets": [
        {
          "datasource": {"type": "prometheus", "uid": "prometheus"},
          "expr": "rate(process_cpu_seconds_total{job=\"mysqld\"}[15s]) * 100",
          "legendFormat": "MySQL CPU %",
          "refId": "A"
        }
      ],
      "title": "MySQL CPU %  (setpoint: 60%)",
      "type": "timeseries"
    }
  ],
  "refresh": "5s",
  "schemaVersion": 38,
  "tags": ["flood-test"],
  "templating": {"list": []},
  "time": {"from": "now-15m", "to": "now"},
  "timepicker": {},
  "timezone": "browser",
  "title": "Flood Test",
  "uid": "flood-test",
  "version": 1
}
```

- [ ] **Step 6: Verify compose file syntax**

```bash
docker compose -f flood-test/docker-compose-flood.yml config --quiet
```

Expected: no output (valid syntax)

- [ ] **Step 7: Commit**

```bash
git add flood-test/docker-compose-flood.yml flood-test/prometheus.yml flood-test/grafana/
git commit -m "feat(flood-test): add docker-compose-flood and grafana dashboard"
```

---

## Task 7: flood-test app config

**Files:**
- Create: `flood-test/config.yml`

- [ ] **Step 1: Create flood-test/config.yml**

```yaml
http:
  addr: ":8080"

kafka:
  brokers:
    - "localhost:9093"
  topic: messages
  group_id: gaardrail

target:
  protocol: sql
  driver: mysql
  dsn: "root:root@tcp(localhost:3307)/gaardrail"

metrics_poller:
  enabled: true
  interval_ms: 3000
  protocol: prometheusapi
  endpoint: "http://localhost:9091/api/v1/query"
  mappings:
    'rate(process_cpu_seconds_total{job="mysqld"}[15s])*100': cpu

pid:
  setpoint: 60.0
  kp: 2.0
  ki: 0.3
  kd: 0.05
  min: 1.0
  max: 200.0
  i_clamp: 50.0
```

- [ ] **Step 2: Commit**

```bash
git add flood-test/config.yml
git commit -m "feat(flood-test): add gaardrail config for flood test"
```

---

## Task 8: Database population script

**Files:**
- Create: `flood-test/scripts/setup-db.sh`

- [ ] **Step 1: Create flood-test/scripts/setup-db.sh**

```bash
#!/bin/bash
set -e

COMPOSE="docker compose -f $(dirname "$0")/../docker-compose-flood.yml"

echo "Waiting for MySQL to be ready..."
until $COMPOSE exec -T mysql mysqladmin ping -uroot -proot --silent 2>/dev/null; do
  sleep 1
done

echo "Creating flood_data table..."
$COMPOSE exec -T mysql mysql -uroot -proot gaardrail <<'SQL'
CREATE TABLE IF NOT EXISTS flood_data (
  id      INT AUTO_INCREMENT PRIMARY KEY,
  val     INT NOT NULL,
  payload VARCHAR(255) NOT NULL
);
SQL

echo "Inserting 500k rows (50 batches of 10k)..."
for i in $(seq 1 50); do
  $COMPOSE exec -T mysql mysql -uroot -proot gaardrail <<'SQL'
INSERT INTO flood_data (val, payload)
SELECT
  FLOOR(RAND() * 1000000),
  LEFT(MD5(RAND()), 255)
FROM information_schema.columns a
CROSS JOIN information_schema.columns b
LIMIT 10000;
SQL
  echo "  Batch $i/50 done"
done

echo "Done. Total rows:"
$COMPOSE exec -T mysql mysql -uroot -proot -N -e "SELECT COUNT(*) FROM gaardrail.flood_data;"
```

- [ ] **Step 2: Make executable**

```bash
chmod +x flood-test/scripts/setup-db.sh
```

- [ ] **Step 3: Commit**

```bash
git add flood-test/scripts/setup-db.sh
git commit -m "feat(flood-test): add database population script"
```

---

## Task 9: Flood message script

**Files:**
- Create: `flood-test/scripts/flood.sh`

- [ ] **Step 1: Create flood-test/scripts/flood.sh**

```bash
#!/bin/bash
set -e

COUNT=${1:-500}
URL="http://localhost:8080/messages"

Q1="SELECT val % 1000 AS bucket, COUNT(*), SUM(val), AVG(val) FROM flood_data GROUP BY bucket ORDER BY COUNT(*) DESC LIMIT 50"
Q2="SELECT a.id, b.val FROM flood_data a JOIN flood_data b ON b.val BETWEEN a.val AND a.val + 100 WHERE a.id % 500 = 0 LIMIT 200"
Q3="SELECT id, val, (SELECT COUNT(*) FROM flood_data b WHERE b.val <= a.val) AS rnk FROM flood_data a WHERE id % 2000 = 0"
QUERIES=("$Q1" "$Q2" "$Q3")

echo "Flooding $URL with $COUNT messages..."
ERRORS=0
for i in $(seq 1 "$COUNT"); do
  Q="${QUERIES[$((RANDOM % 3))]}"
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$URL" \
    -H "Content-Type: application/json" \
    --data-raw "{\"payload\": \"$Q\"}")
  if [ "$STATUS" != "201" ]; then
    ERRORS=$((ERRORS + 1))
  fi
done

echo "Done. Sent: $COUNT  Errors: $ERRORS"
```

- [ ] **Step 2: Make executable**

```bash
chmod +x flood-test/scripts/flood.sh
```

- [ ] **Step 3: Commit**

```bash
git add flood-test/scripts/flood.sh
git commit -m "feat(flood-test): add message flood script"
```

---

## Task 10: Full verification

- [ ] **Step 1: Build all packages**

```bash
go build ./...
```

Expected: no output

- [ ] **Step 2: Run all tests**

```bash
go test ./... 2>&1 | grep -v '"level"'
```

Expected: all packages `ok` or `[no test files]`, no `FAIL`

- [ ] **Step 3: Validate compose**

```bash
docker compose -f flood-test/docker-compose-flood.yml config --quiet
```

Expected: no output

- [ ] **Step 4: Smoke-test infra startup**

```bash
docker compose -f flood-test/docker-compose-flood.yml up -d
sleep 15
docker compose -f flood-test/docker-compose-flood.yml ps
```

Expected: all services `running` or `healthy`. MySQL may take ~20s.

- [ ] **Step 5: Populate the database**

```bash
./flood-test/scripts/setup-db.sh
```

Expected: `Done. Total rows: 500000` (approximately)

- [ ] **Step 6: Start gaardrail locally**

```bash
APP_PATH=./flood-test go run ./cmd/api
```

Expected: app starts, connects to Kafka and MySQL, logs show PID controller active.

- [ ] **Step 7: Run the flood**

In a separate terminal:

```bash
./flood-test/scripts/flood.sh 1000
```

Expected: `Sent: 1000  Errors: 0`

- [ ] **Step 8: Observe convergence in Grafana**

Open `http://localhost:3000` (no login required). Navigate to Flood Test dashboard. Confirm:
- MySQL CPU % rises above 60% after flood starts
- PID controller lowers drain rate (visible in app logs: `orchestrator: updated drain rate`)
- CPU stabilizes near 60% within ~60s

- [ ] **Step 9: Tear down**

```bash
docker compose -f flood-test/docker-compose-flood.yml down -v
```

---

## Run Order (Quick Reference)

```bash
# 1. Start infra
docker compose -f flood-test/docker-compose-flood.yml up -d

# 2. Populate DB (wait ~20s for MySQL healthy first)
./flood-test/scripts/setup-db.sh

# 3. Run gaardrail locally
APP_PATH=./flood-test go run ./cmd/api

# 4. Flood (separate terminal)
./flood-test/scripts/flood.sh 1000

# 5. Watch: http://localhost:3000
```
