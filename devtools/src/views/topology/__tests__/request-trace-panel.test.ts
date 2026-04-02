import { describe, it, expect } from 'vitest';
import {
  normalizeRequestEvent,
  getServiceKey,
  filterRequestsByEdgeServices,
  buildFlows,
  buildFallbackFlows,
  buildResponseSummary,
  mergeFlows,
  computeTimeRange,
  filterFlowsByMethod,
  type RequestFlow,
} from '../request-trace-utils';
import type { TopoNode, TopoEdge } from '../index';
import type { RequestEvent } from '../../../lib/types';

/* ------------------------------------------------------------------ */
/*  Helpers                                                            */
/* ------------------------------------------------------------------ */

function makeNode(overrides: Partial<TopoNode> = {}): TopoNode {
  return {
    id: 'svc:billing',
    label: 'Billing',
    service: 'billing',
    type: 'microservice',
    group: 'Services',
    ...overrides,
  };
}

function makeEdge(source: string, target: string, overrides: Partial<TopoEdge> = {}): TopoEdge {
  return { source, target, ...overrides };
}

function makeRequest(overrides: Partial<RequestEvent> = {}): RequestEvent {
  return {
    id: 'req-1',
    trace_id: 'trace-1',
    service: 'billing',
    action: 'CreateOrder',
    method: 'POST',
    path: '/v1/orders',
    status_code: 200,
    latency_ms: 42,
    timestamp: '2025-06-01T12:00:00.000Z',
    ...overrides,
  };
}

/* ================================================================== */
/*  1. Normalization: PascalCase -> snake_case                        */
/* ================================================================== */

describe('normalizeRequestEvent', () => {
  it('maps PascalCase fields to snake_case', () => {
    const raw = {
      ID: 'abc-123',
      TraceID: 'trace-xyz',
      Service: 'billing',
      Action: 'CreateOrder',
      Method: 'POST',
      Path: '/v1/orders',
      StatusCode: 201,
      LatencyMs: 55,
      Timestamp: '2025-06-01T12:00:00Z',
      CallerID: 'caller-1',
      RequestHeaders: { 'x-auth': 'token' },
      ResponseBody: '{"ok":true}',
      Level: 'app',
    };

    const result = normalizeRequestEvent(raw);

    expect(result.id).toBe('abc-123');
    expect(result.trace_id).toBe('trace-xyz');
    expect(result.service).toBe('billing');
    expect(result.action).toBe('CreateOrder');
    expect(result.method).toBe('POST');
    expect(result.path).toBe('/v1/orders');
    expect(result.status_code).toBe(201);
    expect(result.latency_ms).toBe(55);
    expect(result.timestamp).toBe('2025-06-01T12:00:00Z');
    expect(result.caller_id).toBe('caller-1');
    expect(result.request_headers).toEqual({ 'x-auth': 'token' });
    expect(result.response_body).toBe('{"ok":true}');
    expect(result.source).toBe('app');
  });

  it('falls back to snake_case fields when PascalCase is missing', () => {
    const raw = {
      id: 'def-456',
      trace_id: 'trace-abc',
      service: 'auth',
      action: 'Login',
      method: 'POST',
      path: '/auth/login',
      status_code: 200,
      latency_ms: 10,
      timestamp: '2025-06-01T13:00:00Z',
    };

    const result = normalizeRequestEvent(raw);

    expect(result.id).toBe('def-456');
    expect(result.trace_id).toBe('trace-abc');
    expect(result.service).toBe('auth');
    expect(result.status_code).toBe(200);
  });

  it('uses defaults when both PascalCase and snake_case are missing', () => {
    const result = normalizeRequestEvent({});

    expect(result.id).toBe('');
    expect(result.trace_id).toBe('');
    expect(result.service).toBe('');
    expect(result.status_code).toBe(200);
    expect(result.latency_ms).toBe(0);
    expect(result.source).toBe('infra');
  });
});

/* ================================================================== */
/*  2. Service Key Derivation                                         */
/* ================================================================== */

