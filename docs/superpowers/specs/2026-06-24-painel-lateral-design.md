# Painel lateral de configuração — Design

**Data:** 2026-06-24
**Status:** Aprovado

## Objetivo

Adicionar um drawer lateral (slide-out da esquerda) à interface web do Gaardrail
que centralize a configuração em runtime: troca de controlador, troca de fila,
edição de parâmetros do controlador e disparo de flood / definição de query.
A barra superior fica enxuta e o Grafana ocupa todo o resto quando o drawer está
fechado.

Hoje a troca de controlador já existe inline na barra; a troca de fila **não**
existe em runtime (a fila é definida no boot via config). Este design adiciona a
troca de fila e reorganiza a UI.

## Escopo

**Dentro do escopo:**
- Troca de controlador em runtime: `pid` / `step` (backend já existe).
- Troca de fila em runtime: `inmemory` / `constant` apenas (backend novo).
- Edição dos parâmetros do controlador (mover pro drawer).
- Flood (mover pro drawer) e definição de query constante.
- Drawer deslizante acionado por `⚙` na barra.

**Fora do escopo:**
- Troca para filas `kafka` / `sqs` em runtime (dependências externas; `kafka`
  quebra localmente vs Kafka 4.x, `sqs` exige AWS). Quando a config usa essas
  filas, o app sobe normalmente e o drawer **não** oferece troca de fila.
- Troca de tipo de target (`http`/`sql`) em runtime.
- Ajuste de parâmetros do orquestrador (rate/burst/workers) pela UI.
- Persistência da escolha em arquivo de config (troca vive só no processo, como
  já é o caso do controlador).

## Backend — troca de fila em runtime

Espelha o padrão `switchable` já usado no controlador
(`app/repositories/controllers/switchable`). Toda a lógica de troca e os edge
cases ficam dentro do connector; nenhum consumer (orchestrator, usecases,
handlers) muda.

### `app/repositories/queue/switchable/switchable.go` (novo)

- `Queue` embrulha as filas `inmemory` e `constant`, ambas construídas no boot,
  guardadas num `map[string]queue.Queue` com `active`, `activeType` e um
  `sync.RWMutex`.
- Implementa a interface `queue.Queue` existente (`Enqueue`, `Dequeue`, `Ack`,
  `Size`) delegando para a fila ativa (lida fresca a cada chamada sob `RLock`).
- Expõe a capability de troca:
  - `Type() string`
  - `SetType(t string) error` — troca sob `Lock`; no-op se já é a ativa; tipo
    desconhecido → erro.
  - `Available() []string` — `["inmemory", "constant"]`.
- **Edge case do `Dequeue` bloqueante:** `inmemory.Dequeue` bloqueia em canal
  vazio, então um worker parado nele não perceberia uma troca. O `Dequeue` do
  wrapper roda num loop: a cada iteração cria um child-context cancelado quando
  ocorre um switch (um canal de notificação interno fechado/renovado em
  `SetType`); chama `active.Dequeue(childCtx)`; se o erro for o cancelamento por
  switch, relê a fila ativa e repete; caso contrário retorna o resultado. O
  context original (cancelamento real / shutdown) é respeitado e propagado. Cada
  fila subjacente mantém seu próprio estado entre trocas (mensagens em buffer na
  inmemory permanecem; constant nunca drena).

### Wiring — `cmd/api/modules/queue.go`

- `newQueue` retorna o `switchable.Queue` quando `cfg.Queue.Protocol` é
  `inmemory` ou `constant` (ativa = a configurada). Para `kafka` / `sqs` mantém o
  comportamento atual (retorna a fila real, sem troca), preservando o VPS.
- A fila `constant` continua sendo construída uma vez e compartilhada entre o
  `switchable` e a injeção `queuequery.QueryHolder` existente (o
  `GET/PUT /queue/query` não muda).
- Módulos seguem só com factories/injections fx, sem lógica auxiliar (o `switch`
  por protocolo já é uma factory existente).

### Usecase — `app/usecases/switchqueue/` (novo)

Espelha `switchcontroller`:
- `Queue` interface: `SetType(string) error`, `Type() string`,
  `Available() []string`.
- `UseCase` com `Switch(t string) error`, `Current() string`,
  `Available() []string`.
- Teste de unidade com mock (mockery), como `switchcontroller`.

### Handler — `app/handlers/switchqueue/` (novo)

