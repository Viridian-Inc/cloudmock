import { useState, useEffect } from 'preact/hooks';
import { Drawer } from './Drawer';
import { StatusBadge } from './StatusBadge';
import { JsonView } from './JsonView';
import { getNodeRequests, getNodeTraces, getNodeResources, getStats, getMetrics, api } from '../api';
import { fmtTime, fmtDuration } from '../utils';

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

  const tabs: { key: Tab; label: string }[] = [
    { key: 'overview', label: 'Overview' },
    { key: 'requests', label: `Requests (${requests.length})` },
    { key: 'traces', label: `Traces (${traces.length})` },
    { key: 'connections', label: `Connections (${inbound.length + outbound.length})` },
    { key: 'resource', label: 'Resource' },
  ];

  return (
    <Drawer title={`${node.label}`} onClose={onClose}>
      <div style={{ marginBottom: 12 }}>
        <span class="badge" style={{ marginRight: 6, background: '#E2E8F0', color: '#475569', fontSize: 11 }}>{node.type}</span>
        <span class="badge" style={{ marginRight: 6, background: '#E2E8F0', color: '#475569', fontSize: 11 }}>{node.service}</span>
        <span class="badge" style={{ background: '#E2E8F0', color: '#475569', fontSize: 11 }}>{node.group}</span>
      </div>

      <div class="tabs" style={{ marginBottom: 16, display: 'flex', gap: 0, borderBottom: '1px solid #E2E8F0' }}>
        {tabs.map(t => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            style={{
              padding: '8px 12px',
              fontSize: 12,
              fontWeight: tab === t.key ? 600 : 400,
              color: tab === t.key ? '#3B82F6' : '#64748B',
              background: 'none',
              border: 'none',
              borderBottom: tab === t.key ? '2px solid #3B82F6' : '2px solid transparent',
              cursor: 'pointer',
              whiteSpace: 'nowrap',
            }}
          >
            {t.label}
          </button>
        ))}
      </div>

      {loading ? (
        <div style={{ textAlign: 'center', padding: 32, color: '#94A3B8' }}>Loading...</div>
      ) : (
        <>
          {tab === 'overview' && (
            <OverviewTab reqCount={reqCount} errorRate={errorRate} avgLatency={avgLatency} inbound={inbound.length} outbound={outbound.length} />
          )}
          {tab === 'requests' && <RequestsTab requests={requests} />}
          {tab === 'traces' && <TracesTab traces={traces} />}
          {tab === 'connections' && (
            <ConnectionsTab inbound={inbound} outbound={outbound} nodeMap={nodeMap} onSelectNode={onSelectNode} />
          )}
          {tab === 'resource' && <ResourceTab resources={resources} />}
        </>
      )}
    </Drawer>
  );
}

