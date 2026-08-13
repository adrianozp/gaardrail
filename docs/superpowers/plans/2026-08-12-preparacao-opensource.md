# Preparação Open Source do gaardrail — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deixar `github.com/adrianozp/gaardrail` com mecânica de produto open source: README correto pós-split, releases versionadas via GoReleaser (binários + imagem Docker), CI cobrindo pull requests com lint, docs de comunidade, metadados no GitHub e o rig `gaardrail-flood-test` publicado — fechando com a release `v0.1.0`.

**Architecture:** Nove tasks incrementais no próprio repo (branch `main`, mantenedor único), cada uma com commit e verificação próprios. Publicação de imagem migra do CI-por-push para o fluxo de release por tag. Spec: `docs/superpowers/specs/2026-08-12-preparacao-opensource-design.md`.

**Tech Stack:** Go (versão do `go.mod`), GitHub Actions, GoReleaser v2, golangci-lint v2, Docker, gh CLI.

## Global Constraints

- Commits autorados APENAS como o usuário (`adrianozp <adrianozdp@gmail.com>`, config git local). **NUNCA adicionar trailer `Co-Authored-By` ou similar.**
- Licença permanece MPL-2.0 — não tocar no arquivo LICENSE.
- Todos os arquivos públicos novos (README, CONTRIBUTING, etc.) em inglês.
- `config/config.yaml`: **valores intactos** — somente comentários mudam.
- Nenhuma mudança de comportamento no código Go, exceto correções apontadas pelo lint.
- Imagem Docker: `adrianozdp/gaardrail`. Módulo: `github.com/adrianozp/gaardrail`.
- Trabalho direto em `main` (repo solo, padrão do projeto); push só nas tasks 7-9.
- Diretório de trabalho: `/home/adrianozdp/workspace/ufsc/pfc/gaardrail` (rig em `../gaardrail-flood-test`).

---

## Task 1: Reescrever o README

**Files:**
- Modify: `README.md` (substituição integral)

**Interfaces:**
- Produces: seções `## Quickstart`, `## Validation` e links para `CONTRIBUTING.md`/`SECURITY.md` que as tasks 6 e 8 referenciam; badge de CI que a task 7 valida.

- [ ] **Step 1: Substituir o conteúdo integral de README.md**

O novo README reaproveita as seções técnicas boas do atual (diagrama, orchestrator, PID, tipos, consumo, interfaces) e substitui as seções quebradas pelo split ("Running locally" com infra que não existe mais e todo o "Flood test"). Conteúdo completo:

````markdown
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

The built-in web panel is served at [http://localhost:8080](http://localhost:8080) — live tuning of every controller parameter, controller type switching and disturbance controls.

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
    Dequeue() (entities.Message, error)
    Ack(entities.Message) error
    Size() (int64, error)
}

type Target interface {
    Push(entities.Message) error
}

type MetricsReader interface {
    Read(ctx context.Context) (entities.Metrics, error)
}
```

Current implementations: Kafka and constant (queue), MySQL (target), Prometheus HTTP API (metrics reader).

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

Changing any parameter resets the integral accumulator to prevent windup artifacts. Full schema in [openapi.yaml](openapi.yaml).

## Validation

gaardrail's control loop was validated experimentally — system identification (FOPDT), closed-loop campaigns at T=10s and T=60s, disturbance rejection, and a PI / PI+feedforward / Smith predictor comparison. The full rig (Docker Compose stack, Grafana dashboards, experiment data and analysis scripts) lives at [gaardrail-flood-test](https://github.com/adrianozp/gaardrail-flood-test).

The project started as a final thesis (PFC) in Control and Automation Engineering at UFSC (Brazil).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Security reports: [SECURITY.md](SECURITY.md).

## License

[MPL-2.0](LICENSE)
````

- [ ] **Step 2: Conferir que não sobrou referência a caminhos do rig**

Run: `grep -nE 'flood-test/|make infra|setup-db|disturb\.sh|flood\.sh' README.md`
Expected: sem saída (a única menção a flood-test é a URL do GitHub, que não bate no padrão `flood-test/`).

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "Rewrite README for the post-split, user-facing repo"
```

