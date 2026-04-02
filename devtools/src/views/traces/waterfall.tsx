import { useState, useEffect, useMemo } from 'preact/hooks';
import { api } from '../../lib/api';

/* ====== Types ====== */

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

interface SpanRow {
  span: WaterfallSpan;
  depth: number;
  offsetMs: number;
  durationMs: number;
  isCriticalPath: boolean;
}

interface ServiceAggregate {
  service: string;
  totalMs: number;
  percentage: number;
}

interface WaterfallProps {
  traceId: string | null;
}

/* ====== Service Colors ====== */

const SERVICE_PALETTE = [
  '#4AE5F8', '#a78bfa', '#60a5fa', '#fbbf24', '#818cf8',
  '#4ade80', '#f472b6', '#22d3ee', '#f87171', '#fb923c',
  '#34d399', '#c084fc', '#38bdf8', '#facc15', '#a3e635',
];

const serviceColorCache = new Map<string, string>();
function getServiceColor(service: string): string {
  if (!serviceColorCache.has(service)) {
    let hash = 5381;
    for (let i = 0; i < service.length; i++) {
      hash = ((hash << 5) + hash + service.charCodeAt(i)) >>> 0;
    }
    serviceColorCache.set(service, SERVICE_PALETTE[hash % SERVICE_PALETTE.length]);
  }
  return serviceColorCache.get(service)!;
}

/* ====== Helpers ====== */

function statusClass(status: number): string {
  if (status >= 200 && status < 300) return 'wf-status-2xx';
  if (status >= 400 && status < 500) return 'wf-status-4xx';
  if (status >= 500) return 'wf-status-5xx';
  return '';
}

