import { describe, it, expect } from 'vitest';
import {
  buildSpanTree,
  findCriticalPath,
  computeServiceAggregates,
} from '../span-tree';
import type { WaterfallSpan } from '../span-tree';

function makeSpan(overrides: Partial<WaterfallSpan> = {}): WaterfallSpan {
  return {
    TraceID: 'trace-1',
    SpanID: 'span-1',
    ParentSpanID: '',
    Service: 'api',
    Action: 'GetUser',
    Method: 'GET',
    Path: '/api/users',
    StartTime: '2025-01-01T00:00:00.000Z',
    EndTime: '2025-01-01T00:00:00.100Z',
    Duration: 100000000, // 100ms in nanoseconds
    DurationMs: 100,
    StatusCode: 200,
    Children: null,
    Metadata: null,
    ...overrides,
  };
}

describe('buildSpanTree', () => {
  it('returns empty result for no spans', () => {
    const result = buildSpanTree([]);
    expect(result.rows).toEqual([]);
    expect(result.traceStartMs).toBe(0);
    expect(result.traceDurationMs).toBe(0);
  });

  it('builds a single root span row', () => {
    const spans = [makeSpan()];
    const { rows, traceDurationMs } = buildSpanTree(spans);
    expect(rows.length).toBe(1);
    expect(rows[0].depth).toBe(0);
    expect(rows[0].offsetMs).toBe(0);
    expect(traceDurationMs).toBe(100);
  });

  it('builds parent-child relationships with correct depth', () => {
    const spans = [
      makeSpan({ SpanID: 'root', ParentSpanID: '', StartTime: '2025-01-01T00:00:00.000Z', EndTime: '2025-01-01T00:00:00.200Z', DurationMs: 200 }),
      makeSpan({ SpanID: 'child-1', ParentSpanID: 'root', Service: 'db', StartTime: '2025-01-01T00:00:00.050Z', EndTime: '2025-01-01T00:00:00.150Z', DurationMs: 100 }),
      makeSpan({ SpanID: 'grandchild', ParentSpanID: 'child-1', Service: 'cache', StartTime: '2025-01-01T00:00:00.060Z', EndTime: '2025-01-01T00:00:00.090Z', DurationMs: 30 }),
    ];
    const { rows } = buildSpanTree(spans);
    expect(rows.length).toBe(3);
    expect(rows[0].depth).toBe(0); // root
    expect(rows[1].depth).toBe(1); // child-1
    expect(rows[2].depth).toBe(2); // grandchild
  });

  it('computes offsetMs relative to trace start', () => {
    const spans = [
      makeSpan({ SpanID: 'root', ParentSpanID: '', StartTime: '2025-01-01T00:00:00.000Z', EndTime: '2025-01-01T00:00:00.200Z', DurationMs: 200 }),
      makeSpan({ SpanID: 'child', ParentSpanID: 'root', StartTime: '2025-01-01T00:00:00.050Z', EndTime: '2025-01-01T00:00:00.150Z', DurationMs: 100 }),
    ];
    const { rows } = buildSpanTree(spans);
    expect(rows[0].offsetMs).toBe(0);
    expect(rows[1].offsetMs).toBe(50);
  });

  it('sorts children by start time', () => {
    const spans = [
      makeSpan({ SpanID: 'root', ParentSpanID: '', StartTime: '2025-01-01T00:00:00.000Z', EndTime: '2025-01-01T00:00:00.300Z', DurationMs: 300 }),
      makeSpan({ SpanID: 'child-b', ParentSpanID: 'root', StartTime: '2025-01-01T00:00:00.200Z', EndTime: '2025-01-01T00:00:00.250Z', DurationMs: 50 }),
      makeSpan({ SpanID: 'child-a', ParentSpanID: 'root', StartTime: '2025-01-01T00:00:00.050Z', EndTime: '2025-01-01T00:00:00.100Z', DurationMs: 50 }),
    ];
    const { rows } = buildSpanTree(spans);
    expect(rows[1].span.SpanID).toBe('child-a'); // earlier start time first
    expect(rows[2].span.SpanID).toBe('child-b');
  });

  it('marks critical path spans', () => {
    const spans = [
      makeSpan({ SpanID: 'root', ParentSpanID: '', StartTime: '2025-01-01T00:00:00.000Z', EndTime: '2025-01-01T00:00:00.300Z', DurationMs: 300 }),
      makeSpan({ SpanID: 'child', ParentSpanID: 'root', StartTime: '2025-01-01T00:00:00.050Z', EndTime: '2025-01-01T00:00:00.310Z', DurationMs: 260 }),
    ];
    const { rows } = buildSpanTree(spans);
    // child ends latest (.310Z), walks up to root -- both on critical path
    expect(rows[0].isCriticalPath).toBe(true);
    expect(rows[1].isCriticalPath).toBe(true);
  });
});

