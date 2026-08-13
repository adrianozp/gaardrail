# Mover disturbance e painel web para o rig — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Tornar o `gaardrail` uma API headless — removendo painel web embutido, config `grafana` e o gerador de disturbance — e mover o painel para o `gaardrail-flood-test` como exemplo funcional servido por `panel/serve.py` (stdlib), fechando com a release `v0.2.0`.

**Architecture:** 4 tasks: (1) painel no rig (cópia do HTML sem o card de disturbance + serve.py + README do rig); (2) remoção do disturbance no core; (3) remoção do web/panel + config grafana + README no core; (4) publicação (push dos dois repos, CI, tag v0.2.0, verificação da release). Spec: `docs/superpowers/specs/2026-08-12-mover-disturbance-e-web-para-rig-design.md`.

**Tech Stack:** Go, Python 3 stdlib, GitHub Actions, GoReleaser, gh CLI.

## Global Constraints

- Commits autorados APENAS como o usuário (config git local `adrianozp <adrianozdp@gmail.com>`). **NUNCA adicionar trailer `Co-Authored-By` ou similar.**
- Arquivos públicos em inglês.
- Core: nenhuma mudança de comportamento além das remoções especificadas; `go build ./... && go test ./...` e lint zerados ao fim de cada task de core.
- Rig: zero mudanças nos composes; `serve.py` só stdlib.
- Task 1 roda ANTES das tasks 2-3 (copia `web/` do working tree do core enquanto existe).
- Diretórios: core `/home/adrianozdp/workspace/ufsc/pfc/gaardrail`; rig `/home/adrianozdp/workspace/ufsc/pfc/gaardrail-flood-test`. Ambos em `main` (intencional).

---

## Task 1: Painel no rig (`panel/` + serve.py + README)

**Files:**
- Create: `../gaardrail-flood-test/panel/index.html` (cópia de `web/index.html` do core, sem o card de disturbance)
- Create: `../gaardrail-flood-test/panel/favicon/*` (cópia de `web/favicon/` do core)
- Create: `../gaardrail-flood-test/panel/serve.py`
- Modify: `../gaardrail-flood-test/README.md` (nova seção `## Panel` após "Running locally")

**Interfaces:**
- Consumes: `web/index.html` e `web/favicon/` do working tree do core (existem até a Task 3).
- Produces: painel de exemplo que o README do core (Task 3) referencia.

- [ ] **Step 1: Copiar os arquivos do core**

```bash
cd /home/adrianozdp/workspace/ufsc/pfc/gaardrail-flood-test
mkdir -p panel
cp ../gaardrail/web/index.html panel/index.html
cp -r ../gaardrail/web/favicon panel/favicon
```

- [ ] **Step 2: Remover o card de disturbance do panel/index.html**

Três remoções (localizar por grep, números de linha são do arquivo original):
1. O elemento de card inteiro que contém os botões `onclick="applyDisturbance()"` e `onclick="stopDisturbance()"` (~linhas 341-342) — remover do tag de abertura do card (div que engloba título/inputs/botões da perturbação) até seu fechamento.
2. O bloco JS de `// ---- Disturbance ----` (~linha 715) até o fim da função `stopDisturbance()` (~linha 766), inclusive `loadDisturbance`, `postDisturbance`, `applyDisturbance`.
3. A chamada `loadDisturbance();` na inicialização (~linha 832).

Verificação: `grep -ci disturb panel/index.html` → `0`; `grep -c "applyDisturbance\|postDisturbance\|loadDisturbance" panel/index.html` → `0`. Os placeholders `{{GRAFANA_URL}}`, `{{POLL_INTERVAL_MS}}`, `{{POLL_QUERY}}` permanecem intactos (`grep -c '{{' panel/index.html` ≥ 3).

- [ ] **Step 3: Criar panel/serve.py**

