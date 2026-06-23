# Open Loop Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Adicionar um modo open loop ao controlador PID onde o drain rate é fixo e definido diretamente pelo toolkit, sem interferência do PID — pré-requisito para todos os experimentos de identificação do sistema.

**Architecture:** Adicionar estado `openLoop bool` + `openLoopRate float64` ao `Controller`. Quando ativo, `Compute()` retorna o valor fixo sem acumular estado PID. Um handler HTTP com build tag `tooling` expõe `POST /tooling/open-loop` e `DELETE /tooling/open-loop`. O wiring usa o padrão stub/real com build tags para que o código de tooling nunca entre no binário de produção.

**Tech Stack:** Go 1.21+, `go.uber.org/fx`, `github.com/gin-gonic/gin`, build tags (`-tags tooling`), `mockery` para mocks, `testing` stdlib.

---

## File Structure

| Arquivo | Ação | Responsabilidade |
|---------|------|-----------------|
| `internal/controller/controller.go` | Modify | Adiciona campos e métodos open loop ao Controller |
| `internal/controller/controller_test.go` | Modify | Testes do comportamento open loop |
| `app/handlers/openloop/openloop.go` | Create (`//go:build tooling`) | Handler HTTP GET/POST/DELETE para /tooling/open-loop |
| `app/handlers/openloop/dto.go` | Create (`//go:build tooling`) | Request/response DTOs |
| `app/handlers/openloop/openloop_test.go` | Create (`//go:build tooling`) | Testes do handler com mock |
| `cmd/api/modules/openloop.go` | Create (`//go:build tooling`) | fx factories, injections e endpoints do open loop |
| `cmd/api/options/tooling.go` | Create (`//go:build tooling`) | `ToolingOptions()` com todos os módulos de tooling |
| `cmd/api/options/tooling_stub.go` | Create (`//go:build !tooling`) | `ToolingOptions()` stub que retorna `fx.Options()` |
| `cmd/api/main.go` | Modify | Adiciona `options.ToolingOptions()` ao `fx.New()` |

---

## Task 1: Open Loop state no Controller

**Files:**
- Modify: `internal/controller/controller.go`
- Modify: `internal/controller/controller_test.go`

- [ ] **Step 1: Escrever o teste de open loop (falha esperada)**

Adicionar ao final de `internal/controller/controller_test.go`:

```go
func TestOpenLoopReturnsForcedRate(t *testing.T) {
	c := controller.New(pid(1.0, 1.0, 1.0, 0, 100, 20, 50.0))
	c.OpenLoop(7.5)
	out, err := c.Compute(10.0, t0.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if out != 7.5 {
		t.Errorf("expected forced rate 7.5, got %f", out)
	}
}

func TestOpenLoopDoesNotAccumulateIntegral(t *testing.T) {
	c := controller.New(pid(0, 1.0, 0, 0, 100, 20, 50.0))
	c.OpenLoop(5.0)
	for i := 0; i < 10; i++ {
		c.Compute(10.0, t0.Add(time.Duration(i+1)*5*time.Second))
	}
	c.CloseLoop()
	// After CloseLoop, first PID tick should behave like a fresh controller
	c2 := controller.New(pid(0, 1.0, 0, 0, 100, 20, 50.0))
	out1, err := c.Compute(10.0, t0.Add(60*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	out2, _ := c2.Compute(10.0, t0.Add(5*time.Second))
	if math.Abs(out1-out2) > 0.001 {
		t.Errorf("expected clean state after CloseLoop, got %f vs %f", out1, out2)
	}
}

func TestIsOpenLoop(t *testing.T) {
	c := controller.New(pid(1.0, 0, 0, 0, 100, 20, 50.0))
	if c.IsOpenLoop() {
		t.Error("expected false before OpenLoop()")
	}
	c.OpenLoop(3.0)
	if !c.IsOpenLoop() {
		t.Error("expected true after OpenLoop()")
	}
	c.CloseLoop()
	if c.IsOpenLoop() {
		t.Error("expected false after CloseLoop()")
	}
}
```

