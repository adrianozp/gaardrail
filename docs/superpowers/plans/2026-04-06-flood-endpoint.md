# Flood Endpoint & UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `POST /messages/flood?quantity=N` (fire-and-forget, max 10 000) and a hidden red-dot trigger in the frontend that opens a flood modal.

**Architecture:** New handler package `app/handlers/floodmessage` mirrors the existing `createmessage` handler. It validates `quantity`, responds 202 immediately, then dispatches a goroutine that calls the existing `CreateMessageUseCase.Create()` N times. The frontend adds a semi-transparent red dot to `#bar`; clicking it opens a modal with `payload` + `quantity` inputs.

**Tech Stack:** Go 1.25, Gin, uber/fx, zerolog, testify/mockery v2, vanilla JS (no build step).

---

## File Map

| Action | Path | Responsibility |
|--------|------|---------------|
| Create | `app/handlers/floodmessage/dto.go` | `floodRequest` struct + `toMessage()` |
| Create | `app/handlers/floodmessage/floodmessage.go` | Handler, route registration, `go:generate` directive |
| Create | `app/handlers/floodmessage/mocks/CreateMessageUseCase.go` | Mockery v2 mock (generated) |
| Create | `app/handlers/floodmessage/floodmessage_test.go` | HTTP handler tests |
| Modify | `cmd/api/modules/message.go` | Wire flood handler into fx |
| Modify | `web/index.html` | Red dot + modal UI |

---

## Task 1: DTO and handler skeleton

**Files:**
- Create: `app/handlers/floodmessage/dto.go`
- Create: `app/handlers/floodmessage/floodmessage.go`

- [ ] **Step 1: Create `app/handlers/floodmessage/dto.go`**

```go
package handlers

import (
	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/pkg/clock"
)

type floodRequest struct {
	Payload string `json:"payload" binding:"required"`
}

func (r floodRequest) toMessage() entities.Message {
	return entities.Message{
		Body:      []byte(r.Payload),
		CreatedAt: clock.Now(),
	}
}
```

- [ ] **Step 2: Create `app/handlers/floodmessage/floodmessage.go`**

```go
package handlers

import (
	"net/http"
	"strconv"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

const maxQuantity = 10000

//go:generate mockery --name=CreateMessageUseCase --output=mocks --outpkg=mocks
type CreateMessageUseCase interface {
	Create(entities.Message) (string, error)
}

type FloodMessageHandler struct {
	usecase CreateMessageUseCase
}

func NewFloodMessageHandler(usecase CreateMessageUseCase) *FloodMessageHandler {
	return &FloodMessageHandler{usecase: usecase}
}

func RegisterFloodMessageRoutes(router *gin.Engine, h *FloodMessageHandler) {
	router.POST("/messages/flood", h.Handle)
}

func (h *FloodMessageHandler) Handle(c *gin.Context) {
	quantityStr := c.DefaultQuery("quantity", "1")
	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quantity must be at least 1"})
		return
	}
	if quantity > maxQuantity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quantity exceeds max (10000)"})
		return
	}

	var req floodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"queued": quantity})

	go func() {
		for i := 0; i < quantity; i++ {
			if _, err := h.usecase.Create(req.toMessage()); err != nil {
				log.Error().Err(err).Msg("flood: failed to enqueue message")
			}
		}
	}()
}
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./app/handlers/floodmessage/...
```

Expected: no output (clean build).

---

## Task 2: Mock + tests

**Files:**
- Create: `app/handlers/floodmessage/mocks/CreateMessageUseCase.go`
- Create: `app/handlers/floodmessage/floodmessage_test.go`

- [ ] **Step 1: Generate the mock**

```bash
cd app/handlers/floodmessage && go generate ./...
```

Expected: creates `app/handlers/floodmessage/mocks/CreateMessageUseCase.go`.

If `mockery` is not on PATH, install it first:
```bash
go install github.com/vektra/mockery/v2@latest
```

- [ ] **Step 2: Create `app/handlers/floodmessage/floodmessage_test.go`**