---

## Task 2: Traduzir comentários do config.yaml

**Files:**
- Modify: `config/config.yaml` (só comentários; valores idênticos)

**Interfaces:**
- Consumes: nada. Produces: o arquivo que o README (Task 1) aponta como referência anotada.

- [ ] **Step 1: Substituir o conteúdo integral de config/config.yaml**

```yaml
http:
  addr: ":8080"
queue:
  protocol: constant # constant queue: no Kafka needed, always emits the same query
  capacity: 1000
  query: "SELECT id, val, score FROM flood_data ORDER BY RAND() LIMIT 100"
kafka:
  brokers:
    - "localhost:9092"
  topic: "messages"
  partition: 0
  group_id: "gaardrail"
target:
  protocol: sql
  driver: mysql
  dsn: "root:root@tcp(localhost:3306)/gaardrail"
  tls:
    enabled: false # true: encrypt the database connection (mysql only)
    skip_verify: false # true: encrypt without verifying the server certificate (self-signed certs / tests)
    ca_file: "" # optional: PEM CA bundle for full server verification
    server_name: "" # optional: name used for certificate verification
metrics_poller:
  enabled: true
  interval_ms: 5000 # T=5s (production: 60000 — CloudWatch/Azure/GCP)
  protocol: prometheusapi
  endpoint: "http://localhost:9090/api/v1/query"
  mappings:
    # RAW value (as a real provider delivers it): irate = rate over the last 2 points.
    # Smoothing happens in the controller (filter_size). Window >= 3x scrape for >= 2
    # samples: [15s] with 5s scrape ([3m] with 60s scrape, [1m] with 1s).
    'irate(container_cpu_usage_seconds_total{container_label_com_docker_compose_service="mysql"}[15s])*100': cpu
controller:
  type: "pidff"
orchestrator:
  rate: 0
  workers: 1
  burst: 5
grafana:
  url: "http://localhost:3000/d/flood-test/flood-test?kiosk"
pid: # PIFF baseline at T=10s (PI documented in the regime-10s campaign, truncated at 3 decimals)
  setpoint: 50
  kp: 0.058
  ki: 0.009
  kd: 0
  min: 0
  max: 50 # headroom above the steady-state drain (~22 for setpoint 50%)
  i_clamp: 40 # the integral must reach the steady-state drain (position form)
  filter_size: 1 # raw metric (overshoot is handled by the setpoint soft-start)
  ff_gain: 2.389 # feedforward u_ff=setpoint/K (K = static gain documented at T=10s) -> 0 disables
  setpoint_filter_tau: 10 # soft-start ~2T (T=5s)
smith:
  setpoint: 50.0
  kp: 0.042283 # gentle and stable (exp.12, ts=240s) — high gains oscillate at T=60s
  ki: 0.00781074
  min: 0.0
  max: 40
  i_clamp: 40
  model_k: 2.47465 # static gain of the identified FOPDT (%CPU per msg/s)
  model_tau: 33.1391 # time constant (s)
  model_theta: 61.74 # EFFECTIVE delay (continuous theta + ~T of ZOH/sampling) -> d=1 sample
  sample_seconds: 5.0 # sampling period T (60.0 in production)
  filter_size: 1 # raw metric at 60s is already an average of ~1300 queries/min; no extra MA
autopid: # self-tuning: open-loop step -> FOPDT (2-point) -> automatic gains
  tuning_rule: amigo # amigo (Astrom-Hagglund) | simc (Skogestad)
  mode: pi # pi | pid (rule variant; production runs PI)
  tau_c: 0 # SIMC: desired closed-loop time constant (s); 0 => uses theta_eff (robust)
  baseline_output: 0 # baseline drain rate; 0 => pid.min
  step_output: 0 # drain rate during the step; 0 => pid.max
  baseline_seconds: 120 # baseline duration before the step
  identify_seconds: 600 # step duration (~10 samples at T=60s)
  # operating limits (min/max/i_clamp/setpoint/filter_size) inherited from pid.*
```