describe('findCriticalPath', () => {
  it('returns empty set for no spans', () => {
    expect(findCriticalPath([], 0).size).toBe(0);
  });

  it('returns single span for a single-span trace', () => {
    const spans = [makeSpan({ SpanID: 'only' })];
    const path = findCriticalPath(spans, 0);
    expect(path.has('only')).toBe(true);
    expect(path.size).toBe(1);
  });

  it('walks up the parent chain from the latest-ending span', () => {
    const spans = [
      makeSpan({ SpanID: 'root', ParentSpanID: '', EndTime: '2025-01-01T00:00:00.300Z' }),
      makeSpan({ SpanID: 'fast-child', ParentSpanID: 'root', EndTime: '2025-01-01T00:00:00.100Z' }),
      makeSpan({ SpanID: 'slow-child', ParentSpanID: 'root', EndTime: '2025-01-01T00:00:00.310Z' }),
    ];
    const path = findCriticalPath(spans, 0);
    // slow-child ends latest (.310Z), walks up to root
    expect(path.has('root')).toBe(true);
    expect(path.has('slow-child')).toBe(true);
    expect(path.has('fast-child')).toBe(false);
  });

  it('handles deep chains', () => {
    const spans = [
      makeSpan({ SpanID: 'a', ParentSpanID: '', EndTime: '2025-01-01T00:00:00.400Z' }),
      makeSpan({ SpanID: 'b', ParentSpanID: 'a', EndTime: '2025-01-01T00:00:00.350Z' }),
      makeSpan({ SpanID: 'c', ParentSpanID: 'b', EndTime: '2025-01-01T00:00:00.410Z' }),
    ];
    const path = findCriticalPath(spans, 0);
    // c ends latest (.410Z), walks up through b to a
    expect(path.has('a')).toBe(true);
    expect(path.has('b')).toBe(true);
    expect(path.has('c')).toBe(true);
  });
});

describe('computeServiceAggregates', () => {
  it('groups duration by service', () => {
    const spans = [
      makeSpan({ Service: 'api', DurationMs: 100 }),
      makeSpan({ SpanID: 's2', Service: 'db', DurationMs: 50 }),
      makeSpan({ SpanID: 's3', Service: 'api', DurationMs: 30 }),
    ];
    const aggs = computeServiceAggregates(spans, 200);
    const apiAgg = aggs.find((a) => a.service === 'api');
    const dbAgg = aggs.find((a) => a.service === 'db');
    expect(apiAgg?.totalMs).toBe(130);
    expect(dbAgg?.totalMs).toBe(50);
  });

  it('computes percentage relative to trace duration', () => {
    const spans = [
      makeSpan({ Service: 'api', DurationMs: 100 }),
    ];
    const aggs = computeServiceAggregates(spans, 200);
    expect(aggs[0].percentage).toBe(50);
  });

  it('sorts by totalMs descending', () => {
    const spans = [
      makeSpan({ SpanID: 's1', Service: 'fast', DurationMs: 10 }),
      makeSpan({ SpanID: 's2', Service: 'slow', DurationMs: 100 }),
    ];
    const aggs = computeServiceAggregates(spans, 200);
    expect(aggs[0].service).toBe('slow');
    expect(aggs[1].service).toBe('fast');
  });

  it('returns empty for no spans', () => {
    expect(computeServiceAggregates([], 100)).toEqual([]);
  });

  it('uses "unknown" for spans without Service', () => {
    const spans = [
      makeSpan({ Service: '', DurationMs: 50 }),
    ];
    const aggs = computeServiceAggregates(spans, 100);
    expect(aggs[0].service).toBe('unknown');
  });

  it('handles zero traceDurationMs', () => {
    const spans = [makeSpan({ DurationMs: 10 })];
    const aggs = computeServiceAggregates(spans, 0);
    expect(aggs[0].percentage).toBe(0);
  });
});