- [ ] **Step 2: Rodar os testes para confirmar falha**

```bash
cd /home/adrianozdp/workspace/pfc/gaardrail
go test ./internal/controller/... -run "TestOpenLoop|TestIsOpenLoop" -v
```

Esperado: `FAIL — c.OpenLoop undefined`

- [ ] **Step 3: Adicionar campos e métodos ao Controller**

Em `internal/controller/controller.go`, adicionar campos ao struct `Controller`:

```go
type Controller struct {
	mu         sync.RWMutex
	Kp, Ki, Kd float64
	Min, Max   float64
	IClamp     float64

	setpoint    float64
	i           float64
	prevE       float64
	first       bool
	lastCompute time.Time

	openLoop     bool
	openLoopRate float64
}
```

Adicionar ao início do método `Compute()`, logo após o lock:

```go
func (c *Controller) Compute(measured float64, measureTime time.Time) (float64, error) {
	if math.IsNaN(measured) {
		return 0, errors.New("invalid measured metric")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.openLoop {
		return c.openLoopRate, nil
	}

	// ... resto do método permanece igual
```

Adicionar os três novos métodos após `Reset()`:

```go
// OpenLoop fixes the controller output to drainRate, bypassing PID computation.
// Resets PID state so CloseLoop() starts fresh without transients.
func (c *Controller) OpenLoop(drainRate float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.openLoop = true
	c.openLoopRate = drainRate
	c.i = 0
	c.prevE = 0
	c.first = true
	c.lastCompute = time.Time{}
}

// CloseLoop resumes PID computation and resets state to avoid transients.
func (c *Controller) CloseLoop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.openLoop = false
	c.i = 0
	c.prevE = 0
	c.first = true
	c.lastCompute = time.Time{}
}

// IsOpenLoop reports whether the controller is currently in open loop mode.
func (c *Controller) IsOpenLoop() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.openLoop
}
```

- [ ] **Step 4: Rodar os testes para confirmar aprovação**

```bash
go test ./internal/controller/... -v
```

Esperado: todos os testes PASS, incluindo os novos.

- [ ] **Step 5: Commit**

```bash
git add internal/controller/controller.go internal/controller/controller_test.go
git commit -m "feat(controller): add open loop mode to bypass PID computation"
```

---

## Task 2: HTTP Handler open loop (build tag: tooling)

**Files:**
- Create: `app/handlers/openloop/dto.go`
- Create: `app/handlers/openloop/openloop.go`
- Create: `app/handlers/openloop/openloop_test.go`

- [ ] **Step 1: Criar dto.go**

```go
//go:build tooling

package openloop

type setOpenLoopRequest struct {
	DrainRate float64 `json:"drain_rate" binding:"required"`
}

type openLoopStatusResponse struct {
	OpenLoop  bool    `json:"open_loop"`
	DrainRate float64 `json:"drain_rate,omitempty"`
}
```

- [ ] **Step 2: Escrever os testes do handler (falha esperada)**

Criar `app/handlers/openloop/openloop_test.go`:

