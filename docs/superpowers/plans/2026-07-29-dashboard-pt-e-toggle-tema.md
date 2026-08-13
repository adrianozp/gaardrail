# Dashboard em PT + Toggle Light/Dark — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Dashboard Grafana inteiramente em português (título "Supervisório Gaardrail") e alternância light/dark na web UI, com light como padrão e o iframe do Grafana acompanhando o tema.

**Architecture:** Duas mudanças independentes em dois arquivos. (1) `flood.json`: substituições de strings via Edit tool — apenas `title` do dashboard, `title` de painéis e `legendFormat`; nada estrutural. (2) `web/index.html`: paleta extraída para CSS custom properties (`:root` = light, `[data-theme="dark"]` = dark), botão de toggle no header, persistência em `localStorage`, script anti-flash no `<head>`, e sincronização do query param `theme` na URL do iframe do Grafana.

**Tech Stack:** JSON (Grafana dashboard schema), HTML/CSS/JS vanilla em arquivo único.

## Global Constraints

- Somente 2 arquivos mudam: `flood-test/grafana/dashboards/flood.json` e `web/index.html`. Os `.bak` não são tocados.
- **Não commitar** — o usuário commita manualmente (preferência do projeto). Cada task termina mostrando o diff.
- Termos técnicos permanecem em inglês: setpoint, drain (nomes de métrica), enqueue/dequeue, created→ack, InnoDB Buffer Pool, QPS, slow queries, full scans, working set, RSS, lag, kiosk.
- Tema padrão da web UI: **light**. Dark só via toggle, persistido em `localStorage.theme`.
- Nenhuma query, uid, threshold ou layout do dashboard muda. O uid `flood-test` fica intacto (o link em `pkg/config/grafana.go` resolve por uid).

---

### Task 1: Traduzir flood.json para português

**Files:**
- Modify: `flood-test/grafana/dashboards/flood.json`

**Interfaces:**
- Consumes: nada.
- Produces: nada consumido por outras tasks (Task 2/3 são em outro arquivo).

Contexto: o arquivo é um export do Grafana (~JSON indentado). NÃO reescrever o arquivo inteiro via script (mudaria formatação e sujaria o diff). Usar Edit tool com substituições exatas nas strings abaixo. Strings repetidas em vários painéis usam `replace_all`.

- [ ] **Step 1: Traduzir o título do dashboard**

Substituição única (a chave `"title"` no nível raiz do JSON, valor `"Flood Test"`):

| old_string | new_string |
|---|---|
| `"title": "Flood Test"` | `"title": "Supervisório Gaardrail"` |

Atenção: conferir antes com grep que `"title": "Flood Test"` só ocorre 1 vez; se ocorrer mais de uma, restringir o contexto do Edit (incluir linha vizinha, ex. `"uid"` ou `"tags"`).

- [ ] **Step 2: Traduzir títulos de painéis**

Substituições com Edit (usar `replace_all: true` onde marcado; os demais são únicos):

| old_string | new_string | replace_all |
|---|---|---|
| `"title": "PID Drain Rate"` | `"title": "Taxa de Drenagem (PID)"` | sim (stat id=11 e timeseries id=3) |
| `"title": "PID: CPU % vs Setpoint vs Drain rate"` | `"title": "PID: CPU % vs Setpoint vs Taxa de drenagem"` | não |
| `"title": "PID Terms (P / I / D / Error)"` | `"title": "Termos do PID (P / I / D / Erro)"` | não |
| `"title": "PID Params History"` | `"title": "Histórico de Parâmetros do PID"` | não |
| `"title": "Smith: CPU % vs Setpoint vs Drain rate"` | `"title": "Smith: CPU % vs Setpoint vs Taxa de drenagem"` | não |
| `"title": "Smith Terms (P / I / D / Error)"` | `"title": "Termos do Smith (P / I / Erro)"` | não |
| `"title": "Smith Params History"` | `"title": "Histórico de Parâmetros do Smith"` | não |
| `"title": "Smith Drain Rate"` | `"title": "Taxa de Drenagem (Smith)"` | não |
| `"title": "Messages Processed (total)"` | `"title": "Mensagens Processadas (total)"` | não |
| `"title": "Messages Processed (msg/s)"` | `"title": "Mensagens Processadas (msg/s)"` | não |
| `"title": "Queue Operation Times"` | `"title": "Tempos de Operação da Fila"` | não |
| `"title": "Message Consume Time"` | `"title": "Tempo de Consumo de Mensagem"` | não |
| `"title": "MySQL Row Locks"` | `"title": "MySQL Locks de Linha"` | não |
| `"title": "MySQL Network I/O"` | `"title": "MySQL I/O de Rede"` | não |
| `"title": "MySQL Detalhado"` | (fica — já em PT) | — |
| `"title": "Lag do Consumer"` | (fica — já em PT) | — |
| `"title": "Step: CPU % vs Max"` | (fica — "Max" é nome do parâmetro) | — |

