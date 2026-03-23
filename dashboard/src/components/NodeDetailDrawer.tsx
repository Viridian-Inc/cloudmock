import { useState, useEffect, useRef } from 'preact/hooks';
import { Drawer } from './Drawer';
import { StatusBadge } from './StatusBadge';
import { JsonView } from './JsonView';
import { getNodeRequests, getNodeTraces, getNodeResources, getStats, getMetrics, api } from '../api';
import { fmtTime, fmtDuration } from '../utils';

// --- Types ---

interface TopoNode {
  id: string;
  label: string;
  service: string;
  type: string;
  group: string;
}

interface TopoEdge {
  source: string;
  target: string;
  type: string;
  label: string;
  discovered: string;
  avg_latency_ms?: number;
  call_count?: number;
}

interface NodeDetailDrawerProps {
  node: TopoNode;
  edges: TopoEdge[];
  nodes: TopoNode[];
  onClose: () => void;
  onSelectNode: (node: TopoNode) => void;
}

type Tab = 'overview' | 'requests' | 'traces' | 'connections' | 'resource';

// --- Node type styling ---

const NODE_TYPE_COLORS: Record<string, { bg: string; fg: string; icon: string }> = {
  function:  { bg: '#DBEAFE', fg: '#1D4ED8', icon: 'fn' },
  table:     { bg: '#D1FAE5', fg: '#065F46', icon: 'db' },
  queue:     { bg: '#FEF3C7', fg: '#92400E', icon: 'q' },
  topic:     { bg: '#FCE7F3', fg: '#9D174D', icon: 'sns' },
  bucket:    { bg: '#E0E7FF', fg: '#3730A3', icon: 's3' },
  userpool:  { bg: '#EDE9FE', fg: '#5B21B6', icon: 'id' },
  eventbus:  { bg: '#FFF7ED', fg: '#C2410C', icon: 'ev' },
  client:    { bg: '#F0F9FF', fg: '#0369A1', icon: 'app' },
  plugin:    { bg: '#F1F5F9', fg: '#475569', icon: 'ext' },
  loggroup:  { bg: '#ECFDF5', fg: '#047857', icon: 'log' },
  key:       { bg: '#FEF2F2', fg: '#991B1B', icon: 'key' },
  secret:    { bg: '#FEF2F2', fg: '#991B1B', icon: 'sec' },
  alarm:     { bg: '#FFF1F2', fg: '#BE123C', icon: 'alm' },
  role:      { bg: '#F5F3FF', fg: '#6D28D9', icon: 'iam' },
  stack:     { bg: '#F0FDFA', fg: '#0F766E', icon: 'cfn' },
  api:       { bg: '#EFF6FF', fg: '#1E40AF', icon: 'api' },
};

const DEFAULT_TYPE_STYLE = { bg: '#F1F5F9', fg: '#475569', icon: '?' };

function getTypeStyle(type: string) {
  return NODE_TYPE_COLORS[type] || DEFAULT_TYPE_STYLE;
}

// --- Edge type colors ---

const EDGE_TYPE_COLORS: Record<string, string> = {
  trigger: '#3B82F6',
  read_write: '#10B981',
  publish: '#8B5CF6',
  subscribe: '#EC4899',
  rule: '#F59E0B',
  traffic: '#6366F1',
  config: '#94A3B8',
  alarm: '#EF4444',
  cfn: '#06B6D4',
};

// --- Service colors for waterfall ---

const SPAN_COLORS = ['#3B82F6', '#8B5CF6', '#EC4899', '#F59E0B', '#10B981', '#06B6D4', '#6366F1', '#EF4444'];
function spanColor(service: string): string {
  let hash = 0;
  for (let i = 0; i < service.length; i++) hash = ((hash << 5) - hash + service.charCodeAt(i)) | 0;
  return SPAN_COLORS[Math.abs(hash) % SPAN_COLORS.length];
}

// --- Main component ---

