# E2E Data Wiring — Topology Request Panel

## Problem
Requests don't appear in the topology side panel even though the API returns data.

## Data Flow
```
cloudmock gateway (:4566)
  → LoggingMiddleware captures every request
  → Writes to RequestLog (circular buffer, 1000 entries)
  → Admin API (:4599) serves /api/requests?level=all

Devtools topology panel:
  → Fetches /api/requests?level=all&limit=200
  → Normalizes PascalCase → snake_case (TraceID → trace_id)
  → Filters by outbound/inbound edge services
  → buildFlows() groups by trace_id
  → Falls back to direct flow creation if buildFlows returns empty
  → Renders in left sidebar as clickable list
```

## Known Issues

### Data Path
- [ ] Verify: does `api()` in the panel actually call the right URL?
  - `getAdminBase()` returns `http://localhost:4599` on port 1420
  - The `api()` helper prepends this base
  - Check: does the panel use `api()` or direct `fetch()`?
- [ ] Verify: PascalCase normalization covers ALL fields
  - API returns: ID, TraceID, Service, Method, Path, StatusCode, LatencyMs, Timestamp
  - Normalized to: id, trace_id, service, method, path, status_code, latency_ms, timestamp
- [ ] Verify: edge-based service filter works for each node type
  - External nodes (BFF, GraphQL): svcKey = ID suffix (bff-service, graphql-server)
  - Lambda nodes (ms:Order): svcKey = autotend-order-handler
  - AWS nodes (svc:dynamodb): svcKey = dynamodb
  - Plugin nodes: svcKey = ID suffix (stripe, sendgrid)

### Rendering
- [ ] Verify: buildFlows produces non-empty results
  - Requires trace_id to be non-empty (confirmed: normalization provides this)
  - Groups by trace_id, takes first as primary
  - If all requests share same trace_id, they become one flow (not ideal)
- [ ] Verify: fallback flow creation works when buildFlows fails
- [ ] Verify: sidebar renders flows with correct status/method/path
- [ ] Verify: clicking sidebar item sets highlightedFlowId

### E2E Tests Needed
- [ ] Test: fetch → normalize → filter → buildFlows → render for each node type
- [ ] Test: BFF Service with real BFF traffic
- [ ] Test: DynamoDB node with direct AWS CLI traffic
- [ ] Test: Lambda node (Order/Attendance/etc.)
- [ ] Test: empty state when no traffic
- [ ] Test: AI Explain fetch for a real request ID
- [ ] Test: Replay button sends request to cloudmock gateway

### Integration Test Plan
```typescript
// src/views/topology/__tests__/request-trace-panel.integration.test.ts

describe('RequestTracePanel', () => {
  it('fetches and normalizes requests from API', async () => {
    // Mock fetch to return PascalCase data
    // Verify normalization produces snake_case
  });

  it('filters requests by outbound edge services for BFF', () => {
    // Given: BFF node with edges to dynamodb, cognito, s3
    // When: filter runs on requests with service=dynamodb
    // Then: dynamodb requests are included
  });

  it('builds flows from trace-grouped requests', () => {
    // Given: 3 requests sharing same trace_id
    // When: buildFlows runs
    // Then: produces 1 flow with 1 primary + 2 outbound
  });

  it('falls back to direct flows when buildFlows returns empty', () => {
    // Given: requests without trace_ids
    // When: buildFlows returns []
    // Then: direct flows created from raw requests
  });

  it('renders sidebar with request list', () => {
    // Given: 5 flows
    // When: component renders
    // Then: 5 sidebar items visible
  });

  it('shows AI explain on request click', async () => {
    // Given: selected request
    // When: Explain tab clicked
    // Then: fetches /api/explain/{id}
  });
});
```

## Waterfall Diagram Enhancement
- [ ] Add full request waterfall to the right panel
  - When a request is selected, fetch its trace spans: `/api/traces/{traceId}`
  - Render the same waterfall component from the Traces view
  - Shows: Client → BFF → DynamoDB → response with timing bars
  - Reuse `Waterfall` component from `src/views/traces/waterfall.tsx`

## Client Node "0 Requests" UX
- [ ] Expo App / Admin Portal / Client Portal show "0 requests" because
      cloudmock only sees outbound AWS API calls, not inbound HTTP from clients
- [ ] Fix: show a helpful message instead of empty:
      "Client requests are captured via the SDK. Add @cloudmock/node to
       your app to see inbound requests here."
- [ ] If SDK is connected (source:connected event), show SDK-captured requests

## Infrastructure Overview (Lambda/API GW/AppSync)
- [ ] New view or topology enhancement: "Infrastructure" panel
- [ ] For Lambda: show function list from /api/resources/lambda
      - Invocation count (from traces where RootService=lambda)
      - Error rate, cold start %, avg duration, memory config
- [ ] For API Gateway: show APIs from /api/resources/apigateway
      - Request count, 4xx/5xx rate, latency
- [ ] For AppSync: show GraphQL APIs
      - Query/mutation counts, resolver errors
- [ ] For DynamoDB: show tables from /api/resources/dynamodb
      - Read/write capacity, throttle count
- [ ] Health summary: green/yellow/red per resource
- [ ] Could be a new "🏗️ Infrastructure" view in the icon rail
      or an overlay on the topology nodes
