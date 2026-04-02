import { useState, useEffect, useMemo } from 'preact/hooks';
import { api } from '../../lib/api';
import { Waterfall, type WaterfallSpan } from './waterfall';

interface CompareViewProps {
  traceIdA: string;
  traceIdB: string;
}

function formatDuration(ms: number): string {
  if (ms < 0.01) return `${(ms * 1000).toFixed(0)}us`;
  if (ms < 1) return `${ms.toFixed(2)}ms`;
  if (ms < 1000) return `${Math.round(ms)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

function formatDelta(deltaMs: number): string {
  const sign = deltaMs >= 0 ? '+' : '';
  return `${sign}${formatDuration(Math.abs(deltaMs))}`;
}

/** Compute total trace duration from a list of spans */
function traceDuration(spans: WaterfallSpan[]): number {
  if (spans.length === 0) return 0;
  const starts = spans.map((s) => new Date(s.StartTime).getTime());
  const ends = spans.map((s) => new Date(s.EndTime || s.StartTime).getTime());
  const traceStart = Math.min(...starts);
  const traceEnd = Math.max(...ends);
  return Math.max(traceEnd - traceStart, 0);
}

/** Compare spans by service+action to find slower operations in trace B */
function computeSlowerSpans(
  spansA: WaterfallSpan[],
  spansB: WaterfallSpan[],
): Set<string> {
  // Build avg duration by service+action for trace A
  const aByKey = new Map<string, number[]>();
  for (const s of spansA) {
    const key = `${s.Service}::${s.Action || s.Path}`;
    if (!aByKey.has(key)) aByKey.set(key, []);
    aByKey.get(key)!.push(s.DurationMs);
  }

  const avgA = new Map<string, number>();
  for (const [key, durations] of aByKey) {
    avgA.set(key, durations.reduce((a, b) => a + b, 0) / durations.length);
  }

  // Find spans in B that are slower than A's average for the same key
  const slower = new Set<string>();
  for (const s of spansB) {
    const key = `${s.Service}::${s.Action || s.Path}`;
    const baseline = avgA.get(key);
    if (baseline !== undefined && s.DurationMs > baseline * 1.2) {
      slower.add(s.SpanID);
    }
  }

  return slower;
}

export function CompareView({ traceIdA, traceIdB }: CompareViewProps) {
  const [spansA, setSpansA] = useState<WaterfallSpan[]>([]);
  const [spansB, setSpansB] = useState<WaterfallSpan[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    Promise.all([
      api<WaterfallSpan[]>(`/api/traces/${traceIdA}`).catch(() => []),
      api<WaterfallSpan[]>(`/api/traces/${traceIdB}`).catch(() => []),
    ]).then(([a, b]) => {
      setSpansA(a);
      setSpansB(b);
      setLoading(false);
    });
  }, [traceIdA, traceIdB]);

  const durationA = useMemo(() => traceDuration(spansA), [spansA]);
  const durationB = useMemo(() => traceDuration(spansB), [spansB]);
  const delta = durationB - durationA;

  const slowerSpans = useMemo(
    () => computeSlowerSpans(spansA, spansB),
    [spansA, spansB],
  );

  if (loading) {
    return (
      <div class="compare-view compare-view-loading">
        <div class="wf-placeholder">Loading comparison...</div>
      </div>
    );
  }

  const deltaClass = delta > 0 ? 'compare-delta-slower' : delta < 0 ? 'compare-delta-faster' : '';

  return (
    <div class="compare-view">
      {/* Summary banner */}
      <div class="compare-summary">
        <div class="compare-summary-item">
          <span class="compare-summary-label">Trace A</span>
          <span class="compare-summary-value">{formatDuration(durationA)} total</span>
          <span class="compare-summary-id">{traceIdA.slice(0, 12)}</span>
        </div>
        <div class="compare-summary-item">
          <span class="compare-summary-label">Trace B</span>
          <span class="compare-summary-value">{formatDuration(durationB)} total</span>
          <span class="compare-summary-id">{traceIdB.slice(0, 12)}</span>
        </div>
        <div class={`compare-summary-delta ${deltaClass}`}>
          <span class="compare-delta-symbol">{'\u0394'}</span>
          <span class="compare-delta-value">{formatDelta(delta)}</span>
          {slowerSpans.size > 0 && (
            <span class="compare-delta-detail">
              {slowerSpans.size} span{slowerSpans.size !== 1 ? 's' : ''} slower
            </span>
          )}
        </div>
      </div>

      {/* Stacked waterfalls */}
      <div class="compare-panels">
        <div class="compare-panel">
          <div class="compare-panel-label">A</div>
          <Waterfall traceId={traceIdA} />
        </div>
        <div class="compare-divider" />
        <div class="compare-panel">
          <div class="compare-panel-label compare-panel-label-b">B</div>
          <div class={`compare-panel-waterfall ${slowerSpans.size > 0 ? 'compare-has-slower' : ''}`}>
            <Waterfall traceId={traceIdB} />
          </div>
        </div>
      </div>
    </div>
  );
}