```go
//go:build tooling

package openloop_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adrianozp/gaardrail/app/handlers/openloop"
	"github.com/gin-gonic/gin"
)

type mockController struct {
	openLoopCalled  bool
	closeLoopCalled bool
	isOpenLoop      bool
	drainRate       float64
}

func (m *mockController) OpenLoop(drainRate float64) {
	m.openLoopCalled = true
	m.drainRate = drainRate
	m.isOpenLoop = true
}

func (m *mockController) CloseLoop() {
	m.closeLoopCalled = true
	m.isOpenLoop = false
}

func (m *mockController) IsOpenLoop() bool {
	return m.isOpenLoop
}

func newRouter(h *openloop.Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	openloop.RegisterRoutes(r, h)
	return r
}

func TestHandleSetOpenLoop(t *testing.T) {
	mock := &mockController{}
	h := openloop.New(mock)
	r := newRouter(h)

	body, _ := json.Marshal(map[string]float64{"drain_rate": 7.5})
	req := httptest.NewRequest(http.MethodPost, "/tooling/open-loop", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if !mock.openLoopCalled {
		t.Error("expected OpenLoop() to be called")
	}
	if mock.drainRate != 7.5 {
		t.Errorf("expected drain_rate 7.5, got %f", mock.drainRate)
	}
}

func TestHandleCloseLoop(t *testing.T) {
	mock := &mockController{isOpenLoop: true}
	h := openloop.New(mock)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/tooling/open-loop", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if !mock.closeLoopCalled {
		t.Error("expected CloseLoop() to be called")
	}
}

func TestHandleGetStatus(t *testing.T) {
	mock := &mockController{isOpenLoop: true, drainRate: 3.0}
	h := openloop.New(mock)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/tooling/open-loop", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["open_loop"] != true {
		t.Errorf("expected open_loop true, got %v", resp["open_loop"])
	}
}

func TestHandleSetOpenLoopBadRequest(t *testing.T) {
	mock := &mockController{}
	h := openloop.New(mock)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/tooling/open-loop", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
```

- [ ] **Step 3: Rodar para confirmar falha**

```bash
go test -tags tooling ./app/handlers/openloop/... -v
```

Esperado: `FAIL — package openloop not found`

- [ ] **Step 4: Criar openloop.go**

```go
//go:build tooling

package openloop

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:generate mockery --name=OpenLoopController --output=mocks --outpkg=mocks
type OpenLoopController interface {
	OpenLoop(drainRate float64)
	CloseLoop()
	IsOpenLoop() bool
}

type Handler struct {
	controller OpenLoopController
}

func New(c OpenLoopController) *Handler {
	return &Handler{controller: c}
}

func RegisterRoutes(r *gin.Engine, h *Handler) {
	r.POST("/tooling/open-loop", h.HandleSet)
	r.DELETE("/tooling/open-loop", h.HandleClose)
	r.GET("/tooling/open-loop", h.HandleStatus)
}

func (h *Handler) HandleSet(c *gin.Context) {
	var req setOpenLoopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.controller.OpenLoop(req.DrainRate)
	c.Status(http.StatusNoContent)
}

func (h *Handler) HandleClose(c *gin.Context) {
	h.controller.CloseLoop()
	c.Status(http.StatusNoContent)
}

func (h *Handler) HandleStatus(c *gin.Context) {
	resp := openLoopStatusResponse{OpenLoop: h.controller.IsOpenLoop()}
	c.JSON(http.StatusOK, resp)
}
```

- [ ] **Step 5: Rodar os testes para confirmar aprovação**

```bash
go test -tags tooling ./app/handlers/openloop/... -v
```

Esperado: todos PASS.

- [ ] **Step 6: Commit**

```bash
git add app/handlers/openloop/
git commit -m "feat(tooling): add open loop HTTP handler"
```

---

## Task 3: fx Wiring com build tags

**Files:**
- Create: `cmd/api/modules/openloop.go` (`//go:build tooling`)
- Create: `cmd/api/options/tooling.go` (`//go:build tooling`)
- Create: `cmd/api/options/tooling_stub.go` (`//go:build !tooling`)
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Criar cmd/api/modules/openloop.go**

```go
//go:build tooling

package modules

import (
	openloophandler "github.com/adrianozp/gaardrail/app/handlers/openloop"
	"github.com/adrianozp/gaardrail/internal/controller"
	"go.uber.org/fx"
)

func OpenLoopFactories() fx.Option {
	return fx.Provide(
		openloophandler.New,
	)
}

func OpenLoopInjections() fx.Option {
	return fx.Provide(
		func(c *controller.Controller) openloophandler.OpenLoopController { return c },
	)
}

func OpenLoopEndpoints() fx.Option {
	return fx.Module("openloop",
		fx.Invoke(openloophandler.RegisterRoutes),
	)
}
```