Nota: o título original do painel Smith diz "(P / I / D / Error)" mas o Smith é PI sem derivativo (legendas: P, I, smith_error...) — a tradução corrige para "(P / I / Erro)". É a única mudança além de tradução literal, intencional.

- [ ] **Step 3: Traduzir legendas (`legendFormat`)**

| old_string | new_string | replace_all |
|---|---|---|
| `"legendFormat": "Drain rate"` | `"legendFormat": "Taxa de drenagem"` | sim (ids 11, 102, 301) |
| `"legendFormat": "Drain rate (msgs/s)"` | `"legendFormat": "Taxa de drenagem (msgs/s)"` | sim (ids 3, 304) |
| `"legendFormat": "Processed/s"` | `"legendFormat": "Processadas/s"` | sim (ids 3, 304) |
| `"legendFormat": "Min (controller)"` | `"legendFormat": "Min (controlador)"` | não |
| `"legendFormat": "Max (controller)"` | `"legendFormat": "Max (controlador)"` | sim (ids 3, 304) |
| `"legendFormat": "Raw output (pre-clamp)"` | `"legendFormat": "Saída bruta (pré-clamp)"` | sim (ids 3, 304) |
| `"legendFormat": "Total processed"` | `"legendFormat": "Total processado"` | não |
| `"legendFormat": "Enqueue time"` | `"legendFormat": "Tempo de enqueue"` | não |
| `"legendFormat": "Dequeue time"` | `"legendFormat": "Tempo de dequeue"` | não |
| `"legendFormat": "Consume time (dequeue→ack)"` | `"legendFormat": "Tempo de consumo (dequeue→ack)"` | não |
| `"legendFormat": "Process time (created→ack)"` | `"legendFormat": "Tempo de processamento (created→ack)"` | não |
| `"legendFormat": "Buffer pool hit rate %"` | `"legendFormat": "Taxa de acerto do buffer pool %"` | não |
| `"legendFormat": "Lock waits/s"` | `"legendFormat": "Esperas de lock/s"` | não |
| `"legendFormat": "Avg lock wait (ms)"` | `"legendFormat": "Espera média de lock (ms)"` | não |
| `"legendFormat": "Temp disk tables/s"` | `"legendFormat": "Tabelas temp em disco/s"` | não |
| `"legendFormat": "Queue lag (app)"` | `"legendFormat": "Lag da fila (app)"` | não |
| `"legendFormat": "Consumer lag (msgs)"` | (fica — termo Kafka) | — |
| `"legendFormat": "Working set (RSS)"` | (fica — termo técnico) | — |
| `"legendFormat": "Slow queries/s"`, `"Full joins/s (sem index)"`, `"Full table scans/s"` | (ficam — termos MySQL) | — |
| `"legendFormat": "Msgs/s"`, `"CPU %"`, `"Setpoint"`, `"Tipo"`, `"P"`, `"I"`, `"D"`, `"Kp"`, `"Ki"`, `"Kd"`, `"IClamp"`, `"Max output"`, `"Max"`, `"{{name}}"`, `"smith_*"` | (ficam) | — |

`"Max output"`: fica — é o nome do parâmetro `max` do controlador exibido no histórico.

- [ ] **Step 4: Validar**

```bash
python3 -c "import json; json.load(open('flood-test/grafana/dashboards/flood.json')); print('JSON OK')"
git diff --stat flood-test/grafana/dashboards/flood.json
git diff flood-test/grafana/dashboards/flood.json | grep '^[+-]' | grep -v '^[+-][+-]' | grep -vE '"(title|legendFormat)"' || echo "diff só em title/legendFormat: OK"
```

Expected: `JSON OK`; o terceiro comando imprime `diff só em title/legendFormat: OK` (nenhuma linha do diff fora dessas chaves).

