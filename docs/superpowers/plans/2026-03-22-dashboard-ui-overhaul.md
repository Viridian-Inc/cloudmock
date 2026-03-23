# CloudMock Console Phase 2: Dashboard UI Overhaul

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the topology screen into the CloudMock Console's primary 3-panel layout with health-colored nodes, traffic-sized edges, an advanced request explorer, and an AI-powered service inspector.

**Architecture:** The existing Topology.tsx is split into three focused components — RequestExplorer (left rail), TopologyCanvas (center), and ServiceInspector (right panel). The canvas renders nodes with dynamic sizing/coloring from live metrics. All data flows from existing admin API endpoints with new query params for advanced filtering.

**Tech Stack:** Preact, SVG, SSE (useSSE hook), existing admin API endpoints

**Spec:** `docs/superpowers/specs/2026-03-22-cloudmock-console-product-spec.md` — Section 2.1

---

## File Structure

| Action | File | Responsibility |
|--------|------|---------------|
| Create | `dashboard/src/pages/Console.tsx` | 3-panel layout orchestrator (replaces Topology as main view) |
| Create | `dashboard/src/components/RequestExplorer.tsx` | Left rail: advanced filters, saved views, live tail |
| Create | `dashboard/src/components/TopologyCanvas.tsx` | Center: interactive SVG graph with health coloring |
| Create | `dashboard/src/components/ServiceInspector.tsx` | Right panel: service detail, AI summary, SLO burn |
| Create | `dashboard/src/components/FilterBar.tsx` | Reusable filter controls (dropdowns, text inputs, toggles) |
| Create | `dashboard/src/components/HealthBadge.tsx` | Service health indicator (green/yellow/red) |
| Modify | `dashboard/src/App.tsx` | Add /console route, keep /topology as legacy |
| Modify | `dashboard/src/api.ts` | Add saved views API, extended request filters |
| Modify | `dashboard/src/components/Sidebar.tsx` | Add Console nav item |
| Modify | `cloudmock/pkg/admin/api.go` | Add /api/views CRUD, extended request filters |
| Create | `dashboard/src/hooks/useFilters.ts` | Filter state management hook |
| Create | `dashboard/src/styles/console.css` | Console-specific styles |

---

### Task 1: Backend — Saved Views API + Extended Request Filters

**Files:**
- Modify: `cloudmock/pkg/admin/api.go`

- [ ] **Step 1: Add saved views storage and routes**

Add to API struct:
```go
views   []SavedView
viewsMu sync.RWMutex
```

Add route: `a.mux.HandleFunc("/api/views", a.handleViews)`

Add types and handler:
```go
type SavedView struct {
    ID        string            `json:"id"`
    Name      string            `json:"name"`
    Filters   map[string]string `json:"filters"`
    CreatedBy string            `json:"created_by"`
    CreatedAt string            `json:"created_at"`
}
```

Handler supports GET (list), POST (create), DELETE (by id query param).

- [ ] **Step 2: Add extended request filter params**

Add to `handleRequests`: `tenant_id`, `org_id`, `user_id`, `min_latency_ms`, `max_latency_ms`, `from`, `to` (time range as RFC3339).

Add to `RequestFilter` struct: `TenantID`, `OrgID`, `UserID`, `MinLatencyMs`, `MaxLatencyMs`, `From`, `To`.

Update `RecentFiltered` to check these new fields against `RequestEntry.RequestHeaders` for tenant/org/user and against `LatencyMs`/`Timestamp` for latency/time filters.

- [ ] **Step 3: Build and test**

Run: `cd cloudmock && go build ./... && go test ./pkg/admin/ ./pkg/gateway/ -count=1`
Expected: All pass

- [ ] **Step 4: Commit**

```bash
git add pkg/admin/api.go pkg/gateway/logging.go
git commit -m "feat: saved views API + extended request filters (tenant, org, time range, latency)"
```

---

### Task 2: API Client — New Endpoints + Filter Types