describe('getServiceKey', () => {
  it('returns the service property for a normal microservice node', () => {
    const node = makeNode({ id: 'svc:billing', service: 'billing' });
    expect(getServiceKey(node)).toBe('billing');
  });

  it('strips svc: prefix from id when service is empty', () => {
    const node = makeNode({ id: 'svc:payments', service: '', label: 'Payments' });
    expect(getServiceKey(node)).toBe('payments');
  });

  it('strips ms: prefix from id when service is empty', () => {
    const node = makeNode({ id: 'ms:calendar', service: '', label: 'Calendar' });
    expect(getServiceKey(node)).toBe('calendar');
  });

  it('returns ID suffix after colon for external nodes', () => {
    const node = makeNode({ id: 'external:bff-service', service: 'external', label: 'BFF' });
    expect(getServiceKey(node)).toBe('bff-service');
  });

  it('returns ID suffix after colon for plugin nodes', () => {
    const node = makeNode({ id: 'plugin:auth', service: 'plugin', label: 'Auth Plugin' });
    expect(getServiceKey(node)).toBe('auth');
  });

  it('falls back to label for external node without colon in id', () => {
    const node = makeNode({ id: 'externalnode', service: 'external', label: 'My External' });
    expect(getServiceKey(node)).toBe('My External');
  });

  it('returns full id when no prefix to strip and no service', () => {
    const node = makeNode({ id: 'my-service', service: '' });
    expect(getServiceKey(node)).toBe('my-service');
  });
});

/* ================================================================== */
/*  3. Request Filtering by Edge Services                             */
/* ================================================================== */

describe('filterRequestsByEdgeServices', () => {
  const billingNode = makeNode({ id: 'svc:billing', service: 'billing' });
  const edges: TopoEdge[] = [
    makeEdge('external:bff', 'svc:billing'),      // inbound
    makeEdge('svc:billing', 'svc:dynamodb'),       // outbound
    makeEdge('svc:billing', 'svc:notifications'),  // outbound
  ];

  it('includes requests whose service directly matches the node', () => {
    const requests = [
      makeRequest({ id: 'r1', service: 'billing' }),
      makeRequest({ id: 'r2', service: 'unrelated' }),
    ];
    const filtered = filterRequestsByEdgeServices(requests, 'billing', billingNode, edges);
    expect(filtered.map((r) => r.id)).toContain('r1');
    expect(filtered.map((r) => r.id)).not.toContain('r2');
  });

  it('includes requests matching outbound edge services', () => {
    // Edge targets like "dynamodb" (without svc: prefix) are used as-is since
    // they don't contain a colon. When they do contain a colon (e.g. "svc:dynamodb"),
    // the code splits at ":" and takes the prefix ("svc"), not the service name.
    // Use edges without prefix to test the outbound service matching path.
    const edgesWithPlainTargets: TopoEdge[] = [
      makeEdge('external:bff', 'svc:billing'),
      makeEdge('svc:billing', 'dynamodb'),
      makeEdge('svc:billing', 'notifications'),
    ];
    const requests = [
      makeRequest({ id: 'r1', service: 'dynamodb' }),
      makeRequest({ id: 'r2', service: 'notifications' }),
    ];
    const filtered = filterRequestsByEdgeServices(requests, 'billing', billingNode, edgesWithPlainTargets);
    expect(filtered.length).toBe(2);
  });

  it('includes requests matching inbound edge services', () => {
    const requests = [
      makeRequest({ id: 'r1', service: 'bff' }),
    ];
    // The inbound edge source is "external:bff" — after fix, splits to "bff"
    // which correctly matches the request service
    const filtered = filterRequestsByEdgeServices(requests, 'billing', billingNode, edges);
    expect(filtered.length).toBe(1);
  });

  it('matches stripped lambda handler names', () => {
    const lambdaNode = makeNode({ id: 'svc:autotend-order-handler', service: 'autotend-order-handler' });
    const requests = [
      makeRequest({ id: 'r1', service: 'order' }),
    ];
    const filtered = filterRequestsByEdgeServices(
      requests, 'autotend-order-handler', lambdaNode, [],
    );
    // stripped = "order", rs = "order" -> match
    expect(filtered.length).toBe(1);
  });

  it('returns empty array when no requests match', () => {
    const requests = [
      makeRequest({ id: 'r1', service: 'completely-unrelated' }),
    ];
    const filtered = filterRequestsByEdgeServices(requests, 'billing', billingNode, edges);
    expect(filtered.length).toBe(0);
  });

  // --- Bug 1 regression: colon-prefixed edge IDs must extract service name, not prefix ---

  it('includes requests matching outbound edge services with svc: prefix', () => {
    // Edge target "svc:dynamodb" should extract "dynamodb", not "svc"
    const requests = [
      makeRequest({ id: 'r1', service: 'dynamodb' }),
      makeRequest({ id: 'r2', service: 'notifications' }),
    ];
    const filtered = filterRequestsByEdgeServices(requests, 'billing', billingNode, edges);
    // edges include svc:billing → svc:dynamodb and svc:billing → svc:notifications
    expect(filtered.map((r) => r.id)).toContain('r1');
    expect(filtered.map((r) => r.id)).toContain('r2');
  });

  it('includes requests matching inbound edge services with external: prefix', () => {
    // Edge source "external:bff" should extract "bff", not "external"
    const requests = [
      makeRequest({ id: 'r1', service: 'bff' }),
    ];
    const filtered = filterRequestsByEdgeServices(requests, 'billing', billingNode, edges);
    expect(filtered.map((r) => r.id)).toContain('r1');
  });

  it('matches SDK-captured requests with autotend- prefix against topology node', () => {
    // SDK sends service: "autotend-bff", topology node is "bff-service"
    const bffNode = makeNode({ id: 'external:bff-service', service: 'external', label: 'BFF' });
    const requests = [
      makeRequest({ id: 'r1', service: 'autotend-bff', method: 'POST', path: '/v1/cart/items' }),
    ];
    const filtered = filterRequestsByEdgeServices(requests, 'bff-service', bffNode, []);
    expect(filtered.length).toBe(1);
  });

  it('handles ms: prefixed edge targets correctly', () => {
    const edgesWithMs: TopoEdge[] = [
      makeEdge('svc:billing', 'ms:order-service'),
    ];
    const requests = [
      makeRequest({ id: 'r1', service: 'order-service' }),
    ];
    const filtered = filterRequestsByEdgeServices(requests, 'billing', billingNode, edgesWithMs);
    expect(filtered.length).toBe(1);
  });
});