```python
#!/usr/bin/env python3
"""Serve the gaardrail example panel with same-origin API proxying.

The panel is a single static HTML page that calls the gaardrail API via
relative paths and embeds a Grafana dashboard. This helper serves the page,
fills in the template placeholders and forwards API calls to a running
gaardrail instance, so no CORS or extra infrastructure is needed.

Usage:
  python3 serve.py                       # gaardrail at http://localhost:8080
  python3 serve.py --gaardrail http://my-vps:8080
"""
import argparse
import html
import http.server
import urllib.error
import urllib.request
from pathlib import Path

API_PREFIXES = ("/pid", "/controller", "/queue", "/messages", "/ping")
PANEL_DIR = Path(__file__).resolve().parent

DEFAULT_GRAFANA = "http://localhost:3000/d/flood-test/flood-test?kiosk"
DEFAULT_QUERY = (
    'irate(container_cpu_usage_seconds_total'
    '{container_label_com_docker_compose_service="mysql"}[15s])*100'
)


def build_index(args):
    text = (PANEL_DIR / "index.html").read_text(encoding="utf-8")
    text = text.replace("{{GRAFANA_URL}}", args.grafana)
    text = text.replace("{{POLL_INTERVAL_MS}}", str(args.poll_interval_ms))
    text = text.replace("{{POLL_QUERY}}", html.escape(args.poll_query))
    return text.encode("utf-8")


class Handler(http.server.SimpleHTTPRequestHandler):
    index_html = b""
    upstream = ""

    def __init__(self, *a, **kw):
        super().__init__(*a, directory=str(PANEL_DIR), **kw)

    def _serve_index(self):
        self.send_response(200)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.send_header("Content-Length", str(len(self.index_html)))
        self.end_headers()
        self.wfile.write(self.index_html)

    def _proxy(self):
        length = int(self.headers.get("Content-Length") or 0)
        body = self.rfile.read(length) if length else None
        req = urllib.request.Request(
            self.upstream + self.path, data=body, method=self.command)
        ctype = self.headers.get("Content-Type")
        if ctype:
            req.add_header("Content-Type", ctype)
        try:
            with urllib.request.urlopen(req) as resp:
                payload = resp.read()
                self.send_response(resp.status)
                self.send_header(
                    "Content-Type",
                    resp.headers.get("Content-Type", "application/json"))
                self.send_header("Content-Length", str(len(payload)))
                self.end_headers()
                self.wfile.write(payload)
        except urllib.error.HTTPError as e:
            payload = e.read()
            self.send_response(e.code)
            self.send_header("Content-Length", str(len(payload)))
            self.end_headers()
            self.wfile.write(payload)
        except OSError as e:
            self.send_error(502, f"gaardrail unreachable: {e}")

    def do_GET(self):
        if self.path == "/" or self.path.startswith("/?"):
            return self._serve_index()
        if self.path.startswith(API_PREFIXES):
            return self._proxy()
        return super().do_GET()

    def do_POST(self):
        if self.path.startswith(API_PREFIXES):
            return self._proxy()
        self.send_error(404)

    do_PATCH = do_POST
    do_PUT = do_POST


def main():
    p = argparse.ArgumentParser(
        description=__doc__,
        formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--port", type=int, default=8082)
    p.add_argument("--gaardrail", default="http://localhost:8080",
                   help="base URL of the gaardrail API")
    p.add_argument("--grafana", default=DEFAULT_GRAFANA,
                   help="Grafana dashboard URL embedded in the panel")
    p.add_argument("--poll-interval-ms", type=int, default=5000,
                   help="UI polling interval shown/used by the panel")
    p.add_argument("--poll-query", default=DEFAULT_QUERY,
                   help="PromQL of the process variable, shown in the panel")
    args = p.parse_args()
    Handler.index_html = build_index(args)
    Handler.upstream = args.gaardrail.rstrip("/")
    server = http.server.ThreadingHTTPServer(("", args.port), Handler)
    print(f"panel: http://localhost:{args.port} -> gaardrail {Handler.upstream}")
    server.serve_forever()


if __name__ == "__main__":
    main()
```

- [ ] **Step 4: Seção Panel no README do rig**

Inserir após a seção "Running locally" de `../gaardrail-flood-test/README.md`:

````markdown

## Panel

`panel/` holds the example web panel for gaardrail — live controller tuning,
runtime queue switching and an embedded Grafana dashboard. gaardrail itself
is headless; the panel is served by a small stdlib helper that also proxies
the API calls (no CORS needed):

```
python3 panel/serve.py                            # gaardrail at localhost:8080
python3 panel/serve.py --gaardrail http://<vps>:8080
```

Open http://localhost:8082. See `python3 panel/serve.py --help` for the
Grafana URL and polling options.
````

- [ ] **Step 5: Smoke test do serve.py (sem gaardrail — upstream mock)**

