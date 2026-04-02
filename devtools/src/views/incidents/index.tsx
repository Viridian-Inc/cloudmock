import { useState, useEffect, useCallback } from 'preact/hooks';
import { SplitPanel } from '../../components/panels/split-panel';
import { api } from '../../lib/api';
import './incidents.css';

interface Incident {
  id: string;
  title: string;
  service: string;
  severity: 'critical' | 'warning' | 'info';
  status: 'open' | 'acknowledged' | 'resolved';
  message: string;
  timestamp: string;
  acknowledged_at?: string;
  resolved_at?: string;
  details?: Record<string, unknown>;
  first_seen?: string;
  last_seen?: string;
  affected_services?: string[];
}

// --- Severity color map for timeline dots ---
const SEVERITY_COLORS: Record<string, string> = {
  critical: '#ff4e5e',
  high: '#F7711E',
  warning: '#fad065',
  medium: '#fad065',
  low: '#538eff',
  info: '#538eff',
};

/** Group incidents close in time (within groupWindowMs) */
interface IncidentGroup {
  incidents: Incident[];
  time: number; // average timestamp of group
}

function groupIncidents(incidents: Incident[], groupWindowMs: number): IncidentGroup[] {
  if (incidents.length === 0) return [];

  const sorted = [...incidents].sort(
    (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime(),
  );
  const groups: IncidentGroup[] = [];
  let current: Incident[] = [sorted[0]];
  let lastTs = new Date(sorted[0].timestamp).getTime();

  for (let i = 1; i < sorted.length; i++) {
    const ts = new Date(sorted[i].timestamp).getTime();
    if (ts - lastTs <= groupWindowMs) {
      current.push(sorted[i]);
    } else {
      const avgTime = current.reduce((sum, inc) => sum + new Date(inc.timestamp).getTime(), 0) / current.length;
      groups.push({ incidents: current, time: avgTime });
      current = [sorted[i]];
    }
    lastTs = ts;
  }
  const avgTime = current.reduce((sum, inc) => sum + new Date(inc.timestamp).getTime(), 0) / current.length;
  groups.push({ incidents: current, time: avgTime });

  return groups;
}

/** Horizontal incident timeline */
function IncidentTimeline({
  incidents,
  selectedId,
  onSelect,
}: {
  incidents: Incident[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}) {
  const [hoveredGroup, setHoveredGroup] = useState<IncidentGroup | null>(null);
  const [hoverPos, setHoverPos] = useState<{ x: number; y: number } | null>(null);

  if (incidents.length === 0) return null;

  const now = Date.now();
  const hoursAgo24 = now - 24 * 60 * 60 * 1000;
  const timelineStart = hoursAgo24;
  const timelineEnd = now;
  const timeRange = timelineEnd - timelineStart;

  // Filter incidents within the last 24h
  const recentIncidents = incidents.filter((inc) => {
    const ts = new Date(inc.timestamp).getTime();
    return ts >= timelineStart && ts <= timelineEnd;
  });

  // Group incidents close in time (within 15 minutes)
  const groups = groupIncidents(recentIncidents, 15 * 60 * 1000);

  // Time labels along the bottom
  const timeLabels: { time: number; label: string }[] = [];
  for (let h = 0; h <= 24; h += 4) {
    const ts = timelineStart + h * 60 * 60 * 1000;
    const d = new Date(ts);
    timeLabels.push({
      time: ts,
      label: `${d.getHours().toString().padStart(2, '0')}:00`,
    });
  }

  const SVG_WIDTH = 800;
  const SVG_HEIGHT = 56;
  const PAD_LEFT = 40;
  const PAD_RIGHT = 16;
  const CHART_W = SVG_WIDTH - PAD_LEFT - PAD_RIGHT;
  const DOT_Y = 20;

  return (
    <div class="incident-timeline">
      <div class="incident-timeline-label">Last 24h</div>
      <svg
        class="incident-timeline-svg"
        viewBox={`0 0 ${SVG_WIDTH} ${SVG_HEIGHT}`}
        preserveAspectRatio="none"
        width="100%"
        height={SVG_HEIGHT}
      >
        {/* Baseline */}
        <line
          x1={PAD_LEFT} y1={DOT_Y}
          x2={SVG_WIDTH - PAD_RIGHT} y2={DOT_Y}
          stroke="rgba(255,255,255,0.08)"
          strokeWidth="1"
        />

        {/* Time labels */}
        {timeLabels.map((tl) => {
          const x = PAD_LEFT + ((tl.time - timelineStart) / timeRange) * CHART_W;
          return (
            <g key={tl.time}>
              <line
                x1={x} y1={DOT_Y - 4} x2={x} y2={DOT_Y + 4}
                stroke="rgba(255,255,255,0.12)"
                strokeWidth="1"
              />
              <text
                x={x}
                y={SVG_HEIGHT - 6}
                textAnchor="middle"
                fill="rgba(255,255,255,0.35)"
                fontSize="9"
                fontFamily="var(--font-mono)"
              >
                {tl.label}
              </text>
            </g>
          );
        })}

        {/* Incident dots/groups */}
        {groups.map((group, gi) => {
          const x = PAD_LEFT + ((group.time - timelineStart) / timeRange) * CHART_W;
          const isMulti = group.incidents.length > 1;
          // Use the highest severity color in the group
          const severityPriority = ['critical', 'high', 'warning', 'medium', 'low', 'info'];
          const topSeverity = group.incidents.reduce((best, inc) => {
            const bIdx = severityPriority.indexOf(best);
            const iIdx = severityPriority.indexOf(inc.severity);
            return iIdx < bIdx ? inc.severity : best;
          }, group.incidents[0].severity);
          const color = SEVERITY_COLORS[topSeverity] || '#538eff';

          const isSelected = group.incidents.some((inc) => inc.id === selectedId);

          return (
            <g
              key={gi}
              style={{ cursor: 'pointer' }}
              onClick={() => onSelect(group.incidents[0].id)}
              onMouseEnter={(e: MouseEvent) => {
                setHoveredGroup(group);
                const svg = (e.target as SVGElement).closest('svg');
                if (svg) {
                  const rect = svg.getBoundingClientRect();
                  setHoverPos({ x: e.clientX - rect.left, y: e.clientY - rect.top });
                }
              }}
              onMouseLeave={() => {
                setHoveredGroup(null);
                setHoverPos(null);
              }}
            >
              {isMulti && (
                <circle cx={x} cy={DOT_Y} r={10} fill={color} opacity="0.15" />
              )}
              <circle
                cx={x}
                cy={DOT_Y}
                r={isMulti ? 6 : 5}
                fill={color}
                stroke={isSelected ? '#fff' : 'none'}
                strokeWidth={isSelected ? 2 : 0}
                opacity={isSelected ? 1 : 0.85}
              />
              {isMulti && (
                <text
                  x={x}
                  y={DOT_Y + 3.5}
                  textAnchor="middle"
                  fill="#fff"
                  fontSize="7"
                  fontWeight="700"
                  fontFamily="var(--font-mono)"
                >
                  {group.incidents.length}
                </text>
              )}
            </g>
          );
        })}
      </svg>

      {/* Tooltip */}
      {hoveredGroup && hoverPos && (
        <div
          class="incident-timeline-tooltip"
          style={{ left: `${hoverPos.x}px`, top: `${hoverPos.y - 48}px` }}
        >
          {hoveredGroup.incidents.length === 1 ? (
            <div>
              <div class="incident-timeline-tooltip-title">{hoveredGroup.incidents[0].title}</div>
              <div class="incident-timeline-tooltip-meta">
                {hoveredGroup.incidents[0].service} - {hoveredGroup.incidents[0].severity}
              </div>
            </div>
          ) : (
            <div>
              <div class="incident-timeline-tooltip-title">
                {hoveredGroup.incidents.length} incidents
              </div>
              <div class="incident-timeline-tooltip-meta">
                {hoveredGroup.incidents.map((i) => i.severity).join(', ')}
              </div>
            </div>
          )}
        </div>
      )}

      {recentIncidents.length === 0 && (
        <div class="incident-timeline-empty">No incidents in the last 24h</div>
      )}
    </div>
  );
}

interface RelatedTrace {
  TraceID: string;
  RootService: string;
  RootAction: string;
  Method: string;
  Path: string;
  StatusCode: number;
  DurationMs: number;
  StartTime: string;
}

function formatTime(ts: string): string {
  const d = new Date(ts);
  if (isNaN(d.getTime())) return '--:--:--';
  return d.toLocaleString();
}

function relativeTime(ts: string): string {
  const d = new Date(ts);
  if (isNaN(d.getTime())) return '';
  const diff = Date.now() - d.getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

function severityIcon(severity: string): string {
  switch (severity) {
    case 'critical': return '!!';
    case 'warning': return '!';
    default: return 'i';
  }
}

function IncidentList({
  incidents,
  selectedId,
  onSelect,
}: {
  incidents: Incident[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}) {
  const [filter, setFilter] = useState<string>('all');

  const filtered = incidents.filter((inc) => {
    if (filter === 'all') return true;
    return inc.status === filter;
  });

  const openCount = incidents.filter((i) => i.status === 'open').length;
  const ackCount = incidents.filter((i) => i.status === 'acknowledged').length;

  return (
    <div class="incident-list">
      <div class="incident-list-header">
        <div class="incident-list-title">
          Incidents
          {openCount > 0 && <span class="incident-count-badge">{openCount}</span>}
        </div>
        <div class="incident-list-filters">
          {(['all', 'open', 'acknowledged', 'resolved'] as const).map((f) => (
            <button
              key={f}
              class={`incident-filter-btn ${filter === f ? 'active' : ''}`}
              onClick={() => setFilter(f)}
            >
              {f === 'all' ? 'All' : f === 'acknowledged' ? 'Ack' : f.charAt(0).toUpperCase() + f.slice(1)}
              {f === 'acknowledged' && ackCount > 0 && (
                <span class="incident-filter-count">{ackCount}</span>
              )}
            </button>
          ))}
        </div>
      </div>
      <div class="incident-list-body">
        {filtered.length === 0 && (
          <div class="incident-list-empty">No incidents</div>
        )}
        {filtered.map((inc) => (
          <div
            key={inc.id}
            class={`incident-row ${inc.id === selectedId ? 'incident-row-selected' : ''} incident-severity-${inc.severity}`}
            onClick={() => onSelect(inc.id)}
          >
            <span class={`incident-severity-icon severity-${inc.severity}`}>
              {severityIcon(inc.severity)}
            </span>
            <div class="incident-row-content">
              <div class="incident-row-title">{inc.title}</div>
              <div class="incident-row-meta">
                <span class="incident-row-service">{inc.service}</span>
                <span class="incident-row-time">{relativeTime(inc.timestamp)}</span>
              </div>
            </div>
            <span class={`incident-status-badge status-${inc.status}`}>
              {inc.status}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

function RelatedTraces({ incident }: { incident: Incident }) {
  const [traces, setTraces] = useState<RelatedTrace[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    setTraces([]);

    api<RelatedTrace[]>('/api/traces')
      .then((allTraces) => {
        const firstSeen = incident.first_seen || incident.timestamp;
        const lastSeen = incident.last_seen || incident.timestamp;
        const startMs = new Date(firstSeen).getTime();
        const endMs = new Date(lastSeen).getTime();
        const services = new Set(incident.affected_services ?? [incident.service]);

        const related = allTraces.filter((t) => {
          const traceTime = new Date(t.StartTime).getTime();
          const inWindow = traceTime >= startMs && traceTime <= endMs;
          const matchesService = services.has(t.RootService);
          return inWindow && matchesService;
        });

        setTraces(related);
      })
      .catch(() => setTraces([]))
      .finally(() => setLoading(false));
  }, [incident.id, incident.first_seen, incident.last_seen, incident.timestamp, incident.service]);

  if (loading) {
    return (
      <div class="incident-detail-section">
        <h4 class="incident-section-heading">Related Traces</h4>
        <div class="related-traces-loading">Loading traces...</div>
      </div>
    );
  }

  return (
    <div class="incident-detail-section">
      <h4 class="incident-section-heading">Related Traces ({traces.length})</h4>
      {traces.length === 0 ? (
        <div class="related-traces-empty">No traces found in the incident time window</div>
      ) : (
        <div class="related-traces-table-wrapper">
          <table class="related-traces-table">
            <thead>
              <tr>
                <th>Service</th>
                <th>Method</th>
                <th>Path</th>
                <th>Status</th>
                <th>Duration</th>
              </tr>
            </thead>
            <tbody>
              {traces.map((t) => (
                <tr key={t.TraceID} class="related-trace-row">
                  <td>
                    <span class="related-trace-service">{t.RootService}</span>
                  </td>
                  <td>
                    <span class="related-trace-method">{t.Method || '--'}</span>
                  </td>
                  <td>
                    <span class="related-trace-path">{t.Path || t.RootAction || '--'}</span>
                  </td>
                  <td>
                    <span class={`status-pill ${t.StatusCode >= 400 ? (t.StatusCode >= 500 ? 'status-5xx' : 'status-4xx') : 'status-2xx'}`}>
                      {t.StatusCode}
                    </span>
                  </td>
                  <td>
                    <span class="related-trace-duration">{Math.round(t.DurationMs)}ms</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function RootCauseSuggestion({ incident }: { incident: Incident }) {
  const [suggestion, setSuggestion] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSuggest() {
    setLoading(true);
    setError(null);
    setSuggestion(null);

    const affectedServices = incident.affected_services?.join(', ') || incident.service;
    const prompt = [
      `Incident: ${incident.title}`,
      `Severity: ${incident.severity}`,
      `Service: ${incident.service}`,
      `Affected services: ${affectedServices}`,
      `Message: ${incident.message}`,
      incident.first_seen ? `First seen: ${incident.first_seen}` : '',
      incident.last_seen ? `Last seen: ${incident.last_seen}` : '',
      '',
      'Analyze this incident and suggest the most likely root cause. Consider:',
      '- Common failure patterns for the affected services',
      '- Upstream/downstream dependency failures',
      '- Recent deployment issues',
      '- Resource exhaustion or configuration problems',
    ].filter(Boolean).join('\n');

    try {
      // Collect related trace IDs for context
      let traceContext = '';
      try {
        const traces = await api<RelatedTrace[]>('/api/traces');
        const firstSeen = incident.first_seen || incident.timestamp;
        const lastSeen = incident.last_seen || incident.timestamp;
        const startMs = new Date(firstSeen).getTime();
        const endMs = new Date(lastSeen).getTime();
        const services = new Set(incident.affected_services ?? [incident.service]);
        const related = traces
          .filter((t) => {
            const traceTime = new Date(t.StartTime).getTime();
            return traceTime >= startMs && traceTime <= endMs && services.has(t.RootService);
          })
          .slice(0, 5);

        if (related.length > 0) {
          traceContext = '\n\nRelated traces:\n' + related.map(
            (t) => `- ${t.RootService} ${t.Method} ${t.Path} ${t.StatusCode} (${t.DurationMs}ms)`,
          ).join('\n');
        }
      } catch {
        // Trace context is optional
      }

      const res = await fetch('/api/explain', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          request_id: incident.id,
          prompt: prompt + traceContext,
        }),
      });

      if (!res.ok) {
        if (res.status === 404) {
          setError('AI Debug endpoint not configured. Set up /api/explain to enable root cause suggestions.');
          return;
        }
        throw new Error(`API ${res.status}: ${res.statusText}`);
      }

      const data = await res.json();
      const explanation = data.explanation || data.text || data.content;
      if (!explanation) {
        setError('AI Debug endpoint returned an empty response.');
      } else {
        setSuggestion(explanation);
      }
    } catch (err: any) {
      if (err.message?.includes('Failed to fetch') || err.message?.includes('NetworkError')) {
        setError('AI Debug endpoint not configured. Set up /api/explain to enable root cause suggestions.');
      } else {
        setError(err.message || 'Failed to get root cause suggestion');
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div class="incident-detail-section">
      <h4 class="incident-section-heading">Root Cause Analysis</h4>
      {!suggestion && !loading && !error && (
        <button
          class="btn btn-primary"
          onClick={handleSuggest}
          style={{ marginTop: '4px', fontSize: '12px' }}
        >
          Suggest Root Cause
        </button>
      )}
      {loading && (
        <div style={{ color: 'var(--text-tertiary)', fontSize: '12px', padding: '8px 0' }}>
          Analyzing incident context...
        </div>
      )}
      {error && (
        <div style={{ color: 'var(--text-tertiary)', fontSize: '12px', background: 'rgba(255,78,94,0.06)', borderRadius: '6px', padding: '8px 12px', marginTop: '4px' }}>
          {error}
        </div>
      )}
      {suggestion && (
        <div style={{ fontSize: '12px', color: 'var(--text-secondary)', padding: '8px 12px', background: 'rgba(74,229,248,0.04)', borderRadius: '6px', marginTop: '4px', whiteSpace: 'pre-wrap', lineHeight: '1.5' }}>
          {suggestion}
        </div>
      )}
      {(suggestion || error) && (
        <div style={{ marginTop: '6px', display: 'flex', gap: '8px' }}>
          {suggestion && (
            <button
              class="btn btn-ghost"
              onClick={handleSuggest}
              disabled={loading}
              style={{ fontSize: '11px' }}
            >
              Re-analyze
            </button>
          )}
          <a
            href="#"
            onClick={(e) => {
              e.preventDefault();
              // Navigate to AI Debug view - dispatch a custom event the shell listens for
              window.dispatchEvent(new CustomEvent('navigate-view', { detail: { view: 'ai-debug', requestId: incident.id } }));
            }}
            style={{ fontSize: '11px', color: 'var(--brand-teal)', textDecoration: 'none', display: 'flex', alignItems: 'center', gap: '4px' }}
          >
            Open in AI Debug {'\u2192'}
          </a>
        </div>
      )}
    </div>
  );
}

function IncidentDetail({
  incident,
  onAcknowledge,
  onResolve,
}: {
  incident: Incident | null;
  onAcknowledge: (id: string) => void;
  onResolve: (id: string) => void;
}) {
  if (!incident) {
    return (
      <div class="incident-detail incident-detail-empty">
        <div class="incident-detail-placeholder">Select an incident to view details</div>
      </div>
    );
  }

  return (
    <div class="incident-detail">
      <div class="incident-detail-header">
        <div class="incident-detail-title-row">
          <span class={`incident-severity-icon severity-${incident.severity}`}>
            {severityIcon(incident.severity)}
          </span>
          <h3 class="incident-detail-title">{incident.title}</h3>
        </div>
        <div class="incident-detail-actions">
          {incident.status === 'open' && (
            <button
              class="btn"
              onClick={() => onAcknowledge(incident.id)}
            >
              Acknowledge
            </button>
          )}
          {incident.status !== 'resolved' && (
            <button
              class="btn btn-primary"
              onClick={() => onResolve(incident.id)}
            >
              Resolve
            </button>
          )}
        </div>
      </div>

      <div class="incident-detail-body">
        <div class="incident-detail-field">
          <span class="incident-field-label">Status</span>
          <span class={`incident-status-badge status-${incident.status}`}>
            {incident.status}
          </span>
        </div>
        <div class="incident-detail-field">
          <span class="incident-field-label">Severity</span>
          <span class={`incident-severity-badge severity-${incident.severity}`}>
            {incident.severity}
          </span>
        </div>
        <div class="incident-detail-field">
          <span class="incident-field-label">Service</span>
          <span class="incident-field-value incident-field-accent">{incident.service}</span>
        </div>
        <div class="incident-detail-field">
          <span class="incident-field-label">Created</span>
          <span class="incident-field-value">{formatTime(incident.timestamp)}</span>
        </div>
        {incident.acknowledged_at && (
          <div class="incident-detail-field">
            <span class="incident-field-label">Acknowledged</span>
            <span class="incident-field-value">{formatTime(incident.acknowledged_at)}</span>
          </div>
        )}
        {incident.resolved_at && (
          <div class="incident-detail-field">
            <span class="incident-field-label">Resolved</span>
            <span class="incident-field-value">{formatTime(incident.resolved_at)}</span>
          </div>
        )}

        <div class="incident-detail-section">
          <h4 class="incident-section-heading">Message</h4>
          <div class="incident-message">{incident.message}</div>
        </div>

        {incident.details && Object.keys(incident.details).length > 0 && (
          <div class="incident-detail-section">
            <h4 class="incident-section-heading">Details</h4>
            <pre class="incident-details-json">
              {JSON.stringify(incident.details, null, 2)}
            </pre>
          </div>
        )}

        <RootCauseSuggestion incident={incident} />

        <RelatedTraces incident={incident} />
      </div>
    </div>
  );
}

export function IncidentsView() {
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchIncidents = useCallback(() => {
    api<Incident[]>('/api/incidents')
      .then(setIncidents)
      .catch(() => setIncidents([]))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    fetchIncidents();
  }, [fetchIncidents]);

  const handleAcknowledge = useCallback((id: string) => {
    api<void>(`/api/incidents/${id}/acknowledge`, { method: 'POST' })
      .then(() => fetchIncidents())
      .catch((e) => { console.warn('[Incidents] Acknowledge failed:', e); });
  }, [fetchIncidents]);

  const handleResolve = useCallback((id: string) => {
    api<void>(`/api/incidents/${id}/resolve`, { method: 'POST' })
      .then(() => fetchIncidents())
      .catch((e) => { console.warn('[Incidents] Resolve failed:', e); });
  }, [fetchIncidents]);

  const selected = selectedId
    ? incidents.find((i) => i.id === selectedId) ?? null
    : null;

  if (loading) {
    return (
      <div class="incidents-view incidents-view-empty">
        <div class="incidents-placeholder">Loading incidents...</div>
      </div>
    );
  }

  return (
    <div class="incidents-view">
      <IncidentTimeline
        incidents={incidents}
        selectedId={selectedId}
        onSelect={setSelectedId}
      />
      <div class="incidents-split-area">
        <SplitPanel
          initialSplit={0.38}
          direction="horizontal"
          minSize={260}
          left={
            <IncidentList
              incidents={incidents}
              selectedId={selectedId}
              onSelect={setSelectedId}
            />
          }
          right={
            <IncidentDetail
              incident={selected}
              onAcknowledge={handleAcknowledge}
              onResolve={handleResolve}
            />
          }
        />
      </div>
    </div>
  );
}