**Files:**
- Modify: `dashboard/src/api.ts`
- Create: `dashboard/src/hooks/useFilters.ts`

- [ ] **Step 1: Add API functions**

```typescript
// Saved views
export function getViews() { return api<SavedView[]>('/api/views'); }
export function createView(view: Omit<SavedView, 'id' | 'created_at'>) {
  return api<SavedView>('/api/views', { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify(view) });
}
export function deleteView(id: string) {
  return api('/api/views?id=' + id, { method: 'DELETE' });
}

// Extended request filters
export interface RequestFilters {
  service?: string; path?: string; method?: string;
  caller_id?: string; action?: string; error?: boolean;
  trace_id?: string; level?: string; limit?: number;
  tenant_id?: string; org_id?: string; user_id?: string;
  min_latency_ms?: number; max_latency_ms?: number;
  from?: string; to?: string;
}

export function getFilteredRequests(filters: RequestFilters) {
  const params = new URLSearchParams();
  for (const [k, v] of Object.entries(filters)) {
    if (v !== undefined && v !== '') params.set(k, String(v));
  }
  return api<any[]>('/api/requests?' + params);
}

// SLO status
export function getSLOStatus() { return api<any>('/api/slo'); }
```

- [ ] **Step 2: Create useFilters hook**

```typescript
// dashboard/src/hooks/useFilters.ts
import { useState, useCallback } from 'preact/hooks';
import type { RequestFilters } from '../api';

export function useFilters(initial: Partial<RequestFilters> = {}) {
  const [filters, setFilters] = useState<RequestFilters>({ level: 'app', limit: 100, ...initial });
  const setFilter = useCallback((key: keyof RequestFilters, value: any) => {
    setFilters(prev => ({ ...prev, [key]: value }));
  }, []);
  const clearFilters = useCallback(() => {
    setFilters({ level: 'app', limit: 100 });
  }, []);
  const hasActiveFilters = Object.entries(filters).some(([k, v]) =>
    v !== undefined && v !== '' && k !== 'level' && k !== 'limit'
  );
  return { filters, setFilter, clearFilters, hasActiveFilters };
}
```

- [ ] **Step 3: Commit**

```bash
git add dashboard/src/api.ts dashboard/src/hooks/useFilters.ts
git commit -m "feat: API client for saved views, extended filters, useFilters hook"
```

---

### Task 3: Console Layout — 3-Panel Orchestrator

**Files:**
- Create: `dashboard/src/pages/Console.tsx`
- Create: `dashboard/src/styles/console.css`
- Modify: `dashboard/src/App.tsx`
- Modify: `dashboard/src/components/Sidebar.tsx`

- [ ] **Step 1: Create Console.tsx**

3-panel flex layout:
```
┌────────────┬───────────────────────────┬────────────┐
│  Left Rail │     Center Canvas         │  Right     │
│  300px     │     flex: 1               │  350px     │
│  (explorer)│     (topology SVG)        │  (inspect) │
└────────────┴───────────────────────────┴────────────┘
```

State management:
- `selectedNode` — currently selected topology node
- `selectedRequest` — currently selected request from explorer
- `filters` — from useFilters hook
- `panelState` — which panels are open (left, right, both)

Import existing topology rendering logic from Topology.tsx (extract canvas SVG into TopologyCanvas).

- [ ] **Step 2: Create console.css**

Styles for:
- `.console-layout` — 3-panel flex container, height: 100%
- `.console-left` — 300px, border-right, overflow-y auto
- `.console-center` — flex: 1, overflow: hidden, position: relative
- `.console-right` — 350px, border-left, overflow-y auto
- `.console-panel-toggle` — collapse/expand buttons
- Panel slide animations

- [ ] **Step 3: Wire into App.tsx and Sidebar**

Add `/console` route to App.tsx hash router.
Add "Console" nav item to Sidebar with topology icon.
Make `/console` the default landing (redirect from `/`).