/* ================================================================== */
/*  4. buildFlows Grouping                                            */
/* ================================================================== */

describe('buildFlows', () => {
  const billingNode = makeNode({ id: 'svc:billing', service: 'billing' });
  const allNodes: TopoNode[] = [
    billingNode,
    makeNode({ id: 'external:bff', service: 'external', label: 'BFF' }),
  ];
  const edges: TopoEdge[] = [
    makeEdge('external:bff', 'svc:billing'),
  ];

  it('groups requests by trace_id into a single flow', () => {
    const requests: RequestEvent[] = [
      makeRequest({
        id: 'r1', trace_id: 'trace-1', service: 'billing',
        timestamp: '2025-06-01T12:00:00.000Z', latency_ms: 100,
      }),
      makeRequest({
        id: 'r2', trace_id: 'trace-1', service: 'dynamodb',
        timestamp: '2025-06-01T12:00:00.050Z', latency_ms: 30,
        action: 'PutItem',
      }),
    ];

    const flows = buildFlows(requests, 'billing', allNodes, edges);

    expect(flows.length).toBe(1);
    expect(flows[0].traceId).toBe('trace-1');
    expect(flows[0].outbound.length).toBe(1);
    expect(flows[0].outbound[0].service).toBe('dynamodb');
  });

  it('creates separate flows for different trace IDs', () => {
    const requests: RequestEvent[] = [
      makeRequest({ id: 'r1', trace_id: 'trace-1', service: 'billing', timestamp: '2025-06-01T12:00:00.000Z' }),
      makeRequest({ id: 'r2', trace_id: 'trace-2', service: 'billing', timestamp: '2025-06-01T12:01:00.000Z' }),
    ];

    const flows = buildFlows(requests, 'billing', allNodes, edges);

    expect(flows.length).toBe(2);
    expect(flows.map((f) => f.traceId).sort()).toEqual(['trace-1', 'trace-2']);
  });

  it('creates individual flows for requests without trace IDs when service matches', () => {
    const requests: RequestEvent[] = [
      makeRequest({ id: 'r1', trace_id: undefined, service: 'billing', timestamp: '2025-06-01T12:00:00.000Z' }),
      makeRequest({ id: 'r2', trace_id: undefined, service: 'billing', timestamp: '2025-06-01T12:01:00.000Z' }),
    ];

    const flows = buildFlows(requests, 'billing', allNodes, edges);

    expect(flows.length).toBe(2);
    expect(flows[0].traceId).toBeUndefined();
    expect(flows[1].traceId).toBeUndefined();
  });

  it('skips traceless requests from unconnected services', () => {
    const requests: RequestEvent[] = [
      makeRequest({ id: 'r1', trace_id: undefined, service: 'other-service' }),
    ];

    const flows = buildFlows(requests, 'billing', allNodes, edges);

    expect(flows.length).toBe(0);
  });

  // --- Bug 2 regression: untraceable requests from CONNECTED services should be included ---

  it('includes untraceable requests from outbound-connected services', () => {
    // billing has outbound edge to dynamodb — a dynamodb request without trace_id
    // should still appear as a flow
    const nodesWithDynamo: TopoNode[] = [
      ...allNodes,
      makeNode({ id: 'svc:dynamodb', service: 'dynamodb', label: 'DynamoDB' }),
    ];
    const edgesWithDynamo: TopoEdge[] = [
      ...edges,
      makeEdge('svc:billing', 'svc:dynamodb'),
    ];
    const requests: RequestEvent[] = [
      makeRequest({ id: 'r1', trace_id: undefined, service: 'dynamodb', method: 'POST', path: '/PutItem' }),
    ];

    const flows = buildFlows(requests, 'billing', nodesWithDynamo, edgesWithDynamo);

    expect(flows.length).toBe(1);
    expect(flows[0].path).toBe('/PutItem');
  });

  it('includes untraceable requests from inbound-connected services', () => {
    // bff sends inbound traffic to billing — a bff request without trace_id should appear
    const requests: RequestEvent[] = [
      makeRequest({ id: 'r1', trace_id: undefined, service: 'bff', method: 'GET', path: '/v1/billing' }),
    ];

    // edges already include: external:bff → svc:billing (inbound)
    const flows = buildFlows(requests, 'billing', allNodes, edges);

    expect(flows.length).toBe(1);
  });

  it('sorts flows by timestamp descending', () => {
    const requests: RequestEvent[] = [
      makeRequest({ id: 'r1', trace_id: 'trace-1', service: 'billing', timestamp: '2025-06-01T12:00:00.000Z' }),
      makeRequest({ id: 'r2', trace_id: 'trace-2', service: 'billing', timestamp: '2025-06-01T12:05:00.000Z' }),
      makeRequest({ id: 'r3', trace_id: 'trace-3', service: 'billing', timestamp: '2025-06-01T12:02:00.000Z' }),
    ];

    const flows = buildFlows(requests, 'billing', allNodes, edges);

    expect(flows[0].traceId).toBe('trace-2'); // latest
    expect(flows[1].traceId).toBe('trace-3');
    expect(flows[2].traceId).toBe('trace-1'); // earliest
  });

  it('returns empty array for empty requests', () => {
    expect(buildFlows([], 'billing', allNodes, edges)).toEqual([]);
  });

  it('resolves inbound source from edge + node labels', () => {
    const requests: RequestEvent[] = [
      makeRequest({
        id: 'r1', trace_id: 'trace-1', service: 'billing',
        timestamp: '2025-06-01T12:00:00.000Z',
      }),
    ];

    const flows = buildFlows(requests, 'billing', allNodes, edges);

    // inbound edge points from external:bff -> svc:billing
    // so inboundNode is the BFF node with label "BFF"
    expect(flows[0].inboundSource).toBe('BFF');
  });
});