```go
package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	handlers "github.com/adrianozp/gaardrail/app/handlers/floodmessage"
	"github.com/adrianozp/gaardrail/app/handlers/floodmessage/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupRouter(h *handlers.FloodMessageHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handlers.RegisterFloodMessageRoutes(r, h)
	return r
}

func jsonBody(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}

func TestHandle_ValidRequest_Returns202WithQueuedCount(t *testing.T) {
	done := make(chan struct{}, 3)
	uc := mocks.NewCreateMessageUseCase(t)
	uc.On("Create", mock.AnythingOfType("entities.Message")).
		Run(func(_ mock.Arguments) { done <- struct{}{} }).
		Return("id", nil)

	r := setupRouter(handlers.NewFloodMessageHandler(uc))
	req := httptest.NewRequest(http.MethodPost, "/messages/flood?quantity=3",
		jsonBody(t, map[string]string{"payload": "hello"}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var resp map[string]int
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 3, resp["queued"])

	// drain goroutine calls
	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-t.Context().Done():
			t.Fatal("goroutine did not call Create 3 times")
		}
	}
}

func TestHandle_DefaultQuantityIsOne(t *testing.T) {
	done := make(chan struct{}, 1)
	uc := mocks.NewCreateMessageUseCase(t)
	uc.On("Create", mock.AnythingOfType("entities.Message")).
		Run(func(_ mock.Arguments) { done <- struct{}{} }).
		Return("id", nil)

	r := setupRouter(handlers.NewFloodMessageHandler(uc))
	req := httptest.NewRequest(http.MethodPost, "/messages/flood",
		jsonBody(t, map[string]string{"payload": "x"}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var resp map[string]int
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 1, resp["queued"])

	select {
	case <-done:
	case <-t.Context().Done():
		t.Fatal("goroutine did not call Create")
	}
}

func TestHandle_QuantityExceedsMax_Returns400(t *testing.T) {
	uc := mocks.NewCreateMessageUseCase(t)
	r := setupRouter(handlers.NewFloodMessageHandler(uc))

	req := httptest.NewRequest(http.MethodPost, "/messages/flood?quantity=10001",
		jsonBody(t, map[string]string{"payload": "x"}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "quantity exceeds max (10000)", resp["error"])
}

func TestHandle_QuantityBelowOne_Returns400(t *testing.T) {
	uc := mocks.NewCreateMessageUseCase(t)
	r := setupRouter(handlers.NewFloodMessageHandler(uc))

	req := httptest.NewRequest(http.MethodPost, "/messages/flood?quantity=0",
		jsonBody(t, map[string]string{"payload": "x"}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "quantity must be at least 1", resp["error"])
}

func TestHandle_MissingPayload_Returns400(t *testing.T) {
	uc := mocks.NewCreateMessageUseCase(t)
	r := setupRouter(handlers.NewFloodMessageHandler(uc))

	req := httptest.NewRequest(http.MethodPost, "/messages/flood?quantity=1",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
```

- [ ] **Step 3: Run the tests — expect PASS**

```bash
go test ./app/handlers/floodmessage/... -v
```

Expected: all 5 tests PASS.

- [ ] **Step 4: Commit**

```bash
git add app/handlers/floodmessage/
git commit -m "feat: add flood message handler with tests"
```

---

## Task 3: Wire into fx

**Files:**
- Modify: `cmd/api/modules/message.go`

- [ ] **Step 1: Replace the full content of `cmd/api/modules/message.go`**

```go
package modules

import (
	createMessageHandler "github.com/adrianozp/gaardrail/app/handlers/createmessage"
	floodHandler "github.com/adrianozp/gaardrail/app/handlers/floodmessage"
	queuerepo "github.com/adrianozp/gaardrail/app/repositories/queue"
	"github.com/adrianozp/gaardrail/app/usecases/createmessage"
	"go.uber.org/fx"
)

func MessageFactories() fx.Option {
	return fx.Provide(
		createMessageHandler.NewCreateMessageHandler,
		createmessage.NewCreateMessageUseCase,
		floodHandler.NewFloodMessageHandler,
	)
}

func MessageInjections() fx.Option {
	return fx.Provide(
		func(uc createmessage.CreateMessageUseCase) createMessageHandler.CreateMessageUseCase { return uc },
		func(repo queuerepo.Queue) createmessage.Queue { return repo },
		func(uc createmessage.CreateMessageUseCase) floodHandler.CreateMessageUseCase { return uc },
	)
}

func MessageEndpoints() fx.Option {
	return fx.Module("message",
		fx.Invoke(createMessageHandler.RegisterCreateMessageRoutes),
		fx.Invoke(floodHandler.RegisterFloodMessageRoutes),
	)
}
```

- [ ] **Step 2: Verify the full build**

```bash
go build ./...
```

Expected: no output (clean build).

- [ ] **Step 3: Run all tests**

```bash
go test ./...
```

Expected: all packages PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/api/modules/message.go
git commit -m "feat: wire flood handler into fx modules"
```

---

## Task 4: Frontend — red dot + modal

**Files:**
- Modify: `web/index.html`

- [ ] **Step 1: Add the flood dot to `#bar`**

In `web/index.html`, locate the closing `</div>` of `#actions` and add the flood dot immediately after it, still inside `#bar`:

```html
    <div id="actions">
      <span id="error-msg"></span>
      <button id="btn-cancel" onclick="cancelEdit()"  style="display:none">✕ Cancelar</button>
      <button id="btn-save"   onclick="saveParams()"  style="display:none">✓ Salvar</button>
    </div>
    <div id="flood-dot" onclick="openFlood()" title="flood"></div>
```

Add the flood dot CSS inside `<style>`:

```css
    #flood-dot {
      width: 7px;
      height: 7px;
      border-radius: 50%;
      background: #da363344;
      border: 1px solid #da363388;
      cursor: pointer;
      flex-shrink: 0;
      transition: background 0.2s, border-color 0.2s;
    }
    #flood-dot:hover {
      background: #da363388;
      border-color: #da3633cc;
    }

    #flood-backdrop {
      display: none;
      position: fixed;
      inset: 0;
      background: #0d1117cc;
      z-index: 100;
      align-items: center;
      justify-content: center;
    }
    #flood-backdrop.open { display: flex; }

    #flood-modal {
      background: #161b22;
      border: 1px solid #30363d;
      border-radius: 6px;
      padding: 18px;
      width: 240px;
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
    #flood-modal h3 {
      color: #da3633;
      font-size: 10px;
      letter-spacing: 1px;
      text-transform: uppercase;
      font-family: monospace;
    }
    .flood-field { display: flex; flex-direction: column; gap: 3px; }
    .flood-field label {
      color: #8b949e;
      font-size: 9px;
      text-transform: uppercase;
      letter-spacing: 0.5px;
    }
    .flood-field input {
      background: #0d1117;
      border: 1px solid #30363d;
      border-radius: 3px;
      padding: 4px 7px;
      color: #e6edf3;
      font-family: monospace;
      font-size: 12px;
      outline: none;
    }
    .flood-field input:focus { border-color: #58a6ff; }
    #flood-msg { font-size: 11px; min-height: 14px; }
    #btn-flood {
      background: #da3633;
      border: none;
      color: #fff;
      font-family: monospace;
      font-size: 12px;
      padding: 5px 12px;
      border-radius: 4px;
      cursor: pointer;
    }
    #btn-flood:hover { background: #f85149; }
    #btn-flood:disabled { opacity: 0.5; cursor: default; }
```

- [ ] **Step 2: Add the modal HTML at the end of `<body>`, before `<script>`**

```html
  <div id="flood-backdrop" onclick="backdropClick(event)">
    <div id="flood-modal">
      <h3>⚡ flood queue</h3>
      <div class="flood-field">
        <label>Payload</label>
        <input type="text" id="flood-payload" placeholder="mensagem de teste">
      </div>
      <div class="flood-field">
        <label>Quantity (max 10 000)</label>
        <input type="number" id="flood-qty" value="1000" min="1" max="10000">
      </div>
      <span id="flood-msg"></span>
      <button id="btn-flood" onclick="sendFlood()">▶ flood</button>
    </div>
  </div>
```

- [ ] **Step 3: Add flood JS functions inside `<script>`, before `loadParams()`**

```js
    function openFlood() {
      document.getElementById('flood-msg').textContent = '';
      document.getElementById('flood-msg').style.color = '';
      document.getElementById('btn-flood').disabled = false;
      document.getElementById('btn-flood').textContent = '▶ flood';
      document.getElementById('flood-payload').disabled = false;
      document.getElementById('flood-qty').disabled = false;
      document.getElementById('flood-backdrop').classList.add('open');
    }

    function closeFlood() {
      document.getElementById('flood-backdrop').classList.remove('open');
    }

    function backdropClick(e) {
      if (e.target === document.getElementById('flood-backdrop')) closeFlood();
    }

    async function sendFlood() {
      const payload = document.getElementById('flood-payload').value;
      const qty = parseInt(document.getElementById('flood-qty').value, 10);

      if (!payload) {
        showFloodMsg('payload obrigatório', '#da3633');
        return;
      }
      if (isNaN(qty) || qty < 1 || qty > 10000) {
        showFloodMsg('quantity: 1–10000', '#da3633');
        return;
      }

      setFloodLoading(true);
      try {
        const res = await fetch(`/messages/flood?quantity=${qty}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ payload }),
        });
        const data = await res.json().catch(() => ({}));
        if (!res.ok) {
          showFloodMsg(data.error || `Erro ${res.status}`, '#da3633');
          setFloodLoading(false);
          return;
        }
        showFloodMsg(`✓ queued: ${data.queued}`, '#238636');
        setTimeout(closeFlood, 2000);
      } catch (_) {
        showFloodMsg('erro de rede', '#da3633');
        setFloodLoading(false);
      }
    }

    function setFloodLoading(loading) {
      const btn = document.getElementById('btn-flood');
      btn.disabled = loading;
      btn.textContent = loading ? 'sending...' : '▶ flood';
      document.getElementById('flood-payload').disabled = loading;
      document.getElementById('flood-qty').disabled = loading;
    }

    function showFloodMsg(msg, color) {
      const el = document.getElementById('flood-msg');
      el.textContent = msg;
      el.style.color = color;
    }
```

- [ ] **Step 4: Add `Escape` key handler for the modal**

In the existing `document.addEventListener('keydown', ...)` block, add a close-on-escape check:

```js
    document.addEventListener('keydown', e => {
      if (e.key === 'Escape') closeFlood();
      if (e.key === 'Enter' && editing) saveParams();
    });
```

- [ ] **Step 5: Manual smoke test**

Start the server locally and:
1. Open the UI — confirm a tiny red dot appears at the right side of the topbar
2. Click the dot — confirm modal opens with payload and quantity fields
3. Click outside the modal — confirm it closes
4. Press Escape — confirm it closes
5. Submit with empty payload — confirm validation error
6. Submit with quantity 99999 — confirm client-side validation error
7. Submit valid request (payload: `"test"`, quantity: `5`) — confirm `✓ queued: 5` and modal auto-closes

- [ ] **Step 6: Commit**

```bash
git add web/index.html
git commit -m "feat: add flood trigger dot and modal to frontend"
```
