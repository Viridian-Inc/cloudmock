# Topology & Request Wiring TODO

## Data Flow Issues
- [ ] Add E2E integration test: mock API response → normalize → filter → buildFlows → verify flows rendered
- [ ] Add E2E test: SSE event received → Activity view updates → topology panel updates
- [ ] Add debug panel: show raw API response count, filtered count, flow count in topology sidebar
- [x] Guard against empty serviceName in filterRequestsByEdgeServices (returns all requests if empty)
- [x] Add console.log after buildFlows showing how many flows built vs fallback

## Request Trace Panel Improvements
- [ ] Full request waterfall diagram (DNS → TCP → TLS → TTFB → response for each hop)
- [x] Auto-refresh flows every 5s (already in fetchFlows polling, verify it works)
- [x] Show "last updated X seconds ago" timestamp
- [x] Better empty state with debug link: "Run `curl localhost:4599/api/requests?level=all` to verify API"

## Waterfall Diagram
- [ ] New component: RequestWaterfall showing the full timeline of a request
- [ ] For traced requests: show each span as a bar (client → BFF → DynamoDB → response)
- [ ] For untraced requests: show single bar with timing breakdown
- [ ] Reuse existing Waterfall component from src/views/traces/waterfall.tsx

## Custom Dashboards (Datadog-style)
- [x] Query DSL parser (avg:latency_ms{service:dynamodb})
- [x] Query executor (backend)
- [x] Widget types: timeseries, single-stat, gauge
- [x] Grid layout with resize
- [x] Widget editor
- [x] Preset dashboards with labels
- [x] Favorite/hide/unsave
- [x] localStorage persistence
- [ ] Table widget type
- [ ] Heatmap widget type
- [ ] Dashboard sharing (export/import JSON)
- [ ] Backend API persistence (POST /api/dashboards)

## Integration Tests Needed
- [ ] Test: API returns requests → normalizeRequestEvent produces correct fields
- [ ] Test: filterRequestsByEdgeServices matches BFF node to autotend-bff requests
- [ ] Test: filterRequestsByEdgeServices matches Lambda node to handler requests
- [ ] Test: buildFlows groups by trace_id correctly with outbound calls
- [ ] Test: buildFlows handles requests without trace_id from connected services
- [ ] Test: mergeFlows preserves existing flows across polls
- [ ] Test: time range filter doesn't exclude fresh requests
- [ ] Test: full pipeline mock → fetch → normalize → filter → build → render count
