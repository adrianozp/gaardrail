# Flood Test Design

## Goal

Validate the PID controller end-to-end: generate heavy SQL queries via the HTTP API, observe MySQL CPU rising, and confirm the controller stabilizes drain rate at a target setpoint.

## Architecture

```
flood.sh (curl loop)
    │
    └── POST /messages {"payload": "<sql query>"}
              │
         gaardrail (local, :8080)
              │
         Kafka (enqueue)
              │
         Orchestrator (rate-limited dequeue)
              │
         SQLRepository ──► MySQL (:3307)
                                │
                         mysqld-exporter
                                │
                          Prometheus (:9091)
                                │
         MetricsPoller ◄── /api/v1/query (prometheusapi reader)
              │
         PID Controller ──► SetDrainRate(n msgs/s)
```

## Directory Structure

```
flood-test/
├── docker-compose-flood.yml
├── prometheus.yml
├── grafana/
│   └── dashboard.json
├── config.yml
└── scripts/
    ├── setup-db.sh
    └── flood.sh
```

## Components

### docker-compose-flood.yml

Services (ports shifted to avoid conflicts with the main compose):

| Service         | Image                      | Host Port |
|-----------------|----------------------------|-----------|
| mysql           | mysql:8.0                  | 3307      |
| mysqld-exporter | prom/mysqld-exporter       | 9105      |
| prometheus      | prom/prometheus            | 9091      |
| grafana         | grafana/grafana            | 3000      |
| kafka           | apache/kafka               | 9093      |

MySQL starts with reduced resources to make CPU load more visible:
- `--innodb-buffer-pool-size=128M`
- `--max-connections=50`

### New: `prometheusapi` MetricsReader

Package: `app/repositories/prometheusapi`

Queries the Prometheus HTTP API:
```
GET /api/v1/query?query=<expr>
```

Response format:
```json
{"status":"success","data":{"result":[{"metric":{},"value":[1234567890,"0.47"]}]}}
```

The mapping key is used directly as the PromQL expression (the `query=` parameter). The float value is extracted from `data.result[0].value[1]`.

Registered in `repositories.go` as `case "prometheusapi"`.

### sqlclient MySQL fix

Add `Driver string` to `sqlclient.Config`. Default remains `"postgres"` for backwards compatibility. Flood test uses `"mysql"` with `go-sql-driver/mysql`.

### scripts/setup-db.sh

Connects to MySQL via `docker exec` and runs:

```sql
CREATE DATABASE IF NOT EXISTS gaardrail;
USE gaardrail;

CREATE TABLE IF NOT EXISTS flood_data (
  id      INT AUTO_INCREMENT PRIMARY KEY,
  val     INT NOT NULL,
  payload VARCHAR(255) NOT NULL
);
```

Inserts 500k rows in batches of 10k using a shell loop. Estimated time: ~30s.

### Heavy SQL Queries

Three query types, rotated by the flood script:

**Q1 — full scan + group by + sort:**
```sql
SELECT val % 1000 AS bucket, COUNT(*), SUM(val), AVG(val)
FROM flood_data GROUP BY bucket ORDER BY COUNT(*) DESC LIMIT 50
```

**Q2 — self-join on range:**
```sql
SELECT a.id, b.val FROM flood_data a
JOIN flood_data b ON b.val BETWEEN a.val AND a.val + 100
WHERE a.id % 500 = 0 LIMIT 200
```

**Q3 — correlated subquery:**
```sql
SELECT id, val,
  (SELECT COUNT(*) FROM flood_data b WHERE b.val <= a.val) AS rank
FROM flood_data a WHERE id % 2000 = 0
```

### scripts/flood.sh

Posts messages in a tight loop (no sleep between requests) to fill the Kafka queue quickly. Each iteration picks one of the three queries. Accepts a `COUNT` argument (default: 500 messages).

```bash
./scripts/flood.sh         # posts 500 messages
./scripts/flood.sh 2000    # posts 2000 messages
```

### flood-test/config.yml

```yaml
http:
  addr: ":8080"

kafka:
  brokers: ["localhost:9093"]
  topic: messages
  group_id: gaardrail

target:
  protocol: sql
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

### Grafana Dashboard

Two time-series panels:

1. **MySQL CPU %** — `rate(process_cpu_seconds_total{job="mysqld"}[15s]) * 100` with a horizontal threshold line at the setpoint (60%)
2. **Drain Rate** — sourced from gaardrail logs (manual annotation) or omitted in favor of observing CPU convergence

## Code Changes Required

| File | Change |
|------|--------|
| `internal/sqlclient/sqlclient.go` | Add `Driver` field to Config; use `sql.Open(cfg.Driver, cfg.DSN)` |
| `go.mod` | Add `github.com/go-sql-driver/mysql` |
| `app/repositories/prometheusapi/prometheusapi.go` | New MetricsReader implementation |
| `app/repositories/repositories.go` | Add `case "prometheusapi"` to `NewMetricsReader` switch |
| `app/repositories/repositories.go` | Remove now-redundant code moved to `targets/` package |

## Run Order

```bash
# 1. Start infra
cd flood-test && docker compose -f docker-compose-flood.yml up -d

# 2. Wait for MySQL to be healthy, then populate
./flood-test/scripts/setup-db.sh

# 3. Run gaardrail locally
APP_PATH=./flood-test go run ./cmd/api

# 4. Flood with messages
./flood-test/scripts/flood.sh 1000

# 5. Watch CPU converge in Grafana: http://localhost:3000
```

## Success Criteria

- MySQL CPU rises above setpoint after flood starts
- PID controller reduces drain rate to let CPU fall
- CPU stabilizes within ±10% of setpoint within ~60s
- No message loss (Kafka consumer lag eventually reaches 0 after flood stops)
