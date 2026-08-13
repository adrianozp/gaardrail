# Filtros de setpoint e de medição — plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Filtro de setpoint configurável (tipo + tamanho) em pid/pidff/smith, exposto na API e na interface; filtro de medição movido dos controladores para a cadeia de medição como opção geral.

**Architecture:** Um componente único em `internal/filter` (`Signal`, com tipos none/moving_average/exponential e tamanho em amostras) é usado duas vezes: como prefiltro de referência dentro de pid e smith, e como filtro geral da PV no usecase `processmetrics`. Config plano por seção; legado `setpoint_filter_tau` mapeado para o exponencial equivalente.

**Tech Stack:** Go (fx, viper/mapstructure, testes padrão), web/index.html vanilla JS.

## Global Constraints

- Sem comentários no código; se precisar explicar, extrair função com nome descritivo (orientação do autor).
- Spec: `docs/superpowers/specs/2026-07-26-filtros-setpoint-e-medicao-design.md`.
- Comportamento default preserva o atual: sem chaves novas, pid com `setpoint_filter_tau>0` filtra igual hoje; smith sem filtro; medição sem filtro (`none`).
- `size` sempre em amostras; exponencial usa a = e^(−1/size) por tique (size=round(tau·1000/interval_ms) reproduz o tau legado quando dt=T).
- Commits com autor Adriano <adrianozdp@gmail.com>, sem Co-Authored-By.

---

### Task 1: `internal/filter.Signal` (tipos + tamanho)

**Files:**
- Modify: `internal/filter/filter.go`
- Test: `internal/filter/filter_test.go`

**Interfaces:**
- Produces: `filter.NewReferenceFilter(kind string, size int) (*Signal, error)`, `filter.NewMeasurementFilter(kind string, size int) (*Signal, error)`, métodos `Filter(x float64) float64`, `Seed(v float64)`, `Reset()`, `Kind() string`, `Size() int`. Kinds válidos: `"none"`, `"moving_average"`, `"exponential"` (vazio ≡ none). Referência parte de estado 0; medição semeia com a 1ª amostra.

- [ ] **Step 1: Testes falhando**

```go
func TestSignalNonePassthrough(t *testing.T) {
	s, err := filter.NewReferenceFilter("none", 5)
	if err != nil || s.Filter(50) != 50 {
		t.Fatalf("none deve ser passthrough, err=%v", err)
	}
}

func TestSignalMovingAverageRampaDegrauEmNAmostras(t *testing.T) {
	s, _ := filter.NewReferenceFilter("moving_average", 4)
	got := []float64{s.Filter(50), s.Filter(50), s.Filter(50), s.Filter(50)}
	want := []float64{12.5, 25, 37.5, 50}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-9 {
			t.Fatalf("amostra %d: got %v want %v", i, got[i], want[i])
		}
	}
}

func TestSignalExponentialConstantePorTique(t *testing.T) {
	s, _ := filter.NewReferenceFilter("exponential", 2)
	a := math.Exp(-0.5)
	if got, want := s.Filter(50), (1-a)*50; math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestSignalMeasurementSeedsComPrimeiraAmostra(t *testing.T) {
	s, _ := filter.NewMeasurementFilter("exponential", 4)
	if got := s.Filter(80); got != 80 {
		t.Fatalf("1ª amostra deve passar direto, got %v", got)
	}
}

func TestSignalKindInvalido(t *testing.T) {
	if _, err := filter.NewReferenceFilter("mediana", 3); err == nil {
		t.Fatal("kind inválido deve dar erro")
	}
}

func TestSignalSeedDefineEstado(t *testing.T) {
	s, _ := filter.NewReferenceFilter("exponential", 3)
	s.Seed(50)
	if got := s.Filter(50); math.Abs(got-50) > 1e-9 {
		t.Fatalf("após Seed(50), Filter(50) deve manter 50, got %v", got)
	}
}
```

- [ ] **Step 2: Rodar e ver falhar** — `go test ./internal/filter/` → FAIL (símbolos indefinidos)
- [ ] **Step 3: Implementação mínima**

