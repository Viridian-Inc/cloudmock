import { useState, useEffect, useMemo, useRef, useCallback, useReducer } from 'preact/hooks';
import type { TopoNode, TopoEdge } from './index';
import type { RequestEvent, IncidentInfo } from '../../lib/types';
import { api, getAdminBase } from '../../lib/api';
import { getBlastRadius } from '../../lib/health';
import type { ServiceMetrics } from '../../lib/health';
import { Waterfall } from '../traces/waterfall';
import {
  normalizeRequestEvent,
  getServiceKey,
  buildResponseSummary,
  filterRequestsByEdgeServices,
  buildFlows,
  buildFallbackFlows,
  filterFlowsByMethod,
  type RequestFlow,
  type OutboundCall,
} from './request-trace-utils';
import { panelReducer, initialPanelState } from './request-trace-reducer';

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

interface TraceEntry {
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

/* ------------------------------------------------------------------ */
/*  Props                                                              */
/* ------------------------------------------------------------------ */

export interface RequestTracePanelProps {
  node: TopoNode;
  edges: TopoEdge[];
  allNodes: TopoNode[];
  metrics: ServiceMetrics[];
  incidents: IncidentInfo[];
  onClose: () => void;
}

/* ------------------------------------------------------------------ */
/*  Helpers                                                            */
/* ------------------------------------------------------------------ */

function statusClass(code: number): string {
  if (code >= 500) return 'status-5xx';
  if (code >= 400) return 'status-4xx';
  if (code >= 200 && code < 300) return 'status-2xx';
  return 'status-other';
}

function statusDotClass(code: number): string {
  if (code >= 500) return 'replay-dot-error';
  if (code >= 400) return 'replay-dot-warn';
  return 'replay-dot-ok';
}

function formatMs(ms: number): string {
  if (ms < 1) return `${ms.toFixed(1)}ms`;
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)}s`;
  return `${Math.round(ms)}ms`;
}

function formatTime(iso: string): string {
  try {
    const d = new Date(iso);
    return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  } catch {
    return iso;
  }
}

/* getServiceKey, buildResponseSummary imported from ./request-trace-utils */

/* buildFlows imported from ./request-trace-utils */

/* ------------------------------------------------------------------ */
/*  Replay Timeline Component                                          */
/* ------------------------------------------------------------------ */

interface ReplayTimelineProps {
  flows: RequestFlow[];
  selectedFlowId: string | null;
  onSelectFlow: (id: string) => void;
  timeRange: { start: number; end: number };
  onScrub: (range: { start: number; end: number }) => void;
}

const TIMELINE_HEIGHT = 40;
const TIMELINE_PADDING = 16;

function ReplayTimeline({ flows, selectedFlowId, onSelectFlow, timeRange, onScrub }: ReplayTimelineProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [dragging, setDragging] = useState<'start' | 'end' | null>(null);
  const [dragRange, setDragRange] = useState<{ start: number; end: number } | null>(null);

  const activeRange = dragRange ?? timeRange;
  const allTimestamps = flows.map((f) => new Date(f.timestamp).getTime());
  const minT = allTimestamps.length > 0 ? Math.min(...allTimestamps) : Date.now() - 300000;
  const maxT = Math.max(Date.now(), ...allTimestamps);
  const span = maxT - minT || 1;

  const toX = useCallback((ts: number) => {
    return TIMELINE_PADDING + ((ts - minT) / span) * (300 - TIMELINE_PADDING * 2);
  }, [minT, span]);

  const fromX = useCallback((x: number) => {
    return minT + ((x - TIMELINE_PADDING) / (300 - TIMELINE_PADDING * 2)) * span;
  }, [minT, span]);

  const handleMouseDown = useCallback((edge: 'start' | 'end') => (e: MouseEvent) => {
    e.preventDefault();
    setDragging(edge);
    setDragRange({ ...activeRange });
  }, [activeRange]);

  useEffect(() => {
    if (!dragging) return;

    const handleMouseMove = (e: MouseEvent) => {
      if (!svgRef.current) return;
      const rect = svgRef.current.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const svgWidth = rect.width;
      const scaledX = (x / svgWidth) * 300;
      const ts = fromX(scaledX);

      setDragRange((prev) => {
        if (!prev) return prev;
        if (dragging === 'start') {
          return { start: Math.min(ts, prev.end - 10000), end: prev.end };
        }
        return { start: prev.start, end: Math.max(ts, prev.start + 10000) };
      });
    };

    const handleMouseUp = () => {
      setDragging(null);
      if (dragRange) {
        onScrub(dragRange);
      }
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };
  }, [dragging, dragRange, fromX, onScrub]);

  // Time axis labels
  const labelCount = 5;
  const labels: { x: number; text: string }[] = [];
  for (let i = 0; i < labelCount; i++) {
    const t = minT + (i / (labelCount - 1)) * span;
    labels.push({ x: toX(t), text: formatTime(new Date(t).toISOString()) });
  }

  const rangeStartX = toX(activeRange.start);
  const rangeEndX = toX(activeRange.end);

  return (
    <div class="replay-timeline">
      <div class="replay-timeline-header">
        <span class="replay-timeline-label">Request Timeline</span>
        <span class="replay-timeline-range">
          {formatTime(new Date(activeRange.start).toISOString())} - {formatTime(new Date(activeRange.end).toISOString())}
        </span>
      </div>
      <svg
        ref={svgRef}
        class="replay-timeline-svg"
        viewBox={`0 0 300 ${TIMELINE_HEIGHT}`}
        preserveAspectRatio="none"
      >
        {/* Background */}
        <rect x="0" y="0" width="300" height={TIMELINE_HEIGHT} fill="transparent" />

        {/* Selection range */}
        <rect
          x={rangeStartX}
          y="2"
          width={Math.max(0, rangeEndX - rangeStartX)}
          height={TIMELINE_HEIGHT - 12}
          fill="rgba(74, 229, 248, 0.08)"
          stroke="rgba(74, 229, 248, 0.2)"
          stroke-width="0.5"
          rx="2"
        />

        {/* Drag handles */}
        <rect
          x={rangeStartX - 2}
          y="2"
          width="4"
          height={TIMELINE_HEIGHT - 12}
          fill="rgba(74, 229, 248, 0.4)"
          rx="1"
          style={{ cursor: 'ew-resize' }}
          onMouseDown={handleMouseDown('start')}
        />
        <rect
          x={rangeEndX - 2}
          y="2"
          width="4"
          height={TIMELINE_HEIGHT - 12}
          fill="rgba(74, 229, 248, 0.4)"
          rx="1"
          style={{ cursor: 'ew-resize' }}
          onMouseDown={handleMouseDown('end')}
        />

        {/* Request dots */}
        {flows.map((f) => {
          const ts = new Date(f.timestamp).getTime();
          const x = toX(ts);
          const isSelected = f.id === selectedFlowId;
          return (
            <circle
              key={f.id}
              cx={x}
              cy={(TIMELINE_HEIGHT - 10) / 2}
              r={isSelected ? 4 : 2.5}
              class={`replay-dot ${statusDotClass(f.statusCode)} ${isSelected ? 'replay-dot-selected' : ''}`}
              onClick={() => onSelectFlow(f.id)}
              style={{ cursor: 'pointer' }}
            />
          );
        })}

        {/* Time labels */}
        {labels.map((l, i) => (
          <text
            key={i}
            x={l.x}
            y={TIMELINE_HEIGHT - 1}
            class="replay-timeline-tick"
            text-anchor="middle"
          >
            {l.text}
          </text>
        ))}
      </svg>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Blast Radius Bar Component                                         */
/* ------------------------------------------------------------------ */

interface BlastRadiusBarProps {
  nodeId: string;
  edges: TopoEdge[];
  allNodes: TopoNode[];
  metrics: ServiceMetrics[];
}

function BlastRadiusBar({ nodeId, edges, allNodes, metrics }: BlastRadiusBarProps) {
  const radius = useMemo(() => getBlastRadius(nodeId, edges), [nodeId, edges]);
  const nodeMap = useMemo(() => new Map(allNodes.map((n) => [n.id, n])), [allNodes]);

  if (radius.size === 0) {
    return (
      <div class="blast-radius-bar">
        <div class="blast-radius-bar-header">
          <span class="blast-radius-bar-label">Blast Radius</span>
          <span class="blast-radius-bar-count">0 services</span>
        </div>
        <div class="blast-radius-bar-empty">No downstream services affected</div>
      </div>
    );
  }

  const pills = Array.from(radius).map((id) => {
    const node = nodeMap.get(id);
    const label = node?.label || id.replace(/^svc:|^ms:|^cat:/, '');
    const svcKey = node?.service || id.replace(/^svc:/, '');
    const m = metrics.find((sm) => sm.service === svcKey);
    const hasErrors = m ? m.errorRate > 0.01 : false;
    return { id, label, hasErrors };
  });

  return (
    <div class="blast-radius-bar">
      <div class="blast-radius-bar-header">
        <span class="blast-radius-bar-label">Blast Radius</span>
        <span class="blast-radius-bar-count">
          {radius.size} service{radius.size !== 1 ? 's' : ''}
        </span>
      </div>
      <div class="blast-radius-bar-pills">
        {pills.map((p) => (
          <span key={p.id} class={`blast-radius-pill ${p.hasErrors ? 'blast-radius-pill-error' : ''}`}>
            <span class={`blast-radius-pill-dot ${p.hasErrors ? 'dot-error' : 'dot-ok'}`} />
            {p.label}
          </span>
        ))}
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  AI Explain Section                                                 */
/* ------------------------------------------------------------------ */

interface AiExplainSectionProps {
  flowId: string;
}

function AiExplainSection({ flowId }: AiExplainSectionProps) {
  const [explanation, setExplanation] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleExplain = useCallback(async () => {
    setLoading(true);
    setError(null);
    setExplanation(null);

    try {
      const base = getAdminBase();
      const res = await fetch(`${base}/api/explain/${encodeURIComponent(flowId)}`, {
        headers: { 'Accept': 'application/json' },
      });

      if (!res.ok) {
        const body = await res.text().catch(() => '');
        throw new Error(`API ${res.status}: ${res.statusText}${body ? ` \u2014 ${body}` : ''}`);
      }

      const data = await res.json();
      const text = data.explanation || data.text || data.content || '';
      if (!text) {
        setError('The AI explanation endpoint returned an empty response.');
      } else {
        setExplanation(text);
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to get explanation';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, [flowId]);

  return (
    <div class="ai-explain-section">
      {!explanation && !loading && !error && (
        <button class="ai-explain-btn" onClick={handleExplain}>
          {'\uD83E\uDD16'} AI Explain
        </button>
      )}

      {loading && (
        <div class="ai-explain-loading">
          <div class="ai-explain-loading-bar" />
          <span class="ai-explain-loading-text">Analyzing request...</span>
        </div>
      )}

      {error && (
        <div class="ai-explain-error">
          <span class="ai-explain-error-icon">!</span>
          {error}
          <button class="ai-explain-retry" onClick={handleExplain}>Retry</button>
        </div>
      )}

      {explanation && (
        <div class="ai-explain-result">
          <div class="ai-explain-result-header">
            <span class="ai-explain-result-label">{'\uD83E\uDD16'} AI Explanation</span>
          </div>
          <div class="ai-explain-result-body">
            {renderMarkdown(explanation)}
          </div>
        </div>
      )}
    </div>
  );
}

/**
 * Render simple markdown: headings, lists, code blocks, inline code, bold.
 * Mirrors the approach in the ai-debug view.
 */
function renderMarkdown(text: string): preact.VNode[] {
  const elements: preact.VNode[] = [];
  const codeBlockRegex = /```(\w*)\n([\s\S]*?)```/g;
  let lastIndex = 0;
  let match;

  while ((match = codeBlockRegex.exec(text)) !== null) {
    if (match.index > lastIndex) {
      elements.push(...renderMarkdownLines(text.slice(lastIndex, match.index), lastIndex));
    }
    elements.push(
      <pre class="ai-md-code-block" key={`cb-${match.index}`}>
        <code>{match[2]}</code>
      </pre>,
    );
    lastIndex = match.index + match[0].length;
  }

  if (lastIndex < text.length) {
    elements.push(...renderMarkdownLines(text.slice(lastIndex), lastIndex));
  }

  return elements;
}

function renderMarkdownLines(text: string, keyOffset: number): preact.VNode[] {
  const lines = text.split('\n');
  const elements: preact.VNode[] = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const key = `l-${keyOffset}-${i}`;

    if (line.startsWith('### ')) {
      elements.push(<h4 class="ai-md-h3" key={key}>{line.slice(4)}</h4>);
    } else if (line.startsWith('## ')) {
      elements.push(<h3 class="ai-md-h2" key={key}>{line.slice(3)}</h3>);
    } else if (line.startsWith('# ')) {
      elements.push(<h2 class="ai-md-h1" key={key}>{line.slice(2)}</h2>);
    } else if (line.startsWith('- ') || line.startsWith('* ')) {
      elements.push(
        <div class="ai-md-list-item" key={key}>
          <span class="ai-md-bullet">{'\u2022'}</span>
          <span>{renderInline(line.slice(2))}</span>
        </div>,
      );
    } else if (line.trim() === '') {
      elements.push(<div class="ai-md-spacer" key={key} />);
    } else {
      elements.push(<p class="ai-md-p" key={key}>{renderInline(line)}</p>);
    }
  }

  return elements;
}

function renderInline(text: string): (string | preact.VNode)[] {
  const parts: (string | preact.VNode)[] = [];
  const regex = /(`[^`]+`|\*\*[^*]+\*\*)/g;
  let last = 0;
  let m: RegExpExecArray | null;

  while ((m = regex.exec(text)) !== null) {
    if (m.index > last) parts.push(text.slice(last, m.index));
    const token = m[0];
    if (token.startsWith('`')) {
      parts.push(<code class="ai-md-inline-code" key={m.index}>{token.slice(1, -1)}</code>);
    } else {
      parts.push(<strong key={m.index}>{token.slice(2, -2)}</strong>);
    }
    last = m.index + token.length;
  }

  if (last < text.length) parts.push(text.slice(last));
  return parts;
}