```bash
cd /home/adrianozdp/workspace/ufsc/pfc/gaardrail-flood-test
python3 -m http.server 18080 --directory /tmp >/dev/null 2>&1 &
MOCK=$!
python3 panel/serve.py --port 18082 --gaardrail http://localhost:18080 >/dev/null 2>&1 &
PANEL=$!
sleep 1
curl -fsS localhost:18082/ | grep -c "localhost:3000/d/flood-test"   # >=1: GRAFANA_URL injetada
curl -fsS localhost:18082/ | grep -c '{{'                            # 0: sem placeholder sobrando
curl -fsS -o /dev/null -w '%{http_code}\n' localhost:18082/favicon/  # 200/301: estático ok
curl -s -o /dev/null -w '%{http_code}\n' localhost:18082/pid         # 404 vindo do mock: proxy encaminhou
kill $MOCK $PANEL
```

Expected: os quatro checks conforme os comentários (o `404` do último vem do mock, provando o encaminhamento).

- [ ] **Step 6: Commit no rig**

```bash
cd /home/adrianozdp/workspace/ufsc/pfc/gaardrail-flood-test
git add panel README.md
git commit -m "Add example web panel served by a stdlib proxy helper"
```

---

## Task 2: Core — remover o gerador de disturbance

**Files:**
- Delete: `app/disturbance/`, `app/usecases/setdisturbance/`, `app/handlers/disturbance/`, `cmd/api/modules/disturbance.go`
- Modify: `cmd/api/options/options.go:34-37` (remover as 4 linhas `modules.Disturbance*()`)
- Modify: `app/repositories/targets/targets.go` (comentário de `NewSQLClient`)

**Interfaces:**
- Consumes: nada. Produces: árvore sem `/disturbance` que a Task 3 continua enxugando.

- [ ] **Step 1: Remover código e wiring**

```bash
cd /home/adrianozdp/workspace/ufsc/pfc/gaardrail
git rm -r -q app/disturbance app/usecases/setdisturbance app/handlers/disturbance
git rm -q cmd/api/modules/disturbance.go
```

Em `cmd/api/options/options.go`, remover as linhas:
```go
		modules.DisturbanceFactories(),
		modules.DisturbanceInjections(),
		modules.DisturbanceEndpoints(),
		modules.DisturbanceLifecycle(),
```

Em `app/repositories/targets/targets.go`, trocar o comentário:
`// NewSQLClient builds the SQL client shared by the SQL target and the` / `// disturbance component. It returns nil when the target is not SQL.`
por:
`// NewSQLClient builds the SQL client used by the SQL target.` / `// It returns nil when the target is not SQL.`

- [ ] **Step 2: Build, testes e lint**

```bash
go build ./... && go test ./...
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...
```

Expected: verdes/zerado. Se o lint apontar código que ficou morto com a remoção (ex.: helper usado só pelo disturbance), remover esse código também — nada além disso.

- [ ] **Step 3: Confirmar que nenhuma referência sobrou**

Run: `grep -rn -i "disturbance" --include='*.go' app cmd internal pkg | grep -vi "rejection\|disturbances"`
Expected: sem saída (menções restantes só em comentários de teoria de controle, ex. "disturbance rejection").

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "Remove disturbance generator from the core (rig scripts cover it)"
```

---

## Task 3: Core — remover painel web, config grafana e atualizar README

**Files:**
- Delete: `web/`, `pkg/config/grafana.go`
- Modify: `internal/httpserver/httpserver.go` (remover `registerWeb` e imports associados)
- Modify: `pkg/config/config.go:26` (remover campo `Grafana`)
- Modify: `config/config.yaml` (remover seção `grafana`, 2 linhas)
- Modify: `README.md` (Quickstart)

**Interfaces:**
- Consumes: Task 1 já copiou o painel (a deleção aqui é segura). Produces: core headless final para a Task 4 publicar.

- [ ] **Step 1: Remover web/ e a rota**

```bash
cd /home/adrianozdp/workspace/ufsc/pfc/gaardrail
git rm -r -q web
git rm -q pkg/config/grafana.go
```

Em `internal/httpserver/httpserver.go`: remover a função `registerWeb` inteira (linhas 30-48), a chamada `registerWeb(router, cfg)` (linha 21) e os imports que ficarem sem uso (`bytes`, `htmlpkg "html"`, `io/fs`, `strconv`, `github.com/adrianozp/gaardrail/web`). A assinatura `New(cfg config.Config)` permanece (o import de `config` sai se ficar sem uso — verificar com o compilador).

Em `pkg/config/config.go`: remover a linha `Grafana Grafana \`mapstructure:"grafana"\``.

