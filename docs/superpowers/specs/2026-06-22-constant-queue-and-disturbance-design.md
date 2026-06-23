# Fila constante + Perturbação — Design

Data: 2026-06-22

## Objetivo

Dois recursos para facilitar os experimentos de validação do controlador (PFC):

1. **Fila constante (infinita)**: um connector de `Queue` cujo `Dequeue` sempre
   devolve a mesma query SQL, evitando ter que reenfileirar a mesma mensagem
   repetidamente durante os testes.
2. **Perturbação**: um componente que executa uma query a uma taxa de X
   queries/segundo contra o banco, para subir a CPU e observar o controlador
   atuando. A taxa da perturbação é exibida no gráfico dos controladores, de
   forma análoga ao drain rate.

Ambos são controlados em runtime via endpoints HTTP, no mesmo espírito do
`PATCH /pid`.

## Princípios respeitados

- **Isolamento de connectors/componentes**: nenhum use case ou orchestrator
  existente muda de comportamento. A fila constante é um novo connector; a
  perturbação é um novo componente independente.
- **Módulos fx sem lógica**: módulos contêm apenas factories e injections. Toda
  lógica (ex.: mapeamento `protocol → código`) vive na camada de domínio.
- **Comentários mínimos**.
- **Testes unitários** (mockery) para os use cases e para o componente de
  perturbação.

---

## Feature 1 — Fila constante

### Connector

Novo pacote `app/repositories/queue/constant/constant.go`, implementando a
interface `queue.Queue` (`Enqueue`, `Dequeue`, `Ack`, `Size`).

Estado:

- `query`: a query SQL atual, protegida para acesso concorrente
  (`sync.RWMutex` sobre uma `string`).
- Um mecanismo de sinalização (`chan struct{}` ou `sync.Cond`) para acordar
  `Dequeue` quando a primeira query for configurada.

Comportamento:

- `Dequeue(ctx)`: monta uma `entities.Message` **nova** a cada chamada:
  - `ID`: `pkg/uuid.New()`
  - `CreatedAt`: `clock.Now()`
  - `Body`: `[]byte(query atual)`
  - Enquanto houver query configurada, **nunca bloqueia**.
  - Sem query configurada, espera (respeitando `ctx.Done()`) até a primeira ser
    setada via config ou endpoint.
- `Enqueue(m)`: no-op funcional — retorna `m.ID, nil` (a fila não acumula; o
  payload é fixo). Mantido apenas para satisfazer a interface.
- `Ack(ctx, m)`: no-op — retorna `nil`.
- `Size()`: retorna **`-1`**, sentinela de "fila infinita / não aplicável",
  distinguindo de uma fila genuinamente vazia (`0`).
  - Justificativa: `Size()` é consumido apenas pelo `lagPoller` do orchestrator
    (`app/orchestrator/orchestrator.go`), que publica o gauge `queue_lag`. É
    puramente observacional; nenhuma lógica de controle depende do valor. O `-1`
    aparece no painel "Tamanho da Fila" como marcador claro de N/A.

API interna para o endpoint:

- `SetQuery(string)`: troca a query atual e sinaliza `Dequeue`.
- `Query() string`: lê a query atual.

### Configuração

- `queue.protocol: constant` ativa o connector.
- `queue.query` (novo campo, default vazio): query inicial opcional, para subir
  já com uma query configurada sem precisar do endpoint.

### Wiring

- Novo `case "constant"` em `newQueue` (`cmd/api/modules/queue.go`).
- Injeção que fornece o `*constant.Queue` concreto ao use case do endpoint
  (separado da injeção `queuerepo.Queue` usada pelo orchestrator).

### Endpoint

Análogo ao `/pid` (pacote `app/handlers/...`, use case `app/usecases/...`):

- `PUT /queue/query` — corpo `{ "query": "<sql>" }`, seta a query. `204`.
- `GET /queue/query` — retorna `{ "query": "<sql>" }`.

Use case fino (`Set(query string)` / `Get() string`) que delega ao
`*constant.Queue` via uma interface `QueryHolder { SetQuery(string); Query() string }`.

---

## Feature 2 — Perturbação

### Componente

Novo pacote `app/disturbance/disturbance.go`, no mesmo padrão do
`app/orchestrator`.

Dependências:

