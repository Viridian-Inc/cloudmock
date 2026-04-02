import { describe, it, expect } from 'vitest';
import {
  computeHealthState,
  isUserFacing,
  getBlastRadius,
  formatEdgeLabel,
  getRecentDeploy,
} from '../health';
import type { ServiceMetrics, DeployEvent } from '../health';
import type { TopoNode, TopoEdge } from '../../views/topology/index';

function makeMetrics(overrides: Partial<ServiceMetrics> = {}): ServiceMetrics {
  return {
    service: 'test-svc',
    p50ms: 10,
    p95ms: 50,
    p99ms: 100,
    avgMs: 30,
    errorRate: 0,
    totalCalls: 100,
    errorCalls: 0,
    ...overrides,
  };
}

describe('computeHealthState', () => {
  it('returns green for healthy metrics', () => {
    const m = makeMetrics({ errorRate: 0, p99ms: 50 });
    expect(computeHealthState(m)).toBe('green');
  });

  it('returns yellow for elevated error rate 1-5%', () => {
    const m = makeMetrics({ errorRate: 0.03 });
    expect(computeHealthState(m)).toBe('yellow');
  });

  it('returns red for error rate > 5%', () => {
    const m = makeMetrics({ errorRate: 0.1 });
    expect(computeHealthState(m)).toBe('red');
  });

  it('returns red for p99 > threshold', () => {
    const m = makeMetrics({ p99ms: 300 });
    expect(computeHealthState(m, 200)).toBe('red');
  });

  it('returns yellow for p99 approaching threshold (> 80%)', () => {
    const m = makeMetrics({ p99ms: 170 });
    expect(computeHealthState(m, 200)).toBe('yellow');
  });

  it('returns red when active incident', () => {
    const m = makeMetrics({ errorRate: 0 });
    expect(computeHealthState(m, 200, true)).toBe('red');
  });

  it('returns green for no traffic', () => {
    const m = makeMetrics({ totalCalls: 0 });
    expect(computeHealthState(m)).toBe('green');
  });

  it('returns green for undefined metrics', () => {
    expect(computeHealthState(undefined)).toBe('green');
  });

  it('uses default threshold of 200ms', () => {
    const m = makeMetrics({ p99ms: 250 });
    expect(computeHealthState(m)).toBe('red');
  });
});

describe('isUserFacing', () => {
  const nodes: TopoNode[] = [
    { id: 'client1', label: 'Browser', service: 'browser', type: 'client', group: 'Client' },
    { id: 'api', label: 'API', service: 'api-gw', type: 'server', group: 'API' },
    { id: 'db', label: 'DynamoDB', service: 'dynamodb', type: 'aws-service', group: 'Storage' },
    { id: 'worker', label: 'Worker', service: 'worker', type: 'lambda', group: 'Compute' },
    { id: 'isolated', label: 'Orphan', service: 'orphan', type: 'lambda', group: 'Compute' },
  ];

  const edges: TopoEdge[] = [
    { source: 'client1', target: 'api', type: 'invoke' },
    { source: 'api', target: 'db', type: 'invoke' },
    { source: 'db', target: 'worker', label: 'DDB stream', type: 'trigger' },
  ];

  it('returns true for Client nodes', () => {
    expect(isUserFacing('client1', nodes, edges)).toBe(true);
  });

  it('returns true for nodes reachable from Client via sync edges', () => {
    expect(isUserFacing('api', nodes, edges)).toBe(true);
    expect(isUserFacing('db', nodes, edges)).toBe(true);
  });

  it('returns false for isolated nodes', () => {
    expect(isUserFacing('isolated', nodes, edges)).toBe(false);
  });

  it('skips async edges (DDB streams)', () => {
    // Worker is only reachable via a DDB stream trigger edge
    expect(isUserFacing('worker', nodes, edges)).toBe(false);
  });

  it('returns false for unknown node ID', () => {
    expect(isUserFacing('nonexistent', nodes, edges)).toBe(false);
  });
});

describe('getBlastRadius', () => {
  const edges: TopoEdge[] = [
    { source: 'a', target: 'b' },
    { source: 'b', target: 'c' },
    { source: 'c', target: 'd' },
    { source: 'a', target: 'e' },
  ];

  it('returns downstream nodes', () => {
    const radius = getBlastRadius('a', edges);
    expect(radius).toContain('b');
    expect(radius).toContain('c');
    expect(radius).toContain('d');
    expect(radius).toContain('e');
  });

  it('returns only downstream, not upstream', () => {
    const radius = getBlastRadius('c', edges);
    expect(radius).toContain('d');
    expect(radius).not.toContain('a');
    expect(radius).not.toContain('b');
  });

  it('handles cycles without infinite loop', () => {
    const cycleEdges: TopoEdge[] = [
      { source: 'x', target: 'y' },
      { source: 'y', target: 'z' },
      { source: 'z', target: 'x' },
    ];
    const radius = getBlastRadius('x', cycleEdges);
    expect(radius).toContain('y');
    expect(radius).toContain('z');
    // Should not include self
    expect(radius).not.toContain('x');
  });

  it('returns empty set for a leaf node', () => {
    const radius = getBlastRadius('d', edges);
    expect(radius.size).toBe(0);
  });

  it('returns empty set for unknown node', () => {
    const radius = getBlastRadius('unknown', edges);
    expect(radius.size).toBe(0);
  });
});

describe('getRecentDeploy', () => {
  const deploys: DeployEvent[] = [
    { id: '1', timestamp: '2025-01-01T00:00:00Z', service: 'api', commit: 'abc', author: 'me', message: 'fix', branch: 'main' },
    { id: '2', timestamp: '2025-01-02T00:00:00Z', service: 'worker', commit: 'def', author: 'you', message: 'feat', branch: 'main' },
  ];

  it('finds deploy by service name', () => {
    expect(getRecentDeploy('api', deploys)?.id).toBe('1');
  });

  it('finds deploy by colon-separated ID', () => {
    expect(getRecentDeploy('svc:api', deploys)?.id).toBe('1');
  });

  it('returns undefined for missing service', () => {
    expect(getRecentDeploy('unknown', deploys)).toBeUndefined();
  });
});

describe('formatEdgeLabel', () => {
  it('formats call count and latency', () => {
    expect(formatEdgeLabel(42, 12)).toBe('42 req/s \u00b7 12ms');
  });

  it('returns empty for zero values', () => {
    expect(formatEdgeLabel(0, 0)).toBe('');
  });

  it('formats call count only when latency is zero', () => {
    expect(formatEdgeLabel(10, 0)).toBe('10 req/s');
  });

  it('formats latency only when call count is zero', () => {
    expect(formatEdgeLabel(0, 5)).toBe('5ms');
  });

  it('uses decimal for sub-millisecond latency', () => {
    const result = formatEdgeLabel(1, 0.5);
    expect(result).toContain('0.5ms');
  });
});