- [ ] **Step 2: Verificar que só comentários mudaram**

Run: `diff <(grep -o '^[^#]*' config/config.yaml | sed 's/ *$//') <(git show HEAD:config/config.yaml | grep -o '^[^#]*' | sed 's/ *$//')`
Expected: sem saída (conteúdo não-comentário idêntico).

- [ ] **Step 3: Testes**

Run: `go test ./pkg/config/... ./...`
Expected: tudo `ok`/sem falhas.

- [ ] **Step 4: Commit**

```bash
git add config/config.yaml
git commit -m "Translate config.yaml comments to English"
```

---

## Task 3: golangci-lint — config e correções

**Files:**
- Create: `.golangci.yml`
- Modify: arquivos Go que o lint apontar (correções mínimas, sem mudar comportamento)

**Interfaces:**
- Produces: `.golangci.yml` que o job `lint` do CI (Task 4) consome.

- [ ] **Step 1: Criar .golangci.yml**

```yaml
version: "2"
```

(Linters default do golangci-lint v2: errcheck, govet, ineffassign, staticcheck, unused.)

- [ ] **Step 2: Rodar o lint localmente**

Run: `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...`
Expected: lista de findings (possivelmente vazia).

- [ ] **Step 3: Corrigir cada finding**

Regras de correção (diffs mínimos):
- `errcheck` em erro que deve ser propagado: retornar/logar o erro conforme o padrão do arquivo.
- `errcheck` em erro deliberadamente ignorado (ex.: `defer f.Close()` em leitura, writes de resposta HTTP): trocar por `_ = expr` sem comentário.
- `unused`/`ineffassign`: remover o código morto.
- `staticcheck`/`govet`: aplicar a sugestão do próprio linter.
- Em caso de finding cuja correção mudaria comportamento de forma não trivial: adicionar exceção pontual em `.golangci.yml` (`linters.exclusions.rules`) com o path exato, em vez de mudar a lógica.

- [ ] **Step 4: Re-rodar lint e testes até zerar**

Run: `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./... && go test ./...`
Expected: lint sem findings; testes ok.

- [ ] **Step 5: Commit**

```bash
git add .golangci.yml <arquivos corrigidos>
git commit -m "Add golangci-lint config and fix lint findings"
```

---

## Task 4: CI com pull_request + lint, Dockerfile pinado

**Files:**
- Modify: `.github/workflows/ci.yml` (substituição integral)
- Modify: `Dockerfile:1` e `Dockerfile:11` (pins)

**Interfaces:**
- Consumes: `.golangci.yml` (Task 3). Produces: CI sem publicação de imagem (a publicação passa a ser exclusiva da release, Task 5).

- [ ] **Step 1: Substituir .github/workflows/ci.yml**

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:

concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - run: go build ./...
      - run: go test ./...

      - name: Validate go.mod and go.sum are tidy
        run: |
          go mod tidy
          git diff --exit-code go.mod go.sum

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - uses: golangci/golangci-lint-action@v8
        with:
          version: latest

  docker:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Build image (validation only, no push)
        run: docker build -t gaardrail:ci .
```

- [ ] **Step 2: Pinar bases do Dockerfile**

Em `Dockerfile`, trocar:
- linha 1: `FROM golang:latest AS builder` → `FROM golang:1.25 AS builder`
- linha 11: `FROM alpine:latest` → `FROM alpine:3.22`

- [ ] **Step 3: Validar build da imagem local**

Run: `docker build -t gaardrail:pin-test . && docker image rm gaardrail:pin-test`
Expected: build completa sem erro. (Se `alpine:3.22` não existir mais no registry, usar a mais recente estável `alpine:3.x` disponível e anotar no commit.)

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/ci.yml Dockerfile
git commit -m "Run CI on pull requests with lint; stop publishing images from CI; pin Docker base images"
```

---

## Task 5: GoReleaser (binários + imagem versionada) e workflow de release

**Files:**
- Create: `.goreleaser.yaml`
- Create: `Dockerfile.goreleaser`
- Create: `.github/workflows/release.yml`

