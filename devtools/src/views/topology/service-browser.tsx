import { useState, useMemo, useRef, useCallback } from 'preact/hooks';
import type { TopoNode, TopoEdge } from './index';
import type { ServiceMetrics } from '../../lib/health';
import type { IncidentInfo } from '../../lib/types';
import type { DomainConfig } from '../../lib/domains';
import type { ManifestService } from './endpoints-tab';
import { computeHealthState } from '../../lib/health';
import { getAdminBase } from '../../lib/api';
import { peek } from '../../lib/pane-stack';

interface ServiceBrowserProps {
  nodes: TopoNode[];
  edges: TopoEdge[];
  metrics: ServiceMetrics[];
  incidents: IncidentInfo[];
  domainConfig: DomainConfig | null;
  selectedNodeId: string | null;
  onSelectNode: (node: TopoNode) => void;
  manifest: ManifestService[] | null;
}

const SERVICE_ICONS: Record<string, string> = {
  dynamodb: '\u{1F4BE}',     // floppy disk
  sqs: '\u{1F4EC}',          // mailbox
  sns: '\u{1F4E2}',          // loudspeaker
  lambda: '\u{26A1}',        // lightning
  s3: '\u{1F4E6}',           // package
  cognito: '\u{1F511}',      // key
  eventbridge: '\u{1F504}',  // arrows
  stepfunction: '\u{1F3AF}', // target
};

function getServiceKey(node: TopoNode): string {
  return node.service || node.id.replace(/^svc:/, '');
}

function getServiceIcon(node: TopoNode): string {
  const svc = node.service.toLowerCase();
  return SERVICE_ICONS[svc] || (node.type === 'aws-service' ? '\u2601\uFE0F' : '\u{1F527}');
}

interface DomainGroupData {
  name: string;
  services: ServiceRowData[];
  collapsed: boolean;
}

interface ServiceRowData {
  node: TopoNode;
  svcKey: string;
  health: 'green' | 'yellow' | 'red';
  routeCount: number;
  incidentCount: number;
  icon: string;
}

