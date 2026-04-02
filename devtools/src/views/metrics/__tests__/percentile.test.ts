import { describe, it, expect } from 'vitest';
import {
  percentile,
  p99,
  bucketByMinute,
  formatLatency,
  formatRate,
} from '../percentile';
import type { TraceEntry } from '../percentile';

function makeTrace(overrides: Partial<TraceEntry> = {}): TraceEntry {
  return {
    TraceID: 'trace-1',
    RootService: 'api',
    Method: 'GET',
    Path: '/health',
    DurationMs: 10,
    StatusCode: 200,
    SpanCount: 1,
    HasError: false,
    StartTime: '2025-01-01T00:00:00Z',
    ...overrides,
  };
}

describe('percentile', () => {
  it('returns 0 for empty array', () => {
    expect(percentile([], 50)).toBe(0);
  });

  it('returns the single element for a 1-element array', () => {
    expect(percentile([42], 50)).toBe(42);
    expect(percentile([42], 99)).toBe(42);
  });

  it('computes p50 of sorted array', () => {
    const sorted = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
    expect(percentile(sorted, 50)).toBe(5);
  });

  it('computes p99 of sorted array', () => {
    const sorted = Array.from({ length: 100 }, (_, i) => i + 1);
    expect(percentile(sorted, 99)).toBe(99);
  });

  it('computes p95 of sorted array', () => {
    const sorted = Array.from({ length: 100 }, (_, i) => i + 1);
    expect(percentile(sorted, 95)).toBe(95);
  });

  it('handles unsorted array by trusting the caller to sort', () => {
    // The function expects a pre-sorted array
    const sorted = [1, 5, 10, 50, 100].sort((a, b) => a - b);
    expect(percentile(sorted, 50)).toBe(10);
  });
});

describe('p99', () => {
  it('returns 0 for empty array', () => {
    expect(p99([])).toBe(0);
  });

  it('returns value at 99th percentile index', () => {
    const sorted = Array.from({ length: 100 }, (_, i) => i + 1);
    expect(p99(sorted)).toBe(99);
  });

  it('returns the single element for 1-element array', () => {
    expect(p99([7])).toBe(7);
  });

  it('handles small arrays', () => {
    expect(p99([3, 5])).toBe(5);
  });
});

describe('bucketByMinute', () => {
  it('groups traces into minute buckets', () => {
    const traces = [
      makeTrace({ StartTime: '2025-01-01T00:00:10Z', DurationMs: 10 }),
      makeTrace({ StartTime: '2025-01-01T00:00:30Z', DurationMs: 20 }),
      makeTrace({ StartTime: '2025-01-01T00:01:15Z', DurationMs: 30 }),
    ];
    const buckets = bucketByMinute(traces);
    expect(buckets.length).toBe(2);
    expect(buckets[0].count).toBe(2); // first minute
    expect(buckets[1].count).toBe(1); // second minute
  });

  it('accumulates latencies within a bucket', () => {
    const traces = [
      makeTrace({ StartTime: '2025-01-01T00:00:10Z', DurationMs: 10 }),
      makeTrace({ StartTime: '2025-01-01T00:00:30Z', DurationMs: 20 }),
    ];
    const buckets = bucketByMinute(traces);
    expect(buckets[0].totalLatency).toBe(30);
    expect(buckets[0].latencies).toEqual([10, 20]);
  });

  it('counts errors by HasError flag', () => {
    const traces = [
      makeTrace({ StartTime: '2025-01-01T00:00:10Z', HasError: true }),
      makeTrace({ StartTime: '2025-01-01T00:00:20Z', HasError: false }),
    ];
    const buckets = bucketByMinute(traces);
    expect(buckets[0].errorCount).toBe(1);
  });

  it('counts errors by StatusCode >= 500', () => {
    const traces = [
      makeTrace({ StartTime: '2025-01-01T00:00:10Z', StatusCode: 500, HasError: false }),
      makeTrace({ StartTime: '2025-01-01T00:00:20Z', StatusCode: 200, HasError: false }),
    ];
    const buckets = bucketByMinute(traces);
    expect(buckets[0].errorCount).toBe(1);
  });

  it('returns empty array for no traces', () => {
    expect(bucketByMinute([])).toEqual([]);
  });

  it('returns sorted buckets by time', () => {
    const traces = [
      makeTrace({ StartTime: '2025-01-01T00:05:00Z' }),
      makeTrace({ StartTime: '2025-01-01T00:01:00Z' }),
      makeTrace({ StartTime: '2025-01-01T00:03:00Z' }),
    ];
    const buckets = bucketByMinute(traces);
    for (let i = 1; i < buckets.length; i++) {
      expect(buckets[i].time).toBeGreaterThan(buckets[i - 1].time);
    }
  });

  it('skips traces with invalid StartTime', () => {
    const traces = [
      makeTrace({ StartTime: 'invalid-date' }),
      makeTrace({ StartTime: '2025-01-01T00:00:10Z', DurationMs: 5 }),
    ];
    const buckets = bucketByMinute(traces);
    expect(buckets.length).toBe(1);
    expect(buckets[0].count).toBe(1);
  });
});

describe('formatLatency', () => {
  it('formats microseconds for sub-ms values', () => {
    expect(formatLatency(0.05)).toBe('50us');
  });

  it('formats milliseconds', () => {
    expect(formatLatency(42.5)).toBe('42.5ms');
  });

  it('formats seconds for >= 1000ms', () => {
    expect(formatLatency(2500)).toBe('2.50s');
  });
});

describe('formatRate', () => {
  it('formats rate as percentage with 2 decimals', () => {
    expect(formatRate(0.0532)).toBe('5.32%');
  });

  it('formats zero', () => {
    expect(formatRate(0)).toBe('0.00%');
  });
});
