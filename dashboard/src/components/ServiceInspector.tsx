import { useState, useEffect } from 'preact/hooks';
import { api, getNodeRequests, getNodeTraces, getStats, getMetrics, getBlastRadius, getSLOStatus, captureProfile, fetchProfiles, fetchProfileFlamegraph } from '../api';
import { StatusBadge } from './StatusBadge';
import { JsonView } from './JsonView';
import { FlameGraph } from './FlameGraph';
import { fmtTime, fmtDuration } from '../utils';

interface TopoNode {
  id: string;
  label: string;
  service: string;
  type: string;
  group: string;
  requestService?: string;
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

interface ServiceInspectorProps {
  node: TopoNode;
  edges: TopoEdge[];
  nodes: TopoNode[];
  onSelectNode: (node: TopoNode) => void;
}

type Tab = 'overview' | 'requests' | 'traces' | 'connections' | 'slo' | 'blast' | 'profile';

const NODE_TYPE_COLORS: Record<string, { bg: string; fg: string; icon: string }> = {
  function:  { bg: 'rgba(83,142,255,0.15)', fg: '#538eff', icon: 'fn' },
  table:     { bg: 'rgba(54,217,130,0.15)', fg: '#36d982', icon: 'db' },
  queue:     { bg: 'rgba(250,208,101,0.15)', fg: '#fad065', icon: 'q' },
  topic:     { bg: 'rgba(139,92,246,0.15)', fg: '#8B5CF6', icon: 'sns' },
  bucket:    { bg: 'rgba(9,127,245,0.15)', fg: '#097FF5', icon: 's3' },
  userpool:  { bg: 'rgba(139,92,246,0.15)', fg: '#8B5CF6', icon: 'id' },
  eventbus:  { bg: 'rgba(247,113,30,0.15)', fg: '#F7711E', icon: 'ev' },
  client:    { bg: 'rgba(74,229,248,0.15)', fg: '#4AE5F8', icon: 'app' },
  plugin:    { bg: 'rgba(139,149,165,0.15)', fg: '#8b95a5', icon: 'ext' },
  loggroup:  { bg: 'rgba(54,217,130,0.15)', fg: '#36d982', icon: 'log' },
  key:       { bg: 'rgba(255,78,94,0.15)', fg: '#ff4e5e', icon: 'key' },
  secret:    { bg: 'rgba(255,78,94,0.15)', fg: '#ff4e5e', icon: 'sec' },
  alarm:     { bg: 'rgba(255,78,94,0.15)', fg: '#ff4e5e', icon: 'alm' },
  role:      { bg: 'rgba(139,92,246,0.15)', fg: '#8B5CF6', icon: 'iam' },
  stack:     { bg: 'rgba(74,229,248,0.15)', fg: '#4AE5F8', icon: 'cfn' },
  api:       { bg: 'rgba(9,127,245,0.15)', fg: '#097FF5', icon: 'api' },
};

const DEFAULT_TYPE_STYLE = { bg: 'var(--bg-elevated)', fg: 'var(--text-secondary)', icon: '?' };

function getTypeStyle(type: string) {
  return NODE_TYPE_COLORS[type] || DEFAULT_TYPE_STYLE;
}

const EDGE_TYPE_COLORS: Record<string, string> = {
  trigger: '#3B82F6', read_write: 'var(--success)', publish: '#8B5CF6',
  subscribe: '#EC4899', rule: 'var(--warning)', traffic: '#6366F1',
  config: 'var(--text-secondary)', alarm: 'var(--error)', cfn: '#06B6D4',
};

const SPAN_COLORS = ['#3B82F6', '#8B5CF6', '#EC4899', 'var(--warning)', 'var(--success)', '#06B6D4', '#6366F1', 'var(--error)'];
function spanColor(service: string): string {
  let hash = 0;
  for (let i = 0; i < service.length; i++) hash = ((hash << 5) - hash + service.charCodeAt(i)) | 0;
  return SPAN_COLORS[Math.abs(hash) % SPAN_COLORS.length];
}

function nodeToRequestService(node: TopoNode): string {
  return node.requestService || node.service;
}

function tryParse(s: string): any {
  try { return JSON.parse(s); } catch { return s; }
}

export function ServiceInspector({ node, edges, nodes, onSelectNode }: ServiceInspectorProps) {
  const [tab, setTab] = useState<Tab>('overview');
  const [requests, setRequests] = useState<any[]>([]);
  const [traces, setTraces] = useState<any[]>([]);
  const [stats, setStats] = useState<Record<string, number>>({});
  const [metrics, setMetrics] = useState<any>(null);
  const [sloData, setSloData] = useState<any>(null);
  const [blastData, setBlastData] = useState<any>(null);
  const [explainData, setExplainData] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  // Profile tab state
  const [profileList, setProfileList] = useState<any[]>([]);
  const [flamegraphData, setFlamegraphData] = useState<string>('');
  const [capturing, setCapturing] = useState(false);
  const [captureError, setCaptureError] = useState<string | null>(null);

  const requestService = nodeToRequestService(node);

  useEffect(() => {
    setLoading(true);
    setTab('overview');
    setSloData(null);
    setBlastData(null);
    setExplainData(null);
    setProfileList([]);
    setFlamegraphData('');
    setCaptureError(null);

    Promise.all([
      getNodeRequests(requestService).catch(() => []),
      getNodeTraces(requestService).catch(() => []),
      getStats().catch(() => ({})),
      getMetrics().catch(() => null),
      getSLOStatus().catch(() => null),
    ]).then(([reqs, trs, st, met, slo]) => {
      setRequests(reqs || []);
      setTraces(trs || []);
      setStats(st || {});
      setMetrics(met);
      setSloData(slo);
      setLoading(false);

      // Fetch AI explanation for most recent request
      if (reqs && reqs.length > 0) {
        api(`/api/explain/${reqs[0].id}`).then(setExplainData).catch(() => {});
      }
    });
  }, [node.id, requestService]);

  const inbound = edges.filter(e => e.target === node.id);
  const outbound = edges.filter(e => e.source === node.id);
  const nodeMap = new Map(nodes.map(n => [n.id, n]));

  const reqCount = stats[requestService] || stats[node.service] || 0;
  const errorCount = requests.filter(r => (r.status || r.status_code) >= 400).length;
  const errorRate = requests.length > 0 ? Math.round((errorCount / requests.length) * 100) : 0;
  const svcMetrics = metrics?.services?.[requestService] || metrics?.services?.[node.service];
  const p50 = svcMetrics?.p50 || 0;
  const p95 = svcMetrics?.p95 || 0;
  const p99 = svcMetrics?.p99 || 0;
  const typeStyle = getTypeStyle(node.type);

  // Health badge
  const sloService = sloData?.services?.[requestService] || sloData?.services?.[node.service];
  const healthColor = sloService
    ? sloService.healthy ? 'var(--success)' : sloService.burn_rate > 1 ? 'var(--error)' : 'var(--warning)'
    : 'var(--text-secondary)';

  const tabs: { key: Tab; label: string; count?: number }[] = [
    { key: 'overview', label: 'Overview' },
    { key: 'requests', label: 'Requests', count: requests.length },
    { key: 'traces', label: 'Traces', count: traces.length },
    { key: 'connections', label: 'Connections', count: inbound.length + outbound.length },
    { key: 'slo', label: 'SLO' },
    { key: 'blast', label: 'Blast Radius' },
    { key: 'profile', label: 'Profile' },
  ];

  return (
    <div style={S.panel}>
      {/* Service header */}
      <div style={S.nodeHeader}>
        <div style={{ ...S.typeIcon, background: typeStyle.bg, color: typeStyle.fg }}>
          {typeStyle.icon}
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <div style={S.nodeTitle}>{node.label}</div>
            <span style={{ ...S.healthDot, background: healthColor }} title={sloService?.healthy ? 'Healthy' : 'Degraded'} />
          </div>
          <div style={S.nodeMeta}>
            <span style={S.metaChip}>{node.service}</span>
            <span style={S.metaDot} />
            <span style={S.metaChip}>{node.type}</span>
            <span style={S.metaDot} />
            <span style={{ ...S.metaChip, color: 'var(--text-tertiary)' }}>{node.group}</span>
          </div>
        </div>
      </div>

      {/* Stat cards */}
      <div style={S.statGrid}>
        <StatCard label="Requests" value={reqCount.toLocaleString()} accent={undefined} />
        <StatCard label="Error Rate" value={`${errorRate}%`} accent={errorRate > 0 ? 'var(--error)' : undefined} />
        <StatCard label="P50" value={fmtDuration(p50)} accent={undefined} />
        <StatCard label="P95" value={fmtDuration(p95)} accent={p95 > 1000 ? 'var(--warning)' : undefined} />
      </div>

      {/* AI summary */}
      {explainData?.narrative && (
        <div style={S.aiSummary}>
          <div style={{ fontSize: 10, fontWeight: 600, color: '#8B5CF6', marginBottom: 4, textTransform: 'uppercase' as const, letterSpacing: 0.5 }}>
            AI Summary
          </div>
          <div style={{ fontSize: 12, lineHeight: 1.5, color: 'var(--text-primary)' }}>
            {explainData.narrative.split('\n').slice(0, 3).join(' ').slice(0, 200)}
            {explainData.narrative.length > 200 ? '...' : ''}
          </div>
        </div>
      )}

      {/* Tab bar */}
      <div style={S.tabBar}>
        {tabs.map(t => (
          <button
            key={t.key}
            onClick={() => {
              setTab(t.key);
              if (t.key === 'blast' && !blastData) {
                getBlastRadius(node.id).then(setBlastData).catch(() => {});
              }
              if (t.key === 'profile') {
                fetchProfiles(requestService).then(setProfileList).catch(() => {});
              }
            }}
            style={{
              ...S.tabBtn,
              color: tab === t.key ? 'var(--text-accent)' : 'var(--text-secondary)',
              fontWeight: tab === t.key ? 600 : 400,
            }}
          >
            {t.label}
            {t.count !== undefined && (
              <span style={{
                ...S.tabCount,
                background: tab === t.key ? 'var(--brand-teal)' : 'var(--bg-elevated)',
                color: tab === t.key ? 'var(--bg-primary)' : 'var(--text-secondary)',
              }}>{t.count}</span>
            )}
            {tab === t.key && <div style={S.tabIndicator} />}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div style={S.tabContent}>
        {loading ? <LoadingSkeleton /> : (
          <>
            {tab === 'overview' && (
              <OverviewContent
                p50={p50} p95={p95} p99={p99}
                inbound={inbound.length} outbound={outbound.length}
                requests={requests}
              />
            )}
            {tab === 'requests' && <RequestsContent requests={requests} />}
            {tab === 'traces' && <TracesContent traces={traces} />}
            {tab === 'connections' && (
              <ConnectionsContent inbound={inbound} outbound={outbound} nodeMap={nodeMap} onSelectNode={onSelectNode} />
            )}
            {tab === 'slo' && <SLOContent sloData={sloService} p50={p50} p95={p95} p99={p99} />}
            {tab === 'blast' && <BlastRadiusContent data={blastData} nodeMap={nodeMap} onSelectNode={onSelectNode} />}
            {tab === 'profile' && (
              <ProfileContent
                service={requestService}
                profileList={profileList}
                flamegraphData={flamegraphData}
                capturing={capturing}
                captureError={captureError}
                onCapture={async (type: string) => {
                  setCapturing(true);
                  setCaptureError(null);
                  setFlamegraphData('');
                  try {
                    const data = await captureProfile(requestService, type, type === 'cpu' ? '5s' : undefined);
                    setFlamegraphData(data);
                    fetchProfiles(requestService).then(setProfileList).catch(() => {});
                  } catch (e: any) {
                    setCaptureError(e?.message || 'Capture failed');
                  } finally {
                    setCapturing(false);
                  }
                }}
                onSelectProfile={async (id: string) => {
                  setFlamegraphData('');
                  try {
                    const data = await fetchProfileFlamegraph(id);
                    setFlamegraphData(data);
                  } catch {}
                }}
              />
            )}
          </>
        )}
      </div>
    </div>
  );
}

// --- Sub-components ---

function LoadingSkeleton() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {[1, 2, 3].map(i => (
        <div key={i} style={{
          height: 64, borderRadius: 10, background: 'var(--bg-tertiary)',
          animation: 'pulse 1.5s ease-in-out infinite',
        }} />
      ))}
    </div>
  );
}

function StatCard({ label, value, accent }: { label: string; value: string; accent?: string }) {
  return (
    <div style={S.statCard}>
      <div style={{ fontSize: 18, fontWeight: 700, color: accent || 'var(--text-primary)', lineHeight: 1 }}>
        {value}
      </div>
      <div style={{ fontSize: 10, color: 'var(--text-tertiary)', marginTop: 3, fontWeight: 500 }}>{label}</div>
    </div>
  );
}

function OverviewContent({ p50, p95, p99, inbound, outbound, requests }: {
  p50: number; p95: number; p99: number; inbound: number; outbound: number; requests: any[];
}) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      {/* Latency percentile bars */}
      <div style={S.card}>
        <div style={S.cardLabel}>Latency Percentiles</div>
        <div style={{ display: 'flex', gap: 16, marginTop: 8 }}>
          <LatencyBar label="P50" ms={p50} max={p99 || p50 || 1} color="#3B82F6" />
          <LatencyBar label="P95" ms={p95} max={p99 || p95 || 1} color="#F59E0B" />
          <LatencyBar label="P99" ms={p99} max={p99 || 1} color="#EF4444" />
        </div>
      </div>

      {/* Topology summary */}
      <div style={S.card}>
        <div style={S.cardLabel}>Topology</div>
        <div style={{ display: 'flex', gap: 16, marginTop: 8 }}>
          <div style={{ flex: 1, textAlign: 'center' }}>
            <div style={{ fontSize: 22, fontWeight: 700, color: 'var(--text-primary)' }}>{inbound}</div>
            <div style={{ fontSize: 11, color: 'var(--text-tertiary)', marginTop: 2 }}>Inbound</div>
          </div>
          <div style={{ width: 1, background: 'var(--border-default)' }} />
          <div style={{ flex: 1, textAlign: 'center' }}>
            <div style={{ fontSize: 22, fontWeight: 700, color: 'var(--text-primary)' }}>{outbound}</div>
            <div style={{ fontSize: 11, color: 'var(--text-tertiary)', marginTop: 2 }}>Outbound</div>
          </div>
        </div>
      </div>

      {/* Activity sparkline */}
      {requests.length > 0 && (
        <div style={S.card}>
          <div style={S.cardLabel}>Recent Activity</div>
          <ActivitySparkline requests={requests} />
        </div>
      )}
    </div>
  );
}

