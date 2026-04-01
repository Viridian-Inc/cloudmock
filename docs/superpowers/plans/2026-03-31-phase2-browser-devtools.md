# Phase 2: Browser Devtools Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Drop Tauri from neureaux-devtools. Embed the Preact SPA into the cloudmock Go binary. Users open `localhost:4500` in their browser. One binary = gateway + admin API + devtools UI.

**Architecture:** Build neureaux-devtools as a standalone Preact SPA (`pnpm build` → `dist/`). Copy `dist/` into `cloudmock/pkg/dashboard/dist/`. The existing `embed.FS` in `pkg/dashboard/dashboard.go` serves it. Merge admin API and SPA onto a single port (`:4500`) to eliminate CORS. Remove old cloudmock dashboard.

**Tech Stack:** Preact, Vite, Go `embed.FS`, existing `pkg/dashboard/` infrastructure

**Spec:** `docs/superpowers/specs/2026-03-31-v1-saas-launch-design.md` Phase 2

---

## File Structure

```
neureaux-devtools/
├── package.json                    # Remove @tauri-apps deps
├── vite.config.ts                  # Remove Tauri-specific config
├── src/
│   ├── lib/connection.tsx          # Remove Tauri imports, use pure HTTP
│   ├── hooks/use-topology-metrics.ts  # Remove Tauri invoke for SQLite
│   ├── components/source-bar/     # Remove Tauri source events
│   └── views/profiler/index.tsx   # "Coming soon" empty state

cloudmock/
├── pkg/dashboard/
│   ├── dashboard.go               # Merge admin API + SPA on single port
│   └── dist/                      # Embedded neureaux-devtools build output
├── cmd/gateway/main.go            # Serve merged UI+API on :4500
├── Makefile                       # Update build-dashboard target
├── Dockerfile                     # Update dashboard build stage
└── dashboard/                     # DELETE old React dashboard
```

---

## Task 1: Remove Tauri dependencies from neureaux-devtools

**Files:**
- Modify: `neureaux-devtools/package.json`

- [ ] **Step 1: Read package.json**

Read `neureaux-devtools/package.json` to identify all @tauri-apps dependencies and tauri-related scripts.

- [ ] **Step 2: Remove Tauri deps and scripts**

Remove from `dependencies`:
- `@tauri-apps/api`

Remove from `devDependencies`:
- `@tauri-apps/cli`

Remove from `scripts`:
- `tauri` (and any tauri-related scripts)

- [ ] **Step 3: Run pnpm install to update lockfile**

Run: `cd neureaux-devtools && pnpm install`

- [ ] **Step 4: Verify build still works**

Run: `pnpm build`
Expected: Vite builds successfully to `dist/`

- [ ] **Step 5: Commit**

```bash
git add package.json pnpm-lock.yaml
git commit -m "chore: remove Tauri dependencies from package.json"
```

---

## Task 2: Remove Tauri imports from frontend source

**Files:**
- Modify: `neureaux-devtools/src/lib/connection.tsx`
- Modify: `neureaux-devtools/src/hooks/use-topology-metrics.ts`
- Modify: `neureaux-devtools/src/components/source-bar/source-bar.tsx` (if Tauri imports exist)

- [ ] **Step 1: Read connection.tsx and identify Tauri imports**

Read `src/lib/connection.tsx`. Find all `import` from `@tauri-apps/api` and the try/catch blocks around them. These are already wrapped with fallbacks.

- [ ] **Step 2: Remove Tauri imports from connection.tsx**

Remove the dynamic `import('@tauri-apps/api/...')` calls and their try/catch wrappers. Keep only the fallback behavior (HTTP polling to admin API). The connection provider should:
- Poll `GET /api/health` on the admin base URL every 3 seconds
- No Tauri event listeners
- No Tauri invoke calls

- [ ] **Step 3: Remove Tauri imports from use-topology-metrics.ts**

Read `src/hooks/use-topology-metrics.ts`. Remove any `invoke()` calls for SQLite persistence. Metric data comes from the admin API (`/api/metrics`, `/api/traces`). Historical persistence was in Tauri's SQLite — without it, metrics are ephemeral per session (acceptable for browser mode; the admin API has its own DataPlane persistence).

- [ ] **Step 4: Check source-bar for Tauri imports**

