# Frontend Design — Gaardrail

**Date:** 2026-04-06

## Overview

Single-page frontend for the Gaardrail PID controller, served directly by the Go backend. Allows viewing and editing PID parameters while observing the system behavior through an embedded Grafana dashboard.

## Goals

- View all PID controller parameters at a glance
- Edit parameters inline without leaving the page
- Observe system metrics in real time via embedded Grafana

## Non-Goals

- Multiple pages or routes
- Authentication/authorization
- Frontend build toolchain (no npm, no bundler)
- Multiple dashboard views

## Architecture

A single file `web/index.html` embedded into the Go binary via `//go:embed`. The Go backend registers a new route `GET /` that serves this file. No new dependencies are required.

```
web/
  index.html     ← single file, HTML + CSS + JS inline

internal/httpserver/httpserver.go  ← add GET / handler
```

## Layout

Two vertical sections filling the full viewport height:

1. **Top bar** (fixed height ~56px): displays all PID parameters side by side with an "Edit" button on the right.
2. **Grafana area** (flex: 1): an `<iframe>` filling the remaining height, embedding the provisioned Grafana dashboard.

## PID Parameter Bar

### View mode

Displays all seven parameters in read-only form:

| Field | Description |
|-------|-------------|
| `kp` | Proportional gain |
| `ki` | Integral gain |
| `kd` | Derivative gain |
| `setpoint` | Target value |
| `min` | Controller output minimum |
| `max` | Controller output maximum |
| `i_clamp` | Integral anti-windup clamp |

Values are loaded via `GET /pid` on page load.

### Edit mode

Clicking **✏ Editar** switches the bar to edit mode:
- Each value becomes a `<input type="number">` field in place
- The bar's bottom border turns blue as a visual indicator
- Two action buttons appear on the right: **✕ Cancelar** and **✓ Salvar**

Cancelling restores original values and returns to view mode with no API call.

### Save

Clicking **✓ Salvar** sends `PATCH /pid` with the body containing only the fields whose values differ from the original. On success, the bar border flashes green for 2 seconds and returns to view mode. On error, the border flashes red for 2 seconds and stays in edit mode so the user can retry or cancel.

## Grafana Embed

The dashboard is embedded via:

```html
<iframe
  src="http://localhost:3000/d/flood-test/flood-test?kiosk"
  style="width:100%;height:100%;border:none"
></iframe>
```

- Dashboard UID: `flood-test` (already provisioned in `flood-test/grafana/dashboards/flood.json`)
- `?kiosk` hides the Grafana header and nav bar
- Grafana is configured with `GF_AUTH_ANONYMOUS_ENABLED=true` so no login is required

## API Usage

| Call | When |
|------|------|
| `GET /pid` | Page load — populate bar values |
| `PATCH /pid` | User clicks Salvar — send changed fields only |

The PATCH body omits fields whose value has not changed, taking advantage of the fact that all `updatePIDParamsRequest` fields are optional pointers.

## Backend Changes

1. Add `//go:embed web/index.html` to `internal/httpserver/httpserver.go` (or a new `web` package)
2. Register `GET /` on the gin router to serve the embedded file with `Content-Type: text/html`
3. Add `.superpowers/` to `.gitignore`

## Styling

- Dark theme matching the existing tooling aesthetic (`#0d1117` background, `#161b22` surfaces, `#58a6ff` accent)
- No external CSS libraries — all styles inline in the `<style>` block
- Monospace font throughout

## Error Handling

- `GET /pid` failure: display "Erro ao carregar parâmetros" in the bar with a retry button
- `PATCH /pid` 400/422: bar border flashes red, stays in edit mode
- `PATCH /pid` network error: same as above