- [ ] **Step 4: Build and verify**

Run: `cd cloudmock/dashboard && npm run build`
Expected: Builds successfully

- [ ] **Step 5: Commit**

```bash
git add dashboard/src/pages/Console.tsx dashboard/src/styles/console.css dashboard/src/App.tsx dashboard/src/components/Sidebar.tsx
git commit -m "feat: Console 3-panel layout with left rail, center canvas, right inspector"
```

---

### Task 4: Request Explorer — Advanced Left Rail

**Files:**
- Create: `dashboard/src/components/RequestExplorer.tsx`
- Create: `dashboard/src/components/FilterBar.tsx`

- [ ] **Step 1: Create FilterBar.tsx**

Reusable filter controls:
- `FilterDropdown` — select with label
- `FilterInput` — text input with label
- `FilterToggle` — checkbox toggle
- `FilterTimeRange` — from/to date-time pickers
- `FilterLatencyRange` — min/max ms inputs

All emit filter changes via `onFilterChange(key, value)` callback.

- [ ] **Step 2: Create RequestExplorer.tsx**

Sections:
1. **Header** — "Request Explorer" + collapse button + saved view selector
2. **Quick filters** — service, status, method dropdowns (always visible)
3. **Advanced filters** — collapsible section with:
   - Route/path prefix
   - Tenant ID, Org ID, User ID
   - Feature flag
   - Latency range
   - Time window
   - Error-only toggle
4. **Live indicator** — green dot with LIVE/PAUSED + pause button
5. **Request list** — compact cards (method badge, path, service, status, latency, timestamp)
6. **Saved views** — bottom section with saved/load/delete

Click request → emits `onSelectRequest(req)` which:
- Highlights the service node in the canvas
- Opens the inspector for that service

SSE integration: same as existing RequestPanel (filter infra-level events).

- [ ] **Step 3: Build and verify**

Run: `cd cloudmock/dashboard && npm run build`
Expected: Builds

- [ ] **Step 4: Commit**

```bash
git add dashboard/src/components/RequestExplorer.tsx dashboard/src/components/FilterBar.tsx
git commit -m "feat: RequestExplorer with advanced filters, saved views, live tail"
```

---

### Task 5: Topology Canvas — Health-Colored Interactive Graph

**Files:**
- Create: `dashboard/src/components/TopologyCanvas.tsx`
- Create: `dashboard/src/components/HealthBadge.tsx`

- [ ] **Step 1: Extract canvas from Topology.tsx into TopologyCanvas.tsx**

Move the SVG rendering (groups, nodes, edges, minimap) into a standalone component.

Props:
```typescript
interface TopologyCanvasProps {
  sse: SSEState;
  selectedNodeId?: string;
  highlightService?: string;
  onSelectNode: (node: TopoNode) => void;
}
```

- [ ] **Step 2: Add health-based node coloring**

Fetch SLO status (`/api/slo`) and map to node colors:
- Green: SLO healthy, burn rate < 1x
- Yellow: SLO warning, burn rate 1-5x
- Red: SLO breaching, burn rate > 5x or error rate > threshold

Node rendering:
- `fill` based on health color (gradient)
- Node `width` scaled by log(request_count) from stats
- Small health dot in top-right corner of node rect

- [ ] **Step 3: Add edge health coloring**

Edge rendering:
- `stroke-width` scaled by log(call_count) from topology edge data
- `stroke` color: green (< P95 target), yellow (> P95), red (> P99 or errors)
- Dashed stroke for IaC-only edges (no observed traffic)
- Animated dash for live traffic

- [ ] **Step 4: Create HealthBadge.tsx**

Small colored indicator:
```tsx
function HealthBadge({ status }: { status: 'healthy' | 'warning' | 'critical' }) {
  const colors = { healthy: '#10B981', warning: '#F59E0B', critical: '#EF4444' };
  return <span style={{ width: 8, height: 8, borderRadius: '50%', background: colors[status] }} />;
}
```

