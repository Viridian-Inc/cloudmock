import { useState, useEffect } from 'preact/hooks';
import { Drawer } from './Drawer';
import { StatusBadge } from './StatusBadge';
import { JsonView } from './JsonView';
import { getNodeRequests, getNodeTraces, getNodeResources, getStats, getMetrics } from '../api';
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
  if (requests.length === 0) {
    return <div style={{ textAlign: 'center', padding: 32, color: '#94A3B8' }}>No recent requests</div>;
  }

  return (
    <div style={{ overflow: 'auto', maxHeight: 400 }}>
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
            <tr
              key={r.id}
              style={{ cursor: 'pointer' }}
              onClick={() => { location.hash = `/requests/${r.id}`; }}
            >
              <td style={{ whiteSpace: 'nowrap' }}>{fmtTime(r.timestamp)}</td>
              <td>{r.action}</td>
              <td><StatusBadge code={r.status} /></td>
              <td>{fmtDuration(r.latency_ms)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function TracesTab({ traces }: { traces: any[] }) {
  if (traces.length === 0) {
    return <div style={{ textAlign: 'center', padding: 32, color: '#94A3B8' }}>No recent traces</div>;
  }

  return (
    <div style={{ overflow: 'auto', maxHeight: 400 }}>
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
            <tr
              key={t.trace_id}
              style={{ cursor: 'pointer' }}
              onClick={() => { location.hash = `/traces`; }}
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
          ))}
        </tbody>
      </table>
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
