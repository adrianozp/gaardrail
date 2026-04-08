# Flood Endpoint & UI — Design Spec

**Date:** 2026-04-06

---

## Overview

Add a flood endpoint that enqueues N messages in fire-and-forget fashion, plus a minimally-visible UI trigger in the frontend.

---

## Backend

### Route

```
POST /messages/flood?quantity=N
```

### Request

Body (same as `POST /messages`):
```json
{ "payload": "string" }
```

Query param:
- `quantity` — integer, 1–10000 (default: 1)
- Values > 10000 → `400 Bad Request: {"error": "quantity exceeds max (10000)"}`
- Values < 1 → `400 Bad Request: {"error": "quantity must be at least 1"}`

### Response

```json
{ "queued": 1000 }
```

Returned immediately (HTTP 202 Accepted) before any messages are enqueued.

### Behavior

1. Validate `quantity` and bind JSON body.
2. Respond with `202 {"queued": N}`.
3. Goroutine enqueues N messages by calling `CreateMessageUseCase.Create()` in a loop. Each message gets its own UUID (handled by the use case). Errors per message are logged but do not surface to the caller.

### Code structure

- New package: `app/handlers/floodmessage/`
  - `floodmessage.go` — handler struct, `Handle` method, `RegisterFloodMessageRoutes`
  - `dto.go` — `floodRequest` (same `payload` field as createmessage DTO)
- Declares its own `CreateMessageUseCase` interface (same shape as `createmessage` handler — Go convention), satisfied by `createmessage.CreateMessageUseCase`
- Wired in `cmd/api/modules/message.go`:
  - `RegisterFloodMessageRoutes` added to `MessageEndpoints()`
  - `NewFloodMessageHandler` added to `MessageFactories()`
  - A new binding added to `MessageInjections()`: `func(uc createmessage.CreateMessageUseCase) floodHandler.CreateMessageUseCase { return uc }`

---

## Frontend

### Trigger

A 7px semi-transparent red dot (`background: #da363355; border: 1px solid #da363388`) placed at the far right of `#bar`, after the existing action buttons. It has no label and no tooltip — intentionally subtle.

### Modal

Clicking the dot opens a centered modal with a dark backdrop (`#0d1117cc`):

```
┌─────────────────────────┐
│ ⚡ FLOOD QUEUE           │
│                         │
│ PAYLOAD                 │
│ [________________________] │
│                         │
│ QUANTITY (max 10 000)   │
│ [1000____________________] │
│                         │
│ [      ▶ flood         ] │  ← red background
└─────────────────────────┘
```

- All text in monospace, dark theme matching existing UI
- Click outside or `Escape` closes without action

### Interaction states

| State      | Button text     | Inputs   |
|------------|-----------------|----------|
| Idle       | `▶ flood`       | enabled  |
| Sending    | `sending...`    | disabled |
| Success    | `✓ queued: N`   | disabled |
| Error      | `▶ flood`       | enabled  |

- Success: green text, modal auto-closes after 2s
- Error: red error message shown inside modal, inputs re-enabled

---

## Constraints

- Max quantity: 10,000 (enforced server-side; frontend also caps the input)
- Flood is fire-and-forget — no progress tracking, no cancellation
- Payload is required (same binding rule as `POST /messages`)