- `GET /queue/type` → `{ "type": "...", "available": ["inmemory","constant"] }`.
- `PUT /queue/type` body `{ "type": "..." }`.
- O handler depende da interface `queue.Queue` e faz type-assert para a
  capability `Switcher` (`Type/SetType/Available`). Se a fila configurada não for
  switchable (kafka/sqs), `GET` retorna `available: []` e `type` com o protocolo
  atual; `PUT` responde `400` com erro "fila não suporta troca".
- DTOs em `dto.go`, teste de handler como `switchcontroller`.

### Endpoints existentes reaproveitados

- `GET /pid`, `PATCH /pid` — params do controlador (sem mudança).
- `PUT /controller/type` — troca de controlador (sem mudança).
- `GET /queue/query`, `PUT /queue/query` — query da fila constant (sem mudança).
- `POST /messages/flood?quantity=N` body `{ "payload": "..." }` — flood (sem
  mudança).

## Frontend — drawer deslizante (`web/index.html`)

Arquivo único, vanilla JS, tema escuro monoespaçado atual.

### Barra superior (enxuta)

`GAARDRAIL · ⚙ · <resumo read-only dos params ativos>`

- `⚙` faz toggle do drawer (abre/fecha).
- Resumo read-only dá contexto sem abrir o drawer (ex.: `pid · setpoint 70 · kp 0.5`),
  atualizado após cada save/troca.
- O `flood-dot` e o `#flood-backdrop`/`#flood-modal` são **removidos**.

### Drawer (slide-out da esquerda), de cima pra baixo

1. **Controlador** — `<select>` `pid`/`step` (movido da barra).
2. **Parâmetros** — campos editáveis de `activeFields()` empilhados na vertical
   (`setpoint/kp/ki/kd/min/max/i_clamp` para pid; só `max` para step), com
   **Salvar / Cancelar**. Mesma lógica de `enterEdit`/`saveParams`/`cancelEdit`,
   re-layoutada na vertical (sem o estado "view vs edit" da barra — campos sempre
   editáveis dentro do drawer, com Salvar).
3. **Fila** — `<select>` `inmemory`/`constant`, populado por `GET /queue/type` e
   desabilitado se `available` vier vazio (config kafka/sqs). Abaixo dele, seção
   condicional ao tipo de fila **ativo**:
   - **constant** → campo "query constante" + botão "aplicar query"
     (`PUT /queue/query`).
   - **inmemory** (ou kafka) → "query (payload)" + "quantidade" + botão
     "▶ flood" (`POST /messages/flood?quantity=N`).

### Comportamento JS

- `loadParams()` (GET `/pid`) popula controlador + params, como hoje.
- `loadQueue()` (novo, GET `/queue/type`) popula o select de fila, define o
  `available` (habilita/desabilita o select) e renderiza a seção condicional
  (query constante vs flood).
- `switchController(type)` e `switchQueue(type)` chamam os respectivos `PUT`, com
  o mesmo tratamento de erro/reversão do `<select>` que já existe para o
  controlador.
- Trocar o controlador re-renderiza os params do drawer (segue o tipo novo).
- Trocar a fila re-renderiza a seção condicional (constant ↔ flood).
- `saveParams()` (PATCH `/pid`), `applyQuery()` (PUT `/queue/query`) e
  `sendFlood()` (POST `/messages/flood`) reaproveitam a lógica atual, adaptada ao
  layout do drawer.
- Feedback (borda azul/verde/vermelha) migra do `#bar` para o drawer; a barra
  apenas mantém o resumo atualizado.
- Estado aberto/fechado via classe CSS + toggle no `⚙`. `Esc` fecha o drawer
  (reaproveita o handler de `keydown` que hoje fecha o flood). `Enter` salva
  params quando o foco está num input de parâmetro.

## Testes

- `switchable.Queue`: unit tests — delegação para a ativa, `SetType` (no-op,
  desconhecido, troca real), `Available`, e o unpark do `Dequeue` na troca
  (worker bloqueado na inmemory é liberado e relê a fila ativa após `SetType`).
- `switchqueue` usecase: unit test com mock.
- `switchqueue` handler: unit test (GET com/sem capability, PUT sucesso, PUT em
  fila não-switchable → 400, body inválido).
- Frontend: validação manual no navegador (sem suíte de testes de UI no projeto).

## Notas de implementação

- Seguir os padrões do projeto: `go-gaardrail-standard` (layout hexagonal em
  camadas, fx DI, viper config, mockery), comentários mínimos (só o "porquê" de
  trechos não óbvios, como o unpark do `Dequeue`).
- Não persistir a troca de fila em config (volume read-only em deploy); reseta
  para o tipo configurado no restart, como o controlador.