/* ================================================================== */
/*  5. Fallback Flow Creation                                         */
/* ================================================================== */

describe('buildFallbackFlows', () => {
  it('creates one flow per request event', () => {
    const requests: RequestEvent[] = [
      makeRequest({ id: 'r1', trace_id: 'trace-1' }),
      makeRequest({ id: 'r2', trace_id: 'trace-2' }),
      makeRequest({ id: 'r3' }),
    ];

    const flows = buildFallbackFlows(requests);

    expect(flows.length).toBe(3);
  });

  it('limits to 50 flows', () => {
    const requests: RequestEvent[] = Array.from({ length: 100 }, (_, i) =>
      makeRequest({ id: `r-${i}` }),
    );

    const flows = buildFallbackFlows(requests);

    expect(flows.length).toBe(50);
  });

  it('maps fields correctly from RequestEvent', () => {
    const req = makeRequest({
      id: 'r-x',
      trace_id: 'trace-y',
      method: 'PUT',
      path: '/v1/resources/123',
      status_code: 204,
      latency_ms: 77,
      timestamp: '2025-06-01T14:00:00.000Z',
      caller_id: 'api-gateway',
    });

    const flows = buildFallbackFlows([req]);

    expect(flows[0].id).toBe('r-x');
    expect(flows[0].traceId).toBe('trace-y');
    expect(flows[0].method).toBe('PUT');
    expect(flows[0].path).toBe('/v1/resources/123');
    expect(flows[0].statusCode).toBe(204);
    expect(flows[0].durationMs).toBe(77);
    expect(flows[0].inboundSource).toBe('api-gateway');
    expect(flows[0].outbound).toEqual([]);
    expect(flows[0].responseSummary).toBe('');
  });

  it('generates fallback id from trace_id when id is empty', () => {
    const req = makeRequest({ id: '', trace_id: 'trace-fallback' });
    const flows = buildFallbackFlows([req]);
    expect(flows[0].id).toBe('trace-fallback');
  });

  it('generates fallback id from timestamp+service when both id and trace_id are empty', () => {
    const req = makeRequest({ id: '', trace_id: '', timestamp: '2025-06-01T12:00:00Z', service: 'billing' });
    const flows = buildFallbackFlows([req]);
    expect(flows[0].id).toBe('2025-06-01T12:00:00Z-billing');
  });
});

