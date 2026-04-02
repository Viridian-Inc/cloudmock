import { useState, useEffect, useMemo, useRef, useCallback } from 'preact/hooks';
import { api } from '../../lib/api';
import type { WaterfallSpan } from './waterfall';

/* ====== Types ====== */

interface FlamegraphRow {
  span: WaterfallSpan;
  depth: number;
  offsetMs: number;
  durationMs: number;
  leftPct: number;
  widthPct: number;
}

interface FlamegraphProps {
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

function statusClass(status: number): string {
  if (status >= 200 && status < 300) return 'fg-status-2xx';
  if (status >= 400 && status < 500) return 'fg-status-4xx';
  if (status >= 500) return 'fg-status-5xx';
  return '';
}

/**
 * Build flamegraph rows from spans.
 * X-axis = time (start to end of trace).
 * Y-axis = call depth (root at top, children below).
 */
function buildFlamegraphRows(spans: WaterfallSpan[]): {
  rows: FlamegraphRow[];
  traceStartMs: number;
  traceDurationMs: number;
  maxDepth: number;
} {
  if (spans.length === 0) {
    return { rows: [], traceStartMs: 0, traceDurationMs: 0, maxDepth: 0 };
  }

  const parsed = spans.map((s) => ({
    span: s,
    startMs: new Date(s.StartTime).getTime(),
    endMs: new Date(s.EndTime || s.StartTime).getTime(),
  }));

  const traceStartMs = Math.min(...parsed.map((p) => p.startMs));
  const traceEndMs = Math.max(...parsed.map((p) => p.endMs));
  const traceDurationMs = Math.max(traceEndMs - traceStartMs, 0.001);

  // Build parent-children map
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

  // DFS to build rows with depth
  const rows: FlamegraphRow[] = [];
  let maxDepth = 0;

  function walk(span: WaterfallSpan, depth: number) {
    const startMs = new Date(span.StartTime).getTime();
    const offsetMs = startMs - traceStartMs;
    const durationMs = span.DurationMs > 0 ? span.DurationMs : 0.001;

    const leftPct = (offsetMs / traceDurationMs) * 100;
    const widthPct = Math.max((durationMs / traceDurationMs) * 100, 0.3);

    rows.push({ span, depth, offsetMs, durationMs, leftPct, widthPct });
    if (depth > maxDepth) maxDepth = depth;

    const children = childrenMap.get(span.SpanID) || [];
    for (const child of children) {
      walk(child, depth + 1);
    }
  }

  for (const root of rootSpans) {
    walk(root, 0);
  }

  return { rows, traceStartMs, traceDurationMs, maxDepth };
}

/* ====== Component ====== */

export function Flamegraph({ traceId }: FlamegraphProps) {
  const [spans, setSpans] = useState<WaterfallSpan[]>([]);
  const [loading, setLoading] = useState(false);
  const [hoveredSpanId, setHoveredSpanId] = useState<string | null>(null);
  const [selectedSpanId, setSelectedSpanId] = useState<string | null>(null);
  const [tooltipPos, setTooltipPos] = useState<{ x: number; y: number } | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!traceId) {
      setSpans([]);
      return;
    }
    setLoading(true);
    setSelectedSpanId(null);
    api<WaterfallSpan[]>(`/api/traces/${traceId}`)
      .then(setSpans)
      .catch(() => setSpans([]))
      .finally(() => setLoading(false));
  }, [traceId]);

  const { rows, traceStartMs, traceDurationMs, maxDepth } = useMemo(
    () => buildFlamegraphRows(spans),
    [spans],
  );

  const selectedSpan = useMemo(
    () => (selectedSpanId ? spans.find((s) => s.SpanID === selectedSpanId) ?? null : null),
    [selectedSpanId, spans],
  );

  const hoveredSpan = useMemo(
    () => (hoveredSpanId ? spans.find((s) => s.SpanID === hoveredSpanId) ?? null : null),
    [hoveredSpanId, spans],
  );

  const handleMouseMove = useCallback((e: MouseEvent) => {
    if (!containerRef.current) return;
    const rect = containerRef.current.getBoundingClientRect();
    setTooltipPos({ x: e.clientX - rect.left, y: e.clientY - rect.top });
  }, []);

  if (!traceId) {
    return (
      <div class="fg-container fg-container-empty">
        <div class="fg-placeholder">Select a trace to inspect</div>
      </div>
    );
  }

  if (loading) {
    return (
      <div class="fg-container fg-container-empty">
        <div class="fg-placeholder">Loading flamegraph...</div>
      </div>
    );
  }

  if (spans.length === 0) {
    return (
      <div class="fg-container fg-container-empty">
        <div class="fg-placeholder">No spans found for this trace</div>
      </div>
    );
  }

  // Root span info
  const rootSpan = spans.find((s) => !s.ParentSpanID);
  const ROW_HEIGHT = 24;
  const chartHeight = (maxDepth + 1) * ROW_HEIGHT + 8;

  // Time axis ticks
  const tickCount = 6;
  const ticks: { pct: number; label: string }[] = [];
  for (let i = 0; i <= tickCount; i++) {
    const fraction = i / tickCount;
    const ms = fraction * traceDurationMs;
    ticks.push({ pct: fraction * 100, label: formatDuration(ms) });
  }

  // Collect unique services for legend
  const services = [...new Set(spans.map((s) => s.Service))];

  return (
    <div class="fg-container" ref={containerRef} onMouseMove={handleMouseMove}>
      {/* Header */}
      <div class="fg-header">
        <div class="fg-header-info">
          {rootSpan && (
            <>
              <span class="fg-header-method">{rootSpan.Method}</span>
              <span class="fg-header-path">{rootSpan.Path}</span>
            </>
          )}
          <span class="fg-header-duration">
            Total: <strong>{formatDuration(traceDurationMs)}</strong>
          </span>
        </div>
        <div class="fg-legend">
          {services.map((svc) => (
            <span key={svc} class="fg-legend-item">
              <span class="fg-legend-dot" style={{ background: getServiceColor(svc) }} />
              {svc}
            </span>
          ))}
        </div>
      </div>

      {/* Time axis */}
      <div class="fg-time-axis">
        {ticks.map((tick, i) => (
          <span key={i} class="fg-time-tick" style={{ left: `${tick.pct}%` }}>
            {tick.label}
          </span>
        ))}
      </div>

      {/* Flamegraph chart */}
      <div class="fg-chart" style={{ height: `${chartHeight}px` }}>
        {rows.map((row) => {
          const color = getServiceColor(row.span.Service);
          const isHovered = hoveredSpanId === row.span.SpanID;
          const isSelected = selectedSpanId === row.span.SpanID;
          const top = row.depth * ROW_HEIGHT;

          return (
            <div
              key={row.span.SpanID}
              class={`fg-block ${isHovered ? 'fg-block-hovered' : ''} ${isSelected ? 'fg-block-selected' : ''} ${statusClass(row.span.StatusCode)}`}
              style={{
                left: `${row.leftPct}%`,
                width: `${row.widthPct}%`,
                top: `${top}px`,
                height: `${ROW_HEIGHT - 2}px`,
                background: `${color}30`,
                borderColor: `${color}60`,
              }}
              onMouseEnter={() => setHoveredSpanId(row.span.SpanID)}
              onMouseLeave={() => setHoveredSpanId(null)}
              onClick={() => setSelectedSpanId(
                selectedSpanId === row.span.SpanID ? null : row.span.SpanID,
              )}
            >
              {row.widthPct > 3 && (
                <span class="fg-block-label" style={{ color }}>
                  {row.span.Service}
                  {row.widthPct > 8 && (
                    <span class="fg-block-duration"> {formatDuration(row.durationMs)}</span>
                  )}
                </span>
              )}
            </div>
          );
        })}
      </div>

      {/* Hover tooltip */}
      {hoveredSpan && tooltipPos && (
        <div
          class="fg-tooltip"
          style={{
            left: `${Math.min(tooltipPos.x + 12, (containerRef.current?.clientWidth || 400) - 220)}px`,
            top: `${tooltipPos.y + 12}px`,
          }}
        >
          <div class="fg-tooltip-service" style={{ color: getServiceColor(hoveredSpan.Service) }}>
            {hoveredSpan.Service}
          </div>
          <div class="fg-tooltip-action">{hoveredSpan.Action || hoveredSpan.Path}</div>
          <div class="fg-tooltip-row">
            <span>Duration:</span>
            <span>{formatDuration(hoveredSpan.DurationMs)}</span>
          </div>
          <div class="fg-tooltip-row">
            <span>Status:</span>
            <span class={statusClass(hoveredSpan.StatusCode)}>{hoveredSpan.StatusCode}</span>
          </div>
          <div class="fg-tooltip-row">
            <span>Started:</span>
            <span>{formatTimestamp(hoveredSpan.StartTime)}</span>
          </div>
        </div>
      )}

      {/* Selected span detail */}
      {selectedSpan && (
        <div class="fg-detail">
          <div class="fg-detail-header">
            <span class="fg-detail-service" style={{ color: getServiceColor(selectedSpan.Service) }}>
              {selectedSpan.Service}
            </span>
            <span class="fg-detail-action">{selectedSpan.Action || selectedSpan.Path}</span>
            <span class={`fg-detail-status ${statusClass(selectedSpan.StatusCode)}`}>
              {selectedSpan.StatusCode}
            </span>
          </div>
          <div class="fg-detail-grid">
            <div class="fg-detail-cell">
              <span class="fg-detail-label">Duration</span>
              <span class="fg-detail-value">{formatDuration(selectedSpan.DurationMs)}</span>
            </div>
            <div class="fg-detail-cell">
              <span class="fg-detail-label">Method</span>
              <span class="fg-detail-value">{selectedSpan.Method || '--'}</span>
            </div>
            <div class="fg-detail-cell">
              <span class="fg-detail-label">Path</span>
              <span class="fg-detail-value fg-detail-mono">{selectedSpan.Path || '--'}</span>
            </div>
            <div class="fg-detail-cell">
              <span class="fg-detail-label">Started</span>
              <span class="fg-detail-value">{formatTimestamp(selectedSpan.StartTime)}</span>
            </div>
            <div class="fg-detail-cell">
              <span class="fg-detail-label">Span ID</span>
              <span class="fg-detail-value fg-detail-mono">{selectedSpan.SpanID.slice(0, 16)}</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
