import type { TopoNode, TopoEdge } from './index';

interface RawNode {
  id: string;
  label: string;
  service: string;
  type: string;
  group: string;
  requestService?: string;
}

interface RawEdge {
  source: string;
  target: string;
  label?: string;
  type?: string;
  discovered?: string;
  callCount?: number;
  avgLatencyMs?: number;
}

/**
 * Collapse raw cloudmock topology into a clean service-level graph.
 *
 * Rules:
 * - Nodes with service "external" or "plugin" stay as-is (they're already service-level)
 * - All other nodes collapse by service type: 42 dynamodb nodes -> 1 "DynamoDB" node
 * - Edges remap to collapsed node IDs and dedup
 */
export function collapseTopology(
  rawNodes: RawNode[],
  rawEdges: RawEdge[],
): { nodes: TopoNode[]; edges: TopoEdge[] } {
  const nodes: TopoNode[] = [];
  const nodeIds = new Set<string>();
  const collapseMap = new Map<string, string>(); // raw ID -> collapsed ID

  // Pass 1: Keep external/plugin nodes as-is, collapse AWS resource nodes by service
  const serviceBuckets = new Map<string, { count: number; group: string; label: string }>();

  for (const n of rawNodes) {
    if (n.service === 'external' || n.service === 'plugin') {
      // Keep as-is — these are your actual services
      nodes.push({ ...n });
      nodeIds.add(n.id);
      collapseMap.set(n.id, n.id);
    } else {
      // Collapse: dynamodb:users -> svc:dynamodb
      const collapsedId = `svc:${n.service}`;
      collapseMap.set(n.id, collapsedId);

      if (!serviceBuckets.has(n.service)) {
        serviceBuckets.set(n.service, { count: 0, group: n.group, label: n.service });
      }
      serviceBuckets.get(n.service)!.count++;
    }
  }

  // Create collapsed AWS service nodes
  for (const [svc, info] of serviceBuckets) {
    const id = `svc:${svc}`;
    if (!nodeIds.has(id)) {
      nodes.push({
        id,
        label: svc.charAt(0).toUpperCase() + svc.slice(1),
        service: svc,
        type: 'aws-service',
        group: info.group,
        resourceCount: info.count > 1 ? info.count : undefined,
      });
      nodeIds.add(id);
    }
  }

  // Lambda function name → friendly microservice name (populated dynamically from node labels)
  const LAMBDA_NAMES: Record<string, string> = {};

  // Pass 2: Remap edge endpoints
  function resolveId(id: string): string {
    if (collapseMap.has(id)) return collapseMap.get(id)!;

    const colonIdx = id.indexOf(':');
    if (colonIdx <= 0) return id;

    const svcType = id.substring(0, colonIdx);
    const resourceName = id.substring(colonIdx + 1);

    // Lambda functions → keep as individual microservice nodes
    if (svcType === 'lambda') {
      const friendlyName = LAMBDA_NAMES[resourceName] || resourceName;
      const msId = `ms:${friendlyName}`;
      if (!nodeIds.has(msId)) {
        nodes.push({
          id: msId,
          label: friendlyName,
          service: resourceName,
          type: 'microservice',
          group: 'Compute',
        });
        nodeIds.add(msId);
      }
      collapseMap.set(id, msId);
      return msId;
    }

    // Everything else → collapse to service level
    const collapsedId = `svc:${svcType}`;
    if (!nodeIds.has(collapsedId)) {
      nodes.push({
        id: collapsedId,
        label: svcType.charAt(0).toUpperCase() + svcType.slice(1),
        service: svcType,
        type: 'aws-service',
        group: serviceBuckets.get(svcType)?.group || 'AWS',
      });
      nodeIds.add(collapsedId);
    }
    collapseMap.set(id, collapsedId);
    return collapsedId;
  }

  // Dedup edges
  const edges: TopoEdge[] = [];
  const edgeSeen = new Set<string>();

  for (const e of rawEdges) {
    const source = resolveId(e.source);
    const target = resolveId(e.target);

    if (source === target) continue;
    if (!nodeIds.has(source) || !nodeIds.has(target)) continue;

    const key = `${source}\u2192${target}`;
    if (edgeSeen.has(key)) continue;
    edgeSeen.add(key);

    edges.push({ ...e, source, target });
  }

  return { nodes, edges };
}