- `*sqlclient.Client` **compartilhado** com o target SQL (ver "Wiring do
  sqlclient" abaixo). Premissa: a perturbação é SQL-only; com target HTTP ela
  permanece inativa (rate 0).

Estado controlável:

- `query`: query a executar.
- `rate`: queries/segundo (gated por `golang.org/x/time/rate.Limiter`, mesmo
  padrão do orchestrator).
- `ttl`: duração opcional.

Comportamento:

- `Set(query string, rate float64, ttl time.Duration)`:
  - `rate == 0`: para a perturbação; publica `disturbance_rate = 0`.
  - `rate > 0`: configura query + limiter; inicia/continua o loop de worker que
    executa a query no ritmo configurado; publica `disturbance_rate = rate`.
  - `ttl > 0`: pulso temporizado — agenda auto-stop após `ttl` (reset para
    rate 0 e `disturbance_rate = 0`).
  - `ttl == 0`: persistente até outro `Set` com `rate = 0`.
- Loop de worker: `limiter.Wait(ctx)` → `client.ExecContext(ctx, query)`; erros
  são logados (não derrubam o loop).
- Trocar a configuração enquanto roda substitui de forma limpa o estado anterior
  (cancela o pulso/loop anterior e reinicia com os novos parâmetros).
- Ciclo de vida integrado ao `fx.Lifecycle` (start/stop) como o orchestrator.

### Endpoint

- `POST /disturbance` — corpo `{ "query": "<sql>", "rate": <float>,
  "duration_seconds": <int, opcional> }`. `204`.
- `GET /disturbance` — retorna estado atual `{ query, rate, duration_seconds }`.

Use case fino que delega ao componente de perturbação via interface.

### Wiring do sqlclient

Refatorar o wiring para que o `*sqlclient.Client` do target SQL seja provido via
fx e injetado tanto no `sqlrepo` quanto no componente de perturbação:

- Hoje `targets.NewTarget` cria o `*sqlclient.Client` internamente no
  `case "sql"`.
- Passa a existir um provider do `*sqlclient.Client` (a partir de `cfg.Target`
  quando `protocol == sql`); `NewSQLRepository` e a perturbação recebem o mesmo
  client (um único pool de conexões).
- Com target HTTP não há `*sqlclient.Client`; a perturbação fica inativa. O
  detalhe exato de como o fx lida com a ausência (provider condicional / client
  nulo tratado no componente) será resolvido no plano de implementação,
  preservando o boot do app com target HTTP.

---

## Feature 3 — Dashboard

### Série de perturbação no gráfico dos controladores

- Nova métrica `gaardrail_disturbance_rate` (publicada pelo componente).
- Adicionar a série "Perturbação (queries/s)" aos painéis de controlador em
  `flood-test/grafana/dashboards/flood.json`:
  - Painel PID "CPU % vs Setpoint vs Drain rate".
  - Painel equivalente do controlador Step.
- Estilo análogo ao drain rate (linha própria, eixo/override coerente com o
  drain rate existente).

### Painel "Tipo da Fila"

- Nova métrica numérica `gaardrail_queue_type`, publicada **uma vez no startup**,
  mapeando o protocolo: `kafka → 0`, `inmemory → 1`, `sqs → 2`, `constant → 3`.
- O mapeamento `protocol → código` é uma função no pacote `queue` (camada de
  domínio); o módulo fx apenas a chama em um `fx.Invoke` (sem lógica no módulo).
- Não altera a interface `Recorder` (que hoje só faz `Gauge`/`Incr`, sem
  labels).
- Novo painel **Stat** "Tipo da Fila" em `flood.json`, com *value mappings*
  renderizando o nome do protocolo (ex.: `3 → "constant"`).

---

## Testes

- `constant.Queue`: `Dequeue` retorna a query atual; bloqueio até primeira query;
  `SetQuery` troca o payload; `Size()` retorna `-1`; respeito a `ctx`.
- Use case do `/queue/query`: get/set delegando ao holder (mock).
- Componente de perturbação: `Set` liga/desliga; pulso temporizado expira;
  publica `disturbance_rate` correto; troca de config limpa o estado anterior
  (com client mockado).
- Use case do `/disturbance`: delega corretamente ao componente (mock).

## Fora de escopo

- Aposentar o `disturb.sh` (continua disponível; a perturbação in-app é
  complementar).
- Suporte a perturbação com target HTTP.
- Labels no `Recorder`.
