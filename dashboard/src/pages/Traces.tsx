import { useState, useEffect, useMemo, useCallback, useRef } from 'preact/hooks';
import { api } from '../api';
import { statusClass } from '../components/StatusBadge';
import { RefreshIcon, DownloadIcon, ChevRightIcon, CopyIcon } from '../components/Icons';
import { fmtTime, fmtDuration, copyToClipboard } from '../utils';

interface TraceSummary {
  trace_id: string;
  root_service: string;
  root_action: string;
  method: string;
  path: string;
  duration_ms: number;
  status_code: number;
  span_count: number;
  has_error: boolean;
  start_time: string;
}

interface TimelineSpan {
  span_id: string;
  parent_span_id: string;
  service: string;
  action: string;
  start_offset_ms: number;
  duration_ms: number;
  status_code: number;
  error: string;
  depth: number;
}

interface TraceDetail {
  trace_id: string;
  span_id: string;
  parent_span_id: string;
  service: string;
  action: string;
  method: string;
  path: string;
  start_time: string;
  end_time: string;
  duration_ns: number;
  duration_ms: number;
  status_code: number;
  error: string;
  children: TraceDetail[];
}

interface TracesPageProps {
  showToast: (msg: string) => void;
}

const SERVICE_COLORS: Record<string, string> = {
  dynamodb: '#3B82F6',
  lambda: '#F59E0B',
  sqs: '#F97316',
  sns: '#A855F7',
  s3: '#10B981',
  'cognito-idp': '#8B5CF6',
  events: '#EC4899',
  ses: '#06B6D4',
  kms: '#6366F1',
  sts: '#94A3B8',
  iam: '#64748b',
  logs: '#14B8A6',
  monitoring: '#EC4899',
  ssm: '#6366F1',
  secretsmanager: '#6366F1',
};

function getServiceColor(service: string): string {
  return SERVICE_COLORS[service] || '#64748b';
}