export function NodeDetailDrawer({ node, edges, nodes, onClose, onSelectNode }: NodeDetailDrawerProps) {
  const [tab, setTab] = useState<Tab>('overview');
  const [requests, setRequests] = useState<any[]>([]);
  const [traces, setTraces] = useState<any[]>([]);
  const [resources, setResources] = useState<any>(null);
  const [stats, setStats] = useState<Record<string, number>>({});
  const [metrics, setMetrics] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    setTab('overview');
    Promise.all([
      getNodeRequests(node.service).catch(() => []),
      getNodeTraces(node.service).catch(() => []),
      getNodeResources(node.service).catch(() => null),
      getStats().catch(() => ({})),
      getMetrics().catch(() => null),
    ]).then(([reqs, trs, res, st, met]) => {
      setRequests(reqs || []);
      setTraces(trs || []);
      setResources(res);
      setStats(st || {});
      setMetrics(met);
      setLoading(false);
    });
  }, [node.id]);

  const inbound = edges.filter(e => e.target === node.id);
  const outbound = edges.filter(e => e.source === node.id);
  const nodeMap = new Map(nodes.map(n => [n.id, n]));

  const reqCount = stats[node.service] || 0;
  const errorCount = requests.filter(r => r.status >= 400).length;
  const errorRate = requests.length > 0 ? Math.round((errorCount / requests.length) * 100) : 0;
  const svcMetrics = metrics?.services?.[node.service];
  const avgLatency = svcMetrics?.p50 || 0;
  const typeStyle = getTypeStyle(node.type);

  const tabs: { key: Tab; label: string; count?: number }[] = [
    { key: 'overview', label: 'Overview' },
    { key: 'requests', label: 'Requests', count: requests.length },
    { key: 'traces', label: 'Traces', count: traces.length },
    { key: 'connections', label: 'Connections', count: inbound.length + outbound.length },
    { key: 'resource', label: 'Resource' },
  ];

  return (
    <Drawer title={node.label} onClose={onClose}>
      {/* Node identity header */}
      <div style={S.nodeHeader}>
        <div style={{ ...S.typeIcon, background: typeStyle.bg, color: typeStyle.fg }}>
          {typeStyle.icon}
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={S.nodeTitle}>{node.label}</div>
          <div style={S.nodeMeta}>
            <span style={S.metaChip}>{node.service}</span>
            <span style={S.metaDot} />
            <span style={S.metaChip}>{node.type}</span>
            <span style={S.metaDot} />
            <span style={{ ...S.metaChip, color: 'var(--n400)' }}>{node.group}</span>
          </div>
        </div>
      </div>

      {/* Tab bar */}
      <div style={S.tabBar}>
        {tabs.map(t => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            style={{
              ...S.tabBtn,
              color: tab === t.key ? 'var(--brand-blue, #097FF5)' : 'var(--n500, #64748B)',
              fontWeight: tab === t.key ? 600 : 400,
            }}
          >
            {t.label}
            {t.count !== undefined && (
              <span style={{
                ...S.tabCount,
                background: tab === t.key ? 'var(--brand-blue, #097FF5)' : 'var(--n200, #E2E8F0)',
                color: tab === t.key ? 'white' : 'var(--n500, #64748B)',
              }}>{t.count}</span>
            )}
            {tab === t.key && <div style={S.tabIndicator} />}
          </button>
        ))}
      </div>

      {/* Content */}
      <div style={S.tabContent}>
        {loading ? <LoadingSkeleton /> : (
          <>
            {tab === 'overview' && (
              <OverviewTab
                reqCount={reqCount} errorRate={errorRate} avgLatency={avgLatency}
                p95={svcMetrics?.p95 || 0} p99={svcMetrics?.p99 || 0}
                inbound={inbound.length} outbound={outbound.length}
                requests={requests}
              />
            )}
            {tab === 'requests' && <RequestsTab requests={requests} />}
            {tab === 'traces' && <TracesTab traces={traces} />}
            {tab === 'connections' && (
              <ConnectionsTab inbound={inbound} outbound={outbound} nodeMap={nodeMap} onSelectNode={onSelectNode} />
            )}
            {tab === 'resource' && <ResourceTab resources={resources} nodeType={node.type} />}
          </>
        )}
      </div>
    </Drawer>
  );
}

// --- Loading skeleton ---

function LoadingSkeleton() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {[1, 2, 3].map(i => (
        <div key={i} style={{
          height: 64, borderRadius: 10, background: 'var(--n100, #F1F5F9)',
          animation: 'pulse 1.5s ease-in-out infinite',
        }} />
      ))}
    </div>
  );
}

// --- Overview tab ---

