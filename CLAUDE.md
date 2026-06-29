# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

PagerOps is a macOS desktop app (Wails v2) for monitoring and managing PagerDuty incidents. The backend is Go; the frontend is Svelte 4 + TypeScript + Vite. The Go `App` struct's exported methods are auto-bound and callable from the frontend via generated bindings in `frontend/wailsjs/`.

## Commands

All commands run from the repo root unless noted.

- **Dev (hot reload):** `wails dev` — rebuilds Go and serves the frontend with live reload. The VS Code default build task runs `wails generate module && wails dev -race 2>&1 | tee race.log` (race detector + log to `race.log`).
- **Build:** `wails build` — produces a packaged app in `build/bin/`.
- **Regenerate frontend bindings:** `wails generate module` — run this after changing exported `App` method signatures or any struct returned to the frontend, so `frontend/wailsjs/go/` types stay in sync.
- **Frontend type-check:** `cd frontend && npm run check` (svelte-check).
- **Frontend only (rarely needed directly):** `cd frontend && npm run dev | build`.
- **Go tests:** none exist yet. Use `go test ./...` if you add any.

Requires the `wails` CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`), Go 1.23, and Node/npm. macOS-only runtime features below.

## Architecture

### Backend (Go)

- **`app.go`** — the `App` struct holds all runtime state and exposes every frontend-callable method. This is the central coordination point and by far the largest file. Concurrency-heavy: most state is guarded by dedicated mutexes (`mu`, `pollMu`, `userPollMu`, `resolvedPollMu`, `lastIncidentsMu`, `previousOpenMu`, etc.). Respect the existing lock for each field — don't reuse `a.mu` for state that has its own mutex.
- **`store/`** — PagerDuty API client wrapper.
  - `client.go` — `Client` wraps `go-pagerduty` behind an **`APIQueue`**: every API call is serialized through a single worker goroutine (`processAPIQueue`) that self-throttles to `maxCallsPerMinute` (600). **Always call PagerDuty through `Client` methods**, never `pd` directly, so rate limiting and request typing in `executeAPICall`'s switch are honored. Adding a new API call means adding a case to that switch and a typed request struct.
  - `client_post.go` — write operations (acknowledge, resolve, create note) + `FormatNoteContent`, which renders structured notekit responses/tags into the final note string.
  - `incidents.go` — `convertToIncidentData` (PagerDuty → `database.IncidentData`) and dedup. Note: `UpdatedAt` is sourced from `LastStatusChangeAt`.
  - `types.go` — service-config and sidebar (alerts/notes) types shared with the frontend.
- **`database/schema.go`** — SQLite (`mattn/go-sqlite3`) wrapper. All access goes through `DB` methods guarded by `db.mu`. Tables: `incidents`, `incident_alerts`, `incident_notes`, `incident_sidebar_metadata`, and `app_state` (key/value persistence for things like `latest_resolved_date`, `browser_redirect`, `assigned_incidents_<userID>`). DB lives at `~/Library/Application Support/pager-ops/incidents.db`. Incidents are cleared on startup and re-fetched.
- **`notification.go`** — `NotificationManager` runs worker goroutines for sound (`say` for default / `afplay` for custom files in `assets/sounds/`) and browser-redirect, both rate-limited. Notifications use `terminal-notifier` (falls back to `osascript`). All macOS-specific.
- **`logger.go`** — file logger at `~/Library/Logs/pager-ops/app.log` with dedup of repeated messages and 10MB rotation.

### Polling model (core data flow)

`startup` initializes the DB, keyring, notification manager, and (if an API key exists) starts **three independent polling loops**, each its own goroutine + ticker:

1. **Service polling** (`StartPolling`, 3s) → `fetchServiceIncidents` — open incidents for selected services.
2. **User polling** (`StartUserPolling`, 4s) → `fetchUserIncidents` — incidents assigned to the current user (only when `filterByUser`).
3. **Resolved polling** (`StartResolvedPolling`, 1m) → `fetchResolvedIncidentsSince` — resolved incidents since the last seen date.

All three converge on `processAndUpdateIncidents`, which diffs against DB state, marks stale incidents resolved, persists, and emits the `incidents-updated` Wails event. Reliability is layered: a `CircuitBreaker` (opens after 5 failures, exponential backoff), a `RateLimitTracker` (separate from the queue's own limiter), and a `UserCache` (1h TTL for the current user ID/email). Shutdown is coordinated via `shutdownChan` + `shutdownWg` with a 10s timeout.

**Filtering semantics** (see `GetOpenIncidents` in `app.go`): assigned-mode ON shows the **union** of selected-service incidents and user-assigned incidents; assigned-mode OFF shows only selected services. When both serviceIDs and a userID are passed to `FetchOpenIncidents`, it's an **intersection** instead. Disabled services (`ServiceConfig.Disabled`) are filtered out here.

### Frontend (Svelte)

- **`frontend/src/stores/incidents.ts`** — the heart of the frontend. Svelte stores for incidents/services/UI plus the load functions. It listens for the backend `incidents-updated`, `api-key-configured`, and `services-config-updated` events via `EventsOn` and reloads accordingly. The `isPolling` flag suppresses loading spinners during background refreshes. Sidebar (alert/note) data is fetched lazily and re-fetched only when alert count or `updated_at` changes.
- Backend methods are imported from `../../wailsjs/go/main/App`; runtime helpers from `../../wailsjs/runtime/runtime`. These are generated — never hand-edit.
- Components live in `frontend/src/components/`; `App.svelte` wires ToolBar / Header / tabbed `IncidentPanel`s / side `Panel` / `Settings`.

### Backend ↔ frontend contract

- Communication is two-way: frontend calls bound `App` methods; backend pushes updates via `runtime.EventsEmit(a.ctx, "<event>", ...)`. When adding a backend-driven update, emit an event and add a matching `EventsOn` listener in the relevant store.
- The PagerDuty API key is stored in the OS keyring (`99designs/keyring`, service name `PagerOps`), not in the DB or config files.

### Service configuration (notekit)

Users upload a JSON services config (`UploadServicesConfig`) defining which PagerDuty services to monitor and optional **notekit** templates — `questions` and `tags` (single/multiple choice) per service that drive the structured note-taking UI. `ServiceConfig.ID` is `interface{}` because it may be a string, an array of strings, or a numeric ID; the many `switch id := service.ID.(type)` blocks in `app.go` exist to handle all three. See `service_json/services.notekit.sample.json` for the format.