- [ ] **Step 5: Checkpoint**

Mostrar `git diff --stat` ao usuário. NÃO commitar (usuário commita manualmente).

---

### Task 2: Extrair paleta para CSS variables com light como padrão

**Files:**
- Modify: `web/index.html` (bloco `<style>`, linhas ~11–224, e cores inline no JS)

**Interfaces:**
- Consumes: nada.
- Produces: variáveis CSS que a Task 3 usa: `--bg`, `--surface`, `--border`, `--text`, `--text-muted`, `--text-faint`, `--accent`, `--success`, `--success-hover`, `--error`, `--error-hover`, `--backdrop`. Seletor `[data-theme="dark"]` no `<html>` ativa o dark.

Ordem obrigatória: primeiro os replaces de cor (Step 1), depois inserir os blocos de paleta (Step 2) — senão os replaces corromperiam os hex dentro do bloco `[data-theme="dark"]`.

- [ ] **Step 1: Substituir todas as cores hardcoded pelas variáveis**

Mapeamento (Edit com `replace_all: true` por cor; pega CSS e também as strings do JS, o que é desejado — ver Step 3):

| cor atual | vira |
|---|---|
| `#0d1117aa` | `var(--backdrop)` |
| `#0d1117` | `var(--bg)` |
| `#161b22` | `var(--surface)` |
| `#30363d` | `var(--border)` |
| `#e6edf3` | `var(--text)` |
| `#8b949e` | `var(--text-muted)` |
| `#6e7681` | `var(--text-faint)` |
| `#58a6ff` | `var(--accent)` |
| `#238636` | `var(--success)` |
| `#2ea043` | `var(--success-hover)` |
| `#da3633` | `var(--error)` |
| `#f85149` | `var(--error-hover)` |

Atenção à ordem: `#0d1117aa` antes de `#0d1117` (senão o replace de `#0d1117` corrompe o `aa`). `#fff` (texto dos botões `.btn-save`/`.btn-flood`) fica hardcoded — branco sobre verde/vermelho é legível nos dois temas.

- [ ] **Step 2: Adicionar os blocos de paleta no topo do `<style>`**

Logo após `* { box-sizing: border-box; margin: 0; padding: 0; }` inserir:

```css
:root {
  --bg: #ffffff;
  --surface: #f6f8fa;
  --border: #d0d7de;
  --text: #1f2328;
  --text-muted: #57606a;
  --text-faint: #6e7781;
  --accent: #0969da;
  --success: #1a7f37;
  --success-hover: #1f883d;
  --error: #cf222e;
  --error-hover: #a40e26;
  --backdrop: #1f2328aa;
}

[data-theme="dark"] {
  --bg: #0d1117;
  --surface: #161b22;
  --border: #30363d;
  --text: #e6edf3;
  --text-muted: #8b949e;
  --text-faint: #6e7681;
  --accent: #58a6ff;
  --success: #238636;
  --success-hover: #2ea043;
  --error: #da3633;
  --error-hover: #f85149;
  --backdrop: #0d1117aa;
}
```

- [ ] **Step 3: Conferir as cores inline do JS**

Após o Step 1, as strings JS que eram `'#da3633'` e `'#238636'` viraram `'var(--error)'` e `'var(--success)'`. Isso é válido: `el.style.color = 'var(--error)'` e `style="color:var(--error)"` em HTML gerado resolvem a variável no contexto do elemento. Conferir que não sobrou nenhum hex antigo:

```bash
grep -nE '#(0d1117|161b22|30363d|e6edf3|8b949e|6e7681|58a6ff|238636|2ea043|da3633|f85149)' web/index.html
```

Expected: apenas as linhas dentro do bloco `[data-theme="dark"]` (12 linhas). Qualquer outra ocorrência é um replace faltando.

- [ ] **Step 4: Validar visualmente**

Subir o app local (fila inmemory, sem Kafka):

```bash
APP_QUEUE_PROTOCOL=inmemory go run ./cmd/api &
```

Abrir `http://localhost:8080` (porta do config). Expected: página inteira em tema light (fundo branco, header cinza-claro, texto escuro, accent azul), drawer legível, inputs/selects com borda visível. O iframe do Grafana aparece no tema do servidor (light) — sincronização vem na Task 3. Encerrar o app após conferir.