Read `src/components/source-bar/`. If it imports from `@tauri-apps`, remove those imports and keep fallback behavior.

- [ ] **Step 5: Verify build**

Run: `cd neureaux-devtools && pnpm build`
Expected: Zero TypeScript errors, builds to `dist/`

- [ ] **Step 6: Run tests**

Run: `pnpm test`
Expected: All 269 tests pass

- [ ] **Step 7: Commit**

```bash
git add src/
git commit -m "refactor: remove all Tauri imports, use pure HTTP for browser mode"
```

---

## Task 3: Remove src-tauri directory

**Files:**
- Delete: `neureaux-devtools/src-tauri/` (entire directory)

- [ ] **Step 1: Remove src-tauri**

```bash
cd neureaux-devtools && rm -rf src-tauri/
```

- [ ] **Step 2: Remove Tauri references from vite.config.ts**

Read `vite.config.ts`. Remove any Tauri-specific configuration (e.g., `clearScreen`, Tauri dev server settings). Keep the Preact plugin and proxy config.

- [ ] **Step 3: Verify build**

Run: `pnpm build`
Expected: Builds successfully

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore: remove src-tauri/ directory (browser-only mode)"
```

---

## Task 4: Set Profiler to "Coming soon" empty state

**Files:**
- Modify: `neureaux-devtools/src/views/profiler/index.tsx`

- [ ] **Step 1: Read current profiler view**

Read `src/views/profiler/index.tsx` to understand current state (skeleton with fake flame graph or empty).

- [ ] **Step 2: Replace with honest empty state**

Replace the view content with a clear empty state:
```tsx
export function ProfilerView() {
  return (
    <div class="profiler-view">
      <div class="profiler-empty-state">
        <h2>Profiler</h2>
        <p>CPU, heap, and goroutine profiling.</p>
        <p class="profiler-coming-soon">Coming soon</p>
      </div>
    </div>
  );
}
```

Remove any fake/sample data generation. Keep the CSS file if it has useful styles.

- [ ] **Step 3: Verify build + tests**

Run: `pnpm build && pnpm test`

- [ ] **Step 4: Commit**

```bash
git add src/views/profiler/
git commit -m "feat(profiler): replace skeleton with honest 'Coming soon' empty state"
```

---

## Task 5: Update cloudmock to serve merged UI + API on single port

**Files:**
- Modify: `cloudmock/pkg/dashboard/dashboard.go`
- Modify: `cloudmock/cmd/gateway/main.go`

- [ ] **Step 1: Read current dashboard.go**

Read `pkg/dashboard/dashboard.go` to understand how the SPA is embedded and served. It uses `embed.FS` to embed `dist/` and serves it as a file server.

- [ ] **Step 2: Read current main.go server setup**

Read `cmd/gateway/main.go` to understand how the admin API (`:4599`) and dashboard (`:4500`) are served on separate ports.

- [ ] **Step 3: Merge admin API routes under the dashboard handler**

Modify `dashboard.go` or `main.go` so that a single HTTP server on `:4500` handles:
- `/api/*` routes → admin API handlers (existing mux)
- Everything else → SPA static files (embed.FS)

This eliminates CORS. The admin API mux already handles all `/api/` paths. The dashboard handler serves the SPA for everything else (with `index.html` fallback for client-side routing).

Keep `:4599` as an optional separate admin API port for backward compatibility, but the primary UI experience is `:4500` with everything on one origin.

- [ ] **Step 4: Update the SPA's API base URL detection**

The devtools frontend in `src/lib/api.ts` has a `detectAdminBase()` function. When running on `:4500` (same origin as API), it should return `''` (empty string = same origin). Update the detection:
- Port 1420 (Vite dev) → `http://localhost:4599`
- Port 4500 (production) → `''` (same origin)
- Port 4501 → `http://localhost:4599`

- [ ] **Step 5: Build and test**

```bash
cd cloudmock && go build ./cmd/gateway/
```
Expected: Compiles

- [ ] **Step 6: Commit**

```bash
git add pkg/dashboard/ cmd/gateway/main.go
git commit -m "feat: serve admin API + devtools UI on single port :4500 (no CORS)"
```

---

## Task 6: Update Makefile and Dockerfile for new dashboard build

**Files:**
- Modify: `cloudmock/Makefile`
- Modify: `cloudmock/Dockerfile`

- [ ] **Step 1: Read current Makefile build-dashboard target**

Read `Makefile` to find the `build-dashboard` target that builds from `dashboard/`.

- [ ] **Step 2: Update build-dashboard to build from neureaux-devtools**

Change the target to:
```makefile
.PHONY: build-dashboard
build-dashboard: ## Build devtools UI from neureaux-devtools
	cd ../neureaux-devtools && pnpm install && pnpm build
	rm -rf pkg/dashboard/dist
	cp -r ../neureaux-devtools/dist pkg/dashboard/dist
```

- [ ] **Step 3: Update Dockerfile**

Read `Dockerfile`. Update the dashboard build stage to:
- Copy `neureaux-devtools/` instead of `dashboard/`
- Run `pnpm install && pnpm build` in the neureaux-devtools directory
- Copy `dist/` to `pkg/dashboard/dist/`

- [ ] **Step 4: Build and verify**

```bash
make build-dashboard
go build ./cmd/gateway/
```

- [ ] **Step 5: Commit**

```bash
git add Makefile Dockerfile
git commit -m "build: update dashboard build to use neureaux-devtools SPA"
```

---

## Task 7: Remove old cloudmock dashboard

**Files:**
- Delete: `cloudmock/dashboard/` (entire directory)

- [ ] **Step 1: Verify new dashboard works**

Start the gateway and open `http://localhost:4500` in a browser. Verify the neureaux-devtools UI loads with all 12 views.

- [ ] **Step 2: Remove old dashboard**

```bash
cd cloudmock && rm -rf dashboard/
```

- [ ] **Step 3: Verify build still works**

```bash
go build ./cmd/gateway/
```

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore: remove old cloudmock dashboard (replaced by neureaux-devtools)"
```

---

## Task 8: Update devtools API detection for single-origin mode

**Files:**
- Modify: `neureaux-devtools/src/lib/api.ts`

- [ ] **Step 1: Read current detectAdminBase()**

Read `src/lib/api.ts` to understand the current port-based admin base URL detection.

- [ ] **Step 2: Update detection for :4500 single-origin**

```typescript
function detectAdminBase(): string {
  if (typeof window === 'undefined') return '';
  const port = window.location.port;
  // Vite dev server → proxy or direct to admin API
  if (port === '1420' || port === '4501') {
    return `${window.location.protocol}//${window.location.hostname}:4599`;
  }
  // Production: UI + API on same origin (:4500) → no base needed
  return '';
}
```

- [ ] **Step 3: Rebuild and test**

```bash
cd neureaux-devtools && pnpm build && pnpm test
```

- [ ] **Step 4: Copy to cloudmock**

```bash
cd cloudmock && make build-dashboard
```

- [ ] **Step 5: Commit**

```bash
cd neureaux-devtools && git add src/lib/api.ts && git commit -m "fix: detect single-origin mode when served on :4500"
```

---

## Task 9: Final verification

- [ ] **Step 1: Build everything**

```bash
cd cloudmock && make build-dashboard && go build ./cmd/gateway/
```

- [ ] **Step 2: Start gateway and verify UI**

```bash
./gateway &
```
Open `http://localhost:4500` in browser. Verify:
- All 12 views load
- Activity view shows requests (if cloudmock is processing traffic)
- Topology view renders
- Profiler shows "Coming soon"
- No CORS errors in browser console
- API calls to `/api/*` work on same origin

- [ ] **Step 3: Run all tests**

```bash
cd neureaux-devtools && pnpm test  # 269+ tests
cd cloudmock && make test-all       # 1,876+ tests
```

- [ ] **Step 4: Commit any final fixes**

```bash
git commit -m "feat: Phase 2 complete — browser-only devtools embedded in cloudmock binary"
```

---

## Verification

1. `http://localhost:4500` serves the neureaux-devtools UI in browser
2. `/api/*` routes work on same origin (no CORS)
3. All 12 views functional (Profiler = "Coming soon")
4. No Tauri dependencies in neureaux-devtools
5. No `src-tauri/` directory
6. `pnpm test` — all frontend tests pass
7. `make test-all` — all Go tests pass
8. `go build ./cmd/gateway/` — compiles clean
9. Docker build works with new dashboard
