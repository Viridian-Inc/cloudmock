export interface WaterfallSpan {
  TraceID: string;
  SpanID: string;
  ParentSpanID: string;
  Service: string;
  Action: string;
  Method: string;
  Path: string;
  StartTime: string;
  EndTime: string;
  Duration: number; // nanoseconds
  DurationMs: number;
  StatusCode: number;
  Children: WaterfallSpan[] | null;
  Metadata: Record<string, unknown> | null;
}

export interface SpanRow {
  span: WaterfallSpan;
  depth: number;
  offsetMs: number;
  durationMs: number;
  isCriticalPath: boolean;
}

export interface ServiceAggregate {
  service: string;
  totalMs: number;
  percentage: number;
}

/**
 * Find the critical path: spans whose cumulative duration defines the
 * end-to-end latency. We trace backwards from the span that ends latest.
 */
export function findCriticalPath(spans: WaterfallSpan[], _traceStartMs: number): Set<string> {
  if (spans.length === 0) return new Set();

  const spanMap = new Map<string, WaterfallSpan>();
  for (const s of spans) spanMap.set(s.SpanID, s);

  // Find the span that ends latest
  let latest = spans[0];
  let latestEnd = new Date(latest.EndTime || latest.StartTime).getTime();

  for (const s of spans) {
    const end = new Date(s.EndTime || s.StartTime).getTime();
    if (end > latestEnd) {
      latest = s;
      latestEnd = end;
    }
  }

  // Walk up parent chain
  const path = new Set<string>();
  let current: WaterfallSpan | undefined = latest;
  while (current) {
    path.add(current.SpanID);
    current = current.ParentSpanID ? spanMap.get(current.ParentSpanID) : undefined;
  }

  return path;
}

/**
 * Build a flat list of SpanRows from a tree of spans.
 * Spans are sorted by StartTime, then flattened with indentation based on
 * parent-child depth.
 */
export function buildSpanTree(spans: WaterfallSpan[]): {
  rows: SpanRow[];
  traceStartMs: number;
  traceDurationMs: number;
} {
  if (spans.length === 0) {
    return { rows: [], traceStartMs: 0, traceDurationMs: 0 };
  }

  // Parse start times
  const parsed = spans.map((s) => ({
    span: s,
    startMs: new Date(s.StartTime).getTime(),
    endMs: new Date(s.EndTime || s.StartTime).getTime(),
  }));

  const traceStartMs = Math.min(...parsed.map((p) => p.startMs));
  const traceEndMs = Math.max(...parsed.map((p) => p.endMs));
  const traceDurationMs = Math.max(traceEndMs - traceStartMs, 0.001);

  // Build parent map
  const childrenMap = new Map<string, WaterfallSpan[]>();
  const rootSpans: WaterfallSpan[] = [];

  for (const s of spans) {
    if (!s.ParentSpanID) {
      rootSpans.push(s);
    } else {
      if (!childrenMap.has(s.ParentSpanID)) {
        childrenMap.set(s.ParentSpanID, []);
      }
      childrenMap.get(s.ParentSpanID)!.push(s);
    }
  }

  // Sort children by start time
  for (const children of childrenMap.values()) {
    children.sort((a, b) => new Date(a.StartTime).getTime() - new Date(b.StartTime).getTime());
  }
  rootSpans.sort((a, b) => new Date(a.StartTime).getTime() - new Date(b.StartTime).getTime());

  // Find critical path
  const criticalSpanIds = findCriticalPath(spans, traceStartMs);

  // DFS flatten
  const rows: SpanRow[] = [];

  function walk(span: WaterfallSpan, depth: number) {
    const startMs = new Date(span.StartTime).getTime();
    const offsetMs = startMs - traceStartMs;
    const durationMs = span.DurationMs > 0 ? span.DurationMs : 0.001;

    rows.push({
      span,
      depth,
      offsetMs,
      durationMs,
      isCriticalPath: criticalSpanIds.has(span.SpanID),
    });

    const children = childrenMap.get(span.SpanID) || [];
    for (const child of children) {
      walk(child, depth + 1);
    }
  }

  for (const root of rootSpans) {
    walk(root, 0);
  }

  return { rows, traceStartMs, traceDurationMs };
}

/**
 * Compute per-service aggregate durations.
 */
export function computeServiceAggregates(spans: WaterfallSpan[], traceDurationMs: number): ServiceAggregate[] {
  const map = new Map<string, number>();
  for (const s of spans) {
    const svc = s.Service || 'unknown';
    map.set(svc, (map.get(svc) || 0) + s.DurationMs);
  }

  const aggregates: ServiceAggregate[] = [];
  for (const [service, totalMs] of map) {
    aggregates.push({
      service,
      totalMs,
      percentage: traceDurationMs > 0 ? (totalMs / traceDurationMs) * 100 : 0,
    });
  }

  aggregates.sort((a, b) => b.totalMs - a.totalMs);
  return aggregates;
}
