# Preparação do gaardrail para open source — Design

**Objetivo:** deixar o `gaardrail` com cara e mecânica de produto open source para usuários reais: README correto pós-split, releases versionadas (binários + imagem Docker), CI que cobre pull requests, docs de comunidade e o rig de validação publicado.

**Contexto:** o repo já está público em `github.com/adrianozp/gaardrail`, com LICENSE MPL-2.0 (mantida por decisão deliberada), CI verde (build + test + `go mod tidy` check + push de imagem `latest`/SHA para Docker Hub) e boa descrição no GitHub. O split do ferramental de PFC para `gaardrail-flood-test` (local, sem remote) foi concluído em 2026-08-12.

**Decisões do usuário:** foco em quem vai rodar em produção; MPL-2.0 mantida; distribuição via GoReleaser (binários + Docker versionado); `gaardrail-flood-test` será publicado junto; abordagem A (pacote completo, com lint).

## 1. README (reescrita, em inglês)

Substituir o README atual, que ficou incorreto após o split (instrui a rodar o flood-test que não vive mais aqui e diz que o compose sobe MySQL/Prometheus/Grafana quando ele só tem Kafka).

Estrutura:

- Título + badges: CI (workflow status), última release, licença MPL-2.0.
- Pitch: back-pressure controller para consumers de fila; PID discreto regulando vazão de consumo para manter um setpoint de utilização de recurso (ex.: CPU do banco em 50%).
- **How it works** — aproveitar e enxugar o material atual: orchestrator, tipos de controlador (`pid`, `pidff`, `smith`, `step`, `autopid`), filtros de setpoint/medição, interfaces (queue, target, metrics poller).
- **Quickstart** — honesto sobre pré-requisitos: um alvo SQL (MySQL) e uma fonte de métricas (Prometheus API ou compatível). Duas rotas: `docker run adrianozdp/gaardrail` com config montado, ou `go run ./cmd/api`. Nota: `queue.protocol: constant` dispensa Kafka; o `docker-compose.yml` local sobe só o Kafka (opcional).
- **Configuration** — referência das seções do `config.yaml` (http, queue, kafka, target, metrics_poller, controller, orchestrator, pid/smith, grafana), com defaults e unidades.
- **HTTP API** — endpoints principais + link para `openapi.yaml`; exemplo de `PATCH` de parâmetros em runtime.
- **Web panel** — o painel em `web/` e o que ele controla.
- **Validation** — link para `adrianozp/gaardrail-flood-test` como evidência experimental (identificação de sistema, campanhas de regime, comparação PI/PIFF/Smith); menção de que nasceu como PFC (TCC) na UFSC.
- **Contributing / License** — links para CONTRIBUTING.md e LICENSE.

Fora: instruções operacionais do rig (moram no repo do rig); screenshots (podem entrar depois).

## 2. Releases — GoReleaser + v0.1.0

- `.goreleaser.yaml`:
  - build de `./cmd/api`, binário `gaardrail`, `CGO_ENABLED=0`, `-s -w`;
  - targets: linux/darwin/windows × amd64/arm64;
  - archives tar.gz (zip no Windows) com LICENSE e README;
  - imagem Docker **linux/amd64** `adrianozdp/gaardrail:{{ .Version }}` + `:latest`, usando um `Dockerfile.goreleaser` novo (runtime alpine pinado que copia o binário pronto — não rebuilda);
  - changelog automático agrupado (features/fixes/others) nas GitHub Releases. Sem arquivo CHANGELOG.md manual: o changelog vive nas Releases.
- `.github/workflows/release.yml`: dispara em tag `v*`; checkout com `fetch-depth: 0`, setup-go por `go-version-file`, login no Docker Hub (secrets `DOCKERHUB_USERNAME`/`DOCKERHUB_TOKEN` já existentes), `goreleaser/goreleaser-action` com `release --clean`; `permissions: contents: write`.
- Semântica de tags: `latest` passa a significar "última release estável", não "último push no main".
- Encerramento: tag `v0.1.0` publicada com binários + imagem, após verificação com `goreleaser release --snapshot --clean` local.
- Imagem arm64 e version stamping no binário ficam para depois (fora de escopo desta rodada).

## 3. CI

Editar `.github/workflows/ci.yml`:

- Triggers: `push` em `main` + `pull_request` (PR de fork passa a ser testado); `concurrency` com cancel-in-progress.
- `setup-go` com `go-version-file: go.mod` (hoje está hardcoded "1.25").
- Novo step de lint: `golangci-lint-action` com `.golangci.yml` enxuto (linters default + timeout); corrigir no código o que o lint apontar.
- O job `docker` deixa de publicar: vira um step de `docker build` sem push (valida o Dockerfile em PRs); publicação de imagem passa a ser exclusiva do release.
- `Dockerfile`: pinar bases (`golang:1.25` e `alpine:3.22` em vez de `latest`).

## 4. Docs de comunidade

- `CONTRIBUTING.md`: setup (Go pelo `go.mod`, `make run`, compose do Kafka), como rodar testes e lint, processo de PR (branch, CI verde, descrição), estilo de commit simples. Curto.
- `SECURITY.md`: reporte privado via GitHub Security Advisories (habilitar private vulnerability reporting no repo); sem lista de versões suportadas enquanto só houver v0.x.
- `CODE_OF_CONDUCT.md`: Contributor Covenant 2.1, contato adrianozdp@gmail.com.
- `.github/ISSUE_TEMPLATE/bug_report.yml` e `feature_request.yml` (issue forms) + `.github/PULL_REQUEST_TEMPLATE.md`. Todos mínimos.

## 5. Metadados do GitHub

- Topics via `gh repo edit --add-topic`: `golang`, `pid-controller`, `back-pressure`, `control-theory`, `kafka`, `message-queue`, `prometheus`, `rate-limiting`.
- Habilitar private vulnerability reporting (API do GitHub).
- `docs/superpowers/` permanece no repo como histórico de engenharia.
- Branch protection em `main`: sugerido, configuração manual do usuário (fora do escopo automatizado).

## 6. Publicação do gaardrail-flood-test

- Criar `adrianozp/gaardrail-flood-test` público a partir do repo local (`gh repo create --public --source . --push`).
- Descrição ("PFC validation rig for gaardrail…") + topics básicos.
- Conferir que o README do rig linka o core corretamente (ele já existe desde jul/22) e que o link de "Validation" no README do core resolve.
- Não reescrever docs do rig nesta rodada.

## 7. Polimento do config.yaml

- Traduzir todos os comentários para inglês. **Valores intactos** — nenhuma mudança de comportamento.
- DSN `root:root@tcp(localhost:3306)/gaardrail` permanece como exemplo local explícito.

## Verificação (gate de conclusão)

1. `go build ./... && go test ./...` verdes; lint zerado.
2. `docker build` local do Dockerfile pinado funciona.
3. `goreleaser release --snapshot --clean` local gera binários e imagem sem erro.
4. CI verde no push do main; opcionalmente abrir um PR de teste para ver o trigger de `pull_request` (e fechá-lo).
5. Tag `v0.1.0` → release no GitHub com binários + `docker pull adrianozdp/gaardrail:v0.1.0` funcionando.
6. Repo do rig acessível publicamente; links cruzados corretos.

## Fora de escopo

Screenshots do painel, imagem Docker arm64, version stamping no binário, CHANGELOG.md manual, branch protection automatizada, reescrita dos docs do rig, site/pkg.go.dev.