/* ------------------------------------------------------------------ */
/*  Request Detail Tabs (cloudmock dashboard style)                    */
/* ------------------------------------------------------------------ */

type DetailTab = 'info' | 'request' | 'response' | 'waterfall' | 'explain';

interface RequestDetailTabsProps {
  flow: RequestFlow;
  node: TopoNode;
  allNodes: TopoNode[];
  edges: TopoEdge[];
}

function RequestDetailTabs({ flow, node, allNodes, edges }: RequestDetailTabsProps) {
  const [tab, setTab] = useState<DetailTab>('info');
  const [explainData, setExplainData] = useState<string | null>(null);
  const [explainLoading, setExplainLoading] = useState(false);
  const [explainError, setExplainError] = useState<string | null>(null);

  // Reset tab when flow changes
  useEffect(() => {
    setTab('info');
    setExplainData(null);
    setExplainError(null);
  }, [flow.id]);

  function loadExplain() {
    if (explainData || explainLoading) return;
    setExplainLoading(true);
    setExplainError(null);
    const base = getAdminBase();
    fetch(`${base}/api/explain/${encodeURIComponent(flow.id)}`, {
      headers: { 'Accept': 'application/json' },
    })
      .then((r) => {
        if (!r.ok) throw new Error(`API ${r.status}: ${r.statusText}`);
        return r.json();
      })
      .then((data) => {
        // cloudmock returns { narrative: "...", request: {...}, analysis: {...} }
        const text = data.narrative || data.explanation || data.text || data.content || '';
        if (!text) {
          setExplainError('The AI explanation endpoint returned an empty response.');
        } else {
          setExplainData(text);
        }
      })
      .catch((e) => {
        setExplainError(e instanceof Error ? e.message : 'Failed to get explanation');
      })
      .finally(() => setExplainLoading(false));
  }

  // Derive caller from edges
  const nodeId = allNodes.find(
    (n) => n.service === node.service || n.id === node.id,
  )?.id;
  const inboundEdge = nodeId ? edges.find((e) => e.target === nodeId) : undefined;
  const callerNode = inboundEdge
    ? allNodes.find((n) => n.id === inboundEdge.source)
    : undefined;

  return (
    <div class="request-detail-tabs-container">
      {/* Tab bar */}
      <div class="request-detail-tabs">
        <button
          class={`request-detail-tab ${tab === 'info' ? 'active' : ''}`}
          onClick={() => setTab('info')}
        >
          Info
        </button>
        <button
          class={`request-detail-tab ${tab === 'request' ? 'active' : ''}`}
          onClick={() => setTab('request')}
        >
          Request
        </button>
        <button
          class={`request-detail-tab ${tab === 'response' ? 'active' : ''}`}
          onClick={() => setTab('response')}
        >
          Response
        </button>
        <button
          class={`request-detail-tab ${tab === 'waterfall' ? 'active' : ''}`}
          onClick={() => setTab('waterfall')}
        >
          Waterfall
        </button>
        <button
          class={`request-detail-tab ${tab === 'explain' ? 'active' : ''}`}
          onClick={() => { setTab('explain'); loadExplain(); }}
        >
          {'\u2728'} Explain
        </button>
      </div>

      {/* Tab content */}
      <div class="request-detail-tab-content">
        {tab === 'info' && (
          <div class="request-info-grid">
            <div class="request-info-row">
              <span class="request-info-label">Service</span>
              <span class="request-info-value">{node.label}</span>
            </div>
            <div class="request-info-row">
              <span class="request-info-label">Action</span>
              <span class="request-info-value mono">{flow.outbound[0]?.action || flow.method}</span>
            </div>
            <div class="request-info-row">
              <span class="request-info-label">Method</span>
              <span class="request-info-value mono">{flow.method}</span>
            </div>
            <div class="request-info-row">
              <span class="request-info-label">Path</span>
              <span class="request-info-value mono">{flow.path}</span>
            </div>
            <div class="request-info-row">
              <span class="request-info-label">Status</span>
              <span class={`request-info-status-badge ${statusClass(flow.statusCode)}`}>
                {flow.statusCode}
              </span>
            </div>
            <div class="request-info-row">
              <span class="request-info-label">Latency</span>
              <span class="request-info-value mono">{formatMs(flow.durationMs)}</span>
            </div>
            <div class="request-info-row">
              <span class="request-info-label">Caller</span>
              <span class="request-info-value">
                {flow.inboundSource || callerNode?.label || 'Unknown'}
              </span>
            </div>
            {flow.statusCode >= 400 && (
              <div class="request-info-row">
                <span class="request-info-label">Error</span>
                <span class="request-info-value request-info-error">
                  {flow.statusCode >= 500 ? 'Server Error' : 'Client Error'}
                </span>
              </div>
            )}
            {flow.traceId && (
              <div class="request-info-row">
                <span class="request-info-label">Trace ID</span>
                <span class="request-info-value mono">{flow.traceId}</span>
              </div>
            )}
            {flow.responseSummary && (
              <div class="request-info-row">
                <span class="request-info-label">Response</span>
                <span class="request-info-value">{flow.responseSummary}</span>
              </div>
            )}
            {/* Outbound calls summary */}
            {flow.outbound.length > 0 && (
              <div class="request-info-outbound-section">
                <span class="request-info-label">Outbound Calls ({flow.outbound.length})</span>
                {flow.outbound.map((out, i) => (
                  <div key={i} class="request-info-outbound-row">
                    <span class="request-info-outbound-service">{out.service}</span>
                    <span class="request-info-outbound-action">{out.action}</span>
                    <span class={`request-info-status-badge ${statusClass(out.statusCode)}`}>
                      {out.statusCode}
                    </span>
                    <span class="request-info-outbound-duration">{formatMs(out.durationMs)}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {tab === 'request' && (
          <div class="request-payload-tab">
            {flow.inboundHeaders && Object.keys(flow.inboundHeaders).length > 0 ? (
              <div class="request-payload-section">
                <div class="request-payload-section-title">Headers</div>
                <div class="request-payload-headers-table">
                  {Object.entries(flow.inboundHeaders).map(([k, v]) => (
                    <div key={k} class="request-payload-header-row">
                      <span class="request-payload-header-key">{k}</span>
                      <span class="request-payload-header-val">{v}</span>
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <div class="request-payload-empty">No request headers captured</div>
            )}
          </div>
        )}

        {tab === 'response' && (
          <div class="request-payload-tab">
            {flow.responseSummary ? (
              <div class="request-payload-section">
                <div class="request-payload-section-title">Response Summary</div>
                <div class="request-payload-body-block">
                  <code>{flow.responseSummary}</code>
                </div>
              </div>
            ) : (
              <div class="request-payload-empty">No response body captured</div>
            )}
          </div>
        )}

        {tab === 'waterfall' && (
          <div class="request-payload-tab">
            {flow.traceId ? (
              <Waterfall traceId={flow.traceId} />
            ) : (
              <div class="request-simple-waterfall">
                <div class="request-waterfall-bar">
                  <div class="request-waterfall-fill" style={{ width: '100%', background: 'var(--brand-teal)' }}>
                    {formatMs(flow.durationMs)}
                  </div>
                </div>
                <div class="request-waterfall-legend">
                  <span>Total: {formatMs(flow.durationMs)}</span>
                  <span>Status: {flow.statusCode}</span>
                </div>
              </div>
            )}
          </div>
        )}

        {tab === 'explain' && (
          <div class="request-explain-tab">
            {explainLoading && (
              <div class="ai-explain-loading">
                <div class="ai-explain-loading-bar" />
                <span class="ai-explain-loading-text">Analyzing request...</span>
              </div>
            )}
            {explainError && (
              <div class="ai-explain-error">
                <span class="ai-explain-error-icon">!</span>
                {explainError}
                <button
                  class="ai-explain-retry"
                  onClick={() => { setExplainData(null); setExplainError(null); loadExplain(); }}
                >
                  Retry
                </button>
              </div>
            )}
            {explainData && (
              <div class="ai-explain-result">
                <div class="ai-explain-result-header">
                  <span class="ai-explain-result-label">{'\u2728'} AI Explanation</span>
                </div>
                <div class="ai-explain-result-body">
                  {renderMarkdown(explainData)}
                </div>
              </div>
            )}
            {!explainLoading && !explainError && !explainData && (
              <div class="request-payload-empty">Click the tab to load the AI explanation</div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Sidebar Incidents Section                                          */
/* ------------------------------------------------------------------ */

interface SidebarIncidentsProps {
  incidents: IncidentInfo[];
  serviceName: string;
}

function SidebarIncidents({ incidents, serviceName }: SidebarIncidentsProps) {
  // Filter strictly to incidents that affect this service. Previously this
  // also OR'd in any `status === 'active'` incident, which surfaced every
  // active incident system-wide.
  const serviceIncidents = useMemo(() => {
    const svcLower = serviceName.toLowerCase();
    return incidents.filter((inc) =>
      inc.affected_services.some((s) => s.toLowerCase() === svcLower),
    );
  }, [incidents, serviceName]);

  if (serviceIncidents.length === 0) {
    return (
      <div class="sidebar-section sidebar-section-flex">
        <div class="sidebar-section-title">Incidents</div>
        <div class="sidebar-section-empty">No active incidents</div>
      </div>
    );
  }

  return (
    <div class="sidebar-section sidebar-section-flex">
      <div class="sidebar-section-title">
        Incidents
        <span class="sidebar-section-count">{serviceIncidents.length}</span>
      </div>
      <div class="sidebar-incidents-list">
        {serviceIncidents.map((inc) => (
          <div key={inc.id} class="sidebar-incident-item">
            <span class={`incident-severity-badge severity-${inc.severity}`}>
              {inc.severity}
            </span>
            <span class="sidebar-incident-title">{inc.title}</span>
            <span class="sidebar-incident-status">{inc.status}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Stack Trace Item                                                   */
/* ------------------------------------------------------------------ */

interface StackTraceItemProps {
  flow: RequestFlow;
  isExpanded: boolean;
  onToggle: () => void;
  isHighlighted: boolean;
}

function StackTraceItem({ flow, isExpanded, onToggle, isHighlighted }: StackTraceItemProps) {
  return (
    <div
      class={`request-stack-item ${isHighlighted ? 'request-stack-item-highlighted' : ''}`}
      data-flow-id={flow.id}
    >
      <button class="request-stack-item-header" onClick={onToggle}>
        <span class="request-stack-caret">{isExpanded ? '\u25BC' : '\u25B6'}</span>
        <span class="request-stack-method">{flow.method}</span>
        <span class="request-stack-path" title={flow.path}>{flow.path}</span>
        <span class={`request-stack-status ${statusClass(flow.statusCode)}`}>
          {flow.statusCode}
        </span>
        <span class="request-stack-duration">{formatMs(flow.durationMs)}</span>
        <span class="request-stack-time">{formatTime(flow.timestamp)}</span>
      </button>

      {isExpanded && (
        <div class="request-stack-body">
          {/* Inbound */}
          <div class="request-stack-connector">
            <div class="request-stack-connector-line" />
            <div class="request-stack-leg request-stack-leg-inbound">
              <span class="request-stack-leg-icon">&#x2B07;</span>
              <span class="request-stack-leg-dir">INBOUND</span>
              {flow.inboundSource && (
                <span class="request-stack-leg-source">from {flow.inboundSource}</span>
              )}
            </div>
            {flow.inboundHeaders && Object.keys(flow.inboundHeaders).length > 0 && (
              <div class="request-stack-headers">
                {Object.entries(flow.inboundHeaders).slice(0, 5).map(([k, v]) => (
                  <div key={k} class="request-stack-header-row">
                    <span class="request-stack-header-key">{k}:</span>
                    <span class="request-stack-header-val">{v}</span>
                  </div>
                ))}
              </div>
            )}
            <div class="request-stack-leg-meta">
              Duration: {formatMs(flow.durationMs)} total
            </div>
          </div>

          {/* Outbound calls */}
          {flow.outbound.map((out, i) => (
            <div key={i} class="request-stack-connector">
              <div class="request-stack-connector-line" />
              <div class="request-stack-leg request-stack-leg-outbound">
                <span class="request-stack-leg-icon">&#x2B06;</span>
                <span class="request-stack-leg-dir">OUTBOUND</span>
                <span class="request-stack-leg-target">to {out.service}</span>
              </div>
              <div class="request-stack-leg-meta">
                Action: {out.action}
                {out.detail && <span> &middot; {out.detail}</span>}
              </div>
              <div class="request-stack-leg-meta">
                Duration: {formatMs(out.durationMs)} &middot; Status: {out.statusCode}
              </div>
            </div>
          ))}

          {/* Response */}
          <div class="request-stack-connector request-stack-connector-last">
            <div class="request-stack-connector-line" />
            <div class="request-stack-leg request-stack-leg-response">
              Response: {flow.statusCode} {flow.statusCode < 300 ? 'OK' : flow.statusCode < 500 ? 'Client Error' : 'Server Error'}
              {flow.responseSummary && ` (${flow.responseSummary})`}
            </div>
          </div>

          {flow.traceId && (
            <div class="request-stack-trace-id">
              Trace: {flow.traceId}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Main Panel                                                         */
/* ------------------------------------------------------------------ */

export function RequestTracePanel({
  node,
  edges,
  allNodes,
  metrics,
  incidents,
  onClose,
}: RequestTracePanelProps) {
  // === State machine: single useReducer replaces 10 useState calls ===
  const [state, dispatch] = useReducer(panelReducer, undefined, initialPanelState);
  const { flows, phase, error, expandedIds, selectedFlowId: highlightedFlowId, timeRange, hideOptions, replayStatus, lastFetchedAt } = state;
  const loading = phase === 'loading';

  // Ticking "last updated X seconds ago" label
  const [lastUpdatedLabel, setLastUpdatedLabel] = useState<string>('');
  useEffect(() => {
    if (!lastFetchedAt) { setLastUpdatedLabel(''); return; }
    const tick = () => {
      const secsAgo = Math.floor((Date.now() - lastFetchedAt) / 1000);
      if (secsAgo < 5) setLastUpdatedLabel('just now');
      else if (secsAgo < 60) setLastUpdatedLabel(`${secsAgo}s ago`);
      else setLastUpdatedLabel(`${Math.floor(secsAgo / 60)}m ago`);
    };
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [lastFetchedAt]);

  const stackRef = useRef<HTMLDivElement>(null);
  const serviceName = getServiceKey(node);

  // Refs for allNodes/edges to avoid re-triggering fetch on parent re-renders
  const allNodesRef = useRef(allNodes);
  allNodesRef.current = allNodes;
  const edgesRef = useRef(edges);
  edgesRef.current = edges;

  // Fetch logic — dispatches actions to reducer
  const fetchFlows = useCallback(async (isInitial: boolean) => {
    const currentNodes = allNodesRef.current;
    const currentEdges = edgesRef.current;

    const requestsPromise = api<any[]>(
      `/api/requests?level=all&limit=200`,
    ).then((raw) => raw.map(normalizeRequestEvent)).catch(() => [] as RequestEvent[]);

    const tracesPromise = api<TraceEntry[]>('/api/traces').catch(() => [] as TraceEntry[]);

    const [allRequests, traces] = await Promise.all([requestsPromise, tracesPromise]);

    // Filter requests by service + edge connections
    const requests = filterRequestsByEdgeServices(allRequests, serviceName, node, currentEdges);
    const effectiveRequests = requests.length > 0 ? requests : allRequests.slice(0, 50);

    // Merge trace data into requests
    const traceMap = new Map<string, TraceEntry>();
    for (const t of traces) {
      traceMap.set(t.TraceID, t);
    }

    const enrichedRequests = effectiveRequests.map((r) => {
      if (r.trace_id && traceMap.has(r.trace_id)) {
        const trace = traceMap.get(r.trace_id)!;
        return {
          ...r,
          latency_ms: r.latency_ms || trace.DurationMs,
          status_code: r.status_code || trace.StatusCode,
        };
      }
      return r;
    });

    // Log method breakdown for debugging
    const methodCounts = new Map<string, number>();
    for (const r of enrichedRequests) {
      const m = r.method || 'UNKNOWN';
      methodCounts.set(m, (methodCounts.get(m) || 0) + 1);
    }
    console.log(`[RequestTrace] ${serviceName}: ${allRequests.length} raw → ${enrichedRequests.length} filtered. Methods:`, Object.fromEntries(methodCounts));

    let incoming = buildFlows(enrichedRequests, serviceName, currentNodes, currentEdges);

    if (incoming.length === 0 && enrichedRequests.length > 0) {
      incoming = buildFallbackFlows(enrichedRequests);
    }

    console.log(`[RequestTrace] ${incoming.length} flows built (${incoming.length === 0 && enrichedRequests.length > 0 ? 'using fallback' : 'from traces'})`);


    // Dispatch to reducer — atomic state update
    if (isInitial) {
      dispatch({ type: 'FETCH_SUCCESS', flows: incoming });
    } else {
      dispatch({ type: 'POLL_MERGE', flows: incoming });
    }
  }, [serviceName, node]);

  // Initial fetch
  useEffect(() => {
    let cancelled = false;
    dispatch({ type: 'FETCH_START' });

    fetchFlows(true)
      .catch((err) => {
        if (!cancelled) {
          dispatch({ type: 'FETCH_ERROR', error: err instanceof Error ? err.message : 'Failed to load requests' });
        }
      });

    return () => { cancelled = true; };
  }, [serviceName]);

  // Poll every 5s — merge new flows via reducer
  useEffect(() => {
    const interval = setInterval(() => {
      fetchFlows(false).catch((err) => {
        console.warn('[RequestTrace] Poll error:', err);
      });
    }, 5000);

    return () => clearInterval(interval);
  }, [fetchFlows]);

  // Toggle expand/collapse
  const handleToggle = useCallback((id: string) => {
    dispatch({ type: 'TOGGLE_EXPAND', id });
  }, []);

  // Select flow from timeline or sidebar
  const handleSelectFlow = useCallback((id: string) => {
    dispatch({ type: 'SELECT_FLOW', id });

    // Scroll to the item in sidebar
    requestAnimationFrame(() => {
      const el = stackRef.current?.querySelector(`[data-flow-id="${id}"]`);
      if (el) {
        el.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
      }
    });
  }, []);

  // Filter flows by time range + method filter (derived state).
  // If hiding OPTIONS would empty the list entirely (service only sees CORS
  // preflights, e.g. a pure-API backend), fall back to showing everything —
  // an empty "0 requests" panel is worse than seeing OPTIONS traffic.
  const filteredFlows = useMemo(() => {
    const inRange = flows.filter((f) => {
      const ts = new Date(f.timestamp).getTime();
      return ts >= timeRange.start && ts <= timeRange.end;
    });
    if (!hideOptions) return inRange;
    const nonOptions = filterFlowsByMethod(inRange, new Set(['OPTIONS']));
    return nonOptions.length > 0 ? nonOptions : inRange;
  }, [flows, timeRange, hideOptions]);

  const handleScrub = useCallback((range: { start: number; end: number }) => {
    dispatch({ type: 'SET_TIME_RANGE', range });
  }, []);

  const handleReplay = useCallback(async (flow: RequestFlow) => {
    dispatch({ type: 'REPLAY_START', flowId: flow.id });

    const clearAfter = (id: string, ms: number) => {
      setTimeout(() => dispatch({ type: 'REPLAY_CLEAR', flowId: id }), ms);
    };

    try {
      const adminBase = getAdminBase();
      // Use cloudmock's server-side replay: POST /api/requests/{id}/replay
      // This avoids CORS — admin API replays against gateway internally
      // Use requestId (original cloudmock ID), not flow.id (which may be a trace ID)
      const replayId = flow.requestId || flow.id;
      if (!replayId) {
        dispatch({ type: 'REPLAY_DONE', flowId: flow.id, status: '\u2717 no request ID' });
        clearAfter(flow.id, 3000);
        return;
      }
      const res = await fetch(`${adminBase}/api/requests/${encodeURIComponent(replayId)}/replay`, {
        method: 'POST',
        headers: { 'Accept': 'application/json' },
      });

      if (!res.ok) {
        throw new Error(`Replay API ${res.status}: ${res.statusText}`);
      }

      const data = await res.json();
      const matched = data.match ? 'matched' : 'changed';
      const status = data.replay_status >= 400
        ? `\u2717 ${data.replay_status} (${Math.round(data.replay_latency_ms)}ms, ${matched})`
        : `\u2713 ${data.replay_status} (${Math.round(data.replay_latency_ms)}ms, ${matched})`;
      dispatch({ type: 'REPLAY_DONE', flowId: flow.id, status });
      console.log(`[Replay] ${flow.method} ${flow.path} → ${data.replay_status} (${matched})`);
      clearAfter(flow.id, 5000);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'failed';
      console.warn('[Replay] Failed:', msg);

      // Fallback: copy curl to clipboard
      const gwBase = getAdminBase().replace(':4599', ':4566');
      const curlCmd = `curl -X ${flow.method || 'GET'} '${gwBase}${flow.path || '/'}'`;
      try {
        await navigator.clipboard.writeText(curlCmd);
        dispatch({ type: 'REPLAY_DONE', flowId: flow.id, status: '\uD83D\uDCCB curl copied' });
      } catch {
        dispatch({ type: 'REPLAY_DONE', flowId: flow.id, status: `\u2717 ${msg}` });
      }
      clearAfter(flow.id, 4000);
    }
  }, []);

  const selectedFlow = useMemo(() => {
    if (!highlightedFlowId) return null;
    return filteredFlows.find((f) => f.id === highlightedFlowId) ?? null;
  }, [filteredFlows, highlightedFlowId]);

  return (
    <div class="request-trace-panel">
      {/* Header */}
      <div class="request-trace-panel-header">
        <button class="btn btn-ghost request-trace-back-btn" onClick={onClose}>
          &#x2190; Back
        </button>
        <span class="request-trace-panel-title">
          {node.label}
        </span>
        <span class="request-trace-panel-count">
          {filteredFlows.length} request{filteredFlows.length !== 1 ? 's' : ''}
          {flows.length !== filteredFlows.length && (
            <span style={{ opacity: 0.5 }}> ({flows.length} total)</span>
          )}
        </span>
        {lastUpdatedLabel && (
          <span class="request-trace-panel-updated" style={{ fontSize: '10px', opacity: 0.5, marginLeft: '8px' }}>
            Updated {lastUpdatedLabel}
          </span>
        )}
      </div>

      {/* Two-column layout: left sidebar + right detail */}
      <div class="request-trace-body">
        {/* Left sidebar: timeline + request list + incidents + blast radius */}
        <div class="request-trace-sidebar" ref={stackRef}>
          {/* Compact Replay Timeline */}
          <div class="sidebar-section">
            <ReplayTimeline
              flows={flows}
              selectedFlowId={highlightedFlowId}
              onSelectFlow={handleSelectFlow}
              timeRange={timeRange}
              onScrub={handleScrub}
            />
          </div>

          {/* Request List */}
          <div class="sidebar-section sidebar-section-flex">
            <div class="sidebar-section-title">
              Requests
              <span class="sidebar-section-count">{filteredFlows.length}</span>
              <button
                class={`btn btn-ghost request-filter-toggle ${hideOptions ? 'active' : ''}`}
                onClick={() => dispatch({ type: 'TOGGLE_OPTIONS_FILTER' })}
                title={hideOptions ? 'Show OPTIONS requests' : 'Hide OPTIONS requests'}
                style={{ marginLeft: 'auto', fontSize: '10px', padding: '1px 6px' }}
              >
                {hideOptions ? 'OPTIONS hidden' : 'Show all'}
              </button>
            </div>
            <div class="sidebar-request-list">
              {loading && <div class="request-trace-loading">Loading...</div>}
              {error && <div class="request-trace-error">{error}</div>}
              {!loading && !error && filteredFlows.length === 0 && (
                <div class="request-trace-empty">
                  {hideOptions && flows.length > 0 ? (
                    <div>
                      <div style="margin-bottom: 8px;">
                        {flows.length} OPTIONS request{flows.length !== 1 ? 's' : ''} hidden (CORS preflights).
                      </div>
                      <button
                        class="btn btn-ghost"
                        onClick={() => dispatch({ type: 'TOGGLE_OPTIONS_FILTER' })}
                        style={{ fontSize: '11px' }}
                      >
                        Show OPTIONS requests
                      </button>
                    </div>
                  ) : node.group === 'Client' ? (
                    <div>
                      <div style="margin-bottom: 8px;">No requests captured for this client app.</div>
                      <div style="font-size: 10px; color: var(--text-tertiary); line-height: 1.5;">
                        Client requests are captured via the SDK.<br />
                        Add <code>@cloudmock/node</code> (or Swift/Kotlin SDK) to see inbound requests here.
                      </div>
                    </div>
                  ) : (
                    <div>
                      <div style="margin-bottom: 8px;">No requests found in the last 5 minutes.</div>
                      <div style="font-size: 10px; color: var(--text-tertiary);">
                        Generate traffic to see requests here.
                      </div>
                      <div style="font-size: 10px; color: var(--text-tertiary); margin-top: 8px;">
                        Debug: Open browser console for [RequestTrace] logs, or run:<br/>
                        <code>curl localhost:4599/api/requests?level=all</code>
                      </div>
                    </div>
                  )}
                </div>
              )}
              {!loading && filteredFlows.map((flow) => {
                const isActive = flow.id === highlightedFlowId;
                const statusColor = flow.statusCode < 300 ? '#22c55e'
                  : flow.statusCode < 500 ? '#fbbf24' : '#ef4444';
                return (
                  <div
                    key={flow.id}
                    class={`request-sidebar-item ${isActive ? 'active' : ''}`}
                    data-flow-id={flow.id}
                    onClick={() => handleSelectFlow(flow.id)}
                  >
                    <div class="request-sidebar-item-header">
                      <span class="request-sidebar-status" style={{ color: statusColor }}>
                        {flow.statusCode}
                      </span>
                      <span class="request-sidebar-method">{flow.method}</span>
                      <span class="request-sidebar-path">
                        {flow.path && flow.path !== '/'
                          ? flow.path.split('/').pop() || '/'
                          : flow.outbound[0]?.action || flow.path || '/'}
                      </span>
                    </div>
                    <div class="request-sidebar-item-meta">
                      <span>{flow.inboundSource || flow.outbound[0]?.service || '?'}</span>
                      <span>{flow.durationMs ? `${Math.round(flow.durationMs)}ms` : ''}</span>
                    </div>
                    <div class="request-sidebar-actions">
                      {replayStatus[flow.id] ? (
                        <span class={`request-sidebar-replay-status ${replayStatus[flow.id].startsWith('\u2713') ? 'success' : replayStatus[flow.id] === 'replaying...' ? 'loading' : 'error'}`}>
                          {replayStatus[flow.id]}
                        </span>
                      ) : (
                        <button
                          class="request-sidebar-replay-btn"
                          onClick={(e) => { e.stopPropagation(); handleReplay(flow); }}
                          title="Replay this request to cloudmock"
                        >
                          ▶ Replay
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Incidents */}
          <SidebarIncidents incidents={incidents} serviceName={serviceName} />

          {/* Blast Radius — equal-height flex section with its own scroll */}
          <div class="sidebar-section sidebar-section-flex">
            <div class="sidebar-blast-scroll">
              <BlastRadiusBar
                nodeId={node.id}
                edges={edges}
                allNodes={allNodes}
                metrics={metrics}
              />
            </div>
          </div>
        </div>

        {/* Right: tabbed detail panel (cloudmock dashboard style) */}
        <div class="request-trace-detail">
          {selectedFlow ? (
            <RequestDetailTabs
              flow={selectedFlow}
              node={node}
              allNodes={allNodes}
              edges={edges}
            />
          ) : (
            <div class="request-trace-select-hint">
              {'\u2190'} Select a request to see details
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