/* ================================================================== */
/*  6. Waterfall Tab Availability                                     */
/* ================================================================== */

describe('waterfall tab availability', () => {
  it('flow with traceId should be considered waterfall-capable', () => {
    const req = makeRequest({ id: 'r1', trace_id: 'trace-abc' });
    const flows = buildFallbackFlows([req]);
    expect(flows[0].traceId).toBe('trace-abc');
    // Component renders <Waterfall traceId={flow.traceId} /> when traceId is truthy
    expect(!!flows[0].traceId).toBe(true);
  });

  it('flow without traceId should not be waterfall-capable', () => {
    const req = makeRequest({ id: 'r1', trace_id: undefined });
    const flows = buildFallbackFlows([req]);
    expect(flows[0].traceId).toBeUndefined();
    expect(!!flows[0].traceId).toBe(false);
  });

  it('flow with empty string traceId should not be waterfall-capable', () => {
    const req = makeRequest({ id: 'r1', trace_id: '' });
    const flows = buildFallbackFlows([req]);
    // Empty string is falsy
    expect(!!flows[0].traceId).toBe(false);
  });
});

/* ================================================================== */
/*  buildResponseSummary                                              */
/* ================================================================== */

describe('buildResponseSummary', () => {
  it('returns item count for array body', () => {
    expect(buildResponseSummary('[1,2,3]')).toBe('3 items');
  });

  it('returns singular for single-element array', () => {
    expect(buildResponseSummary('[1]')).toBe('1 item');
  });

  it('returns nested array key count', () => {
    expect(buildResponseSummary('{"users":[{"id":1},{"id":2}]}')).toBe('2 users');
  });

  it('returns field count for plain object', () => {
    expect(buildResponseSummary('{"name":"test","age":10}')).toBe('2 fields');
  });

  it('returns empty string for undefined', () => {
    expect(buildResponseSummary(undefined)).toBe('');
  });

  it('returns empty string for invalid JSON', () => {
    expect(buildResponseSummary('not json')).toBe('');
  });
});

