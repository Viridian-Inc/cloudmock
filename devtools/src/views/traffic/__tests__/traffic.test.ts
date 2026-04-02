import { describe, it, expect } from 'vitest';
import {
  buildCompareRows,
  computeDelta,
  computeDeltaPercent,
  isDeltaImproved,
  isDeltaRegressed,
  formatDelta,
  formatDeltaPercent,
} from '../helpers';
import type { ReplayRun } from '../helpers';

function makeRun(overrides: Partial<ReplayRun> = {}): ReplayRun {
  return {
    id: 'run-1',
    recording_id: 'rec-1',
    recording_name: 'checkout-flow',
    status: 'completed',
    speed_multiplier: 1,
    total_requests: 100,
    sent_requests: 100,
    error_count: 0,
    latency_stats: { p50_ms: 10, p95_ms: 50, p99_ms: 100, avg_ms: 25 },
    started_at: '2025-01-15T10:00:00Z',
    completed_at: '2025-01-15T10:01:00Z',
    ...overrides,
  };
}

describe('computeDelta', () => {
  it('computes positive delta (B > A)', () => {
    expect(computeDelta(100, 150)).toBe(50);
  });

  it('computes negative delta (B < A)', () => {
    expect(computeDelta(150, 100)).toBe(-50);
  });

  it('returns zero for equal values', () => {
    expect(computeDelta(100, 100)).toBe(0);
  });
});

describe('computeDeltaPercent', () => {
  it('computes positive percentage change', () => {
    expect(computeDeltaPercent(100, 150)).toBe(50);
  });

  it('computes negative percentage change', () => {
    expect(computeDeltaPercent(200, 100)).toBe(-50);
  });

  it('returns 0 when base is 0 (avoids division by zero)', () => {
    expect(computeDeltaPercent(0, 100)).toBe(0);
  });

  it('handles equal values', () => {
    expect(computeDeltaPercent(100, 100)).toBe(0);
  });

  it('computes 100% for double', () => {
    expect(computeDeltaPercent(50, 100)).toBe(100);
  });
});

describe('isDeltaImproved', () => {
  it('latency: negative delta is improvement (lower is better)', () => {
    expect(isDeltaImproved('P50 Latency (ms)', -10)).toBe(true);
  });

  it('latency: positive delta is not improvement', () => {
    expect(isDeltaImproved('P99 Latency (ms)', 10)).toBe(false);
  });

  it('errors: negative delta is improvement (lower is better)', () => {
    expect(isDeltaImproved('Errors', -5)).toBe(true);
  });

  it('requests sent: positive delta is improvement (higher is better)', () => {
    expect(isDeltaImproved('Requests Sent', 10)).toBe(true);
  });

  it('requests sent: negative delta is not improvement', () => {
    expect(isDeltaImproved('Requests Sent', -10)).toBe(false);
  });

  it('zero delta is not improvement', () => {
    expect(isDeltaImproved('Errors', 0)).toBe(false);
  });
});

describe('isDeltaRegressed', () => {
  it('latency: positive delta is regression', () => {
    expect(isDeltaRegressed('P95 Latency (ms)', 20)).toBe(true);
  });

  it('latency: negative delta is not regression', () => {
    expect(isDeltaRegressed('Avg Latency (ms)', -10)).toBe(false);
  });

  it('requests sent: negative delta is regression', () => {
    expect(isDeltaRegressed('Requests Sent', -5)).toBe(true);
  });

  it('requests sent: positive delta is not regression', () => {
    expect(isDeltaRegressed('Requests Sent', 5)).toBe(false);
  });
});

describe('formatDelta', () => {
  it('formats positive delta with + prefix', () => {
    expect(formatDelta(50)).toBe('+50');
  });

  it('formats negative delta', () => {
    expect(formatDelta(-30)).toBe('-30');
  });

  it('returns -- for zero delta', () => {
    expect(formatDelta(0)).toBe('--');
  });

  it('rounds fractional values', () => {
    expect(formatDelta(12.7)).toBe('+13');
  });
});

describe('formatDeltaPercent', () => {
  it('formats positive percentage', () => {
    const result = formatDeltaPercent(100, 50);
    expect(result).toBe(' (+50%)');
  });

  it('formats negative percentage', () => {
    const result = formatDeltaPercent(200, -50);
    expect(result).toBe(' (-25%)');
  });

  it('returns empty string when delta is zero', () => {
    expect(formatDeltaPercent(100, 0)).toBe('');
  });

  it('returns empty string when base is zero (avoids division by zero)', () => {
    expect(formatDeltaPercent(0, 50)).toBe('');
  });
});

describe('buildCompareRows', () => {
  it('builds 6 comparison rows from two runs', () => {
    const runA = makeRun({ latency_stats: { p50_ms: 10, p95_ms: 50, p99_ms: 100, avg_ms: 25 }, error_count: 2, sent_requests: 100 });
    const runB = makeRun({ latency_stats: { p50_ms: 15, p95_ms: 60, p99_ms: 120, avg_ms: 30 }, error_count: 1, sent_requests: 100 });
    const rows = buildCompareRows(runA, runB);
    expect(rows).toHaveLength(6);
    expect(rows[0].label).toBe('P50 Latency (ms)');
    expect(rows[0].a).toBe(10);
    expect(rows[0].b).toBe(15);
  });

  it('handles null latency_stats gracefully', () => {
    const runA = makeRun({ latency_stats: null as any });
    const runB = makeRun({ latency_stats: null as any });
    const rows = buildCompareRows(runA, runB);
    expect(rows[0].a).toBe(0);
    expect(rows[0].b).toBe(0);
  });
});