**Interfaces:**
- Consumes: secrets `DOCKERHUB_USERNAME`/`DOCKERHUB_TOKEN` (já existem no repo, eram usados pelo antigo job docker do CI).
- Produces: pipeline de release que a Task 9 dispara com a tag `v0.1.0`.

- [ ] **Step 1: Criar .goreleaser.yaml**

```yaml
version: 2

project_name: gaardrail

before:
  hooks:
    - go mod tidy

builds:
  - main: ./cmd/api
    binary: gaardrail
    env:
      - CGO_ENABLED=0
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w

archives:
  - formats: [tar.gz]
    format_overrides:
      - goos: windows
        formats: [zip]
    files:
      - LICENSE
      - README.md

dockers:
  - goos: linux
    goarch: amd64
    image_templates:
      - "adrianozdp/gaardrail:{{ .Tag }}"
      - "adrianozdp/gaardrail:latest"
    dockerfile: Dockerfile.goreleaser
    build_flag_templates:
      - "--platform=linux/amd64"

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs"
      - "^ci"
      - "^test"
```

- [ ] **Step 2: Criar Dockerfile.goreleaser**

(Contexto do GoReleaser contém só o binário pronto — sem rebuild.)

```dockerfile
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY gaardrail .

EXPOSE 8080

ENTRYPOINT ["./gaardrail"]
```

- [ ] **Step 3: Criar .github/workflows/release.yml**

```yaml
name: Release

on:
  push:
    tags: ["v*"]

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Log in to Docker Hub
        uses: docker/login-action@v3
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}

      - uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 4: Validar config e snapshot local**

```bash
go run github.com/goreleaser/goreleaser/v2@latest check
go run github.com/goreleaser/goreleaser/v2@latest release --snapshot --clean
```

Expected: `check` sem erros; snapshot gera `dist/` com binários para os 6 targets e a imagem local `adrianozdp/gaardrail:<snapshot>`. Se o daemon Docker não estiver disponível, rodar com `--skip docker` e anotar que a imagem será validada na release real (Task 9).

- [ ] **Step 5: Limpar dist e commitar**

```bash
rm -rf dist
echo "dist/" >> .gitignore
git add .goreleaser.yaml Dockerfile.goreleaser .github/workflows/release.yml .gitignore
git commit -m "Add GoReleaser release pipeline (binaries + versioned Docker image)"
```

---

## Task 6: Docs de comunidade e templates

**Files:**
- Create: `CONTRIBUTING.md`
- Create: `SECURITY.md`
- Create: `CODE_OF_CONDUCT.md`
- Create: `.github/ISSUE_TEMPLATE/bug_report.yml`
- Create: `.github/ISSUE_TEMPLATE/feature_request.yml`
- Create: `.github/PULL_REQUEST_TEMPLATE.md`

**Interfaces:**
- Consumes: links `CONTRIBUTING.md`/`SECURITY.md` já referenciados no README (Task 1).

- [ ] **Step 1: Criar CONTRIBUTING.md**

````markdown
# Contributing to gaardrail

Thanks for your interest in contributing!

## Development setup

Requirements: Go (version from `go.mod`); Docker only for the optional local Kafka and image builds.

```bash
git clone https://github.com/adrianozp/gaardrail
cd gaardrail
go build ./...
go test ./...
```

Run the app with `make run` (reads `config/config.yaml`). The default config uses `queue.protocol: constant`, which needs no broker. For a real queue: `make kafka/up && make kafka/setup`, then set `queue.protocol: kafka`.

## Linting

CI runs [golangci-lint](https://golangci-lint.run). Locally:

```bash
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...
```

## Pull requests

1. Fork and branch from `main`.
2. Keep changes focused; add or update tests for behavior changes.
3. Make sure `go build ./...`, `go test ./...` and the linter pass.
4. Open the PR describing the motivation and the change.

## Commit style

Short imperative subject lines ("Add X", "Fix Y"). Reference issues when applicable.

## Bugs and features

Use the issue templates. For security issues see [SECURITY.md](SECURITY.md) — do not open a public issue.

## License

By contributing you agree that your contributions are licensed under the [MPL-2.0](LICENSE).
````

- [ ] **Step 2: Criar SECURITY.md**

````markdown
# Security Policy

## Reporting a vulnerability

Please do not open public issues for security vulnerabilities.

Use GitHub's private vulnerability reporting (repository **Security** tab → **Report a vulnerability**) or email adrianozdp@gmail.com. Include a description, steps to reproduce and the affected version. You should get a response within a week.

## Supported versions

gaardrail is pre-1.0: only the latest release receives security fixes.
````

- [ ] **Step 3: Criar CODE_OF_CONDUCT.md (Contributor Covenant 2.1)**

```bash
curl -fsSL -o CODE_OF_CONDUCT.md https://www.contributor-covenant.org/version/2/1/code_of_conduct/code_of_conduct.md
sed -i 's/\[INSERT CONTACT METHOD\]/adrianozdp@gmail.com/' CODE_OF_CONDUCT.md
grep -c 'adrianozdp@gmail.com' CODE_OF_CONDUCT.md
```

Expected: download ok e `grep` imprime `1`. (Se o download falhar, copiar o texto 2.1 de https://github.com/EthicalSource/contributor_covenant/blob/release/content/version/2/1/code_of_conduct.md e aplicar o mesmo sed.)

- [ ] **Step 4: Criar .github/ISSUE_TEMPLATE/bug_report.yml**

```yaml
name: Bug report
description: Report a problem with gaardrail
labels: [bug]
body:
  - type: textarea
    id: what-happened
    attributes:
      label: What happened?
      description: What did you expect to happen instead?
    validations:
      required: true
  - type: input
    id: version
    attributes:
      label: gaardrail version
      placeholder: v0.1.0
    validations:
      required: true
  - type: textarea
    id: config
    attributes:
      label: Relevant config
      description: Relevant parts of your config.yaml (redact credentials).
      render: yaml
  - type: textarea
    id: logs
    attributes:
      label: Logs
      render: shell