- [ ] **Step 2: Criar cmd/api/options/tooling.go**

```go
//go:build tooling

package options

import (
	"github.com/adrianozp/gaardrail/cmd/api/modules"
	"go.uber.org/fx"
)

func ToolingOptions() fx.Option {
	return fx.Options(
		modules.OpenLoopFactories(),
		modules.OpenLoopInjections(),
		modules.OpenLoopEndpoints(),
	)
}
```

- [ ] **Step 3: Criar cmd/api/options/tooling_stub.go**

```go
//go:build !tooling

package options

import "go.uber.org/fx"

func ToolingOptions() fx.Option {
	return fx.Options()
}
```

- [ ] **Step 4: Modificar cmd/api/main.go**

```go
package main

import (
	"os"

	"github.com/adrianozp/gaardrail/cmd/api/options"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	fx.New(
		options.Options(),
		options.ToolingOptions(),
	).Run()
}
```

- [ ] **Step 5: Verificar build de produção (sem tooling)**

```bash
go build ./cmd/api/...
```

Esperado: compila sem erros. Sem nenhuma referência ao pacote `openloop`.

- [ ] **Step 6: Verificar build com tooling**

```bash
go build -tags tooling ./cmd/api/...
```

Esperado: compila sem erros. Endpoints de tooling incluídos.

- [ ] **Step 7: Rodar todos os testes**

```bash
go test ./...
go test -tags tooling ./...
```

Esperado: todos PASS nos dois modos.

- [ ] **Step 8: Smoke test manual**

```bash
# Terminal 1 — rodar com tooling
go run -tags tooling ./cmd/api/

# Terminal 2 — testar endpoints
curl -s http://localhost:8080/tooling/open-loop | jq .
# Esperado: {"open_loop":false}

curl -s -X POST http://localhost:8080/tooling/open-loop \
  -H "Content-Type: application/json" \
  -d '{"drain_rate": 5.0}' -w "%{http_code}"
# Esperado: 204

curl -s http://localhost:8080/tooling/open-loop | jq .
# Esperado: {"open_loop":true}

curl -s -X DELETE http://localhost:8080/tooling/open-loop -w "%{http_code}"
# Esperado: 204

curl -s http://localhost:8080/tooling/open-loop | jq .
# Esperado: {"open_loop":false}
```

- [ ] **Step 9: Commit**

```bash
git add cmd/api/modules/openloop.go cmd/api/options/tooling.go cmd/api/options/tooling_stub.go cmd/api/main.go
git commit -m "feat(tooling): wire open loop mode into fx with build tag isolation"
```

---

## Self-Review

**Spec coverage:**
- ✅ Open Loop Mode pausa o PID — `Controller.OpenLoop()` retorna drain rate fixo em `Compute()`
- ✅ Controle direto do drain rate pelo toolkit — `POST /tooling/open-loop` com `drain_rate`
- ✅ Retorno ao loop fechado — `DELETE /tooling/open-loop` chama `CloseLoop()` + reset de estado
- ✅ Build tag isolado — handler e wiring só compilam com `-tags tooling`
- ✅ Estado PID preservado corretamente — `OpenLoop()` e `CloseLoop()` ambos fazem reset para evitar transientes
- ✅ Interface no padrão do projeto — gin handler, fx wiring, mesmo estilo do `/pid`

**Placeholder scan:** nenhum encontrado — todos os steps têm código completo.

**Type consistency:**
- `OpenLoopController` interface em `openloop.go` declara `OpenLoop(float64)`, `CloseLoop()`, `IsOpenLoop() bool`
- `*controller.Controller` implementa os três — confirmado na Task 1
- `openloop.New` aceita `OpenLoopController` — injetado via `func(c *controller.Controller) openloophandler.OpenLoopController` na Task 3