```go
type Signal struct {
	kind          string
	size          int
	seedFromFirst bool
	seeded        bool
	ma            *MovingAverage
	state         float64
}

func NewReferenceFilter(kind string, size int) (*Signal, error) {
	return newSignal(kind, size, false)
}

func NewMeasurementFilter(kind string, size int) (*Signal, error) {
	return newSignal(kind, size, true)
}

func newSignal(kind string, size int, seedFromFirst bool) (*Signal, error) {
	if kind == "" {
		kind = "none"
	}
	if kind != "none" && kind != "moving_average" && kind != "exponential" {
		return nil, fmt.Errorf("filter: tipo desconhecido %q", kind)
	}
	if size < 1 {
		size = 1
	}
	return &Signal{kind: kind, size: size, seedFromFirst: seedFromFirst, ma: NewMovingAverage(size)}, nil
}

func (s *Signal) Filter(x float64) float64 {
	switch s.kind {
	case "moving_average":
		return s.filterMovingAverage(x)
	case "exponential":
		return s.filterExponential(x)
	default:
		return x
	}
}

func (s *Signal) filterMovingAverage(x float64) float64 {
	if s.seedFromFirst && !s.seeded {
		s.Seed(x)
		return x
	}
	return s.ma.FilterFullWindow(x)
}

func (s *Signal) filterExponential(x float64) float64 {
	if s.seedFromFirst && !s.seeded {
		s.Seed(x)
		return x
	}
	a := math.Exp(-1.0 / float64(s.size))
	s.state = a*s.state + (1-a)*x
	s.seeded = true
	return s.state
}

func (s *Signal) Seed(v float64) {
	s.state = v
	s.seeded = true
	s.ma.fill(v)
}

func (s *Signal) Reset() { s.state = 0; s.seeded = false; s.ma.Reset() }
func (s *Signal) Kind() string { return s.kind }
func (s *Signal) Size() int    { return s.size }
```

Nota de implementação: a rampa da média móvel exige janela cheia desde o início
(média sobre `size` posições, buffer começando em zero), diferente do
`MovingAverage.Filter` atual (média das amostras vistas). Adicionar ao
`MovingAverage` os métodos `FilterFullWindow(x)` (divide sempre por `size`) e
`fill(v)` (preenche o buffer com `v`), sem tocar no `Filter` existente.

- [ ] **Step 4: Rodar e ver passar** — `go test ./internal/filter/` → PASS (incluindo os testes antigos do MovingAverage)
- [ ] **Step 5: Commit** — `git add internal/filter && git -c user.name=Adriano -c user.email=adrianozdp@gmail.com commit -m "filter: Signal com tipo+tamanho (none/média móvel/exponencial)"`

### Task 2: config (pid, smith, metrics_poller)

**Files:**
- Modify: `pkg/config/pid.go`, `pkg/config/smith.go`, `pkg/config/metricspoller.go`

**Interfaces:**
- Produces: `PID.SetpointFilterType string`, `PID.SetpointFilterSize int` (mantém `SetpointFilterTau`); `Smith.SetpointFilterType/Size`; `MetricsPoller.FilterType string`, `MetricsPoller.FilterSize int`. `PID.FilterSize` e `Smith.FilterSize` removidos.

- [ ] **Step 1:** Em `PID`: remover `FilterSize`; adicionar `SetpointFilterType string \`mapstructure:"setpoint_filter_type" default:""\`` e `SetpointFilterSize int \`mapstructure:"setpoint_filter_size" default:"0"\``. Atualizar `envKeys` do init (tirar `pid.filter_size`, incluir as duas novas).
- [ ] **Step 2:** Em `Smith`: remover `FilterSize`; adicionar os mesmos dois campos; atualizar `envKeys`.
- [ ] **Step 3:** Em `MetricsPoller`: adicionar `FilterType string \`mapstructure:"filter_type" default:"none"\`` e `FilterSize int \`mapstructure:"filter_size" default:"1"\``; atualizar `envKeys`.
- [ ] **Step 4:** `go build ./...` → falhas esperadas nos usos de `FilterSize` (corrigidos nas Tasks 3–4).
- [ ] **Step 5:** Commit junto com a Task 3 (o build precisa fechar).

### Task 3: pid.go e smith.go (prefiltro de referência; medição removida)

**Files:**
- Modify: `app/repositories/controllers/pid/pid.go`, `app/repositories/controllers/smith/smith.go`
- Test: `app/repositories/controllers/pid/pid_test.go`, `app/repositories/controllers/smith/smith_test.go`

**Interfaces:**
- Consumes: `filter.NewReferenceFilter`, `Signal.Seed`.
- Produces: `Controller.refFilter *filter.Signal` interno; `GetParams` devolve `SetpointFilterType/Size` e não devolve mais `FilterSize`; `SetParams` aceita `SetpointFilterType/Size` (erro em tipo inválido).

