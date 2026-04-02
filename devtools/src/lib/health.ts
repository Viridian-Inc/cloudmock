import type { TopoNode, TopoEdge } from '../views/topology/index';

export type HealthState = 'green' | 'yellow' | 'red';

export interface ServiceMetrics {
  service: string;
  p50ms: number;
  p95ms: number;
  p99ms: number;
  avgMs: number;
  errorRate: number;
  totalCalls: number;
  errorCalls: number;
}

export interface DeployEvent {
  id: string;
  timestamp: string;
  service: string;
  commit: string;
  author: string;
  message: string;
  branch: string;
  pr?: string;
}

const DEFAULT_P99_THRESHOLD = 200; // ms
const DEFAULT_P99_WARNING = 0.8; // 80% of threshold

/**
 * Compute health state for a service based on its metrics.
 */
export function computeHealthState(
  metrics: ServiceMetrics | undefined,
  sloThreshold: number = DEFAULT_P99_THRESHOLD,
  hasActiveIncident: boolean = false,
): HealthState {
  if (hasActiveIncident) return 'red';
  if (!metrics || metrics.totalCalls === 0) return 'green'; // no traffic = assumed healthy

  const { errorRate, p99ms } = metrics;

  // Red: error rate > 5% OR p99 > SLO
  if (errorRate > 0.05 || p99ms > sloThreshold) return 'red';

  // Yellow: error rate 1-5% OR p99 approaching SLO (within 80%)
  if (errorRate > 0.01 || p99ms > sloThreshold * DEFAULT_P99_WARNING) return 'yellow';

  return 'green';
}

/**
 * Determine if a node is on a user-facing path.
 * Uses BFS from Client-group nodes through synchronous (non-stream) edges.
 */
export function isUserFacing(
  nodeId: string,
  nodes: TopoNode[],
  edges: TopoEdge[],
): boolean {
  // Client nodes are always user-facing
  const clientNodeIds = new Set(
    nodes.filter((n) => n.group === 'Client').map((n) => n.id),
  );
  if (clientNodeIds.has(nodeId)) return true;

  // BFS from client nodes through synchronous edges
  const visited = new Set<string>();
  const queue = [...clientNodeIds];

  // Build adjacency (source → targets) for non-stream edges
  const adj = new Map<string, string[]>();
  for (const e of edges) {
    // Skip async edges (DDB streams, SNS fanout)
    if (e.type === 'trigger' || e.label === 'DDB stream') continue;
    if (!adj.has(e.source)) adj.set(e.source, []);
    adj.get(e.source)!.push(e.target);
  }

  while (queue.length > 0) {
    const current = queue.shift()!;
    if (visited.has(current)) continue;
    visited.add(current);

    if (current === nodeId) return true;

    const neighbors = adj.get(current) || [];
    for (const next of neighbors) {
      if (!visited.has(next)) queue.push(next);
    }
  }

  return false;
}

/**
 * Find all downstream nodes from a given node (blast radius).
 */
export function getBlastRadius(
  nodeId: string,
  edges: TopoEdge[],
): Set<string> {
  const downstream = new Set<string>();
  const adj = new Map<string, string[]>();
  for (const e of edges) {
    if (!adj.has(e.source)) adj.set(e.source, []);
    adj.get(e.source)!.push(e.target);
  }

  const queue = [nodeId];
  while (queue.length > 0) {
    const current = queue.shift()!;
    const neighbors = adj.get(current) || [];
    for (const next of neighbors) {
      if (!downstream.has(next) && next !== nodeId) {
        downstream.add(next);
        queue.push(next);
      }
    }
  }

  return downstream;
}

/**
 * Get the most recent deploy for a service.
 */
export function getRecentDeploy(
  serviceId: string,
  deploys: DeployEvent[],
): DeployEvent | undefined {
  // Match by service name from the node ID
  const parts = serviceId.split(':');
  const svcName = parts[parts.length - 1];

  return deploys.find(
    (d) => d.service === svcName || d.service === serviceId,
  );
}

/**
 * Format a metric for edge display: "42 req/s · 12ms"
 */
export function formatEdgeLabel(callCount: number, avgLatencyMs: number): string {
  if (callCount === 0 && avgLatencyMs === 0) return '';
  const parts: string[] = [];
  if (callCount > 0) parts.push(`${callCount} req/s`);
  if (avgLatencyMs > 0) parts.push(`${avgLatencyMs < 1 ? avgLatencyMs.toFixed(1) : Math.round(avgLatencyMs)}ms`);
  return parts.join(' · ');
}
