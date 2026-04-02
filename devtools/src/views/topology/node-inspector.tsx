import { useState, useMemo } from 'preact/hooks';
import type { TopoNode, TopoEdge } from './index';
import type { ServiceMetrics, DeployEvent } from '../../lib/health';
import type { IncidentInfo } from '../../lib/types';
import { computeHealthState, getBlastRadius } from '../../lib/health';
import { getRoutingConfig, type DomainConfig } from '../../lib/domains';
import { getAdminBase } from '../../lib/api';
import { Sparkline } from './sparkline';
import { EndpointsTab, type ManifestService } from './endpoints-tab';
import { DeployDetail } from './deploy-detail';
import { RequestTracePanel } from './request-trace-panel';

type TabId = 'metrics' | 'endpoints' | 'deploys' | 'incidents' | 'connections';

interface NodeInspectorProps {
  node: TopoNode | null;
  edges: TopoEdge[];
  allNodes: TopoNode[];
  metrics: ServiceMetrics[];
  deploys: DeployEvent[];
  incidents: IncidentInfo[];
  domainConfig: DomainConfig | null;
  metricsHistory: Map<string, number[]>;
  manifest: ManifestService[] | null;
}

const TABS: { id: TabId; label: string }[] = [
  { id: 'metrics', label: 'Metrics' },
  { id: 'endpoints', label: 'Endpoints' },
  { id: 'deploys', label: 'Deploys' },
  { id: 'incidents', label: 'Incidents' },
  { id: 'connections', label: 'Connections' },
];

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const secs = Math.floor(diff / 1000);
  if (secs < 60) return `${secs}s ago`;
  const mins = Math.floor(secs / 60);
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  return `${days}d ago`;
}

function getServiceKey(node: TopoNode): string {
  // For IaC nodes (external/plugin), use the ID suffix which is more specific
  // e.g., "external:bff-service" → "bff-service", not "external"
  if (node.service === 'external' || node.service === 'plugin') {
    const colonIdx = node.id.indexOf(':');
    if (colonIdx >= 0) {
      return node.id.substring(colonIdx + 1); // "bff-service"
    }
    return node.label; // fallback to label
  }
  // For AWS service nodes, use the service field
  return node.service || node.id.replace(/^svc:|^ms:/, '');
}

export function NodeInspector({
  node, edges, allNodes, metrics, deploys, incidents, domainConfig, metricsHistory, manifest,
}: NodeInspectorProps) {
  const [activeTab, setActiveTab] = useState<TabId>('metrics');
  const [showRequestTrace, setShowRequestTrace] = useState(false);

  if (!node) {
    return (
      <div class="node-inspector node-inspector-empty">
        <div class="node-inspector-placeholder">Click a node to inspect</div>
      </div>
    );
  }

  const svcKey = getServiceKey(node);

  // When "View Requests" is clicked, show the request trace panel inline
  if (showRequestTrace) {
    return (
      <div class="node-inspector">
        <RequestTracePanel
          node={node}
          edges={edges}
          allNodes={allNodes}
          metrics={metrics}
          incidents={incidents}
          onClose={() => setShowRequestTrace(false)}
        />
      </div>
    );
  }

  return (
    <div class="node-inspector">
      <div class="node-inspector-header">
        <div class="node-inspector-title">{node.label}</div>
        <div class="node-inspector-subtitle">{node.group} · {node.type}</div>
      </div>
      <div class="inspector-tab-bar">
        {TABS.map((t) => (
          <button
            key={t.id}
            class={`inspector-tab ${activeTab === t.id ? 'active' : ''}`}
            onClick={() => setActiveTab(t.id)}
          >
            {t.label}
          </button>
        ))}
      </div>
      <div class="inspector-tab-content">
        {activeTab === 'metrics' && (
          <MetricsTab
            svcKey={svcKey}
            node={node}
            edges={edges}
            allNodes={allNodes}
            metrics={metrics}
            incidents={incidents}
            domainConfig={domainConfig}
            metricsHistory={metricsHistory}
            onViewRequests={() => setShowRequestTrace(true)}
          />
        )}
        {activeTab === 'endpoints' && <EndpointsTab svcKey={svcKey} manifest={manifest} />}
        {activeTab === 'deploys' && <DeploysTab svcKey={svcKey} deploys={deploys} />}
        {activeTab === 'incidents' && <IncidentsTab svcKey={svcKey} incidents={incidents} />}
        {activeTab === 'connections' && (
          <ConnectionsTab node={node} edges={edges} allNodes={allNodes} />
        )}
      </div>
    </div>
  );
}

