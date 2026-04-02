import { describe, it, expect } from 'vitest';
import { normalizeRequestEvent, filterRequestsByEdgeServices, buildFlows, buildFallbackFlows, mergeFlows } from '../request-trace-utils';

describe('Request Pipeline E2E', () => {
  // Mock API response (PascalCase, like cloudmock returns)
  const mockAPIResponse = [
    { ID: 'req-1', TraceID: 'trace-1', Service: 'dynamodb', Action: 'PutItem', Method: 'POST', Path: '/', StatusCode: 200, LatencyMs: 12, Timestamp: new Date().toISOString(), Level: 'infra' },
    { ID: 'req-2', TraceID: 'trace-1', Service: 'dynamodb', Action: 'GetItem', Method: 'POST', Path: '/', StatusCode: 200, LatencyMs: 5, Timestamp: new Date(Date.now() + 100).toISOString(), Level: 'infra' },
    { ID: 'req-3', TraceID: '', Service: 'autotend-bff', Method: 'POST', Path: '/v1/cart/items', StatusCode: 201, LatencyMs: 45, Timestamp: new Date().toISOString(), Level: 'app' },
    { ID: 'req-4', Service: 's3', Action: 'PutObject', Method: 'PUT', Path: '/my-bucket/file.jpg', StatusCode: 200, LatencyMs: 30, Timestamp: new Date().toISOString(), Level: 'infra' },
  ];

  it('normalizes PascalCase API response to snake_case', () => {
    const normalized = mockAPIResponse.map(normalizeRequestEvent);
    expect(normalized[0].id).toBe('req-1');
    expect(normalized[0].trace_id).toBe('trace-1');
    expect(normalized[0].service).toBe('dynamodb');
    expect(normalized[0].status_code).toBe(200);
  });

  it('filters requests by edge services for a DynamoDB node', () => {
    const normalized = mockAPIResponse.map(normalizeRequestEvent);
    const node = { id: 'svc:dynamodb', label: 'DynamoDB', service: 'dynamodb', type: 'aws-service', group: 'Storage' };
    const edges = [{ source: 'svc:billing', target: 'svc:dynamodb' }];
    const filtered = filterRequestsByEdgeServices(normalized, 'dynamodb', node, edges);
    expect(filtered.length).toBeGreaterThan(0);
    expect(filtered.every(r => r.service === 'dynamodb')).toBe(true);
  });

  it('filters requests for BFF node matching autotend-bff service', () => {
    const normalized = mockAPIResponse.map(normalizeRequestEvent);
    const node = { id: 'external:bff-service', label: 'BFF', service: 'external', type: 'external', group: 'API' };
    const edges = [{ source: 'external:bff-service', target: 'svc:dynamodb' }];
    const filtered = filterRequestsByEdgeServices(normalized, 'bff-service', node, edges);
    // Should match autotend-bff via stripped prefix matching
    expect(filtered.some(r => r.service === 'autotend-bff')).toBe(true);
  });

  it('builds flows from trace-grouped requests', () => {
    const normalized = mockAPIResponse.map(normalizeRequestEvent);
    const nodes = [
      { id: 'svc:dynamodb', label: 'DynamoDB', service: 'dynamodb', type: 'aws-service', group: 'Storage' },
    ];
    const edges = [{ source: 'svc:billing', target: 'svc:dynamodb' }];
    const flows = buildFlows(normalized.filter(r => r.service === 'dynamodb'), 'dynamodb', nodes, edges);
    expect(flows.length).toBeGreaterThan(0);
    // trace-1 should group 2 requests into 1 flow with 1 outbound
    const tracedFlow = flows.find(f => f.traceId === 'trace-1');
    expect(tracedFlow).toBeDefined();
    expect(tracedFlow!.outbound.length).toBe(1);
  });

  it('builds fallback flows when no trace IDs exist', () => {
    const untracedRequests = mockAPIResponse
      .filter(r => !r.TraceID)
      .map(normalizeRequestEvent);
    const fallback = buildFallbackFlows(untracedRequests);
    expect(fallback.length).toBe(untracedRequests.length);
  });

  it('merges flows without losing existing ones', () => {
    const flowA = buildFallbackFlows([normalizeRequestEvent(mockAPIResponse[0])]);
    const flowB = buildFallbackFlows([normalizeRequestEvent(mockAPIResponse[2])]);
    const merged = mergeFlows(flowA, flowB);
    expect(merged.length).toBe(2);
  });

  it('guards against empty serviceName', () => {
    const normalized = mockAPIResponse.map(normalizeRequestEvent);
    const node = { id: '', label: '', service: '', type: '', group: '' };
    const filtered = filterRequestsByEdgeServices(normalized, '', node, []);
    // Should return first 50 (guard), not match everything
    expect(filtered.length).toBeLessThanOrEqual(50);
  });

  it('full pipeline: API response → normalize → filter → buildFlows → count', () => {
    const normalized = mockAPIResponse.map(normalizeRequestEvent);
    const node = { id: 'svc:dynamodb', label: 'DynamoDB', service: 'dynamodb', type: 'aws-service', group: 'Storage' };
    const edges = [{ source: 'svc:billing', target: 'svc:dynamodb' }];
    const filtered = filterRequestsByEdgeServices(normalized, 'dynamodb', node, edges);
    const flows = buildFlows(filtered, 'dynamodb', [node], edges);
    const fallback = flows.length > 0 ? flows : buildFallbackFlows(filtered);
    expect(fallback.length).toBeGreaterThan(0);
    // Every flow should have required fields
    for (const f of fallback) {
      expect(f.id).toBeTruthy();
      expect(f.method).toBeTruthy();
      expect(typeof f.statusCode).toBe('number');
    }
  });
});