function LatencyBar({ label, ms, max, color }: { label: string; ms: number; max: number; color: string }) {
  const pct = max > 0 ? Math.min((ms / max) * 100, 100) : 0;
  return (
    <div style={{ flex: 1 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
        <span style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-secondary)' }}>{label}</span>
        <span style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--text-secondary)' }}>{fmtDuration(ms)}</span>
      </div>
      <div style={{ height: 6, background: 'var(--border-subtle)', borderRadius: 3 }}>
        <div style={{ height: '100%', width: `${pct}%`, background: color, borderRadius: 3, transition: 'width 0.3s ease' }} />
      </div>
    </div>
  );
}

function ActivitySparkline({ requests }: { requests: any[] }) {
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
          background: count > 0 ? 'var(--brand-blue)' : 'var(--border-subtle)',
          borderRadius: 2,
          opacity: count > 0 ? 0.7 : 0.4,
          transition: 'height 0.2s ease',
        }} />
      ))}
    </div>
  );
}

function RequestsContent({ requests }: { requests: any[] }) {
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
    return <EmptyState icon={'\u21C5'} message="No recent requests" />;
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      {requests.map((r: any) => {
        const isExpanded = expanded === r.id;
        const isError = (r.status || r.status_code) >= 400;
        return (
          <div key={r.id} style={{
            borderRadius: 8,
            border: `1px solid ${isExpanded ? 'var(--brand-blue, #097FF5)20' : 'var(--border-default)'}`,
            background: isExpanded ? 'var(--bg-active)' : 'var(--bg-tertiary)',
            overflow: 'hidden', transition: 'all 0.15s ease',
          }}>
            <div
              onClick={() => toggleExpand(r.id)}
              style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '8px 10px', cursor: 'pointer', fontSize: 12 }}
            >
              <span style={{
                display: 'inline-block', fontSize: 14, fontWeight: 700, color: 'var(--text-tertiary)',
                transition: 'transform 0.15s ease', transform: isExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
                width: 12, textAlign: 'center' as const,
              }}>{'\u203A'}</span>
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--text-tertiary)', minWidth: 55 }}>{fmtTime(r.timestamp)}</span>
              <span style={{ fontWeight: 500, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' as const }}>{r.action || r.path}</span>
              <StatusBadge code={r.status || r.status_code} />
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: isError ? 'var(--error)' : 'var(--text-tertiary)', minWidth: 40, textAlign: 'right' as const }}>
                {fmtDuration(r.latency_ms)}
              </span>
            </div>
            {isExpanded && detail && (
              <div style={{ borderTop: '1px solid var(--border-subtle)', padding: 10 }}>
                <div style={{ display: 'grid', gridTemplateColumns: '80px 1fr', gap: '4px 0', fontSize: 11 }}>
                  <span style={{ color: 'var(--text-tertiary)' }}>Service</span><span>{detail.service || r.service}</span>
                  <span style={{ color: 'var(--text-tertiary)' }}>Method</span><span style={{ fontFamily: 'var(--font-mono)', fontWeight: 600 }}>{detail.method || r.method}</span>
                  <span style={{ color: 'var(--text-tertiary)' }}>Path</span><span style={{ fontFamily: 'var(--font-mono)', fontSize: 10 }}>{detail.path || r.path}</span>
                  {detail.error && <><span style={{ color: 'var(--text-tertiary)' }}>Error</span><span style={{ color: 'var(--error)' }}>{detail.error}</span></>}
                </div>
                {detail.request_body && (
                  <div style={{ marginTop: 8 }}>
                    <div style={{ fontSize: 10, fontWeight: 600, color: 'var(--text-tertiary)', marginBottom: 4 }}>Request Body</div>
                    <div style={{ maxHeight: 150, overflow: 'auto', borderRadius: 6 }}>
                      <JsonView data={tryParse(detail.request_body)} />
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

function TracesContent({ traces }: { traces: any[] }) {
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
    return <EmptyState icon={'\u2B21'} message="No recent traces" />;
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      {traces.map((t: any) => {
        const isExpanded = expanded === t.trace_id;
        return (
          <div key={t.trace_id} style={{
            borderRadius: 8,
            border: `1px solid ${isExpanded ? 'var(--brand-blue, #097FF5)20' : 'var(--border-default)'}`,
            background: isExpanded ? 'var(--bg-active)' : 'var(--bg-tertiary)',
            overflow: 'hidden', transition: 'all 0.15s ease',
          }}>
            <div
              onClick={() => toggleExpand(t.trace_id)}
              style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '8px 10px', cursor: 'pointer', fontSize: 12 }}
            >
              <span style={{
                display: 'inline-block', fontSize: 14, fontWeight: 700, color: 'var(--text-tertiary)',
                transition: 'transform 0.15s ease', transform: isExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
                width: 12, textAlign: 'center' as const,
              }}>{'\u203A'}</span>
              <span style={{ fontWeight: 500, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' as const }}>
                {t.root_action || t.root_service}
              </span>
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--text-tertiary)' }}>{t.span_count} spans</span>
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--text-secondary)' }}>{fmtDuration(t.duration_ms)}</span>
              {t.has_error ? (
                <span style={{ padding: '1px 8px', borderRadius: 10, background: 'var(--error-50)', color: 'var(--error)', fontSize: 10, fontWeight: 600 }}>Error</span>
              ) : (
                <StatusBadge code={t.status_code} />
              )}
            </div>
            {isExpanded && (
              <WaterfallTimeline spans={timeline} totalMs={t.duration_ms} />
            )}
          </div>
        );
      })}
    </div>
  );
}

function WaterfallTimeline({ spans, totalMs }: { spans: any[]; totalMs: number }) {
  if (spans.length === 0) {
    return (
      <div style={{ padding: 16, borderTop: '1px solid var(--border-subtle)', textAlign: 'center' }}>
        <div style={S.spinner} />
      </div>
    );
  }

  const maxMs = totalMs || Math.max(...spans.map((s: any) => (s.start_offset_ms || 0) + (s.duration_ms || 0)), 1);

  return (
    <div style={{ borderTop: '1px solid var(--border-subtle)', padding: '10px 10px 6px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
        <span style={{ fontSize: 10, fontWeight: 600, color: 'var(--text-secondary)' }}>Waterfall</span>
        <span style={{ fontSize: 9, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)' }}>{spans.length} spans / {fmtDuration(totalMs)}</span>
      </div>
      {spans.map((s: any, i: number) => {
        const left = ((s.start_offset_ms || 0) / maxMs) * 100;
        const width = Math.max(((s.duration_ms || 0) / maxMs) * 100, 0.5);
        const indent = (s.depth || 0) * 12;
        const color = s.error ? 'var(--error)' : (s.status_code || 0) >= 400 ? 'var(--warning)' : spanColor(s.service || '');

        return (
          <div key={i} style={{ display: 'flex', alignItems: 'center', marginBottom: 3, fontSize: 10 }}>
            <div style={{
              width: 110, flexShrink: 0, paddingLeft: indent,
              overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' as const,
              color: 'var(--text-secondary)',
            }}>
              <span style={{ display: 'inline-block', width: 5, height: 5, borderRadius: '50%', background: color, marginRight: 4, verticalAlign: 'middle' }} />
              {s.action || s.service}
            </div>
            <div style={{ flex: 1, height: 14, background: 'var(--border-subtle)', borderRadius: 3, position: 'relative', overflow: 'hidden' }}>
              <div style={{
                position: 'absolute', left: `${left}%`, width: `${width}%`,
                height: '100%', background: color, borderRadius: 3, opacity: 0.85,
              }}>
                {width > 10 && (
                  <span style={{ position: 'absolute', left: 3, top: 1, fontSize: 8, color: 'white', fontWeight: 600, fontFamily: 'var(--font-mono)' }}>
                    {fmtDuration(s.duration_ms)}
                  </span>
                )}
              </div>
            </div>
            {width <= 10 && (
              <div style={{ width: 40, textAlign: 'right' as const, flexShrink: 0, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)', fontSize: 9 }}>
                {fmtDuration(s.duration_ms)}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

function ConnectionsContent({ inbound, outbound, nodeMap, onSelectNode }: {
  inbound: TopoEdge[]; outbound: TopoEdge[];
  nodeMap: Map<string, TopoNode>; onSelectNode: (node: TopoNode) => void;
}) {
  if (inbound.length === 0 && outbound.length === 0) {
    return <EmptyState icon={'\u2194'} message="No connections" />;
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      {inbound.length > 0 && (
        <ConnectionGroup label="Inbound" icon={'\u2190'} edges={inbound} direction="in" nodeMap={nodeMap} onSelect={onSelectNode} />
      )}
      {outbound.length > 0 && (
        <ConnectionGroup label="Outbound" icon={'\u2192'} edges={outbound} direction="out" nodeMap={nodeMap} onSelect={onSelectNode} />
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
      <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-secondary)', marginBottom: 6, display: 'flex', alignItems: 'center', gap: 6 }}>
        <span>{icon}</span> {label}
        <span style={{ fontWeight: 400, color: 'var(--text-tertiary)' }}>({edges.length})</span>
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
        {edges.map((edge, i) => {
          const peerId = direction === 'in' ? edge.source : edge.target;
          const peer = nodeMap.get(peerId);
          const peerType = peer ? getTypeStyle(peer.type) : DEFAULT_TYPE_STYLE;
          const edgeColor = EDGE_TYPE_COLORS[edge.type] || 'var(--text-secondary)';

          return (
            <div
              key={i}
              onClick={() => peer && onSelect(peer)}
              style={{
                display: 'flex', alignItems: 'center', gap: 8, padding: '7px 10px',
                borderRadius: 8, border: '1px solid var(--border-default)',
                cursor: peer ? 'pointer' : 'default', background: 'var(--bg-tertiary)',
                transition: 'all 0.1s ease',
              }}
            >
              <div style={{
                width: 24, height: 24, borderRadius: 6, display: 'flex',
                alignItems: 'center', justifyContent: 'center',
                fontSize: 8, fontWeight: 700, fontFamily: 'var(--font-mono)',
                background: peerType.bg, color: peerType.fg, textTransform: 'uppercase' as const,
              }}>{peerType.icon}</div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 11, fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' as const }}>
                  {peer?.label || peerId}
                </div>
                <div style={{ fontSize: 9, color: 'var(--text-tertiary)', display: 'flex', gap: 6, marginTop: 1 }}>
                  <span style={{ color: edgeColor, fontWeight: 600 }}>{edge.type}</span>
                </div>
              </div>
              <div style={{ textAlign: 'right' as const, flexShrink: 0 }}>
                <div style={{ fontSize: 10, fontFamily: 'var(--font-mono)', color: 'var(--text-secondary)' }}>
                  {edge.avg_latency_ms ? fmtDuration(edge.avg_latency_ms) : '\u2014'}
                </div>
                {(edge.call_count || 0) > 0 && (
                  <div style={{ fontSize: 9, color: 'var(--text-tertiary)' }}>{edge.call_count} calls</div>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function SLOContent({ sloData, p50, p95, p99 }: {
  sloData: any; p50: number; p95: number; p99: number;
}) {
  if (!sloData) {
    return <EmptyState icon={'\u2691'} message="No SLO data available" />;
  }

  const burnRate = sloData.burn_rate ?? 0;
  const burnColor = burnRate > 1 ? 'var(--error)' : burnRate > 0.5 ? 'var(--warning)' : 'var(--success)';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {/* Burn rate */}
      <div style={S.card}>
        <div style={S.cardLabel}>Burn Rate</div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginTop: 8 }}>
          <div style={{ fontSize: 32, fontWeight: 700, color: burnColor }}>{burnRate.toFixed(2)}x</div>
          <div style={{ fontSize: 11, color: 'var(--text-secondary)', lineHeight: 1.4 }}>
            {burnRate > 1 ? 'Exceeding error budget' : burnRate > 0.5 ? 'Elevated burn' : 'Within budget'}
          </div>
        </div>
      </div>

      {/* Targets vs actuals */}
      <div style={S.card}>
        <div style={S.cardLabel}>Latency: Targets vs Actuals</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginTop: 8 }}>
          <TargetRow label="P50" actual={p50} target={sloData.p50_target} />
          <TargetRow label="P95" actual={p95} target={sloData.p95_target} />
          <TargetRow label="P99" actual={p99} target={sloData.p99_target} />
        </div>
      </div>

      {/* Availability */}
      {sloData.availability !== undefined && (
        <div style={S.card}>
          <div style={S.cardLabel}>Availability</div>
          <div style={{ fontSize: 28, fontWeight: 700, color: sloData.availability >= 99.9 ? 'var(--success)' : 'var(--warning)', marginTop: 6 }}>
            {sloData.availability.toFixed(2)}%
          </div>
        </div>
      )}
    </div>
  );
}

function TargetRow({ label, actual, target }: { label: string; actual: number; target?: number }) {
  const isOver = target !== undefined && actual > target;
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
      <span style={{ width: 30, fontSize: 11, fontWeight: 600, color: 'var(--text-secondary)' }}>{label}</span>
      <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: isOver ? 'var(--error)' : 'var(--text-primary)', fontWeight: 600 }}>
        {fmtDuration(actual)}
      </span>
      {target !== undefined && (
        <>
          <span style={{ fontSize: 10, color: 'var(--text-tertiary)' }}>/ {fmtDuration(target)}</span>
          <span style={{ fontSize: 9, fontWeight: 600, color: isOver ? 'var(--error)' : 'var(--success)' }}>
            {isOver ? 'OVER' : 'OK'}
          </span>
        </>
      )}
    </div>
  );
}

function BlastRadiusContent({ data, nodeMap, onSelectNode }: {
  data: any; nodeMap: Map<string, TopoNode>; onSelectNode: (n: TopoNode) => void;
}) {
  if (!data) {
    return (
      <div style={{ textAlign: 'center', padding: 20 }}>
        <div style={S.spinner} />
      </div>
    );
  }

  const upstream: string[] = data.upstream || [];
  const downstream: string[] = data.downstream || [];

  if (upstream.length === 0 && downstream.length === 0) {
    return <EmptyState icon={'\u26A1'} message="No blast radius data" />;
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      {upstream.length > 0 && (
        <ImpactGroup label="Upstream (affected by)" nodes={upstream} nodeMap={nodeMap} onSelect={onSelectNode} color="#F59E0B" />
      )}
      {downstream.length > 0 && (
        <ImpactGroup label="Downstream (will affect)" nodes={downstream} nodeMap={nodeMap} onSelect={onSelectNode} color="#EF4444" />
      )}
    </div>
  );
}

function ImpactGroup({ label, nodes, nodeMap, onSelect, color }: {
  label: string; nodes: string[]; nodeMap: Map<string, TopoNode>;
  onSelect: (n: TopoNode) => void; color: string;
}) {
  return (
    <div>
      <div style={{ fontSize: 11, fontWeight: 600, color, marginBottom: 6 }}>
        {label} ({nodes.length})
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
        {nodes.map((nodeId: string) => {
          const node = nodeMap.get(nodeId);
          const style = node ? getTypeStyle(node.type) : DEFAULT_TYPE_STYLE;
          return (
            <div
              key={nodeId}
              onClick={() => node && onSelect(node)}
              style={{
                display: 'flex', alignItems: 'center', gap: 8, padding: '6px 10px',
                borderRadius: 6, border: '1px solid var(--border-default)', cursor: node ? 'pointer' : 'default',
                background: 'var(--bg-tertiary)', fontSize: 11,
              }}
            >
              <div style={{
                width: 20, height: 20, borderRadius: 4, display: 'flex',
                alignItems: 'center', justifyContent: 'center',
                fontSize: 7, fontWeight: 700, fontFamily: 'var(--font-mono)',
                background: style.bg, color: style.fg, textTransform: 'uppercase' as const,
              }}>{style.icon}</div>
              <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' as const }}>
                {node?.label || nodeId}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function EmptyState({ icon, message }: { icon: string; message: string }) {
  return (
    <div style={{ textAlign: 'center', padding: '32px 16px' }}>
      <div style={{ fontSize: 28, marginBottom: 6, opacity: 0.3 }}>{icon}</div>
      <div style={{ fontSize: 12, color: 'var(--text-tertiary)' }}>{message}</div>
    </div>
  );
}

function ProfileContent({ service, profileList, flamegraphData, capturing, captureError, onCapture, onSelectProfile }: {
  service: string;
  profileList: any[];
  flamegraphData: string;
  capturing: boolean;
  captureError: string | null;
  onCapture: (type: string) => void;
  onSelectProfile: (id: string) => void;
}) {
  const profileTypes = [
    { type: 'cpu', label: 'CPU Profile (5s)' },
    { type: 'heap', label: 'Heap Snapshot' },
    { type: 'goroutine', label: 'Goroutines' },
  ];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      {/* Capture buttons */}
      <div style={S.card}>
        <div style={S.cardLabel}>Capture Profile</div>
        <div style={{ display: 'flex', gap: 8, marginTop: 10, flexWrap: 'wrap' as const }}>
          {profileTypes.map(({ type, label }) => (
            <button
              key={type}
              disabled={capturing}
              onClick={() => onCapture(type)}
              style={{
                padding: '6px 12px', borderRadius: 6, border: '1px solid var(--border-default)',
                background: capturing ? 'var(--bg-elevated)' : 'var(--bg-tertiary)', cursor: capturing ? 'not-allowed' : 'pointer',
                fontSize: 12, fontWeight: 500, color: 'var(--text-primary)',
                display: 'flex', alignItems: 'center', gap: 6,
                opacity: capturing ? 0.6 : 1,
              }}
            >
              {capturing && type === 'cpu' ? (
                <span style={S.spinner} />
              ) : null}
              {label}
            </button>
          ))}
        </div>
        {capturing && (
          <div style={{ marginTop: 10, display: 'flex', alignItems: 'center', gap: 8, fontSize: 12, color: 'var(--text-secondary)' }}>
            <span style={S.spinner} />
            Capturing...
          </div>
        )}
        {captureError && (
          <div style={{ marginTop: 8, padding: '6px 10px', borderRadius: 6, background: 'var(--error-50)', color: 'var(--error)', fontSize: 12 }}>
            {captureError}
          </div>
        )}
      </div>

      {/* FlameGraph */}
      {flamegraphData && (
        <div style={S.card}>
          <div style={S.cardLabel}>Flame Graph</div>
          <div style={{ marginTop: 10 }}>
            <FlameGraph data={flamegraphData} />
          </div>
        </div>
      )}

      {/* Recent profiles list */}
      <div style={S.card}>
        <div style={S.cardLabel}>Recent Profiles</div>
        {profileList.length === 0 ? (
          <div style={{ padding: '16px 0', textAlign: 'center', fontSize: 12, color: 'var(--text-tertiary)', opacity: 0.7 }}>
            No profiles captured yet.
          </div>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 11, marginTop: 8 }}>
            <thead>
              <tr style={{ borderBottom: '1px solid var(--border-default)' }}>
                {['Type', 'Captured At', 'Size'].map(h => (
                  <th key={h} style={{ padding: '5px 6px', textAlign: h === 'Size' ? 'right' : 'left', fontWeight: 500, color: 'var(--text-secondary)' }}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {profileList.map((p: any) => (
                <tr
                  key={p.id}
                  onClick={() => onSelectProfile(p.id)}
                  style={{ borderBottom: '1px solid var(--border-subtle)', cursor: 'pointer' }}
                >
                  <td style={{ padding: '6px', fontWeight: 600, color: 'var(--brand-blue)' }}>{p.type}</td>
                  <td style={{ padding: '6px', fontFamily: 'var(--font-mono)', color: 'var(--text-secondary)' }}>
                    {new Date(p.captured_at || p.timestamp || p.created_at).toLocaleString()}
                  </td>
                  <td style={{ padding: '6px', textAlign: 'right', fontFamily: 'var(--font-mono)', color: 'var(--text-secondary)' }}>
                    {p.size_bytes ? `${(p.size_bytes / 1024).toFixed(1)} KB` : '\u2014'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}

// --- Styles ---

const S = {
  panel: {
    display: 'flex', flexDirection: 'column' as const, height: '100%', overflow: 'hidden',
  },
  nodeHeader: {
    display: 'flex', alignItems: 'center', gap: 10, padding: '14px 16px 10px',
    borderBottom: '1px solid var(--border-subtle)', flexShrink: 0,
  },
  typeIcon: {
    width: 36, height: 36, borderRadius: 8, display: 'flex',
    alignItems: 'center', justifyContent: 'center',
    fontSize: 11, fontWeight: 700, fontFamily: 'var(--font-mono)',
    flexShrink: 0, textTransform: 'uppercase' as const, letterSpacing: 0.5,
  },
  healthDot: {
    width: 8, height: 8, borderRadius: '50%', flexShrink: 0,
  },
  nodeTitle: { fontSize: 14, fontWeight: 700, color: 'var(--text-primary)', lineHeight: 1.2 },
  nodeMeta: {
    display: 'flex', alignItems: 'center', gap: 5, marginTop: 3, fontSize: 11, color: 'var(--text-secondary)',
  },
  metaChip: { fontSize: 10, fontWeight: 500 },
  metaDot: { width: 3, height: 3, borderRadius: '50%', background: 'var(--text-tertiary)', flexShrink: 0 },
  statGrid: {
    display: 'grid', gridTemplateColumns: '1fr 1fr 1fr 1fr', gap: 8,
    padding: '10px 16px', flexShrink: 0,
  },
  statCard: {
    padding: '8px 6px', borderRadius: 8, border: '1px solid var(--border-default)',
    background: 'var(--bg-tertiary)', textAlign: 'center' as const,
  },
  aiSummary: {
    margin: '0 16px 8px', padding: '10px 12px', borderRadius: 8,
    background: 'rgba(139, 92, 246, 0.1)', border: '1px solid rgba(139, 92, 246, 0.2)',
  },
  tabBar: {
    display: 'flex', gap: 0, borderBottom: '1px solid var(--border-default)',
    paddingLeft: 16, paddingRight: 16, flexShrink: 0, overflowX: 'auto' as const,
  },
  tabBtn: {
    position: 'relative' as const, padding: '7px 10px 9px', fontSize: 11,
    background: 'none', border: 'none', cursor: 'pointer',
    whiteSpace: 'nowrap' as const, display: 'flex', alignItems: 'center', gap: 5,
  },
  tabCount: {
    display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
    minWidth: 16, height: 16, borderRadius: 8, fontSize: 9, fontWeight: 600,
    padding: '0 4px', lineHeight: 1,
  },
  tabIndicator: {
    position: 'absolute' as const, bottom: -1, left: 10, right: 10, height: 2,
    background: 'var(--brand-teal)', borderRadius: '2px 2px 0 0',
  },
  tabContent: {
    flex: 1, overflowY: 'auto' as const, padding: '12px 16px',
  },
  card: {
    padding: 12, borderRadius: 10, border: '1px solid var(--border-subtle)', background: 'var(--bg-tertiary)',
  },
  cardLabel: {
    fontSize: 10, fontWeight: 600, color: 'var(--text-tertiary)', textTransform: 'uppercase' as const, letterSpacing: 0.5,
  },
  spinner: {
    width: 20, height: 20, border: '2px solid var(--border-default)', borderTopColor: 'var(--brand-teal)',
    borderRadius: '50%', animation: 'spin 0.6s linear infinite', display: 'inline-block',
  },
};