function OverviewTab({ reqCount, errorRate, avgLatency, inbound, outbound }: {
  reqCount: number; errorRate: number; avgLatency: number; inbound: number; outbound: number;
}) {
  const stats = [
    { label: 'Requests', value: reqCount.toLocaleString() },
    { label: 'Error Rate', value: `${errorRate}%`, color: errorRate > 0 ? '#EF4444' : undefined },
    { label: 'Avg Latency', value: fmtDuration(avgLatency) },
    { label: 'Inbound', value: inbound },
    { label: 'Outbound', value: outbound },
  ];

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
      {stats.map(s => (
        <div key={s.label} style={{ padding: 12, background: '#F8FAFC', borderRadius: 8, border: '1px solid #E2E8F0' }}>
          <div style={{ fontSize: 11, color: '#94A3B8', marginBottom: 4 }}>{s.label}</div>
          <div style={{ fontSize: 20, fontWeight: 700, color: s.color || '#1E293B' }}>{s.value}</div>
        </div>
      ))}
    </div>
  );
}

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
    return <div style={{ textAlign: 'center', padding: 32, color: '#94A3B8' }}>No recent requests</div>;
  }

  return (
    <div style={{ overflow: 'auto', maxHeight: 500 }}>
      <table class="table" style={{ fontSize: 12, width: '100%' }}>
        <thead>
          <tr>
            <th>Time</th>
            <th>Action</th>
            <th>Status</th>
            <th>Latency</th>
          </tr>
        </thead>
        <tbody>
          {requests.map((r: any) => (
            <>
              <tr
                key={r.id}
                style={{ cursor: 'pointer', background: expanded === r.id ? '#F1F5F9' : undefined }}
                onClick={() => toggleExpand(r.id)}
              >
                <td style={{ whiteSpace: 'nowrap' }}>{fmtTime(r.timestamp)}</td>
                <td>{r.action}</td>
                <td><StatusBadge code={r.status} /></td>
                <td>{fmtDuration(r.latency_ms)}</td>
              </tr>
              {expanded === r.id && (
                <tr key={`${r.id}-detail`}>
                  <td colSpan={4} style={{ padding: 0 }}>
                    <RequestInlineDetail req={detail || r} />
                  </td>
                </tr>
              )}
            </>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function RequestInlineDetail({ req }: { req: any }) {
  const [tab, setTab] = useState<'overview' | 'request' | 'response'>('overview');

  const tabStyle = (active: boolean) => ({
    padding: '4px 10px',
    fontSize: 11,
    fontWeight: active ? 600 : 400,
    color: active ? '#3B82F6' : '#64748B',
    background: active ? '#EFF6FF' : 'none',
    border: '1px solid ' + (active ? '#BFDBFE' : '#E2E8F0'),
    borderRadius: 4,
    cursor: 'pointer' as const,
  });

  return (
    <div style={{ padding: 12, background: '#F8FAFC', borderTop: '1px solid #E2E8F0' }}>
      <div style={{ display: 'flex', gap: 6, marginBottom: 10 }}>
        <button style={tabStyle(tab === 'overview')} onClick={() => setTab('overview')}>Overview</button>
        <button style={tabStyle(tab === 'request')} onClick={() => setTab('request')}>Request</button>
        <button style={tabStyle(tab === 'response')} onClick={() => setTab('response')}>Response</button>
      </div>

      {tab === 'overview' && (
        <div style={{ fontSize: 12, display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '6px 16px' }}>
          <div><span style={{ color: '#94A3B8' }}>Service:</span> {req.service}</div>
          <div><span style={{ color: '#94A3B8' }}>Action:</span> {req.action}</div>
          <div><span style={{ color: '#94A3B8' }}>Method:</span> {req.method}</div>
          <div><span style={{ color: '#94A3B8' }}>Path:</span> <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}>{req.path}</span></div>
          <div><span style={{ color: '#94A3B8' }}>Status:</span> <StatusBadge code={req.status} /></div>
          <div><span style={{ color: '#94A3B8' }}>Latency:</span> {fmtDuration(req.latency_ms)}</div>
          {req.error && <div style={{ gridColumn: '1/3', color: '#EF4444' }}>Error: {req.error}</div>}
        </div>
      )}

      {tab === 'request' && (
        <div style={{ maxHeight: 200, overflow: 'auto' }}>
          {req.request_body ? <JsonView data={tryParse(req.request_body)} /> : <span style={{ color: '#94A3B8' }}>No request body</span>}
        </div>
      )}

      {tab === 'response' && (
        <div style={{ maxHeight: 200, overflow: 'auto' }}>
          {req.response_body ? <JsonView data={tryParse(req.response_body)} /> : <span style={{ color: '#94A3B8' }}>No response body</span>}
        </div>
      )}
    </div>
  );
}

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
    return <div style={{ textAlign: 'center', padding: 32, color: '#94A3B8' }}>No recent traces</div>;
  }

  return (
    <div style={{ overflow: 'auto', maxHeight: 500 }}>
      <table class="table" style={{ fontSize: 12, width: '100%' }}>
        <thead>
          <tr>
            <th>Root Action</th>
            <th>Spans</th>
            <th>Duration</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {traces.map((t: any) => (
            <>
              <tr
                key={t.trace_id}
                style={{ cursor: 'pointer', background: expanded === t.trace_id ? '#F1F5F9' : undefined }}
                onClick={() => toggleExpand(t.trace_id)}
              >
                <td>{t.root_action || t.root_service}</td>
                <td>{t.span_count}</td>
                <td>{fmtDuration(t.duration_ms)}</td>
                <td>
                  {t.has_error ? (
                    <span style={{ color: '#EF4444', fontWeight: 600 }}>Error</span>
                  ) : (
                    <StatusBadge code={t.status_code} />
                  )}
                </td>
              </tr>
              {expanded === t.trace_id && (
                <tr key={`${t.trace_id}-detail`}>
                  <td colSpan={4} style={{ padding: 0 }}>
                    <TraceInlineDetail spans={timeline} totalMs={t.duration_ms} />
                  </td>
                </tr>
              )}
            </>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function TraceInlineDetail({ spans, totalMs }: { spans: any[]; totalMs: number }) {
  if (spans.length === 0) {
    return <div style={{ padding: 12, color: '#94A3B8', fontSize: 12 }}>Loading timeline...</div>;
  }

  const maxMs = totalMs || Math.max(...spans.map(s => (s.start_offset_ms || 0) + (s.duration_ms || 0)), 1);

  return (
    <div style={{ padding: 12, background: '#F8FAFC', borderTop: '1px solid #E2E8F0' }}>
      <div style={{ fontSize: 11, color: '#94A3B8', marginBottom: 8 }}>Waterfall ({spans.length} spans, {fmtDuration(totalMs)})</div>
      {spans.map((s: any, i: number) => {
        const left = ((s.start_offset_ms || 0) / maxMs) * 100;
        const width = Math.max(((s.duration_ms || 0) / maxMs) * 100, 1);
        const indent = (s.depth || 0) * 12;
        const color = s.error ? '#EF4444' : s.status_code >= 400 ? '#F59E0B' : '#3B82F6';

        return (
          <div key={i} style={{ display: 'flex', alignItems: 'center', marginBottom: 3, fontSize: 11 }}>
            <div style={{ width: 120, flexShrink: 0, paddingLeft: indent, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {s.service}/{s.action}
            </div>
            <div style={{ flex: 1, height: 14, background: '#E2E8F0', borderRadius: 2, position: 'relative' }}>
              <div style={{
                position: 'absolute',
                left: `${left}%`,
                width: `${width}%`,
                height: '100%',
                background: color,
                borderRadius: 2,
                opacity: 0.8,
              }} />
            </div>
            <div style={{ width: 50, textAlign: 'right', flexShrink: 0, color: '#64748B', fontSize: 10 }}>
              {fmtDuration(s.duration_ms)}
            </div>
          </div>
        );
      })}
    </div>
  );
}

function ConnectionsTab({ inbound, outbound, nodeMap, onSelectNode }: {
  inbound: TopoEdge[];
  outbound: TopoEdge[];
  nodeMap: Map<string, TopoNode>;
  onSelectNode: (node: TopoNode) => void;
}) {
  function EdgeRow({ edge, direction }: { edge: TopoEdge; direction: 'in' | 'out' }) {
    const peerId = direction === 'in' ? edge.source : edge.target;
    const peer = nodeMap.get(peerId);
    return (
      <tr
        style={{ cursor: peer ? 'pointer' : 'default' }}
        onClick={() => peer && onSelectNode(peer)}
      >
        <td style={{ fontWeight: 500 }}>{peer?.label || peerId}</td>
        <td>
          <span class="badge" style={{ fontSize: 10, background: '#E2E8F0', color: '#475569' }}>
            {edge.type}
          </span>
        </td>
        <td>{edge.avg_latency_ms ? fmtDuration(edge.avg_latency_ms) : '—'}</td>
        <td>{edge.call_count || 0}</td>
      </tr>
    );
  }

  return (
    <div>
      {inbound.length > 0 && (
        <>
          <h4 style={{ fontSize: 13, fontWeight: 600, marginBottom: 8, color: '#475569' }}>
            Inbound ({inbound.length})
          </h4>
          <table class="table" style={{ fontSize: 12, width: '100%', marginBottom: 16 }}>
            <thead>
              <tr><th>Source</th><th>Type</th><th>Latency</th><th>Calls</th></tr>
            </thead>
            <tbody>
              {inbound.map((e, i) => <EdgeRow key={i} edge={e} direction="in" />)}
            </tbody>
          </table>
        </>
      )}

      {outbound.length > 0 && (
        <>
          <h4 style={{ fontSize: 13, fontWeight: 600, marginBottom: 8, color: '#475569' }}>
            Outbound ({outbound.length})
          </h4>
          <table class="table" style={{ fontSize: 12, width: '100%' }}>
            <thead>
              <tr><th>Target</th><th>Type</th><th>Latency</th><th>Calls</th></tr>
            </thead>
            <tbody>
              {outbound.map((e, i) => <EdgeRow key={i} edge={e} direction="out" />)}
            </tbody>
          </table>
        </>
      )}

      {inbound.length === 0 && outbound.length === 0 && (
        <div style={{ textAlign: 'center', padding: 32, color: '#94A3B8' }}>No connections</div>
      )}
    </div>
  );
}

function tryParse(s: string): any {
  try { return JSON.parse(s); } catch { return s; }
}

function ResourceTab({ resources }: { resources: any }) {
  if (!resources || (Array.isArray(resources) && resources.length === 0)) {
    return <div style={{ textAlign: 'center', padding: 32, color: '#94A3B8' }}>No resource details available</div>;
  }

  return (
    <div style={{ overflow: 'auto', maxHeight: 500 }}>
      <JsonView data={resources} />
    </div>
  );
}