/* ================================================================== */
/*  7. mergeFlows — accumulate across fetches, don't replace          */
/* ================================================================== */

describe('mergeFlows', () => {
  function makeFlow(overrides: Partial<RequestFlow> = {}): RequestFlow {
    return {
      id: 'flow-1',
      requestId: 'flow-1',
      traceId: 'trace-1',
      method: 'POST',
      path: '/v1/orders',
      statusCode: 201,
      durationMs: 42,
      timestamp: '2025-06-01T12:00:00.000Z',
      inboundSource: 'BFF',
      inboundHeaders: undefined,
      outbound: [],
      responseSummary: '',
      ...overrides,
    };
  }

  it('keeps existing flows when incoming is empty', () => {
    const existing = [makeFlow({ id: 'f1' }), makeFlow({ id: 'f2' })];
    const result = mergeFlows(existing, []);
    expect(result.length).toBe(2);
    expect(result.map((f) => f.id)).toEqual(['f1', 'f2']);
  });

  it('adds new flows from incoming', () => {
    const existing = [makeFlow({ id: 'f1' })];
    const incoming = [makeFlow({ id: 'f2' }), makeFlow({ id: 'f3' })];
    const result = mergeFlows(existing, incoming);
    expect(result.length).toBe(3);
    expect(result.map((f) => f.id).sort()).toEqual(['f1', 'f2', 'f3']);
  });

  it('deduplicates by id — incoming updates existing', () => {
    const existing = [makeFlow({ id: 'f1', statusCode: 200 })];
    const incoming = [makeFlow({ id: 'f1', statusCode: 201 })];
    const result = mergeFlows(existing, incoming);
    expect(result.length).toBe(1);
    expect(result[0].statusCode).toBe(201); // incoming wins
  });

  it('preserves order: most recent first', () => {
    const existing = [
      makeFlow({ id: 'f1', timestamp: '2025-06-01T12:00:00.000Z' }),
    ];
    const incoming = [
      makeFlow({ id: 'f2', timestamp: '2025-06-01T12:05:00.000Z' }),
      makeFlow({ id: 'f3', timestamp: '2025-06-01T12:02:00.000Z' }),
    ];
    const result = mergeFlows(existing, incoming);
    expect(result[0].id).toBe('f2'); // latest
    expect(result[1].id).toBe('f3');
    expect(result[2].id).toBe('f1'); // earliest
  });

  it('caps at 200 flows to prevent unbounded growth', () => {
    const existing = Array.from({ length: 150 }, (_, i) =>
      makeFlow({ id: `existing-${i}`, timestamp: new Date(Date.UTC(2025, 5, 1, 12, 0, 0) + i * 1000).toISOString() }),
    );
    const incoming = Array.from({ length: 100 }, (_, i) =>
      makeFlow({ id: `incoming-${i}`, timestamp: new Date(Date.UTC(2025, 5, 1, 13, 0, 0) + i * 1000).toISOString() }),
    );
    const result = mergeFlows(existing, incoming);
    expect(result.length).toBe(200);
    // Most recent (incoming at 13:00+) should be first
    expect(result[0].id).toContain('incoming');
  });

  it('handles both arrays empty', () => {
    expect(mergeFlows([], [])).toEqual([]);
  });
});

/* ================================================================== */
/*  8. computeTimeRange — sliding window, not frozen                  */
/* ================================================================== */