function formatDuration(ms: number): string {
  if (ms < 0.01) return `${(ms * 1000).toFixed(0)}us`;
  if (ms < 1) return `${ms.toFixed(2)}ms`;
  if (ms < 1000) return `${Math.round(ms)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

function formatTimestamp(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return '--';
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

/**
 * Build a flat list of SpanRows from a tree of spans.
 * Spans are sorted by StartTime, then flattened with indentation based on
 * parent-child depth.
 */
function buildSpanTree(spans: WaterfallSpan[]): {
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

  // Find critical path: the longest chain of sequential spans ending at the trace end
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
 * Find the critical path: spans whose cumulative duration defines the
 * end-to-end latency. We trace backwards from the span that ends latest.
 */
function findCriticalPath(spans: WaterfallSpan[], traceStartMs: number): Set<string> {
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
 * Compute per-service aggregate durations.
 */
function computeServiceAggregates(spans: WaterfallSpan[], traceDurationMs: number): ServiceAggregate[] {
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

/**
 * Detect gaps between parent span start/end and children (network/queue time).
 */
function computeGaps(
  rows: SpanRow[],
  traceDurationMs: number,
  traceStartMs: number,
): { offsetPct: number; widthPct: number; parentService: string }[] {
  const gaps: { offsetPct: number; widthPct: number; parentService: string }[] = [];

  // Build parent-to-children mapping from rows
  const childrenOfParent = new Map<string, SpanRow[]>();
  for (const row of rows) {
    const pid = row.span.ParentSpanID;
    if (pid) {
      if (!childrenOfParent.has(pid)) childrenOfParent.set(pid, []);
      childrenOfParent.get(pid)!.push(row);
    }
  }

  // For each parent span, find gaps between its start and first child, and between children
  for (const row of rows) {
    const children = childrenOfParent.get(row.span.SpanID);
    if (!children || children.length === 0) continue;

    const parentStart = row.offsetMs;
    const parentEnd = row.offsetMs + row.durationMs;

    // Sort children by offset
    const sorted = [...children].sort((a, b) => a.offsetMs - b.offsetMs);

    // Gap before first child
    const firstChildStart = sorted[0].offsetMs;
    if (firstChildStart > parentStart + 0.01) {
      const gapMs = firstChildStart - parentStart;
      gaps.push({
        offsetPct: (parentStart / traceDurationMs) * 100,
        widthPct: (gapMs / traceDurationMs) * 100,
        parentService: row.span.Service,
      });
    }

    // Gap after last child
    const lastChild = sorted[sorted.length - 1];
    const lastChildEnd = lastChild.offsetMs + lastChild.durationMs;
    if (parentEnd > lastChildEnd + 0.01) {
      const gapMs = parentEnd - lastChildEnd;
      gaps.push({
        offsetPct: (lastChildEnd / traceDurationMs) * 100,
        widthPct: (gapMs / traceDurationMs) * 100,
        parentService: row.span.Service,
      });
    }
  }

  return gaps;
}

/* ====== Components ====== */

function BreakdownSummary({
  spans,
  traceDurationMs,
  rootMethod,
  rootPath,
}: {
  spans: WaterfallSpan[];
  traceDurationMs: number;
  rootMethod: string;
  rootPath: string;
}) {
  const aggregates = useMemo(
    () => computeServiceAggregates(spans, traceDurationMs),
    [spans, traceDurationMs],
  );

  const bottleneck = aggregates.length > 0 ? aggregates[0] : null;

  // Compute upstream vs downstream split
  // "Upstream" = root/client span, "Downstream" = everything else
  const rootSpans = spans.filter((s) => !s.ParentSpanID);
  const rootDuration = rootSpans.reduce((acc, s) => acc + s.DurationMs, 0);
  const childDuration = spans.filter((s) => s.ParentSpanID).reduce((acc, s) => acc + s.DurationMs, 0);
  const networkOverhead = Math.max(0, traceDurationMs - childDuration);

  return (
    <div class="wf-breakdown">
      <div class="wf-breakdown-header">
        <div class="wf-breakdown-title">
          <span class="wf-breakdown-method">{rootMethod}</span>
          <span class="wf-breakdown-path">{rootPath}</span>
        </div>
        <div class="wf-breakdown-total">
          Total: <strong>{formatDuration(traceDurationMs)}</strong>
        </div>
      </div>

      <div class="wf-breakdown-body">
        {/* Per-service bars */}
        <div class="wf-breakdown-services">
          {aggregates.map((agg) => (
            <div key={agg.service} class="wf-breakdown-svc-row">
              <span
                class="wf-breakdown-svc-dot"
                style={{ background: getServiceColor(agg.service) }}
              />
              <span class="wf-breakdown-svc-name">{agg.service}</span>
              <div class="wf-breakdown-svc-bar-track">
                <div
                  class="wf-breakdown-svc-bar-fill"
                  style={{
                    width: `${Math.max(agg.percentage, 1)}%`,
                    background: getServiceColor(agg.service),
                  }}
                />
              </div>
              <span class="wf-breakdown-svc-value">{formatDuration(agg.totalMs)}</span>
              <span class="wf-breakdown-svc-pct">{agg.percentage.toFixed(1)}%</span>
            </div>
          ))}
        </div>

        {/* Indicators row */}
        <div class="wf-breakdown-indicators">
          {bottleneck && (
            <div class="wf-breakdown-bottleneck">
              <span
                class="wf-breakdown-bottleneck-dot"
                style={{ background: getServiceColor(bottleneck.service) }}
              />
              Bottleneck: {bottleneck.service} ({bottleneck.percentage.toFixed(0)}% of total)
            </div>
          )}

          <div class="wf-breakdown-split">
            <span class="wf-breakdown-split-label">Upstream:</span>
            <span class="wf-breakdown-split-value">{formatDuration(networkOverhead)}</span>
            <span class="wf-breakdown-split-sep">/</span>
            <span class="wf-breakdown-split-label">Downstream:</span>
            <span class="wf-breakdown-split-value">{formatDuration(childDuration)}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

function SpanDetail({ span }: { span: WaterfallSpan }) {
  return (
    <div class="wf-span-detail">
      <div class="wf-span-detail-header">
        <span class="wf-span-detail-service">{span.Service}</span>
        <span class="wf-span-detail-action">{span.Action || span.Path}</span>
        <span class={`wf-span-detail-status ${statusClass(span.StatusCode)}`}>
          {span.StatusCode}
        </span>
      </div>
      <div class="wf-span-detail-grid">
        <div class="wf-span-detail-cell">
          <span class="wf-span-detail-label">Duration</span>
          <span class="wf-span-detail-value">{formatDuration(span.DurationMs)}</span>
        </div>
        <div class="wf-span-detail-cell">
          <span class="wf-span-detail-label">Method</span>
          <span class="wf-span-detail-value">{span.Method || '--'}</span>
        </div>
        <div class="wf-span-detail-cell">
          <span class="wf-span-detail-label">Path</span>
          <span class="wf-span-detail-value wf-span-detail-mono">{span.Path || '--'}</span>
        </div>
        <div class="wf-span-detail-cell">
          <span class="wf-span-detail-label">Started</span>
          <span class="wf-span-detail-value">{formatTimestamp(span.StartTime)}</span>
        </div>
        <div class="wf-span-detail-cell">
          <span class="wf-span-detail-label">Span ID</span>
          <span class="wf-span-detail-value wf-span-detail-mono">{span.SpanID.slice(0, 16)}</span>
        </div>
        {span.ParentSpanID && (
          <div class="wf-span-detail-cell">
            <span class="wf-span-detail-label">Parent</span>
            <span class="wf-span-detail-value wf-span-detail-mono">{span.ParentSpanID.slice(0, 16)}</span>
          </div>
        )}
      </div>
      {span.Metadata && Object.keys(span.Metadata).length > 0 && (
        <div class="wf-span-detail-meta">
          <span class="wf-span-detail-label">Metadata</span>
          <pre class="wf-span-detail-meta-pre">
            {JSON.stringify(span.Metadata, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
}

/* ====== Main Waterfall Component ====== */

export function Waterfall({ traceId }: WaterfallProps) {
  const [spans, setSpans] = useState<WaterfallSpan[]>([]);
  const [loading, setLoading] = useState(false);
  const [hoveredSpanId, setHoveredSpanId] = useState<string | null>(null);
  const [selectedSpanId, setSelectedSpanId] = useState<string | null>(null);

  useEffect(() => {
    if (!traceId) {
      setSpans([]);
      return;
    }
    setLoading(true);
    setSelectedSpanId(null);
    api<WaterfallSpan[] | WaterfallSpan>(`/api/traces/${traceId}`)
      .then((data) => {
        // API may return single span or array
        if (Array.isArray(data)) {
          setSpans(data);
        } else if (data && typeof data === 'object') {
          setSpans([data as WaterfallSpan]);
        } else {
          setSpans([]);
        }
      })
      .catch((e) => { console.warn('[Waterfall] Failed to fetch spans:', e); setSpans([]); })
      .finally(() => setLoading(false));
  }, [traceId]);

  const { rows, traceStartMs, traceDurationMs } = useMemo(
    () => buildSpanTree(spans),
    [spans],
  );

  const gaps = useMemo(
    () => computeGaps(rows, traceDurationMs, traceStartMs),
    [rows, traceDurationMs, traceStartMs],
  );

  const selectedSpan = useMemo(
    () => (selectedSpanId ? spans.find((s) => s.SpanID === selectedSpanId) ?? null : null),
    [selectedSpanId, spans],
  );

  // Root span info for the header
  const rootSpan = spans.find((s) => !s.ParentSpanID);
  const rootMethod = rootSpan?.Method || '';
  const rootPath = rootSpan?.Path || '';

  if (!traceId) {
    return (
      <div class="wf-container wf-container-empty">
        <div class="wf-placeholder">Select a trace to inspect</div>
      </div>
    );
  }

  if (loading) {
    return (
      <div class="wf-container wf-container-empty">
        <div class="wf-placeholder">Loading waterfall...</div>
      </div>
    );
  }

  if (spans.length === 0) {
    return (
      <div class="wf-container wf-container-empty">
        <div class="wf-placeholder">No spans found for this trace</div>
      </div>
    );
  }

  // Generate time axis ticks
  const tickCount = 6;
  const ticks: { pct: number; label: string }[] = [];
  for (let i = 0; i <= tickCount; i++) {
    const fraction = i / tickCount;
    const ms = fraction * traceDurationMs;
    ticks.push({ pct: fraction * 100, label: formatDuration(ms) });
  }

  return (
    <div class="wf-container">
      {/* Round-trip breakdown summary */}
      <BreakdownSummary
        spans={spans}
        traceDurationMs={traceDurationMs}
        rootMethod={rootMethod}
        rootPath={rootPath}
      />

      {/* Time axis */}
      <div class="wf-time-axis">
        {ticks.map((tick, i) => (
          <span
            key={i}
            class="wf-time-tick"
            style={{ left: `calc(200px + ${tick.pct}% * (100% - 200px - 80px) / 100)` }}
          >
            {tick.label}
          </span>
        ))}
      </div>

      {/* Waterfall rows */}
      <div class="wf-rows">
        {rows.map((row) => {
          const color = getServiceColor(row.span.Service);
          const offsetPct = traceDurationMs > 0
            ? (row.offsetMs / traceDurationMs) * 100
            : 0;
          const widthPct = traceDurationMs > 0
            ? Math.max((row.durationMs / traceDurationMs) * 100, 0.5)
            : 1;
          const isHovered = hoveredSpanId === row.span.SpanID;
          const isSelected = selectedSpanId === row.span.SpanID;

          return (
            <div
              key={row.span.SpanID}
              class={`wf-row ${isHovered ? 'wf-row-hovered' : ''} ${isSelected ? 'wf-row-selected' : ''} ${row.isCriticalPath ? 'wf-row-critical' : ''}`}
              onMouseEnter={() => setHoveredSpanId(row.span.SpanID)}
              onMouseLeave={() => setHoveredSpanId(null)}
              onClick={() => setSelectedSpanId(
                selectedSpanId === row.span.SpanID ? null : row.span.SpanID,
              )}
            >
              {/* Label section */}
              <div
                class="wf-row-label"
                style={{ paddingLeft: `${12 + row.depth * 16}px` }}
              >
                {row.depth > 0 && (
                  <span class="wf-row-connector" style={{ color }}>
                    {'\u2514\u2500'}
                  </span>
                )}
                <span class="wf-row-service" style={{ color }}>
                  {row.span.Service}
                </span>
                <span class="wf-row-action">
                  {row.span.Action || ''}
                </span>
              </div>

              {/* Bar section */}
              <div class="wf-row-bar-track">
                {/* Gap indicators for this row's span */}
                {gaps
                  .filter((g) => g.parentService === row.span.Service)
                  .map((g, i) => (
                    <div
                      key={`gap-${i}`}
                      class="wf-row-gap"
                      style={{
                        left: `${g.offsetPct}%`,
                        width: `${Math.max(g.widthPct, 0.3)}%`,
                        background: `${color}15`,
                      }}
                    />
                  ))}

                {/* The span bar */}
                <div
                  class={`wf-row-bar ${statusClass(row.span.StatusCode)}`}
                  style={{
                    left: `${offsetPct}%`,
                    width: `${widthPct}%`,
                    background: `${color}40`,
                    borderColor: `${color}80`,
                  }}
                >
                  {widthPct > 4 && (
                    <span class="wf-row-bar-label">
                      {formatDuration(row.durationMs)}
                    </span>
                  )}
                </div>
              </div>

              {/* Duration column */}
              <div class="wf-row-duration">
                <span class={statusClass(row.span.StatusCode) || undefined}>
                  {formatDuration(row.durationMs)}
                </span>
              </div>
            </div>
          );
        })}
      </div>

      {/* Selected span detail panel */}
      {selectedSpan && <SpanDetail span={selectedSpan} />}
    </div>
  );
}