/* ---------- Metrics Tab ---------- */

interface MetricsTabProps {
  svcKey: string;
  node: TopoNode;
  edges: TopoEdge[];
  allNodes: TopoNode[];
  metrics: ServiceMetrics[];
  incidents: IncidentInfo[];
  domainConfig: DomainConfig | null;
  metricsHistory: Map<string, number[]>;
  onViewRequests: () => void;
}

function MetricsTab({ svcKey, node, edges, allNodes, metrics, incidents, domainConfig, metricsHistory, onViewRequests }: MetricsTabProps) {
  const m = metrics.find((s) => s.service === svcKey);
  const hasIncident = incidents.some((i) => i.affected_services.includes(svcKey));
  const health = computeHealthState(m, undefined, hasIncident);
  const routing = domainConfig ? getRoutingConfig(svcKey, domainConfig) : undefined;
  const currentRoute = routing?.mode ?? 'local';
  const [routingMode, setRoutingMode] = useState<'local' | 'cloud'>(currentRoute as 'local' | 'cloud');

  const reqHistory = metricsHistory.get(`${svcKey}:req`) ?? [];
  const latHistory = metricsHistory.get(`${svcKey}:p99`) ?? [];
  const errHistory = metricsHistory.get(`${svcKey}:err`) ?? [];

  const healthColor = health === 'green' ? '#22c55e' : health === 'yellow' ? '#fbbf24' : '#ef4444';

  // Compute inbound/outbound request counts from topology edges
  const inboundEdges = edges.filter((e) => e.target === node.id);
  const outboundEdges = edges.filter((e) => e.source === node.id);
  const inboundTotal = inboundEdges.reduce((sum, e) => sum + (e.callCount ?? 0), 0);
  const outboundTotal = outboundEdges.reduce((sum, e) => sum + (e.callCount ?? 0), 0);
  const maxTraffic = Math.max(inboundTotal, outboundTotal, 1);

  const handleViewRequests = () => {
    onViewRequests();
  };

  return (
    <div>
      {/* Inbound / Outbound request meters */}
      <div class="io-meters">
        <div class="io-meter">
          <div class="io-meter-header">
            <span class="io-meter-label">⬇ Inbound</span>
            <span class="io-meter-value">{inboundTotal}</span>
          </div>
          <div class="io-meter-bar-track">
            <div
              class="io-meter-bar-fill io-inbound"
              style={{ width: `${(inboundTotal / maxTraffic) * 100}%` }}
            />
          </div>
          <div class="io-meter-sources">
            {inboundEdges.filter((e) => (e.callCount ?? 0) > 0).map((e, i) => {
              const src = allNodes.find((n) => n.id === e.source);
              return (
                <span key={i} class="io-meter-source">
                  {src?.label || e.source} ({e.callCount})
                </span>
              );
            })}
          </div>
        </div>
        <div class="io-meter">
          <div class="io-meter-header">
            <span class="io-meter-label">⬆ Outbound</span>
            <span class="io-meter-value">{outboundTotal}</span>
          </div>
          <div class="io-meter-bar-track">
            <div
              class="io-meter-bar-fill io-outbound"
              style={{ width: `${(outboundTotal / maxTraffic) * 100}%` }}
            />
          </div>
          <div class="io-meter-sources">
            {outboundEdges.filter((e) => (e.callCount ?? 0) > 0).slice(0, 5).map((e, i) => {
              const tgt = allNodes.find((n) => n.id === e.target);
              return (
                <span key={i} class="io-meter-source">
                  {tgt?.label || e.target} ({e.callCount})
                </span>
              );
            })}
            {outboundEdges.filter((e) => (e.callCount ?? 0) > 0).length > 5 && (
              <span class="io-meter-source io-meter-more">
                +{outboundEdges.filter((e) => (e.callCount ?? 0) > 0).length - 5} more
              </span>
            )}
          </div>
        </div>
      </div>

      {/* Request count — use max of metrics totalCalls OR topology edge traffic */}
      {(() => {
        const effectiveCount = Math.max(m?.totalCalls ?? 0, inboundTotal, outboundTotal);
        return (
          <div class="metrics-request-summary">
            <span class="metrics-request-count">{effectiveCount}</span>
            <span class="metrics-request-label">requests in last 15m</span>
            <button class="btn btn-ghost metrics-view-requests-btn" onClick={handleViewRequests}>
              View Requests
            </button>
          </div>
        );
      })()}

      <Sparkline
        data={reqHistory}
        color="#4AE5F8"
        label="req/s"
        value={m ? `${m.totalCalls}` : `${Math.max(inboundTotal, outboundTotal)}`}
      />
      <Sparkline
        data={latHistory}
        color="#a78bfa"
        label="p99 latency"
        value={m ? `${m.p99ms < 1 ? m.p99ms.toFixed(1) : Math.round(m.p99ms)}ms` : '--'}
      />
      <Sparkline
        data={errHistory}
        color={healthColor}
        label="error rate"
        value={m ? `${(m.errorRate * 100).toFixed(2)}%` : '--'}
      />

      {m && (
        <div class={`slo-status-row ${health === 'green' ? 'slo-ok' : health === 'yellow' ? 'slo-warn' : 'slo-breach'}`}>
          <span>{health === 'green' ? 'SLO OK' : health === 'yellow' ? 'SLO Warning' : 'SLO Breach'}</span>
          <span style={{ marginLeft: 'auto', fontFamily: 'var(--font-mono)', fontSize: '10px' }}>
            p99={m.p99ms < 1 ? m.p99ms.toFixed(1) : Math.round(m.p99ms)}ms · err={((m.errorRate) * 100).toFixed(2)}%
          </span>
        </div>
      )}

      {routing && (
        <div class="routing-toggle">
          <span>Routing:</span>
          <div class="routing-toggle-pill">
            <button
              class={`routing-toggle-option ${routingMode === 'local' ? 'active' : ''}`}
              onClick={() => {
                setRoutingMode('local');
                fetch(`${getAdminBase()}/api/routing`, {
                  method: 'POST',
                  headers: { 'Content-Type': 'application/json' },
                  body: JSON.stringify({ service: svcKey, mode: 'local' }),
                }).catch((err) => console.warn('[Topology] Failed to update routing:', err));
              }}
            >
              Local
            </button>
            <button
              class={`routing-toggle-option ${routingMode === 'cloud' ? 'active' : ''}`}
              onClick={() => {
                setRoutingMode('cloud');
                fetch(`${getAdminBase()}/api/routing`, {
                  method: 'POST',
                  headers: { 'Content-Type': 'application/json' },
                  body: JSON.stringify({ service: svcKey, mode: 'cloud' }),
                }).catch((err) => console.warn('[Topology] Failed to update routing:', err));
              }}
            >
              Cloud
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

/* ---------- Deploys Tab ---------- */

interface DeploysTabProps {
  svcKey: string;
  deploys: DeployEvent[];
}

function DeploysTab({ svcKey, deploys }: DeploysTabProps) {
  const [selectedDeploy, setSelectedDeploy] = useState<DeployEvent | null>(null);

  // Match deploys flexibly — strip common prefixes/suffixes for fuzzy matching.
  const svcLower = svcKey.toLowerCase();
  const stripped = svcLower.replace(/-handler$/, '').replace(/-sync$/, '').replace(/-service$/, '');
  const mappedName: string | undefined = undefined;
  const filtered = deploys.filter((d) => {
    const ds = (d.service || '').toLowerCase();
    return ds === svcLower ||             // exact: "bff" === "bff"
      ds === stripped ||                   // stripped: "bff" matches "bff-service" → "bff"
      svcLower.includes(ds) ||            // contains: "bff-service" includes "bff"
      ds.includes(stripped) ||            // reverse: "billing" includes "order"? no, but catches others
      (mappedName && ds === mappedName);  // Lambda map: "billing" for order-handler
  });

  if (filtered.length === 0) {
    return <div class="inspector-placeholder">No recent deploys for this service.</div>;
  }

  return (
    <div>
      {filtered.map((d) => {
        // DeployEvent is already normalized by the topology-metrics hook,
        // but guard against missing optional fields
        const time = d.timestamp;
        const commitDisplay = d.commit ? d.commit.slice(0, 8) : '--';
        const author = d.author || 'unknown';
        const message = d.message || 'No description';
        const branch = d.branch || '';

        return (
          <div
            key={d.id}
            class="deploy-item deploy-item-clickable"
            onClick={() => setSelectedDeploy(d)}
          >
            <div class="deploy-item-header">
              <span class="deploy-item-time">{time ? relativeTime(time) : '--'}</span>
              {branch && <span class="deploy-item-branch">{branch}</span>}
            </div>
            <div class="deploy-item-message">{message}</div>
            <div class="deploy-item-author">
              {author} · {commitDisplay}
            </div>
          </div>
        );
      })}

      {selectedDeploy && (
        <DeployDetail
          deploy={selectedDeploy}
          onClose={() => setSelectedDeploy(null)}
        />
      )}
    </div>
  );
}

/* ---------- Incidents Tab ---------- */

interface IncidentsTabProps {
  svcKey: string;
  incidents: IncidentInfo[];
}

function IncidentsTab({ svcKey, incidents }: IncidentsTabProps) {
  const filtered = incidents.filter((i) => i.affected_services.includes(svcKey));

  if (filtered.length === 0) {
    return <div class="inspector-placeholder">No active incidents for this service.</div>;
  }

  return (
    <div>
      {filtered.map((inc) => (
        <div key={inc.id} class="incident-item">
          <div class="incident-item-header">
            <span class={`incident-severity-badge severity-${inc.severity}`}>
              {inc.severity}
            </span>
            <span class="incident-status-badge">{inc.status}</span>
          </div>
          <div class="incident-item-title">{inc.title}</div>
          <div class="incident-item-times">
            first: {relativeTime(inc.first_seen)} · last: {relativeTime(inc.last_seen)}
          </div>
        </div>
      ))}
    </div>
  );
}

/* ---------- Connections Tab ---------- */

interface ConnectionsTabProps {
  node: TopoNode;
  edges: TopoEdge[];
  allNodes: TopoNode[];
}

function ConnectionsTab({ node, edges, allNodes }: ConnectionsTabProps) {
  const nodeMap = useMemo(() => new Map(allNodes.map((n) => [n.id, n])), [allNodes]);
  const inbound = edges.filter((e) => e.target === node.id);
  const outbound = edges.filter((e) => e.source === node.id);
  const blastRadius = useMemo(() => getBlastRadius(node.id, edges), [node.id, edges]);

  const handleViewTraces = (serviceId: string) => {
    const svc = nodeMap.get(serviceId);
    const serviceName = svc?.service || serviceId.replace(/^svc:/, '');
    document.dispatchEvent(
      new CustomEvent('neureaux:navigate-traces', {
        detail: { traceId: serviceName },
      }),
    );
  };

  return (
    <div>
      {inbound.length > 0 && (
        <div class="node-inspector-section">
          <div class="node-inspector-label">Inbound ({inbound.length})</div>
          {inbound.map((e, i) => {
            const src = nodeMap.get(e.source);
            return (
              <div key={i} class="node-connection">
                <span class="node-connection-arrow">&larr;</span>
                <span class="node-connection-name">{src?.label || e.source}</span>
                {(e.callCount ?? 0) > 0 && (
                  <span class="node-connection-count">{e.callCount} req/s</span>
                )}
                {(e.avgLatencyMs ?? 0) > 0 && (
                  <span class="node-connection-latency">{e.avgLatencyMs!.toFixed(1)}ms</span>
                )}
                <button
                  class="btn btn-ghost btn-view-traces"
                  onClick={() => handleViewTraces(e.source)}
                >
                  View Traces
                </button>
              </div>
            );
          })}
        </div>
      )}

      {outbound.length > 0 && (
        <div class="node-inspector-section">
          <div class="node-inspector-label">Outbound ({outbound.length})</div>
          {outbound.map((e, i) => {
            const tgt = nodeMap.get(e.target);
            return (
              <div key={i} class="node-connection">
                <span class="node-connection-arrow">&rarr;</span>
                <span class="node-connection-name">{tgt?.label || e.target}</span>
                {(e.callCount ?? 0) > 0 && (
                  <span class="node-connection-count">{e.callCount} req/s</span>
                )}
                {(e.avgLatencyMs ?? 0) > 0 && (
                  <span class="node-connection-latency">{e.avgLatencyMs!.toFixed(1)}ms</span>
                )}
                <button
                  class="btn btn-ghost btn-view-traces"
                  onClick={() => handleViewTraces(e.target)}
                >
                  View Traces
                </button>
              </div>
            );
          })}
        </div>
      )}

      {inbound.length === 0 && outbound.length === 0 && (
        <div class="inspector-placeholder">No connections</div>
      )}

      <div class="blast-radius-row">
        Blast radius: <span class="blast-radius-count">{blastRadius.size}</span> downstream service{blastRadius.size !== 1 ? 's' : ''}
      </div>
    </div>
  );
}