- [ ] **Step 1: Testes falhando (pid)**

```go
func TestSetpointFilterMovingAverageRampaReferencia(t *testing.T) {
	cfg := config.Config{PID: config.PID{Kp: 1, Max: 100, Min: -100, IClamp: 100,
		Setpoint: 50, SetpointFilterType: "moving_average", SetpointFilterSize: 4}}
	c := pid.New(cfg)
	base := time.Now()
	out1, _ := c.Compute(0, base.Add(1*time.Second))
	if math.Abs(out1-12.5) > 1e-6 {
		t.Fatalf("1º tique: referência deve ser 12,5 (P=12,5), got %v", out1)
	}
}

func TestLegacyTauViraExponencialEquivalente(t *testing.T) {
	cfg := config.Config{MetricsPoller: config.MetricsPoller{IntervalMs: 5000},
		PID: config.PID{Kp: 1, Max: 100, Min: -100, IClamp: 100, Setpoint: 50,
			SetpointFilterTau: 10}}
	c := pid.New(cfg)
	base := time.Now()
	out1, _ := c.Compute(0, base.Add(5*time.Second))
	want := (1 - math.Exp(-0.5)) * 50
	if math.Abs(out1-want) > 1e-6 {
		t.Fatalf("tau=2T deve equivaler a size=2: got %v want %v", out1, want)
	}
}
```

(Nos dois testes acima Ki=0 e medição 0, então a saída é só P = referência filtrada.)

- [ ] **Step 2: Rodar e ver falhar** — `go test ./app/repositories/controllers/pid/`
- [ ] **Step 3: Implementar em pid.go**
  - Remover `filter *filter.MovingAverage` e a linha `measured = c.filter.Filter(measured)`; remover `spFilterTau/spFiltered`.
  - Novo campo `refFilter *filter.Signal`; resolução na construção:

```go
func referenceFilterFromConfig(cfg config.Config) *filter.Signal {
	p := cfg.PID
	kind, size := p.SetpointFilterType, p.SetpointFilterSize
	if kind == "" && p.SetpointFilterTau > 0 {
		kind = "exponential"
		size = equivalentSamples(p.SetpointFilterTau, cfg.MetricsPoller.IntervalMs)
	}
	f, err := filter.NewReferenceFilter(kind, size)
	if err != nil {
		panic("pid.New: " + err.Error())
	}
	return f
}

func equivalentSamples(tauSeconds float64, intervalMs int) int {
	if intervalMs <= 0 {
		return 1
	}
	return max(int(math.Round(tauSeconds*1000/float64(intervalMs))), 1)
}
```

  - Em `Compute`: `spEff := c.refFilter.Filter(c.setpoint)` (o `dt` deixa de ser usado pelo filtro).
  - `GetParams`: trocar `FilterSize` por `SetpointFilterType *string` e `SetpointFilterSize *int`.
  - `SetParams`: aplicar os dois campos quando não nulos; em mudança, recriar via `NewReferenceFilter` (erro propaga) e `Seed` com a referência efetiva atual para não saltar.
- [ ] **Step 4: Mesmo tratamento em smith.go** (não há legado tau na seção; default none = degrau cru atual). Remover o filtro de medição (`filter` field, `FilterSize`) por completo. Teste:

```go
func TestSmithSemFiltroDeSetpointMantemDegrau(t *testing.T) {
	c := smith.New(cfg)
	out, _ := c.Compute(0, base.Add(time.Second))
	// setpoint 50, kp 1: erro cheio no 1º tique
}

func TestSmithComFiltroExponencialRampa(t *testing.T) { /* análogo ao do pid */ }
```

- [ ] **Step 5: Rodar** — `go test ./app/repositories/controllers/...` → PASS (ajustar testes existentes que referenciem FilterSize)
- [ ] **Step 6: Commit** — `config+pid+smith: filtro de setpoint tipo+tamanho; medição sai dos controladores`

### Task 4: processmetrics (filtro geral da cadeia)

**Files:**
- Modify: `app/usecases/processmetrics/processmetrics.go`, `cmd/api/modules/metricspoller.go` (se a assinatura pedir), mocks se necessário (`go generate ./...`)
- Test: `app/usecases/processmetrics/processmetrics_test.go`

