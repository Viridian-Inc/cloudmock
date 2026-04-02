import type { ServiceInfo } from '../../lib/types';

export interface AppService {
  id: string;
  name: string;
  icon: string;
  type: string;
  group: string;
  awsDeps: string[];
}

export interface TopologyNode {
  id: string;
  label: string;
  type: string;
  group: string;
}

export interface TopologyEdge {
  source: string;
  target: string;
}

const TYPE_ICONS: Record<string, string> = {
  client: '\uD83D\uDCF1',
  server: '\uD83D\uDDA5\uFE0F',
  plugin: '\uD83D\uDD0C',
};

const GROUP_ICONS: Record<string, string> = {
  Client: '\uD83D\uDCF1',
  API: '\uD83D\uDD00',
  Compute: '\u2699\uFE0F',
  Plugins: '\uD83D\uDD0C',
};

/** Derive icon for a topology node based on type then group */
export function nodeIcon(type: string, group: string): string {
  return TYPE_ICONS[type] || GROUP_ICONS[group] || '\u2699\uFE0F';
}

/**
 * Build AppService entries from IaC topology nodes + edges.
 * Pure function that converts raw topology data into the app's service model.
 */
export function buildAppServices(
  nodes: TopologyNode[],
  edges: TopologyEdge[],
): AppService[] {
  return nodes.map((n) => {
    const deps = edges
      .filter((e) => e.source === n.id)
      .map((e) => e.target)
      .filter((t: string) => !t.startsWith('external:') && !t.startsWith('plugin:'));

    return {
      id: n.id,
      name: n.label,
      icon: nodeIcon(n.type, n.group),
      type: n.type,
      group: n.group,
      awsDeps: deps,
    };
  });
}

/**
 * Group app services by their domain/group.
 * Returns a Map from group name to filtered services (matching search query).
 */
export function groupByDomain(
  appServices: AppService[],
  searchQuery: string,
): Map<string, AppService[]> {
  const q = searchQuery.toLowerCase();
  const groups = new Map<string, AppService[]>();
  for (const app of appServices) {
    if (!app.name.toLowerCase().includes(q)) continue;
    if (!groups.has(app.group)) groups.set(app.group, []);
    groups.get(app.group)!.push(app);
  }
  return groups;
}

/**
 * Derive health dot CSS class from ServiceInfo.
 * Returns 'healthy' or 'unhealthy'.
 */
export function healthDotClass(info: ServiceInfo | undefined): string {
  return info?.healthy !== false ? 'healthy' : 'unhealthy';
}

/**
 * Parse an AWS dependency string ("service:resource") and resolve its health
 * using the services list.
 */
export function parseAwsDep(
  dep: string,
  awsServices: ServiceInfo[],
): { svcName: string; resourceName: string; healthy: boolean } {
  const parts = dep.split(':');
  const svcName = parts[0];
  const resourceName = parts.slice(1).join(':');
  const info = awsServices.find((s) => s.name === svcName);
  return {
    svcName,
    resourceName: resourceName || svcName,
    healthy: info?.healthy !== false,
  };
}

/**
 * Split AWS services into active (action_count > 0) and stub (action_count === 0),
 * filtered by search query. Active sorted by action_count desc, stubs alphabetical.
 */
export function splitAwsServices(
  services: ServiceInfo[],
  searchQuery: string,
): { active: ServiceInfo[]; stubs: ServiceInfo[] } {
  const q = searchQuery.toLowerCase();
  const filtered = services.filter((s) => s.name.toLowerCase().includes(q));
  const active = filtered
    .filter((s) => s.action_count > 0)
    .sort((a, b) => b.action_count - a.action_count);
  const stubs = filtered
    .filter((s) => s.action_count === 0)
    .sort((a, b) => a.name.localeCompare(b.name));
  return { active, stubs };
}