export function TracesPage({ showToast }: TracesPageProps) {
  const [traces, setTraces] = useState<TraceSummary[]>([]);
  const [expandedTraceId, setExpandedTraceId] = useState<string | null>(null);
  const [timeline, setTimeline] = useState<TimelineSpan[]>([]);
  const [traceDetail, setTraceDetail] = useState<TraceDetail | null>(null);
  const [svcFilter, setSvcFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [textFilter, setTextFilter] = useState('');
  const [services, setServices] = useState<string[]>([]);
  const [hoveredSpan, setHoveredSpan] = useState<string | null>(null);
  const [selectedSpan, setSelectedSpan] = useState<TimelineSpan | null>(null);

  const loadTraces = useCallback(() => {
    let url = '/api/traces?limit=200';
    if (svcFilter) url += `&service=${encodeURIComponent(svcFilter)}`;
    if (statusFilter === 'error') url += '&error=true';
    if (statusFilter === 'success') url += '&error=false';
    api(url).then(setTraces).catch(() => {});
  }, [svcFilter, statusFilter]);

  useEffect(() => {
    loadTraces();
    api('/api/services').then((s: any[]) => setServices(s.map((x: any) => x.name).sort())).catch(() => {});
  }, []);

  useEffect(() => {
    loadTraces();
  }, [svcFilter, statusFilter]);

  useEffect(() => {
    const iv = setInterval(loadTraces, 5000);
    return () => clearInterval(iv);
  }, [loadTraces]);

  const filtered = useMemo(() => {
    if (!textFilter) return traces;
    const q = textFilter.toLowerCase();
    return traces.filter((t) => {
      const haystack = `${t.trace_id} ${t.root_service} ${t.root_action} ${t.method} ${t.path}`.toLowerCase();
      return haystack.includes(q);
    });
  }, [traces, textFilter]);

  function toggleTrace(traceId: string) {
    if (expandedTraceId === traceId) {
      setExpandedTraceId(null);
      setTimeline([]);
      setTraceDetail(null);
      setSelectedSpan(null);
      return;
    }
    setExpandedTraceId(traceId);
    setSelectedSpan(null);
    api(`/api/traces/${traceId}/timeline`).then(setTimeline).catch(() => setTimeline([]));
    api(`/api/traces/${traceId}`).then(setTraceDetail).catch(() => setTraceDetail(null));
  }

  function exportTrace(e: Event, traceId: string) {
    e.stopPropagation();
    api(`/api/traces/${traceId}`).then((data: any) => {
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `trace-${traceId}.json`;
      a.click();
      URL.revokeObjectURL(url);
      showToast('Trace exported');
    }).catch(() => showToast('Export failed'));
  }

  return (
    <div>
      <div class="flex items-center justify-between mb-6">
        <div>
          <h1 class="page-title">Distributed Traces</h1>
          <p class="page-desc">Request correlation and cross-service tracing</p>
        </div>
        <button class="btn btn-ghost btn-sm" onClick={loadTraces}>
          <RefreshIcon /> Refresh
        </button>
      </div>

      <div class="filters-bar">
        <select class="select" value={svcFilter} onChange={(e) => setSvcFilter((e.target as HTMLSelectElement).value)}>
          <option value="">All Services</option>
          {services.map(s => <option value={s}>{s}</option>)}
        </select>
        <select class="select" value={statusFilter} onChange={(e) => setStatusFilter((e.target as HTMLSelectElement).value)}>
          <option value="">All Status</option>
          <option value="success">Success</option>
          <option value="error">Errors</option>
        </select>
        <input class="input input-search" placeholder="Search traces..." value={textFilter} onInput={(e) => setTextFilter((e.target as HTMLInputElement).value)} />
        <span class="text-sm text-muted ml-auto">{filtered.length} traces</span>
      </div>

      <div class="card">
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th style="width:40px"></th>
                <th style="width:100px">Time</th>
                <th>Service</th>
                <th>Action</th>
                <th style="width:80px">Status</th>
                <th style="width:80px">Duration</th>
                <th style="width:60px">Spans</th>
                <th style="width:40px"></th>
              </tr>
            </thead>
            <tbody>
              {filtered.length === 0 ? (
                <tr><td colSpan={8} class="empty-state">No traces recorded yet. Make some API requests to generate traces.</td></tr>
              ) : filtered.map((t) => (
                <>
                  <tr
                    class={`clickable ${expandedTraceId === t.trace_id ? 'expanded' : ''}`}
                    onClick={() => toggleTrace(t.trace_id)}
                    key={t.trace_id}
                  >
                    <td>
                      <span class={`chev-icon ${expandedTraceId === t.trace_id ? 'chev-open' : ''}`}>
                        <ChevRightIcon />
                      </span>
                    </td>
                    <td class="font-mono text-sm">{fmtTime(t.start_time)}</td>
                    <td><span style="font-weight:600">{t.root_service || '(unknown)'}</span></td>
                    <td class="font-mono text-sm">{t.root_action}</td>
                    <td>
                      <span class={`status-pill ${t.has_error ? 'status-5xx' : 'status-2xx'}`}>
                        {t.status_code}
                      </span>
                    </td>
                    <td class="font-mono text-sm">{fmtDuration(t.duration_ms)}</td>
                    <td class="text-center">
                      <span class="trace-span-count">{t.span_count}</span>
                    </td>
                    <td>
                      <button class="btn-icon btn-sm btn-ghost" title="Export as JSON" onClick={(e) => exportTrace(e, t.trace_id)}>
                        <DownloadIcon />
                      </button>
                    </td>
                  </tr>
                  {expandedTraceId === t.trace_id && (
                    <tr>
                      <td colSpan={8} style="padding:0">
                        <WaterfallView
                          timeline={timeline}
                          traceDetail={traceDetail}
                          traceSummary={t}
                          hoveredSpan={hoveredSpan}
                          setHoveredSpan={setHoveredSpan}
                          selectedSpan={selectedSpan}
                          setSelectedSpan={setSelectedSpan}
                          showToast={showToast}
                        />
                      </td>
                    </tr>
                  )}
                </>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

function CompareButton({ traceId }: { traceId: string }) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('click', handler, true);
    return () => document.removeEventListener('click', handler, true);
  }, [open]);

  const compareWithAnother = () => {
    const otherId = prompt('Enter trace ID to compare with:');
    if (otherId) {
      location.hash = `#/traces/compare?a=${traceId}&b=${otherId}`;
    }
    setOpen(false);
  };

  const compareWithBaseline = () => {
    location.hash = `#/traces/compare?a=${traceId}&baseline=true`;
    setOpen(false);
  };

  return (
    <div ref={ref} style={{ position: 'relative', display: 'inline-block' }}>
      <button
        onClick={(e) => { e.stopPropagation(); setOpen(!open); }}
        style={{
          padding: '3px 8px', borderRadius: '4px', border: '1px solid var(--border)',
          background: 'transparent', color: 'var(--text-primary)', cursor: 'pointer', fontSize: '11px',
          whiteSpace: 'nowrap',
        }}
      >
        Compare
      </button>
      {open && (
        <div style={{
          position: 'absolute', right: 0, top: '100%', marginTop: '4px', zIndex: 100,
          background: 'var(--bg-primary)', border: '1px solid var(--border)', borderRadius: '6px',
          boxShadow: '0 4px 12px rgba(0,0,0,0.15)', minWidth: '200px', overflow: 'hidden',
        }}>
          <button
            onClick={compareWithAnother}
            style={{
              display: 'block', width: '100%', padding: '8px 12px', border: 'none',
              background: 'transparent', color: 'var(--text-primary)', cursor: 'pointer',
              fontSize: '12px', textAlign: 'left',
            }}
            onMouseEnter={(e: any) => e.target.style.background = 'var(--bg-secondary)'}
            onMouseLeave={(e: any) => e.target.style.background = 'transparent'}
          >
            Compare with another trace
          </button>
          <button
            onClick={compareWithBaseline}
            style={{
              display: 'block', width: '100%', padding: '8px 12px', border: 'none',
              background: 'transparent', color: 'var(--text-primary)', cursor: 'pointer',
              fontSize: '12px', textAlign: 'left',
            }}
            onMouseEnter={(e: any) => e.target.style.background = 'var(--bg-secondary)'}
            onMouseLeave={(e: any) => e.target.style.background = 'transparent'}
          >
            Compare with baseline
          </button>
        </div>
      )}
    </div>
  );
}

interface WaterfallViewProps {
  timeline: TimelineSpan[];
  traceDetail: TraceDetail | null;
  traceSummary: TraceSummary;
  hoveredSpan: string | null;
  setHoveredSpan: (id: string | null) => void;
  selectedSpan: TimelineSpan | null;
  setSelectedSpan: (span: TimelineSpan | null) => void;
  showToast: (msg: string) => void;
}

function WaterfallView({ timeline, traceDetail, traceSummary, hoveredSpan, setHoveredSpan, selectedSpan, setSelectedSpan, showToast }: WaterfallViewProps) {
  if (timeline.length === 0) {
    return (
      <div class="waterfall-container" style="padding:24px;text-align:center;color:var(--n400)">
        Loading trace timeline...
      </div>
    );
  }

  const totalDuration = traceSummary.duration_ms || Math.max(...timeline.map(s => s.start_offset_ms + s.duration_ms), 1);
  const barAreaWidth = 500;
  const labelWidth = 280;
  const durationLabelWidth = 80;
  const totalWidth = labelWidth + barAreaWidth + durationLabelWidth;
  const rowHeight = 32;
  const rulerHeight = 24;
  const totalHeight = rulerHeight + timeline.length * rowHeight;

  // Generate ruler ticks
  const tickCount = 5;
  const ticks: number[] = [];
  for (let i = 0; i <= tickCount; i++) {
    ticks.push((totalDuration / tickCount) * i);
  }

  return (
    <div class="waterfall-container">
      <div class="waterfall-header">
        <div class="flex items-center justify-between" style="padding:12px 16px">
          <div>
            <span class="font-mono text-sm" style="color:var(--n400)">Trace: </span>
            <span class="font-mono text-sm">{traceSummary.trace_id}</span>
            <button class="btn-icon btn-sm btn-ghost" style="margin-left:4px" title="Copy trace ID"
              onClick={() => { copyToClipboard(traceSummary.trace_id); showToast('Copied trace ID'); }}>
              <CopyIcon />
            </button>
          </div>
          <div class="flex items-center" style="gap:8px">
            <div class="text-sm text-muted">
              {timeline.length} span{timeline.length !== 1 ? 's' : ''} | {fmtDuration(totalDuration)}
            </div>
            <CompareButton traceId={traceSummary.trace_id} />
          </div>
        </div>
      </div>
      <div class="waterfall-scroll" style={`overflow-x:auto`}>
        <svg width={totalWidth} height={totalHeight} class="waterfall-svg">
          {/* Ruler */}
          <g transform={`translate(${labelWidth}, 0)`}>
            <line x1="0" y1={rulerHeight} x2={barAreaWidth} y2={rulerHeight}
              stroke="var(--n200)" stroke-width="1" />
            {ticks.map((t, i) => {
              const x = (t / totalDuration) * barAreaWidth;
              return (
                <g key={i}>
                  <line x1={x} y1={rulerHeight - 4} x2={x} y2={rulerHeight}
                    stroke="var(--n400)" stroke-width="1" />
                  <text x={x} y={rulerHeight - 8} text-anchor="middle"
                    fill="var(--n400)" font-size="10" font-family="monospace">
                    {fmtDuration(t)}
                  </text>
                  {/* Gridline */}
                  <line x1={x} y1={rulerHeight} x2={x} y2={totalHeight}
                    stroke="var(--n200)" stroke-width="0.5" stroke-dasharray="4,4" opacity="0.5" />
                </g>
              );
            })}
          </g>

          {/* Spans */}
          {timeline.map((span, i) => {
            const y = rulerHeight + i * rowHeight;
            const barX = (span.start_offset_ms / totalDuration) * barAreaWidth;
            const barW = Math.max((span.duration_ms / totalDuration) * barAreaWidth, 2);
            const color = getServiceColor(span.service);
            const isHovered = hoveredSpan === span.span_id;
            const isSelected = selectedSpan?.span_id === span.span_id;
            const isError = span.status_code >= 400 || !!span.error;
            const indent = span.depth * 16;

            return (
              <g key={span.span_id}
                onMouseEnter={() => setHoveredSpan(span.span_id)}
                onMouseLeave={() => setHoveredSpan(null)}
                onClick={() => setSelectedSpan(isSelected ? null : span)}
                style="cursor:pointer"
              >
                {/* Row background */}
                <rect x="0" y={y} width={totalWidth} height={rowHeight}
                  fill={isSelected ? 'var(--n100)' : isHovered ? 'var(--n50)' : 'transparent'} />

                {/* Row border */}
                <line x1="0" y1={y + rowHeight} x2={totalWidth} y2={y + rowHeight}
                  stroke="var(--n200)" stroke-width="0.5" opacity="0.3" />

                {/* Service + Action label */}
                <g transform={`translate(${8 + indent}, ${y})`}>
                  {span.depth > 0 && (
                    <g>
                      <line x1="-4" y1="0" x2="-4" y2={rowHeight / 2}
                        stroke="var(--n400)" stroke-width="1" opacity="0.4" />
                      <line x1="-4" y1={rowHeight / 2} x2="4" y2={rowHeight / 2}
                        stroke="var(--n400)" stroke-width="1" opacity="0.4" />
                    </g>
                  )}
                  <circle cx="8" cy={rowHeight / 2} r="4" fill={color} opacity="0.9" />
                  <text x="18" y={rowHeight / 2 + 4} fill="var(--n800)"
                    font-size="12" font-weight="600" font-family="var(--font-sans, system-ui)">
                    {span.service || '(root)'}
                  </text>
                  <text x="18" y={rowHeight / 2 + 4} fill="var(--n400)"
                    font-size="11" font-family="monospace" dx={`${(span.service || '(root)').length * 7.2 + 6}px`}>
                    {span.action}
                  </text>
                </g>

                {/* Bar */}
                <g transform={`translate(${labelWidth}, ${y})`}>
                  <rect x={barX} y={6} width={barW} height={rowHeight - 12} rx="3"
                    fill={isError ? 'var(--error)' : color}
                    opacity={isHovered || isSelected ? 1 : 0.8} />
                </g>

                {/* Duration label */}
                <text x={labelWidth + barAreaWidth + 8} y={y + rowHeight / 2 + 4}
                  fill="var(--n400)" font-size="11" font-family="monospace">
                  {fmtDuration(span.duration_ms)}
                </text>
              </g>
            );
          })}
        </svg>
      </div>

      {/* Selected span detail */}
      {selectedSpan && (
        <div class="waterfall-detail">
          <div class="waterfall-detail-header">
            <span style="font-weight:600">{selectedSpan.service}</span>
            <span class="font-mono" style="margin-left:8px">{selectedSpan.action}</span>
            <span class={`status-pill ${statusClass(selectedSpan.status_code)}`} style="margin-left:8px">
              {selectedSpan.status_code}
            </span>
          </div>
          <table class="waterfall-detail-table">
            <tbody>
              <tr><td>Span ID</td><td class="font-mono text-sm">{selectedSpan.span_id}</td></tr>
              {selectedSpan.parent_span_id && <tr><td>Parent Span</td><td class="font-mono text-sm">{selectedSpan.parent_span_id}</td></tr>}
              <tr><td>Start Offset</td><td>{fmtDuration(selectedSpan.start_offset_ms)}</td></tr>
              <tr><td>Duration</td><td>{fmtDuration(selectedSpan.duration_ms)}</td></tr>
              <tr><td>Status</td><td>{selectedSpan.status_code}</td></tr>
              {selectedSpan.error && <tr><td>Error</td><td style="color:var(--error)">{selectedSpan.error}</td></tr>}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