**Interfaces:**
- Consumes: `filter.NewMeasurementFilter`, `cfg.MetricsPoller.FilterType/FilterSize`.
- Produces: `NewProcessMetricsUseCase(c Controller, o Orchestrator, cfg config.Config) ProcessMetricsUseCase`.

- [ ] **Step 1: Teste falhando**

```go
func TestProcessAplicaFiltroGeralNaPV(t *testing.T) {
	cfg := config.Config{MetricsPoller: config.MetricsPoller{FilterType: "moving_average", FilterSize: 2}}
	ctrl := mocks.NewController(t)
	orch := mocks.NewOrchestrator(t)
	u := processmetrics.NewProcessMetricsUseCase(ctrl, orch, cfg)
	ctrl.On("Compute", 40.0, mock.Anything).Return(10.0, nil)
	orch.On("SetDrainRate", 10.0).Return(nil)
	u.Process(entities.Metrics{Metrics: map[string]float64{"cpu": 40}, MeasureTime: time.Now()})
	ctrl.On("Compute", 60.0, mock.Anything).Return(10.0, nil)
	u.Process(entities.Metrics{Metrics: map[string]float64{"cpu": 80}, MeasureTime: time.Now()})
}
```

(1ª amostra semeia e passa 40; 2ª = média(40,80)=60.)

- [ ] **Step 2: Ver falhar**, **Step 3: implementar** (campo `pvFilter *filter.Signal`; `cpuPercentage = u.pvFilter.Filter(cpuPercentage)` antes do gauge e do Compute), **Step 4: ver passar**
- [ ] **Step 5: Commit** — `processmetrics: filtro geral da PV (metrics_poller.filter_type/size)`

### Task 5: entities, DTO, persistência

**Files:**
- Modify: `app/entities/entities.go`, `app/handlers/controllerparams/dto.go`, `app/usecases/controllerparams/controllerparams.go`
- Test: `app/handlers/controllerparams/*_test.go`, `app/usecases/controllerparams/*_test.go`

**Interfaces:**
- Produces: `ControllerParams.SetpointFilterType *string`, `SetpointFilterSize *int` (sem `FilterSize`); request/response JSON `setpoint_filter_type`/`setpoint_filter_size`; persistência grava `pid.setpoint_filter_type` e `pid.setpoint_filter_size`.

- [ ] **Step 1:** Testes: PATCH com os dois campos chega ao controller (mock) e persiste as duas chaves; GET reflete.
- [ ] **Step 2:** Ver falhar. **Step 3:** aplicar nos três arquivos (request, response com `setpoint_filter_type string`/`setpoint_filter_size int`, `pidUpdates` com as duas chaves novas; remover `FilterSize` do response).
- [ ] **Step 4:** `go test ./app/...` PASS. **Step 5:** Commit — `api: setpoint_filter_type/size expostos e persistidos; filter_size removido`

### Task 6: interface (web/index.html)

**Files:**
- Modify: `web/index.html`

- [ ] **Step 1:** `FIELDS` ganha `'setpoint_filter_size'`; `SMITH_FIELDS` ganha `'setpoint_filter_size'`; remover `'filter_size'` de `SMITH_MODEL`.
- [ ] **Step 2:** Em `renderParams()`, antes do grid, um `<select id="input-setpoint_filter_type">` com `none|moving_average|exponential` (valor de `current.setpoint_filter_type`), exibido para pid/pidff/smith.
- [ ] **Step 3:** Em `saveParams()`, tratar o select como string: `const t = document.getElementById('input-setpoint_filter_type'); if (t && t.value !== current.setpoint_filter_type) body.setpoint_filter_type = t.value;`
- [ ] **Step 4:** Smoke manual: `go run ./cmd/api`, abrir `/`, trocar tipo/tamanho, salvar, recarregar, conferir GET.
- [ ] **Step 5:** Commit — `ui: filtro de setpoint no painel (tipo+tamanho); filter_size sai do grupo do smith`

### Task 7: regressão e conformidade

- [ ] `go build ./... && go vet ./... && go test ./...` limpos.
- [ ] Rodar a skill `go-gaardrail-review` sobre o diff e corrigir violações.
- [ ] Smoke no rig: subir app com config demo intocado (legado tau=10) e confirmar partida idêntica (referência rampa); PATCH `{"setpoint_filter_type":"moving_average","setpoint_filter_size":4}` e observar rampa linear no supervisório.
- [ ] Commit final se houver ajustes.