export function ServiceBrowser({
  nodes, edges, metrics, incidents, domainConfig, selectedNodeId, onSelectNode, manifest,
}: ServiceBrowserProps) {
  const [filter, setFilter] = useState('');
  const [collapsedDomains, setCollapsedDomains] = useState<Set<string>>(new Set());
  const listRef = useRef<HTMLDivElement>(null);

  // Routing toggles: service → 'local' | 'cloud'
  const [routingModes, setRoutingModes] = useState<Record<string, 'local' | 'cloud'>>(() => {
    try {
      const saved = localStorage.getItem('neureaux-devtools:routing');
      return saved ? JSON.parse(saved) : {};
    } catch { return {}; }
  });

  const toggleRouting = useCallback((svcKey: string) => {
    setRoutingModes((prev) => {
      const current = prev[svcKey] || 'local';
      const next = current === 'local' ? 'cloud' : 'local';
      const updated: Record<string, 'local' | 'cloud'> = { ...prev, [svcKey]: next };
      localStorage.setItem('neureaux-devtools:routing', JSON.stringify(updated));
      // POST to admin API
      const base = getAdminBase();
      fetch(`${base}/api/routing`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ service: svcKey, mode: next }),
      }).catch(() => {});
      return updated;
    });
  }, []);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      const container = listRef.current;
      if (!container) return;
      const items = container.querySelectorAll<HTMLElement>('.domain-service-row');
      if (items.length === 0) return;

      const focused = container.querySelector<HTMLElement>('.domain-service-row:focus');
      const idx = focused ? Array.from(items).indexOf(focused) : -1;

      if (e.key === 'ArrowDown') {
        e.preventDefault();
        const next = idx < items.length - 1 ? idx + 1 : 0;
        items[next].focus();
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        const prev = idx > 0 ? idx - 1 : items.length - 1;
        items[prev].focus();
      } else if (e.key === 'Enter' && focused) {
        e.preventDefault();
        focused.click();
      }
    },
    [],
  );

  // Build metrics lookup
  const metricsMap = useMemo(() => {
    const m = new Map<string, ServiceMetrics>();
    for (const sm of metrics) m.set(sm.service, sm);
    return m;
  }, [metrics]);

  // Build manifest route count lookup
  const routeCountMap = useMemo(() => {
    const m = new Map<string, number>();
    if (manifest) {
      for (const svc of manifest) {
        m.set(svc.name, svc.routes.length);
        m.set(svc.name.toLowerCase(), svc.routes.length);
      }
    }
    return m;
  }, [manifest]);

  // Build service row data
  const serviceRows = useMemo(() => {
    const rows: ServiceRowData[] = [];
    for (const node of nodes) {
      const svcKey = getServiceKey(node);
      const m = metricsMap.get(svcKey);
      const hasIncident = incidents.some((i) => i.affected_services.includes(svcKey));
      const health = computeHealthState(m, undefined, hasIncident);
      const incidentCount = incidents.filter((i) => i.affected_services.includes(svcKey)).length;
      const routeCount = routeCountMap.get(svcKey) || routeCountMap.get(svcKey.toLowerCase()) || 0;

      rows.push({
        node,
        svcKey,
        health,
        routeCount,
        incidentCount,
        icon: getServiceIcon(node),
      });
    }
    return rows;
  }, [nodes, metricsMap, incidents, routeCountMap]);

  // Group by domain
  const domainGroups: DomainGroupData[] = useMemo(() => {
    const filterLower = filter.toLowerCase();
    const filtered = filterLower
      ? serviceRows.filter((r) =>
          r.node.label.toLowerCase().includes(filterLower) ||
          r.svcKey.toLowerCase().includes(filterLower))
      : serviceRows;

    const groups: DomainGroupData[] = [];
    const assigned = new Set<string>();

    if (domainConfig) {
      for (const domain of domainConfig.domains) {
        const domainServices = filtered.filter((r) =>
          domain.services.includes(r.svcKey) ||
          domain.services.includes(r.node.label) ||
          domain.services.includes(r.node.service));

        if (domainServices.length > 0) {
          groups.push({
            name: domain.name,
            services: domainServices.sort((a, b) => a.node.label.localeCompare(b.node.label)),
            collapsed: collapsedDomains.has(domain.name),
          });
          for (const s of domainServices) assigned.add(s.node.id);
        }
      }
    }

    // Remaining services under "Other"
    const other = filtered.filter((r) => !assigned.has(r.node.id));
    if (other.length > 0) {
      groups.push({
        name: 'Other',
        services: other.sort((a, b) => a.node.label.localeCompare(b.node.label)),
        collapsed: collapsedDomains.has('Other'),
      });
    }

    return groups;
  }, [serviceRows, domainConfig, filter, collapsedDomains]);

  const toggleDomain = (name: string) => {
    setCollapsedDomains((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  };

  const HEALTH_COLORS = {
    green: '#22c55e',
    yellow: '#fbbf24',
    red: '#ef4444',
  };

  return (
    <div class="service-browser">
      <div class="service-browser-header">
        <input
          type="text"
          class="service-browser-search"
          placeholder="Filter services..."
          value={filter}
          onInput={(e) => setFilter((e.target as HTMLInputElement).value)}
        />
      </div>
      <div class="service-browser-list" ref={listRef} tabIndex={0} onKeyDown={handleKeyDown}>
        {domainGroups.map((group) => (
          <div key={group.name} class="domain-group">
            <button
              class="domain-header"
              onClick={() => toggleDomain(group.name)}
            >
              <span class="domain-header-arrow">{group.collapsed ? '\u25B6' : '\u25BC'}</span>
              <span class="domain-header-name">{group.name}</span>
              <span class="domain-header-count">{group.services.length}</span>
            </button>
            {!group.collapsed && group.services.map((svc) => (
              <div
                key={svc.node.id}
                class={`domain-service-row ${selectedNodeId === svc.node.id ? 'selected' : ''}`}
                tabIndex={-1}
                onClick={() => onSelectNode(svc.node)}
              >
                <span
                  class="domain-service-health-dot"
                  style={{ background: HEALTH_COLORS[svc.health] }}
                />
                <span class="domain-service-icon">{svc.icon}</span>
                <span class="domain-service-name">{svc.node.label}</span>
                {svc.routeCount > 0 && (
                  <span class="domain-service-routes">{svc.routeCount} routes</span>
                )}
                {svc.incidentCount > 0 && (
                  <span class="domain-service-incident-badge">{svc.incidentCount}</span>
                )}
                <button
                  class="domain-service-peek-btn"
                  onClick={(e) => {
                    e.stopPropagation();
                    peek({
                      view: 'activity',
                      params: { service: svc.svcKey },
                      title: svc.node.label,
                    });
                  }}
                  title={`Peek ${svc.node.label} activity in a new pane`}
                  aria-label={`Peek ${svc.node.label}`}
                >
                  {'\u2197'}
                </button>
                <button
                  class={`domain-service-route-toggle ${(routingModes[svc.svcKey] || 'local') === 'cloud' ? 'cloud' : 'local'}`}
                  onClick={(e) => { e.stopPropagation(); toggleRouting(svc.svcKey); }}
                  title={`Switch to ${(routingModes[svc.svcKey] || 'local') === 'local' ? 'cloud' : 'local'}`}
                >
                  {(routingModes[svc.svcKey] || 'local') === 'local' ? 'Local' : 'Cloud'}
                </button>
              </div>
            ))}
          </div>
        ))}
        {domainGroups.length === 0 && (
          <div class="inspector-placeholder" style={{ padding: '16px' }}>
            No services match filter.
          </div>
        )}
      </div>
    </div>
  );
}