- [ ] **Step 5: Checkpoint**

Mostrar `git diff --stat web/index.html` ao usuário. NÃO commitar.

---

### Task 3: Botão de toggle, persistência e sincronização do iframe

**Files:**
- Modify: `web/index.html` (header `#bar`, `<head>`, fim do `<script>`)

**Interfaces:**
- Consumes: variáveis CSS e seletor `[data-theme="dark"]` da Task 2.
- Produces: `localStorage.theme` (`"light"` | `"dark"`); funções `toggleTheme()`, `setTheme(theme)`, `syncGrafanaTheme(theme)`.

- [ ] **Step 1: Script anti-flash no `<head>`**

Inserir logo após a linha `<link rel="manifest" ...>` (antes do `<style>`):

```html
<script>
  if (localStorage.getItem('theme') === 'dark') {
    document.documentElement.dataset.theme = 'dark';
  }
</script>
```

- [ ] **Step 2: Botão no header**

Em `#bar`, logo após o botão `#gear`:

```html
<button id="theme-toggle" onclick="toggleTheme()" title="tema">🌙</button>
```

E no CSS, estender os seletores do gear (as duas regras existentes):

```css
#gear, #theme-toggle {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-muted);
  font-size: 14px;
  line-height: 1;
  padding: 5px 9px;
}
#gear:hover, #theme-toggle:hover { border-color: var(--accent); color: var(--accent); }
```

(Substituir as regras `#gear { ... }` e `#gear:hover { ... }` existentes por essas.)

- [ ] **Step 3: JS do toggle no fim do `<script>` principal**

Adicionar antes das chamadas de init (`renderPoll(); loadParams(); ...`):

```js
// ---- Theme ----

function toggleTheme() {
  const dark = document.documentElement.dataset.theme === 'dark';
  setTheme(dark ? 'light' : 'dark');
}

function setTheme(theme) {
  if (theme === 'dark') document.documentElement.dataset.theme = 'dark';
  else delete document.documentElement.dataset.theme;
  localStorage.setItem('theme', theme);
  document.getElementById('theme-toggle').textContent = theme === 'dark' ? '☀' : '🌙';
  syncGrafanaTheme(theme);
}

function syncGrafanaTheme(theme) {
  const iframe = document.querySelector('#grafana iframe');
  if (!iframe) return;
  const src = iframe.src;
  const next = /[?&]theme=/.test(src)
    ? src.replace(/([?&])theme=[^&]*/, '$1theme=' + theme)
    : src + (src.includes('?') ? '&' : '?') + 'theme=' + theme;
  if (next !== src) iframe.src = next;
}

function initTheme() {
  const theme = document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light';
  document.getElementById('theme-toggle').textContent = theme === 'dark' ? '☀' : '🌙';
  syncGrafanaTheme(theme);
}
```

E acrescentar `initTheme();` junto às chamadas de init existentes:

```js
initTheme();
renderPoll();
loadParams();
loadQueue();
loadDisturbance();
```

Notas: o ícone mostra o tema alvo (🌙 no light = "mudar para dark"; ☀ no dark). `syncGrafanaTheme` só reatribui `src` se mudou, evitando reload desnecessário no init quando o param já está correto. Manipulação de string em vez de `new URL`/`searchParams`: o re-serialize de URLSearchParams transformaria `?kiosk` em `?kiosk=`, que o Grafana trata como kiosk desligado.

- [ ] **Step 4: Validar o fluxo completo**

Subir o app de novo (`APP_QUEUE_PROTOCOL=inmemory go run ./cmd/api &`) e conferir no browser:

1. Primeira visita (localStorage limpo — usar aba anônima ou `localStorage.clear()`): página light, botão 🌙, iframe do Grafana com `theme=light` na URL.
2. Clicar no toggle: página vira dark sem reload, botão vira ☀, iframe recarrega com `theme=dark`.
3. F5: página abre direto em dark, sem flash light.
4. Toggle de volta + F5: abre em light.
5. Drawer nos três estados (borda azul ao editar, verde no sucesso, vermelho em erro — forçar erro salvando com o app derrubado) legível nos dois temas.

Encerrar o app.

- [ ] **Step 5: Checkpoint final**

Mostrar `git diff --stat` dos dois arquivos. NÃO commitar. Avisar que o efeito no VPS só aparece após deploy manual (`gh workflow run Deploy`).