```

- [ ] **Step 5: Criar .github/ISSUE_TEMPLATE/feature_request.yml**

```yaml
name: Feature request
description: Suggest an idea for gaardrail
labels: [enhancement]
body:
  - type: textarea
    id: problem
    attributes:
      label: What problem would this solve?
    validations:
      required: true
  - type: textarea
    id: solution
    attributes:
      label: Proposed solution
    validations:
      required: true
  - type: textarea
    id: alternatives
    attributes:
      label: Alternatives considered
```

- [ ] **Step 6: Criar .github/PULL_REQUEST_TEMPLATE.md**

```markdown
## What does this PR do?

## Why?

## Checklist

- [ ] `go build ./...` and `go test ./...` pass
- [ ] Linter passes
- [ ] Tests added/updated for behavior changes
```

- [ ] **Step 7: Commit**

```bash
git add CONTRIBUTING.md SECURITY.md CODE_OF_CONDUCT.md .github/ISSUE_TEMPLATE .github/PULL_REQUEST_TEMPLATE.md
git commit -m "Add contributing guide, security policy, code of conduct and issue/PR templates"
```

---

## Task 7: Push, CI verde e metadados do GitHub

**Files:** nenhum (operações git/gh)

**Interfaces:**
- Consumes: todos os commits das tasks 1-6.
- Produces: `main` publicado com CI verde — pré-requisito da release (Task 9).

- [ ] **Step 1: Push do main**

Run: `git push origin main`
Expected: push aceito.

- [ ] **Step 2: Acompanhar o CI**

Run: `gh run watch $(gh run list --workflow=CI --limit 1 --json databaseId -q '.[0].databaseId') --exit-status`
Expected: exit 0 (jobs test, lint e docker verdes). Se falhar: corrigir, commitar, push, repetir.

- [ ] **Step 3: Topics**

```bash
gh repo edit adrianozp/gaardrail \
  --add-topic golang --add-topic pid-controller --add-topic back-pressure \
  --add-topic control-theory --add-topic kafka --add-topic message-queue \
  --add-topic prometheus --add-topic rate-limiting