Em `config/config.yaml`: remover a seção:
```yaml
grafana:
  url: "http://localhost:3000/d/flood-test/flood-test?kiosk"
```

- [ ] **Step 2: README — Quickstart headless**

Substituir a frase (linha ~71):
`The built-in web panel is served at [http://localhost:8080](http://localhost:8080) — live tuning of every controller parameter, controller type switching and disturbance controls.`
por:
`gaardrail is headless — an example web panel (live tuning, controller switching, embedded Grafana) lives in the [gaardrail-flood-test](https://github.com/adrianozp/gaardrail-flood-test) repo under `panel/`, alongside disturbance-generation scripts for experiments.`

- [ ] **Step 3: Build, testes e lint**

```bash
go build ./... && go test ./...
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...
```

Expected: verdes/zerado. Se testes de `pkg/config` (persist) referenciarem grafana, ajustar o teste removendo a referência (sem mudar semântica do persist).

- [ ] **Step 4: Confirmar remoção completa**

Run: `grep -rn -i "grafana\|GRAFANA\|IndexHTML\|registerWeb" --include='*.go' . ; grep -n "grafana" config/config.yaml README.md`
Expected: sem saída em código e config; no README só a menção da seção Validation/painel do rig (a frase nova) — nenhuma outra.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "Remove embedded web panel and grafana config (panel moved to the rig)"
```

---

## Task 4: Publicar — push dos dois repos, CI, release v0.2.0

**Files:** nenhum (operações git/gh; inclui os commits locais pré-existentes de spec/plano no core)

**Interfaces:**
- Consumes: Tasks 1-3 commitadas.
- Produces: v0.2.0 no ar; rig atualizado.

- [ ] **Step 1: Push do core e CI verde**

```bash
cd /home/adrianozdp/workspace/ufsc/pfc/gaardrail
git push origin main
sleep 10
gh run watch $(gh run list --workflow=CI --limit 1 --json databaseId -q '.[0].databaseId') --exit-status
```

Expected: exit 0. Se falhar: corrigir minimamente, commitar (autoria do usuário, sem trailers), push, repetir (máx. 3 tentativas; depois BLOCKED).

- [ ] **Step 2: Push do rig**

```bash
cd /home/adrianozdp/workspace/ufsc/pfc/gaardrail-flood-test
git push origin main
```

- [ ] **Step 3: Tag e release v0.2.0**

```bash
cd /home/adrianozdp/workspace/ufsc/pfc/gaardrail
git tag -a v0.2.0 -m "Headless API: web panel and disturbance moved to the rig"
git push origin v0.2.0
gh run watch $(gh run list --workflow=Release --limit 1 --json databaseId -q '.[0].databaseId') --exit-status
```

Expected: exit 0.

- [ ] **Step 4: Verificar a release**

```bash
gh release view v0.2.0 --json assets -q '.assets[].name'
docker pull adrianozdp/gaardrail:v0.2.0 && docker pull adrianozdp/gaardrail:latest
docker image inspect adrianozdp/gaardrail:v0.2.0 -f '{{.Config.Entrypoint}}'
```

Expected: 6 archives + checksums; pulls ok; entrypoint `[./gaardrail]`.

- [ ] **Step 5: Confirmar 404 dos endpoints removidos na imagem nova**

```bash
docker run -d --rm --name gr-check -p 18085:8080 \
  -v /home/adrianozdp/workspace/ufsc/pfc/gaardrail/config/config.yaml:/app/config/config.yaml \
  adrianozdp/gaardrail:v0.2.0
sleep 3
curl -s -o /dev/null -w '/ -> %{http_code}\n' localhost:18085/
curl -s -o /dev/null -w '/disturbance -> %{http_code}\n' localhost:18085/disturbance
curl -s -o /dev/null -w '/ping -> %{http_code}\n' localhost:18085/ping
docker stop gr-check
```

Expected: `/ -> 404`, `/disturbance -> 404`, `/ping -> 200`. (Se o container abortar por dependência externa antes do curl, registrar os logs e considerar o check atendido se o processo registrou as rotas — mas com o config montado usando `queue: constant`, deve subir.)
