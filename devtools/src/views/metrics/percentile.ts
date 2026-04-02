export interface TraceEntry {
  TraceID: string;
  RootService: string;
  Method: string;
  Path: string;
  DurationMs: number;
  StatusCode: number;
  SpanCount: number;
  HasError: boolean;
  StartTime: string;
}

export interface TimeBucket {
  time: number; // minute timestamp
  count: number;
  totalLatency: number;
  errorCount: number;
  latencies: number[];
}

export function percentile(sorted: number[], p: number): number {
  if (sorted.length === 0) return 0;
  const idx = Math.ceil((p / 100) * sorted.length) - 1;
  return sorted[Math.max(0, idx)];
}

/** Compute P99 from a sorted array of latencies */
export function p99(sorted: number[]): number {
  if (sorted.length === 0) return 0;
  const idx = Math.ceil(0.99 * sorted.length) - 1;
  return sorted[Math.max(0, idx)];
}

/** Bucket traces by minute for time-series charts */
export function bucketByMinute(traces: TraceEntry[]): TimeBucket[] {
  const buckets = new Map<number, TimeBucket>();

  for (const t of traces) {
    const ts = new Date(t.StartTime).getTime();
    if (isNaN(ts)) continue;
    // Truncate to minute
    const minuteTs = Math.floor(ts / 60000) * 60000;

    let bucket = buckets.get(minuteTs);
    if (!bucket) {
      bucket = { time: minuteTs, count: 0, totalLatency: 0, errorCount: 0, latencies: [] };
      buckets.set(minuteTs, bucket);
    }

    bucket.count++;
    bucket.totalLatency += t.DurationMs;
    bucket.latencies.push(t.DurationMs);
    if (t.HasError || t.StatusCode >= 500) {
      bucket.errorCount++;
    }
  }

  // Sort by time
  return Array.from(buckets.values()).sort((a, b) => a.time - b.time);
}

export function formatLatency(ms: number): string {
  if (ms < 1) return `${(ms * 1000).toFixed(0)}us`;
  if (ms < 1000) return `${ms.toFixed(1)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

export function formatRate(rate: number): string {
  return `${(rate * 100).toFixed(2)}%`;
}