```

Expected: comando sai 0; `gh repo view --json repositoryTopics` lista os 8 topics.

- [ ] **Step 4: Habilitar private vulnerability reporting**

Run: `gh api -X PUT repos/adrianozp/gaardrail/private-vulnerability-reporting`
Expected: HTTP 204 (sem corpo).

---

## Task 8: Publicar o gaardrail-flood-test

**Files:** nenhum neste repo (opera em `../gaardrail-flood-test`)

**Interfaces:**
- Produces: o repo público que o link "Validation" do README (Task 1) aponta.

- [ ] **Step 1: Conferir o README do rig**

Run: `grep -n "github.com/adrianozp/gaardrail" ../gaardrail-flood-test/README.md`
Expected: link para o repo core presente. Se apontar para caminho local ou URL errada, corrigir e commitar lá (`git -C ../gaardrail-flood-test commit -am "Fix core repo link"`).

- [ ] **Step 2: Criar o repo público e pushar**

```bash
cd ../gaardrail-flood-test
gh repo create adrianozp/gaardrail-flood-test --public --source . --push \
  --description "PFC (thesis) validation rig for gaardrail — closed-loop control experiments, system identification, Grafana dashboards"
```

Expected: repo criado e branch `main` pushada.

- [ ] **Step 3: Topics do rig**

```bash
gh repo edit adrianozp/gaardrail-flood-test \
  --add-topic control-theory --add-topic system-identification \
  --add-topic grafana --add-topic prometheus --add-topic golang
```

- [ ] **Step 4: Verificar links cruzados**

Run: `gh repo view adrianozp/gaardrail-flood-test --json url -q .url && curl -fsSL -o /dev/null -w "%{http_code}\n" https://github.com/adrianozp/gaardrail-flood-test`
Expected: URL impressa e HTTP 200.

---

## Task 9: Release v0.1.0 e verificação final

**Files:** nenhum (tag + verificações)

**Interfaces:**
- Consumes: pipeline de release (Task 5), main verde (Task 7).

- [ ] **Step 1: Criar e pushar a tag**

```bash
cd /home/adrianozdp/workspace/ufsc/pfc/gaardrail
git tag -a v0.1.0 -m "First public release"
git push origin v0.1.0
```

- [ ] **Step 2: Acompanhar o workflow de release**

Run: `gh run watch $(gh run list --workflow=Release --limit 1 --json databaseId -q '.[0].databaseId') --exit-status`
Expected: exit 0.

- [ ] **Step 3: Verificar a release no GitHub**

Run: `gh release view v0.1.0 --json assets -q '.assets[].name'`
Expected: 6 archives (linux/darwin/windows × amd64/arm64) + checksums.

- [ ] **Step 4: Verificar a imagem Docker**

```bash
docker pull adrianozdp/gaardrail:v0.1.0
docker pull adrianozdp/gaardrail:latest
docker image inspect adrianozdp/gaardrail:v0.1.0 -f '{{.Config.Entrypoint}} {{.Config.ExposedPorts}}'
```

Expected: pulls ok; entrypoint `[./gaardrail]`, porta `8080/tcp`.

- [ ] **Step 5: Smoke test do binário da release**

```bash
cd $(mktemp -d)
gh release download v0.1.0 -R adrianozp/gaardrail -p 'gaardrail_*linux_amd64.tar.gz'
tar xzf gaardrail_*linux_amd64.tar.gz
./gaardrail & sleep 2; curl -fsS http://localhost:8080/ping; kill %1
```

Expected: `/ping` responde 200. (O binário sobe mesmo sem MySQL/Prometheus acessíveis; se o processo abortar por dependência externa indisponível, considerar o teste satisfeito com o processo tendo iniciado e logado a tentativa de conexão — anotar o comportamento observado.)

- [ ] **Step 6: Badge check**

Run: `curl -fsS https://img.shields.io/github/v/release/adrianozp/gaardrail | grep -o 'v0.1.0' | head -1`
Expected: `v0.1.0`.
