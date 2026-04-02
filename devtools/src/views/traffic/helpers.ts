/**
 * Pure functions extracted from TrafficView for testing.
 * Compare delta calculation and percentage computation.
 */

export interface LatencyStats {
  p50_ms: number;
  p95_ms: number;
  p99_ms: number;
  avg_ms: number;
}

export interface ReplayRun {
  id: string;
  recording_id: string;
  recording_name: string;
  status: string;
  speed_multiplier: number;
  total_requests: number;
  sent_requests: number;
  error_count: number;
  latency_stats: LatencyStats;
  started_at: string;
  completed_at: string | null;
}

export interface CompareRow {
  label: string;
  a: number;
  b: number;
}

/**
 * Build compare rows from two replay runs.
 */
export function buildCompareRows(runA: ReplayRun, runB: ReplayRun): CompareRow[] {
  const statsA = runA.latency_stats || { p50_ms: 0, p95_ms: 0, p99_ms: 0, avg_ms: 0 };
  const statsB = runB.latency_stats || { p50_ms: 0, p95_ms: 0, p99_ms: 0, avg_ms: 0 };

  return [
    { label: 'P50 Latency (ms)', a: statsA.p50_ms, b: statsB.p50_ms },
    { label: 'P95 Latency (ms)', a: statsA.p95_ms, b: statsB.p95_ms },
    { label: 'P99 Latency (ms)', a: statsA.p99_ms, b: statsB.p99_ms },
    { label: 'Avg Latency (ms)', a: statsA.avg_ms, b: statsB.avg_ms },
    { label: 'Errors', a: runA.error_count, b: runB.error_count },
    { label: 'Requests Sent', a: runA.sent_requests, b: runB.sent_requests },
  ];
}

/**
 * Compute delta between two values (b - a).
 */
export function computeDelta(a: number, b: number): number {
  return b - a;
}

/**
 * Compute percentage change from a to b.
 * Returns 0 when a is 0 (avoids division by zero).
 */
export function computeDeltaPercent(a: number, b: number): number {
  if (a === 0) return 0;
  return ((b - a) / a) * 100;
}

/**
 * Determine if a delta represents an improvement or regression.
 * For latency and errors: lower is better (negative delta = improvement).
 * For requests sent: higher is better (positive delta = improvement).
 */
export function isDeltaImproved(label: string, delta: number): boolean {
  const isLowerBetter = label !== 'Requests Sent';
  return isLowerBetter ? delta < 0 : delta > 0;
}

/**
 * Determine if a delta represents a regression.
 */
export function isDeltaRegressed(label: string, delta: number): boolean {
  const isLowerBetter = label !== 'Requests Sent';
  return isLowerBetter ? delta > 0 : delta < 0;
}

/**
 * Format a delta value for display.
 * Returns '--' for zero delta.
 */
export function formatDelta(delta: number): string {
  if (delta === 0) return '--';
  return `${delta > 0 ? '+' : ''}${Math.round(delta)}`;
}

/**
 * Format delta percentage for display.
 * Returns empty string when not applicable (zero base or zero delta).
 */
export function formatDeltaPercent(a: number, delta: number): string {
  if (delta === 0 || a === 0) return '';
  const pct = (delta / a) * 100;
  return ` (${pct > 0 ? '+' : ''}${Math.round(pct)}%)`;
}