function OverviewTab({ reqCount, errorRate, avgLatency, p95, p99, inbound, outbound, requests }: {
  reqCount: number; errorRate: number; avgLatency: number; p95: number; p99: number;
  inbound: number; outbound: number; requests: any[];
}) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Stat cards */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 10 }}>
        <StatCard label="Requests" value={reqCount.toLocaleString()} icon="reqs" />
        <StatCard label="Error Rate" value={`${errorRate}%`} icon="err" accent={errorRate > 0 ? '#EF4444' : undefined} />
        <StatCard label="P50 Latency" value={fmtDuration(avgLatency)} icon="lat" />
      </div>

      {/* Latency breakdown */}
      <div style={S.card}>
        <div style={S.cardLabel}>Latency Percentiles</div>
        <div style={{ display: 'flex', gap: 20, marginTop: 8 }}>
          <LatencyBar label="P50" ms={avgLatency} max={p99 || avgLatency || 1} color="#3B82F6" />
          <LatencyBar label="P95" ms={p95} max={p99 || p95 || 1} color="#F59E0B" />
          <LatencyBar label="P99" ms={p99} max={p99 || 1} color="#EF4444" />
        </div>
      </div>

      {/* Connection summary */}
      <div style={S.card}>
        <div style={S.cardLabel}>Topology</div>
        <div style={{ display: 'flex', gap: 16, marginTop: 8 }}>
          <div style={{ flex: 1, textAlign: 'center' }}>
            <div style={{ fontSize: 24, fontWeight: 700, color: 'var(--n800, #1E293B)' }}>{inbound}</div>
            <div style={{ fontSize: 11, color: 'var(--n400)', marginTop: 2 }}>Inbound</div>
          </div>
          <div style={{ width: 1, background: 'var(--n200, #E2E8F0)' }} />
          <div style={{ flex: 1, textAlign: 'center' }}>
            <div style={{ fontSize: 24, fontWeight: 700, color: 'var(--n800, #1E293B)' }}>{outbound}</div>
            <div style={{ fontSize: 11, color: 'var(--n400)', marginTop: 2 }}>Outbound</div>
          </div>
        </div>
      </div>

      {/* Recent activity sparkline */}
      {requests.length > 0 && (
        <div style={S.card}>
          <div style={S.cardLabel}>Recent Activity</div>
          <MiniTimeline requests={requests} />
        </div>
      )}
    </div>
  );
}

function StatCard({ label, value, icon, accent }: { label: string; value: string | number; icon: string; accent?: string }) {
  const iconBg = accent ? `${accent}15` : 'var(--brand-blue-50, #E6F2FF)';
  const iconFg = accent || 'var(--brand-blue, #097FF5)';
  const icons: Record<string, string> = { reqs: '\u21C5', err: '\u26A0', lat: '\u23F1' };

  return (
    <div style={S.statCard}>
      <div style={{ ...S.statIcon, background: iconBg, color: iconFg }}>{icons[icon] || '\u2022'}</div>
      <div style={{ fontSize: 22, fontWeight: 700, color: accent || 'var(--n800, #1E293B)', lineHeight: 1, marginTop: 8 }}>
        {value}
      </div>
      <div style={{ fontSize: 11, color: 'var(--n400, #94A3B8)', marginTop: 4, fontWeight: 500 }}>{label}</div>
    </div>
  );
}

