# Dashboard em português + toggle light/dark na web UI

Data: 2026-07-29

## Objetivo

1. Padronizar o dashboard Grafana (`flood-test/grafana/dashboards/flood.json`) inteiramente em português.
2. Adicionar alternância de tema light/dark na web UI (`web/index.html`), com light como padrão.

## 1. Tradução do dashboard para português

Escopo: somente `flood.json`. Os backups (`flood.json.bak-*`) não são alterados. Nada estrutural muda — queries, uids, thresholds, layout e datasources ficam intactos; apenas strings visíveis (títulos de painéis, legendas, descriptions).

Traduções:

| Atual | Novo |
|---|---|
| PID Drain Rate (stat e timeseries) | Taxa de Drenagem (PID) |
| PID: CPU % vs Setpoint vs Drain rate | PID: CPU % vs Setpoint vs Taxa de drenagem |
| PID Terms (P / I / D / Error) | Termos do PID (P / I / D / Erro) |
| PID Params History | Histórico de Parâmetros do PID |
| Messages Processed (total) | Mensagens Processadas (total) |
| Messages Processed (msg/s) | Mensagens Processadas (msg/s) |
| Queue Operation Times | Tempos de Operação da Fila |
| Message Consume Time | Tempo de Consumo de Mensagem |
| MySQL Row Locks | MySQL Locks de Linha |
| MySQL Network I/O | MySQL I/O de Rede |
| legenda "Processed/s" | Processadas/s |
| legenda "Total processed" | Total processado |
| legenda "Process time (created→ack)" | Tempo de processamento (created→ack) |
| título do dashboard "Flood Test" | Supervisório Gaardrail |

Regras:
- Strings já em português permanecem.
- Termos técnicos consagrados permanecem em inglês: setpoint, InnoDB Buffer Pool, QPS, Kafka, drain (em nomes de métricas), created→ack.
- Descriptions de painéis em inglês, se houver, seguem as mesmas regras.
- Terminologia alinhada aos docs do projeto: drain rate = "taxa de drenagem" (cf. GUIA-CONTROLE-gaardrail.md).

Validação: JSON válido (parse), diff contendo apenas mudanças em `title`, `legendFormat` e `description`; dashboard carrega no Grafana local.

## 2. Toggle light/dark na web UI

Escopo: somente `web/index.html` (arquivo único, CSS e JS inline).

Abordagem: CSS custom properties.

- A paleta hardcoded atual (estilo GitHub dark: `#0d1117`, `#161b22`, `#e6edf3`, `#58a6ff`, etc.) é extraída para variáveis em `:root`: `--bg`, `--surface`, `--border`, `--text`, `--text-muted`, `--text-faint`, `--accent`, `--success`, `--error`.
- `:root` define a paleta **light** (padrão), estilo GitHub light: fundo `#ffffff`, superfície `#f6f8fa`, borda `#d0d7de`, texto `#1f2328`, muted `#57606a`, accent `#0969da`, success `#1a7f37`, error `#cf222e`.
- `[data-theme="dark"]` redefine as mesmas variáveis com a paleta atual.
- Todos os usos de cor no CSS passam a referenciar as variáveis; nenhuma cor hardcoded permanece fora dos dois blocos de paleta (exceção: cores intrinsecamente fixas, como texto branco sobre botão de ação, se legível nos dois temas).
- Estados do drawer (editing/success/error) usam as variáveis de accent/success/error e devem ser legíveis nos dois temas.

Toggle:
- Botão sol/lua no header, ao lado do gear, com o mesmo estilo dos botões existentes. Ícone mostra o tema alvo (lua no tema light, sol no dark).
- JS: no carregamento, lê `localStorage.theme`; se `"dark"`, aplica `data-theme="dark"` no `<html>`. Sem valor salvo → light (padrão). O clique alterna o atributo e persiste a escolha.
- O script de aplicação do tema roda inline no `<head>`, antes do CSS renderizar, para evitar flash do tema errado.
- Sem detecção de `prefers-color-scheme` (decisão: light é o padrão explícito do projeto).
- **Iframe do Grafana acompanha o tema**: a página embute o dashboard via `<iframe src="{{GRAFANA_URL}}">`. Ao aplicar/alternar o tema, o JS define o query param `theme=light|dark` na URL do iframe (o Grafana aceita `theme` na URL e ele prevalece sobre o default do servidor). O iframe recarrega ao alternar — aceitável.
- Cores hardcoded no JS (`#da3633`, `#238636` em `style.color` e em HTML gerado) passam a referenciar as variáveis (`var(--error)`, `var(--success)`).

Validação: conferência visual dos dois temas (header, summary, drawer nos três estados, selects e inputs), persistência após reload, ausência de flash no carregamento. O projeto não tem testes automatizados para a web UI; nenhum será adicionado.

## Fora de escopo

- Tema do Grafana (já resolvido via `GF_USERS_DEFAULT_THEME` nos compose).
- Tradução dos backups do dashboard.
- Qualquer mudança estrutural no dashboard ou na API.