describe('computeTimeRange', () => {
  it('returns a window ending at now when no flows exist', () => {
    const before = Date.now();
    const range = computeTimeRange([], 5 * 60 * 1000);
    const after = Date.now();
    expect(range.end).toBeGreaterThanOrEqual(before);
    expect(range.end).toBeLessThanOrEqual(after);
    expect(range.end - range.start).toBe(5 * 60 * 1000);
  });

  it('extends end to now even when latest flow is in the past', () => {
    const fiveMinAgo = Date.now() - 5 * 60 * 1000;
    const flows = [
      { id: 'f1', timestamp: new Date(fiveMinAgo).toISOString() } as RequestFlow,
    ];
    const range = computeTimeRange(flows, 5 * 60 * 1000);
    // End should be at or near Date.now(), not frozen at the flow timestamp
    expect(range.end).toBeGreaterThanOrEqual(Date.now() - 1000);
  });

  it('start is at least windowMs before end', () => {
    const now = Date.now();
    const flows = [
      { id: 'f1', timestamp: new Date(now - 1000).toISOString() } as RequestFlow,
    ];
    const range = computeTimeRange(flows, 5 * 60 * 1000);
    expect(range.end - range.start).toBeGreaterThanOrEqual(5 * 60 * 1000);
  });

  it('start encompasses oldest flow if within double the window', () => {
    const now = Date.now();
    const eightMinAgo = now - 8 * 60 * 1000;
    const flows = [
      { id: 'f1', timestamp: new Date(eightMinAgo).toISOString() } as RequestFlow,
      { id: 'f2', timestamp: new Date(now - 1000).toISOString() } as RequestFlow,
    ];
    const range = computeTimeRange(flows, 5 * 60 * 1000);
    // Should extend start back to include the oldest flow
    expect(range.start).toBeLessThanOrEqual(eightMinAgo);
  });
});

/* ================================================================== */
/*  9. filterFlowsByMethod — exclude methods like OPTIONS              */
/* ================================================================== */

describe('filterFlowsByMethod', () => {
  function makeFlow(overrides: Partial<RequestFlow> = {}): RequestFlow {
    return {
      id: 'flow-1',
      requestId: 'flow-1',
      traceId: 'trace-1',
      method: 'POST',
      path: '/v1/orders',
      statusCode: 201,
      durationMs: 42,
      timestamp: '2025-06-01T12:00:00.000Z',
      inboundSource: 'BFF',
      inboundHeaders: undefined,
      outbound: [],
      responseSummary: '',
      ...overrides,
    };
  }

  it('returns all flows when excludedMethods is empty', () => {
    const flows = [
      makeFlow({ id: 'f1', method: 'GET' }),
      makeFlow({ id: 'f2', method: 'OPTIONS' }),
      makeFlow({ id: 'f3', method: 'POST' }),
    ];
    const result = filterFlowsByMethod(flows, new Set());
    expect(result.length).toBe(3);
  });

  it('excludes OPTIONS when specified', () => {
    const flows = [
      makeFlow({ id: 'f1', method: 'GET' }),
      makeFlow({ id: 'f2', method: 'OPTIONS' }),
      makeFlow({ id: 'f3', method: 'POST' }),
      makeFlow({ id: 'f4', method: 'OPTIONS' }),
    ];
    const result = filterFlowsByMethod(flows, new Set(['OPTIONS']));
    expect(result.length).toBe(2);
    expect(result.map((f) => f.id)).toEqual(['f1', 'f3']);
  });

  it('is case-insensitive', () => {
    const flows = [
      makeFlow({ id: 'f1', method: 'options' }),
      makeFlow({ id: 'f2', method: 'Options' }),
      makeFlow({ id: 'f3', method: 'POST' }),
    ];
    const result = filterFlowsByMethod(flows, new Set(['OPTIONS']));
    expect(result.length).toBe(1);
    expect(result[0].id).toBe('f3');
  });

  it('can exclude multiple methods', () => {
    const flows = [
      makeFlow({ id: 'f1', method: 'GET' }),
      makeFlow({ id: 'f2', method: 'OPTIONS' }),
      makeFlow({ id: 'f3', method: 'HEAD' }),
      makeFlow({ id: 'f4', method: 'POST' }),
    ];
    const result = filterFlowsByMethod(flows, new Set(['OPTIONS', 'HEAD']));
    expect(result.length).toBe(2);
    expect(result.map((f) => f.id)).toEqual(['f1', 'f4']);
  });

  it('returns empty array for empty input', () => {
    expect(filterFlowsByMethod([], new Set(['OPTIONS']))).toEqual([]);
  });
});