function LatencyBar({ label, ms, max, color }: { label: string; ms: number; max: number; color: string }) {
  const pct = max > 0 ? Math.min((ms / max) * 100, 100) : 0;
  return (
    <div style={{ flex: 1 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
        <span style={{ fontSize: 11, fontWeight: 600, color: 'var(--n600)' }}>{label}</span>
        <span style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--n500)' }}>{fmtDuration(ms)}</span>
      </div>
      <div style={{ height: 6, background: 'var(--n100, #F1F5F9)', borderRadius: 3 }}>
        <div style={{ height: '100%', width: `${pct}%`, background: color, borderRadius: 3, transition: 'width 0.3s ease' }} />
      </div>
    </div>
  );
}

function MiniTimeline({ requests }: { requests: any[] }) {
  const buckets = new Array(20).fill(0);
  const now = Date.now();
  const windowMs = 5 * 60 * 1000;
  requests.forEach(r => {
    const age = now - new Date(r.timestamp).getTime();
    if (age < windowMs) {
      const idx = Math.min(Math.floor((age / windowMs) * 20), 19);
      buckets[19 - idx]++;
    }
  });
  const max = Math.max(...buckets, 1);

  return (
    <div style={{ display: 'flex', alignItems: 'flex-end', gap: 2, height: 32, marginTop: 8 }}>
      {buckets.map((count, i) => (
        <div key={i} style={{
          flex: 1,
          height: `${Math.max((count / max) * 100, 4)}%`,
          background: count > 0 ? 'var(--brand-blue, #097FF5)' : 'var(--n100, #F1F5F9)',
          borderRadius: 2,
          opacity: count > 0 ? 0.7 : 0.4,
          transition: 'height 0.2s ease',
        }} />
      ))}
    </div>
  );
}

// --- Requests tab ---

function RequestsTab({ requests }: { requests: any[] }) {
  const [expanded, setExpanded] = useState<string | null>(null);
  const [detail, setDetail] = useState<any>(null);

  function toggleExpand(id: string) {
    if (expanded === id) {
      setExpanded(null);
      setDetail(null);
    } else {
      setExpanded(id);
      setDetail(null);
      api(`/api/requests/${id}`).then(setDetail).catch(() => {});
    }
  }

  if (requests.length === 0) {
    return <EmptyState icon="\u21C5" message="No recent requests" />;
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
      {requests.map((r: any) => {
        const isExpanded = expanded === r.id;
        const isError = r.status >= 400;
        return (
          <div key={r.id} style={{
            borderRadius: 8,
            border: `1px solid ${isExpanded ? 'var(--brand-blue, #097FF5)20' : 'var(--n200, #E2E8F0)'}`,
            background: isExpanded ? 'var(--brand-blue-50, #F0F7FF)' : 'white',
            overflow: 'hidden',
            transition: 'all 0.15s ease',
          }}>
            <div
              onClick={() => toggleExpand(r.id)}
              style={{
                display: 'flex', alignItems: 'center', gap: 10, padding: '10px 12px',
                cursor: 'pointer', fontSize: 12,
              }}
            >
              <span style={{
                ...S.expandChevron,
                transform: isExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
              }}>{'\u203A'}</span>
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--n400)', minWidth: 62 }}>{fmtTime(r.timestamp)}</span>
              <span style={{ fontWeight: 500, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{r.action}</span>
              <StatusBadge code={r.status} />
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: isError ? '#EF4444' : 'var(--n400)', minWidth: 44, textAlign: 'right' }}>
                {fmtDuration(r.latency_ms)}
              </span>
            </div>
            {isExpanded && <RequestInlineDetail req={detail || r} />}
          </div>
        );
      })}
    </div>
  );
}

function RequestInlineDetail({ req }: { req: any }) {
  const [subTab, setSubTab] = useState<'info' | 'request' | 'response' | 'explain'>('info');
  const [explainData, setExplainData] = useState<any>(null);
  const [explainLoading, setExplainLoading] = useState(false);

  function loadExplain() {
    if (explainData || explainLoading) return;
    setExplainLoading(true);
    api(`/api/explain/${req.id}`).then(setExplainData).catch(() => {}).finally(() => setExplainLoading(false));
  }

  return (
    <div style={{ borderTop: '1px solid var(--n100, #F1F5F9)' }}>
      <div style={{ display: 'flex', gap: 0, background: 'var(--n50, #F8FAFC)', borderBottom: '1px solid var(--n100)' }}>
        {(['info', 'request', 'response', 'explain'] as const).map(t => (
          <button key={t} onClick={() => { setSubTab(t); if (t === 'explain') loadExplain(); }} style={{
            ...S.subTab,
            color: subTab === t ? (t === 'explain' ? '#8B5CF6' : 'var(--brand-blue, #097FF5)') : 'var(--n500)',
            fontWeight: subTab === t ? 600 : 400,
            borderBottom: subTab === t ? `2px solid ${t === 'explain' ? '#8B5CF6' : 'var(--brand-blue, #097FF5)'}` : '2px solid transparent',
          }}>{t === 'explain' ? '\u2728 Explain' : t.charAt(0).toUpperCase() + t.slice(1)}</button>
        ))}
      </div>

      <div style={{ padding: 12 }}>
        {subTab === 'info' && (
          <div style={{ display: 'grid', gridTemplateColumns: '100px 1fr', gap: '8px 0', fontSize: 12 }}>
            <InfoRow label="Service" value={req.service} />
            <InfoRow label="Action" value={req.action} />
            <InfoRow label="Method" value={<span style={{ fontFamily: 'var(--font-mono)', fontWeight: 600 }}>{req.method}</span>} />
            <InfoRow label="Path" value={<span style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}>{req.path}</span>} />
            <InfoRow label="Status" value={<StatusBadge code={req.status} />} />
            <InfoRow label="Latency" value={fmtDuration(req.latency_ms)} />
            {req.caller_id && <InfoRow label="Caller" value={req.caller_id} />}
            {req.error && <InfoRow label="Error" value={<span style={{ color: '#EF4444' }}>{req.error}</span>} />}
          </div>
        )}
        {subTab === 'request' && (
          <div style={{ maxHeight: 220, overflow: 'auto', borderRadius: 6 }}>
            {req.request_body ? <JsonView data={tryParse(req.request_body)} /> : <span style={{ color: 'var(--n400)', fontSize: 12 }}>No request body</span>}
          </div>
        )}
        {subTab === 'response' && (
          <div style={{ maxHeight: 220, overflow: 'auto', borderRadius: 6 }}>
            {req.response_body ? <JsonView data={tryParse(req.response_body)} /> : <span style={{ color: 'var(--n400)', fontSize: 12 }}>No response body</span>}
          </div>
        )}
        {subTab === 'explain' && <ExplainPanel data={explainData} loading={explainLoading} />}
      </div>
    </div>
  );
}

