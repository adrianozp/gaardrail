# Mover disturbance e painel web para o rig — Design

**Objetivo:** o core `gaardrail` vira uma API headless (direção futura: operator Kubernetes) — o painel web e o gerador de perturbação saem do core. O painel passa a viver no `gaardrail-flood-test` como exemplo funcional servido por um script Python mínimo; a função de perturbação fica com os scripts que o rig já tem.

**Contexto:** o front embutido de hoje se resume a três papéis do binário: (1) servir `web/index.html` + favicons em `/`; (2) ser a origem das chamadas de API do painel (fetch relativos para `/pid`, `/controller/type`, `/queue/query`, `/queue/type`, `/messages/flood`, `/disturbance`); (3) substituir `{{GRAFANA_URL}}` no HTML — um `<iframe>` kiosk (index.html:349). O Grafana em si sempre foi o container do compose do rig. O "sobe tudo" da VPS é o `docker-compose.vps.yml` do rig e permanece intacto.

**Decisões do usuário:** sem nginx, sem micro-serviço, sem mudanças nos composes; painel = exemplo + `serve.py` (~30-60 linhas, stdlib) com URL do gaardrail e do Grafana parametrizáveis; disturbance removido de vez (o rig já tem `disturb.sh`/`disturb-vps.sh` equivalentes) e o card correspondente sai da cópia do painel; release v0.2.0 na sequência.

## 1. Core (gaardrail) — remoções

- `web/` inteiro (index.html, web.go, favicon/).
- Em `internal/httpserver/httpserver.go`: rota `GET /`, servir de favicons, injeção de `{{GRAFANA_URL}}` e o import de `web`.
- Seção `grafana` do config: struct em `pkg/config`, entrada no `config/config.yaml` e qualquer persistência associada (verificar `pkg/config` persist).
- Disturbance completo: `app/disturbance/`, `app/usecases/setdisturbance/`, `app/handlers/disturbance/`, `cmd/api/modules/disturbance.go` e o registro do módulo no boot (`cmd/api`).
- `app/repositories/targets/targets.go`: `NewSQLClient` permanece (o target SQL o usa); atualizar o comentário que menciona o disturbance.
- README: remover a frase do painel no Quickstart; nota curta apontando que o painel de exemplo e os scripts de perturbação vivem no repo do rig; referência de configuração sem `grafana`.
- `openapi.yaml`: sem mudança (nunca documentou `/` nem `/disturbance`).
- Gate: `go build ./... && go test ./...` e lint zerado; nenhum outro endpoint muda.

## 2. Rig — `panel/`

- `panel/index.html` + `panel/favicon/` copiados do estado atual do core, com o card de disturbance e seu JS removidos (nenhum `fetch('/disturbance')` restante). Nenhuma outra mudança no HTML; o placeholder `{{GRAFANA_URL}}` permanece.
- `panel/serve.py` (stdlib apenas, sem dependências):
  - serve o estático (index + favicons) na porta `--port` (default `8082`);
  - substitui os três placeholders que o `httpserver` do core injetava (httpserver.go:31-40): `{{GRAFANA_URL}}` (`--grafana`, default `http://localhost:3000/d/flood-test/flood-test?kiosk`), `{{POLL_INTERVAL_MS}}` (`--poll-interval-ms`, default `5000`) e `{{POLL_QUERY}}` (`--poll-query`, default a PromQL padrão do rig, HTML-escapada como no Go);
  - repassa `/pid`, `/controller/*`, `/queue/*`, `/messages/*`, `/ping` para `--gaardrail` (default `http://localhost:8080`), preservando método, corpo, query string e status.
- README do rig: seção "Panel" curta — uso local (`python3 panel/serve.py`) e uso contra a VPS (`--gaardrail http://<vps>:8080` ou túnel SSH).
- Zero mudanças em `docker-compose-flood.yml` e `docker-compose.vps.yml`.

## 3. Versionamento e publicação

- Core: commit(s) da remoção, push, CI verde, tag `v0.2.0` (release notes citam a remoção do painel/disturbance e para onde foram).
- Rig: commits de `panel/` + README, push.

## Verificação (gate de conclusão)

1. Core: build/test/lint verdes; num run local, `GET /` e `GET /disturbance` retornam 404 e `GET /ping`/`GET /pid` respondem normalmente.
2. Rig: `python3 panel/serve.py` contra um gaardrail local — painel carrega em `:8082`, favicons ok, iframe com a URL do Grafana injetada, um `PATCH /pid` disparado pelo painel chega no gaardrail; `grep` confirma ausência de `/disturbance` no HTML copiado.
3. Release v0.2.0 publicada (binários + `docker pull adrianozdp/gaardrail:v0.2.0`).
4. READMEs dos dois repos consistentes entre si.

## Fora de escopo

Operator Kubernetes (direção futura registrada, nada implementado agora); painel hospedado permanente na VPS (se desejado depois, é um container estático de ~10 linhas no compose de lá — sem retrabalho); CORS no core; TLS/auth no `serve.py`; mudanças no JS do painel além da remoção do card de disturbance; CI para o repo do rig.