- [ ] **Step 5: Build and verify**

Run: `cd cloudmock/dashboard && npm run build`
Expected: Builds

- [ ] **Step 6: Commit**

```bash
git add dashboard/src/components/TopologyCanvas.tsx dashboard/src/components/HealthBadge.tsx
git commit -m "feat: TopologyCanvas with health-colored nodes, traffic-sized edges"
```

---

### Task 6: Service Inspector — Right Panel

**Files:**
- Create: `dashboard/src/components/ServiceInspector.tsx`

- [ ] **Step 1: Create ServiceInspector.tsx**

Replaces and enhances NodeDetailDrawer as a persistent right panel (not a drawer overlay).

Sections:
1. **Service header** — name, type icon, health badge, group
2. **Health dashboard** — stat cards (requests, errors, P50/P95/P99, SLO burn)
3. **AI summary** — one-paragraph summary from explain data for the most recent slow/error request
4. **Tabs:**
   - Overview (stats + latency bars + activity sparkline)
   - Requests (filtered by service, inline expand with explain)
   - Traces (filtered, inline waterfall)
   - Connections (inbound/outbound with latency + click-to-navigate)
   - SLO (burn rate chart, budget remaining, threshold config)
   - Blast Radius (upstream/downstream impact list)

Reuse existing components from NodeDetailDrawer.tsx:
- `RequestInlineDetail` with Explain tab
- `WaterfallTimeline`
- `ConnectionsTab`
- `OverviewTab` stats

- [ ] **Step 2: Build and verify**

Run: `cd cloudmock/dashboard && npm run build`
Expected: Builds

- [ ] **Step 3: Commit**

```bash
git add dashboard/src/components/ServiceInspector.tsx
git commit -m "feat: ServiceInspector right panel with AI summary, SLO burn, blast radius"
```

---

### Task 7: Integration + Build + Deploy

**Files:**
- Modify: `dashboard/src/pages/Console.tsx` (wire all components)
- Modify: `cloudmock/pkg/dashboard/dist/` (rebuild embedded assets)

- [ ] **Step 1: Wire Console.tsx with all three panels**

Connect:
- RequestExplorer `onSelectRequest` → highlights canvas node + opens inspector
- TopologyCanvas `onSelectNode` → opens inspector for that service
- ServiceInspector `onSelectNode` (from connections tab) → updates canvas + explorer filter
- Shared filter state from useFilters

- [ ] **Step 2: Full build**

Run: `cd cloudmock/dashboard && npm run build`
Expected: Builds successfully with no TS errors

- [ ] **Step 3: Copy to embedded dist**

Run: `cp -r dashboard/dist/* pkg/dashboard/dist/`

- [ ] **Step 4: Go build and full test**

Run: `cd cloudmock && go build ./... && go test ./... 2>&1 | tail -5`
Expected: All packages pass

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: CloudMock Console Phase 2 — 3-panel topology with health coloring, advanced explorer, AI inspector"
```

- [ ] **Step 6: Restart and verify**

Restart services, open dashboard at `/console`, verify:
1. Left rail shows filtered app requests with live tail
2. Center canvas shows health-colored nodes with traffic-sized edges
3. Click a node → right panel opens with service details
4. Click a request → highlights node + opens inspector
5. Filters work across all panels
6. Saved views persist
7. SLO burn rate visible in inspector
8. AI summary shows for degraded services

---

## Verification Checklist

1. `cd cloudmock && go test ./...` — all packages pass
2. `cd cloudmock/dashboard && npm run build` — no errors
3. Console loads at `/#/console` with 3-panel layout
4. Request explorer shows live BFF/app requests (no DynamoDB spam)
5. Topology nodes colored by SLO health
6. Edge thickness reflects traffic volume
7. Click request → node highlights + inspector opens
8. Filter by tenant_id → only that tenant's requests shown
9. Save a view → reload → view persists
10. AI summary visible when clicking a node with recent errors