function ExplainPanel({ data, loading }: { data: any; loading: boolean }) {
  if (loading) {
    return <div style={{ textAlign: 'center', padding: 20, color: 'var(--n400)' }}><div style={S.spinner} /> Analyzing...</div>;
  }
  if (!data) {
    return <div style={{ textAlign: 'center', padding: 20, color: 'var(--n400)' }}>No analysis available</div>;
  }

  const a = data.analysis;
  const anomalies = a.anomalies || [];

  return (
    <div style={{ fontSize: 12 }}>
      {/* Anomalies */}
      {anomalies.length > 0 && (
        <div style={{ marginBottom: 12 }}>
          {anomalies.map((msg: string, i: number) => (
            <div key={i} style={{
              padding: '8px 10px', marginBottom: 4, borderRadius: 6,
              background: '#FEF2F2', border: '1px solid #FECACA', color: '#991B1B', fontSize: 11,
            }}>
              {'\u26A0'} {msg}
            </div>
          ))}
        </div>
      )}

      {/* Latency context */}
      <div style={{ ...S.card, marginBottom: 10 }}>
        <div style={S.cardLabel}>Latency Context</div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 8, marginTop: 8 }}>
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: 16, fontWeight: 700 }}>{fmtDuration(a.p50_ms)}</div>
            <div style={{ fontSize: 10, color: 'var(--n400)' }}>P50</div>
          </div>
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: 16, fontWeight: 700, color: '#F59E0B' }}>{fmtDuration(a.p95_ms)}</div>
            <div style={{ fontSize: 10, color: 'var(--n400)' }}>P95</div>
          </div>
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: 16, fontWeight: 700, color: '#EF4444' }}>{fmtDuration(a.p99_ms)}</div>
            <div style={{ fontSize: 10, color: 'var(--n400)' }}>P99</div>
          </div>
        </div>
        <div style={{ textAlign: 'center', marginTop: 8, fontSize: 11, color: a.is_slow ? '#EF4444' : 'var(--n500)' }}>
          This request: {fmtDuration(data.request?.latency_ms)} ({a.latency_ratio?.toFixed(1)}x P50)
        </div>
      </div>

      {/* Health */}
      <div style={{ ...S.card, marginBottom: 10 }}>
        <div style={S.cardLabel}>Service Health</div>
        <div style={{ display: 'flex', gap: 16, marginTop: 8 }}>
          <div>
            <span style={{ color: 'var(--n400)' }}>Error rate: </span>
            <span style={{ fontWeight: 600, color: a.error_rate > 0.1 ? '#EF4444' : 'var(--n700)' }}>
              {(a.error_rate * 100).toFixed(0)}%
            </span>
          </div>
          <div>
            <span style={{ color: 'var(--n400)' }}>Spans: </span>
            <span style={{ fontWeight: 600 }}>{a.span_count}</span>
          </div>
          {a.slowest_span && (
            <div>
              <span style={{ color: 'var(--n400)' }}>Slowest: </span>
              <span style={{ fontWeight: 600, fontFamily: 'var(--font-mono)', fontSize: 11 }}>{a.slowest_span}</span>
            </div>
          )}
        </div>
      </div>

      {/* Similar requests */}
      {data.similar_recent && data.similar_recent.length > 0 && (
        <div style={{ ...S.card }}>
          <div style={S.cardLabel}>Similar Requests ({data.similar_recent.length})</div>
          <div style={{ marginTop: 8, maxHeight: 120, overflow: 'auto' }}>
            {data.similar_recent.slice(0, 8).map((r: any, i: number) => (
              <div key={i} style={{ display: 'flex', gap: 8, fontSize: 11, padding: '3px 0', borderBottom: '1px solid var(--n100)' }}>
                <span style={{ color: 'var(--n400)', fontFamily: 'var(--font-mono)', width: 62, flexShrink: 0 }}>{fmtTime(r.timestamp)}</span>
                <StatusBadge code={r.status_code || r.status} />
                <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--n500)' }}>{fmtDuration(r.latency_ms)}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// --- Traces tab ---

function TracesTab({ traces }: { traces: any[] }) {
  const [expanded, setExpanded] = useState<string | null>(null);
  const [timeline, setTimeline] = useState<any[]>([]);

  function toggleExpand(traceId: string) {
    if (expanded === traceId) {
      setExpanded(null);
      setTimeline([]);
    } else {
      setExpanded(traceId);
      setTimeline([]);
      api(`/api/traces/${traceId}/timeline`).then(setTimeline).catch(() => {});
    }
  }

  if (traces.length === 0) {
    return <EmptyState icon="\u2B21" message="No recent traces" />;
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
      {traces.map((t: any) => {
        const isExpanded = expanded === t.trace_id;
        return (
          <div key={t.trace_id} style={{
            borderRadius: 8,
            border: `1px solid ${isExpanded ? 'var(--brand-blue, #097FF5)20' : 'var(--n200, #E2E8F0)'}`,
            background: isExpanded ? 'var(--brand-blue-50, #F0F7FF)' : 'white',
            overflow: 'hidden',
            transition: 'all 0.15s ease',
          }}>
            <div
              onClick={() => toggleExpand(t.trace_id)}
              style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 12px', cursor: 'pointer', fontSize: 12 }}
            >
              <span style={{
                ...S.expandChevron,
                transform: isExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
              }}>{'\u203A'}</span>
              <span style={{ fontWeight: 500, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                {t.root_action || t.root_service}
              </span>
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--n400)' }}>{t.span_count} spans</span>
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--n500)' }}>{fmtDuration(t.duration_ms)}</span>
              {t.has_error ? (
                <span style={{ padding: '1px 8px', borderRadius: 10, background: '#FEE2E2', color: '#DC2626', fontSize: 10, fontWeight: 600 }}>Error</span>
              ) : (
                <StatusBadge code={t.status_code} />
              )}
            </div>
            {isExpanded && <WaterfallTimeline spans={timeline} totalMs={t.duration_ms} />}
          </div>
        );
      })}
    </div>
  );
}

function WaterfallTimeline({ spans, totalMs }: { spans: any[]; totalMs: number }) {
  if (spans.length === 0) {
    return (
      <div style={{ padding: 16, borderTop: '1px solid var(--n100)', textAlign: 'center' }}>
        <div style={{ ...S.spinner }} />
      </div>
    );
  }

  const maxMs = totalMs || Math.max(...spans.map(s => (s.start_offset_ms || 0) + (s.duration_ms || 0)), 1);

  return (
    <div style={{ borderTop: '1px solid var(--n100)', padding: '12px 12px 8px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 10 }}>
        <span style={{ fontSize: 11, fontWeight: 600, color: 'var(--n600)' }}>Waterfall</span>
        <span style={{ fontSize: 10, fontFamily: 'var(--font-mono)', color: 'var(--n400)' }}>{spans.length} spans / {fmtDuration(totalMs)}</span>
      </div>
      {spans.map((s: any, i: number) => {
        const left = ((s.start_offset_ms || 0) / maxMs) * 100;
        const width = Math.max(((s.duration_ms || 0) / maxMs) * 100, 0.5);
        const indent = (s.depth || 0) * 16;
        const color = s.error ? '#EF4444' : s.status_code >= 400 ? '#F59E0B' : spanColor(s.service || '');

        return (
          <div key={i} style={{ display: 'flex', alignItems: 'center', marginBottom: 4, fontSize: 11 }}>
            <div style={{
              width: 130, flexShrink: 0, paddingLeft: indent,
              overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
              color: 'var(--n600)',
            }}>
              <span style={{ display: 'inline-block', width: 6, height: 6, borderRadius: '50%', background: color, marginRight: 5, verticalAlign: 'middle' }} />
              {s.action || s.service}
            </div>
            <div style={{ flex: 1, height: 16, background: 'var(--n100, #F1F5F9)', borderRadius: 3, position: 'relative', overflow: 'hidden' }}>
              <div style={{
                position: 'absolute', left: `${left}%`, width: `${width}%`,
                height: '100%', background: color, borderRadius: 3,
                opacity: 0.85,
                boxShadow: `0 0 0 1px ${color}30`,
                transition: 'all 0.2s ease',
              }}>
                {width > 8 && (
                  <span style={{ position: 'absolute', left: 4, top: 1, fontSize: 9, color: 'white', fontWeight: 600, fontFamily: 'var(--font-mono)' }}>
                    {fmtDuration(s.duration_ms)}
                  </span>
                )}
              </div>
            </div>
            {width <= 8 && (
              <div style={{ width: 44, textAlign: 'right', flexShrink: 0, fontFamily: 'var(--font-mono)', color: 'var(--n400)', fontSize: 10 }}>
                {fmtDuration(s.duration_ms)}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

// --- Connections tab ---

function ConnectionsTab({ inbound, outbound, nodeMap, onSelectNode }: {
  inbound: TopoEdge[]; outbound: TopoEdge[];
  nodeMap: Map<string, TopoNode>; onSelectNode: (node: TopoNode) => void;
}) {
  if (inbound.length === 0 && outbound.length === 0) {
    return <EmptyState icon="\u2194" message="No connections" />;
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {inbound.length > 0 && (
        <ConnectionGroup label="Inbound" icon="\u2190" edges={inbound} direction="in" nodeMap={nodeMap} onSelect={onSelectNode} />
      )}
      {outbound.length > 0 && (
        <ConnectionGroup label="Outbound" icon="\u2192" edges={outbound} direction="out" nodeMap={nodeMap} onSelect={onSelectNode} />
      )}
    </div>
  );
}

function ConnectionGroup({ label, icon, edges, direction, nodeMap, onSelect }: {
  label: string; icon: string; edges: TopoEdge[];
  direction: 'in' | 'out';
  nodeMap: Map<string, TopoNode>; onSelect: (n: TopoNode) => void;
}) {
  return (
    <div>
      <div style={{ fontSize: 12, fontWeight: 600, color: 'var(--n600)', marginBottom: 8, display: 'flex', alignItems: 'center', gap: 6 }}>
        <span>{icon}</span> {label}
        <span style={{ fontWeight: 400, color: 'var(--n400)' }}>({edges.length})</span>
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
        {edges.map((edge, i) => {
          const peerId = direction === 'in' ? edge.source : edge.target;
          const peer = nodeMap.get(peerId);
          const peerType = peer ? getTypeStyle(peer.type) : DEFAULT_TYPE_STYLE;
          const edgeColor = EDGE_TYPE_COLORS[edge.type] || '#94A3B8';

          return (
            <div
              key={i}
              onClick={() => peer && onSelect(peer)}
              style={{
                display: 'flex', alignItems: 'center', gap: 10, padding: '8px 10px',
                borderRadius: 8, border: '1px solid var(--n200, #E2E8F0)',
                cursor: peer ? 'pointer' : 'default',
                transition: 'all 0.1s ease',
                background: 'white',
              }}
              onMouseEnter={(e) => { (e.currentTarget as HTMLElement).style.borderColor = edgeColor + '60'; (e.currentTarget as HTMLElement).style.background = edgeColor + '08'; }}
              onMouseLeave={(e) => { (e.currentTarget as HTMLElement).style.borderColor = ''; (e.currentTarget as HTMLElement).style.background = 'white'; }}
            >
              <div style={{ ...S.typeIconSm, background: peerType.bg, color: peerType.fg }}>{peerType.icon}</div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 12, fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {peer?.label || peerId}
                </div>
                <div style={{ fontSize: 10, color: 'var(--n400)', display: 'flex', gap: 8, marginTop: 2 }}>
                  <span style={{ color: edgeColor, fontWeight: 600 }}>{edge.type}</span>
                  {edge.discovered && <span>{edge.discovered}</span>}
                </div>
              </div>
              <div style={{ textAlign: 'right', flexShrink: 0 }}>
                <div style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--n500)' }}>
                  {edge.avg_latency_ms ? fmtDuration(edge.avg_latency_ms) : '\u2014'}
                </div>
                {(edge.call_count || 0) > 0 && (
                  <div style={{ fontSize: 10, color: 'var(--n400)' }}>{edge.call_count} calls</div>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// --- Resource tab ---

function ResourceTab({ resources, nodeType }: { resources: any; nodeType: string }) {
  if (!resources || (Array.isArray(resources) && resources.length === 0)) {
    return <EmptyState icon="\u2699" message="No resource details available" />;
  }

  return (
    <div style={{ overflow: 'auto', maxHeight: 500, borderRadius: 8, border: '1px solid var(--n200)', background: 'var(--n50, #F8FAFC)' }}>
      <div style={{ padding: 12 }}>
        <JsonView data={resources} />
      </div>
    </div>
  );
}

// --- Shared components ---

function EmptyState({ icon, message }: { icon: string; message: string }) {
  return (
    <div style={{ textAlign: 'center', padding: '40px 20px' }}>
      <div style={{ fontSize: 32, marginBottom: 8, opacity: 0.3 }}>{icon}</div>
      <div style={{ fontSize: 13, color: 'var(--n400, #94A3B8)' }}>{message}</div>
    </div>
  );
}

function InfoRow({ label, value }: { label: string; value: any }) {
  return (
    <>
      <div style={{ color: 'var(--n400)', fontSize: 12, padding: '2px 0' }}>{label}</div>
      <div style={{ color: 'var(--n700, #334155)', fontSize: 12, padding: '2px 0' }}>{value}</div>
    </>
  );
}

function tryParse(s: string): any {
  try { return JSON.parse(s); } catch { return s; }
}

// --- Styles ---

const S = {
  nodeHeader: {
    display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16,
    padding: '12px 14px', borderRadius: 10,
    background: 'var(--n50, #F8FAFC)', border: '1px solid var(--n200, #E2E8F0)',
  } as const,
  typeIcon: {
    width: 36, height: 36, borderRadius: 8, display: 'flex',
    alignItems: 'center', justifyContent: 'center',
    fontSize: 11, fontWeight: 700, fontFamily: 'var(--font-mono)',
    flexShrink: 0, textTransform: 'uppercase' as const, letterSpacing: 0.5,
  } as const,
  typeIconSm: {
    width: 26, height: 26, borderRadius: 6, display: 'flex',
    alignItems: 'center', justifyContent: 'center',
    fontSize: 9, fontWeight: 700, fontFamily: 'var(--font-mono)',
    flexShrink: 0, textTransform: 'uppercase' as const, letterSpacing: 0.3,
  } as const,
  nodeTitle: { fontSize: 15, fontWeight: 700, color: 'var(--n800, #1E293B)', lineHeight: 1.2 } as const,
  nodeMeta: {
    display: 'flex', alignItems: 'center', gap: 6, marginTop: 4, fontSize: 12, color: 'var(--n500, #64748B)',
  } as const,
  metaChip: { fontSize: 11, fontWeight: 500 } as const,
  metaDot: {
    width: 3, height: 3, borderRadius: '50%', background: 'var(--n300, #CBD5E1)', flexShrink: 0,
  } as const,
  tabBar: {
    display: 'flex', gap: 0, borderBottom: '1px solid var(--n200, #E2E8F0)',
    marginBottom: 16, marginLeft: -20, marginRight: -20, paddingLeft: 20, paddingRight: 20,
    overflowX: 'auto' as const,
  } as const,
  tabBtn: {
    position: 'relative' as const, padding: '8px 14px 10px', fontSize: 12,
    background: 'none', border: 'none', cursor: 'pointer',
    whiteSpace: 'nowrap' as const, display: 'flex', alignItems: 'center', gap: 6,
  } as const,
  tabCount: {
    display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
    minWidth: 18, height: 18, borderRadius: 9, fontSize: 10, fontWeight: 600,
    padding: '0 5px', lineHeight: 1, transition: 'all 0.15s ease',
  } as const,
  tabIndicator: {
    position: 'absolute' as const, bottom: -1, left: 14, right: 14, height: 2,
    background: 'var(--brand-blue, #097FF5)', borderRadius: '2px 2px 0 0',
  } as const,
  tabContent: { minHeight: 200 } as const,
  card: {
    padding: 14, borderRadius: 10, border: '1px solid var(--n200, #E2E8F0)',
    background: 'white',
  } as const,
  cardLabel: { fontSize: 11, fontWeight: 600, color: 'var(--n400, #94A3B8)', textTransform: 'uppercase' as const, letterSpacing: 0.5 } as const,
  statCard: {
    padding: 14, borderRadius: 10, border: '1px solid var(--n200, #E2E8F0)',
    background: 'white', textAlign: 'center' as const,
  } as const,
  statIcon: {
    width: 28, height: 28, borderRadius: 8, display: 'inline-flex',
    alignItems: 'center', justifyContent: 'center', fontSize: 14,
  } as const,
  expandChevron: {
    display: 'inline-block', fontSize: 16, fontWeight: 700, color: 'var(--n400)',
    transition: 'transform 0.15s ease', lineHeight: 1, width: 12, textAlign: 'center' as const,
  } as const,
  subTab: {
    padding: '6px 12px', fontSize: 11, background: 'none', border: 'none',
    cursor: 'pointer' as const, whiteSpace: 'nowrap' as const,
  } as const,
  spinner: {
    width: 20, height: 20, border: '2px solid var(--n200)', borderTopColor: 'var(--brand-blue, #097FF5)',
    borderRadius: '50%', animation: 'spin 0.6s linear infinite', display: 'inline-block',
  } as const,
};
