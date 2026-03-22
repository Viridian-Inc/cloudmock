import { useState, useEffect, useRef, useCallback, useMemo } from 'preact/hooks';
import { api } from '../api';
import type { SSEState } from '../hooks/useSSE';
import { NodeDetailDrawer } from '../components/NodeDetailDrawer';

// Internal developer dashboard -- SVG content is generated programmatically
// from our own service API data, not from user input.

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
}

interface TopoGroup {
  id: string;
  label: string;
  color: string;
}

interface TopoData {
  nodes: TopoNode[];
  edges: TopoEdge[];
  groups: TopoGroup[];
}

interface TopologyPageProps {
  sse: SSEState;
}

// --- Positioned types ---

interface PositionedNode {
  id: string;
  label: string;
  service: string;
  type: string;
  group: string;
  x: number;
  y: number;
}

interface PositionedGroup {
  id: string;
  label: string;
  color: string;
  x: number;
  y: number;
  width: number;
  height: number;
  nodeCount: number;
}

// --- Constants ---

const RES_W = 160;
const RES_H = 44;
const RES_GAP_X = 14;
const RES_GAP_Y = 12;
const COLS = 1;
const CLUSTER_PAD_X = 16;
const CLUSTER_PAD_Y = 16;
const CLUSTER_HEADER = 32;
const GROUP_GAP_Y = 36;
const LAYER_GAP = 100;

// --- Grouping config types ---

type GroupMode = 'service' | 'domain' | 'flow' | 'hybrid';

interface GroupConfig {
  nodeToGroup: (node: TopoNode) => string;
  groups: { id: string; label: string; color: string }[];
  layers: Record<string, number>;
  order: Record<string, number>;
}

// --- "By Service" config (default -- uses the server-provided groups) ---

const SERVICE_CONFIG: GroupConfig = {
  nodeToGroup: (n: TopoNode) => n.group, // use server assignment
  groups: [
    { id: 'Client', label: 'Client Apps', color: '#6366F1' },
    { id: 'Plugins', label: 'External Services', color: '#94A3B8' },
    { id: 'API', label: 'API Layer', color: '#06B6D4' },
    { id: 'Auth', label: 'Auth & Identity', color: '#8B5CF6' },
    { id: 'Compute', label: 'Compute', color: '#3B82F6' },
    { id: 'Core Data', label: 'Core Domain', color: '#10B981' },
    { id: 'Features', label: 'Features', color: '#059669' },
    { id: 'Admin', label: 'Admin & Analytics', color: '#6366F1' },
    { id: 'Integrations', label: 'Integrations', color: '#A855F7' },
    { id: 'Facilities', label: 'Facilities', color: '#14B8A6' },
    { id: 'Messaging', label: 'Messaging', color: '#F97316' },
    { id: 'Storage', label: 'Storage', color: '#F59E0B' },
    { id: 'Security', label: 'Security & Config', color: '#6366F1' },
    { id: 'Monitoring', label: 'Monitoring', color: '#EC4899' },
  ],
  layers: {
    Client: 0, Plugins: 0,
    API: 1, Auth: 1,
    Compute: 2,
    'Core Data': 3, Features: 3, Admin: 3, Integrations: 3, Facilities: 3,
    Messaging: 4, Storage: 4,
    Security: 5, Monitoring: 5,
  },
  order: {
    Client: 0, Plugins: 1,
    API: 0, Auth: 1,
    Compute: 0,
    'Core Data': 0, Features: 1, Admin: 2, Integrations: 3, Facilities: 4,
    Messaging: 0, Storage: 1,
    Security: 0, Monitoring: 1,
  },
};

// --- "By Domain" config ---

const DOMAIN_TABLE_MAP: Record<string, string> = {
  attendance: 'Attendance', attendancePolicy: 'Attendance', attendanceOverride: 'Attendance',
  order: 'Orders',
  membership: 'Membership', resourceMembership: 'Membership', userGroup: 'Membership', invitation: 'Membership',
  enterprise: 'Enterprise', resource: 'Enterprise', building: 'Enterprise', roomBlueprint: 'Enterprise',
  session: 'Sessions', calendar: 'Sessions', eventInstance: 'Sessions', event: 'Sessions', personalEvent: 'Sessions',
  notification: 'Notifications', webhook: 'Notifications', webhookDelivery: 'Notifications',
  analytics: 'Analytics', analyticsConsent: 'Analytics', healthMetrics: 'Analytics',
  release: 'AdminDomain', deployment: 'AdminDomain', rolloutStage: 'AdminDomain', auditLog: 'AdminDomain', approval: 'AdminDomain',
  lmsIntegration: 'LMS', lmsCourseMapping: 'LMS', lmsSyncLog: 'LMS', integration: 'LMS', dispute: 'LMS', dataRequest: 'LMS',
  identityProvider: 'AuthDomain', customDomain: 'AuthDomain',
};

function domainNodeToGroup(n: TopoNode): string {
  // Client apps
  if (n.id.startsWith('external:expo') || n.id.startsWith('external:admin') || n.id.startsWith('external:client')) return 'ClientApps';
  if (n.id === 'external:bff-service' || n.id === 'external:graphql-server') return 'ClientApps';
  if (n.id === 'external:calendar-service') return 'Sessions';

  // Auth
  if (n.service === 'cognito-idp' || n.service === 'iam' || n.service === 'sts') return 'AuthDomain';

  // Infra
  if (n.service === 's3' || n.service === 'rds' || n.service === 'kms' || n.service === 'secretsmanager' ||
      n.service === 'ssm' || n.service === 'cloudformation' || n.service === 'monitoring' || n.service === 'logs') return 'Infrastructure';

  // Plugins
  if (n.service === 'plugin') {
    if (n.id === 'plugin:posthog') return 'Analytics';
    return 'ClientApps';
  }

  // API Gateway
  if (n.service === 'apigateway') return 'ClientApps';

  // DynamoDB tables -> domain mapping
  if (n.service === 'dynamodb') {
    const mapped = DOMAIN_TABLE_MAP[n.label];
    if (mapped) return mapped;
    // Fallback by name patterns
    if (n.label.includes('seat') || n.label.includes('building') || n.label.includes('room')) return 'Enterprise';
    if (n.label.includes('feature') || n.label.includes('color') || n.label.includes('class') ||
        n.label.includes('report') || n.label.includes('apiKey') || n.label.includes('tinyUrl') ||
        n.label.includes('userMetadata')) return 'Enterprise';
    return 'Enterprise';
  }

  // Lambda -> domain by function name
  if (n.service === 'lambda') {
    const fn = n.label.toLowerCase();
    if (fn.includes('attendance')) return 'Attendance';
    if (fn.includes('order')) return 'Orders';
    if (fn.includes('membership')) return 'Membership';
    if (fn.includes('notification')) return 'Notifications';
    return 'Enterprise';
  }

  // SQS -> domain by queue name
  if (n.service === 'sqs') {
    const q = n.label.toLowerCase();
    if (q.includes('attendance')) return 'Attendance';
    if (q.includes('order')) return 'Orders';
    if (q.includes('notification')) return 'Notifications';
    return 'Enterprise';
  }

  // SNS, SES -> Notifications
  if (n.service === 'sns' || n.service === 'ses') return 'Notifications';

  // EventBridge -> Enterprise
  if (n.service === 'events') return 'Enterprise';

  return 'Enterprise';
}

const DOMAIN_CONFIG: GroupConfig = {
  nodeToGroup: domainNodeToGroup,
  groups: [
    { id: 'ClientApps', label: 'Client Apps', color: '#6366F1' },
    { id: 'AuthDomain', label: 'Auth', color: '#8B5CF6' },
    { id: 'Attendance', label: 'Attendance', color: '#EF4444' },
    { id: 'Orders', label: 'Orders', color: '#F97316' },
    { id: 'Membership', label: 'Membership', color: '#3B82F6' },
    { id: 'Enterprise', label: 'Enterprise', color: '#10B981' },
    { id: 'Sessions', label: 'Sessions', color: '#06B6D4' },
    { id: 'Notifications', label: 'Notifications', color: '#F59E0B' },
    { id: 'Analytics', label: 'Analytics', color: '#EC4899' },
    { id: 'AdminDomain', label: 'Admin', color: '#6366F1' },
    { id: 'LMS', label: 'LMS', color: '#A855F7' },
    { id: 'Infrastructure', label: 'Infrastructure', color: '#64748B' },
  ],
  layers: {
    ClientApps: 0,
    AuthDomain: 1, Attendance: 1, Orders: 1,
    Membership: 2, Enterprise: 2, Sessions: 2,
    Notifications: 3, Analytics: 3, AdminDomain: 3, LMS: 3,
    Infrastructure: 4,
  },
  order: {
    ClientApps: 0,
    AuthDomain: 0, Attendance: 1, Orders: 2,
    Membership: 0, Enterprise: 1, Sessions: 2,
    Notifications: 0, Analytics: 1, AdminDomain: 2, LMS: 3,
    Infrastructure: 0,
  },
};

// --- "By Flow" config ---

function flowNodeToGroup(n: TopoNode): string {
  // Clients
  if (n.id.startsWith('external:expo') || n.id.startsWith('external:admin') || n.id.startsWith('external:client')) return 'FlowClients';
  if (n.service === 'plugin') return 'FlowClients';

  // API
  if (n.id === 'external:bff-service' || n.id === 'external:graphql-server' || n.service === 'apigateway') return 'FlowAPI';

  // Compute
  if (n.service === 'lambda' || n.id === 'external:calendar-service') return 'FlowCompute';

  // Data
  if (n.service === 'dynamodb' || n.service === 'rds') return 'FlowData';

  // Events
  if (n.service === 'sqs' || n.service === 'sns' || n.service === 'ses' || n.service === 'events') return 'FlowEvents';

  // Infrastructure (everything else)
  return 'FlowInfra';
}

const FLOW_CONFIG: GroupConfig = {
  nodeToGroup: flowNodeToGroup,
  groups: [
    { id: 'FlowClients', label: 'Clients', color: '#6366F1' },
    { id: 'FlowAPI', label: 'API', color: '#06B6D4' },
    { id: 'FlowCompute', label: 'Compute', color: '#3B82F6' },
    { id: 'FlowData', label: 'Data', color: '#10B981' },
    { id: 'FlowEvents', label: 'Events', color: '#F97316' },
    { id: 'FlowInfra', label: 'Infrastructure', color: '#64748B' },
  ],
  layers: {
    FlowClients: 0,
    FlowAPI: 1,
    FlowCompute: 2,
    FlowData: 3,
    FlowEvents: 4,
    FlowInfra: 5,
  },
  order: {
    FlowClients: 0,
    FlowAPI: 0,
    FlowCompute: 0,
    FlowData: 0,
    FlowEvents: 0,
    FlowInfra: 0,
  },
};

// --- "Hybrid" config ---

function hybridNodeToGroup(n: TopoNode): string {
  // Layer 0: Clients
  if (n.id.startsWith('external:expo') || n.id.startsWith('external:admin') || n.id.startsWith('external:client')) return 'HybridClients';
  if (n.service === 'plugin') return 'HybridClients';

  // Layer 1: API Gateway + Cognito
  if (n.service === 'apigateway' || n.id === 'external:bff-service' || n.id === 'external:graphql-server') return 'HybridAPIGW';
  if (n.service === 'cognito-idp' || n.service === 'iam' || n.service === 'sts') return 'HybridCognito';

  // Layer 2: Lambda
  if (n.service === 'lambda' || n.id === 'external:calendar-service') return 'HybridLambda';

  // Layer 3: DynamoDB + RDS
  if (n.service === 'dynamodb') return 'HybridDynamo';
  if (n.service === 'rds') return 'HybridRDS';

  // Layer 4: SQS + SNS + EventBridge
  if (n.service === 'sqs') return 'HybridSQS';
  if (n.service === 'sns' || n.service === 'ses') return 'HybridSNS';
  if (n.service === 'events') return 'HybridEB';

  // Layer 5: KMS, Secrets, CloudWatch, S3
  if (n.service === 'kms') return 'HybridKMS';
  if (n.service === 'secretsmanager' || n.service === 'ssm') return 'HybridSecrets';
  if (n.service === 'monitoring' || n.service === 'logs') return 'HybridCW';
  if (n.service === 's3') return 'HybridS3';
  if (n.service === 'cloudformation') return 'HybridCW';

  return 'HybridLambda';
}

const HYBRID_CONFIG: GroupConfig = {
  nodeToGroup: hybridNodeToGroup,
  groups: [
    { id: 'HybridClients', label: 'Clients', color: '#6366F1' },
    { id: 'HybridAPIGW', label: 'API Gateway', color: '#06B6D4' },
    { id: 'HybridCognito', label: 'Cognito & IAM', color: '#8B5CF6' },
    { id: 'HybridLambda', label: 'Lambda', color: '#3B82F6' },
    { id: 'HybridDynamo', label: 'DynamoDB', color: '#10B981' },
    { id: 'HybridRDS', label: 'RDS', color: '#059669' },
    { id: 'HybridSQS', label: 'SQS', color: '#F97316' },
    { id: 'HybridSNS', label: 'SNS & SES', color: '#EAB308' },
    { id: 'HybridEB', label: 'EventBridge', color: '#F43F5E' },
    { id: 'HybridKMS', label: 'KMS', color: '#6366F1' },
    { id: 'HybridSecrets', label: 'Secrets & SSM', color: '#8B5CF6' },
    { id: 'HybridCW', label: 'CloudWatch & CFN', color: '#EC4899' },
    { id: 'HybridS3', label: 'S3', color: '#F59E0B' },
  ],
  layers: {
    HybridClients: 0,
    HybridAPIGW: 1, HybridCognito: 1,
    HybridLambda: 2,
    HybridDynamo: 3, HybridRDS: 3,
    HybridSQS: 4, HybridSNS: 4, HybridEB: 4,
    HybridKMS: 5, HybridSecrets: 5, HybridCW: 5, HybridS3: 5,
  },
  order: {
    HybridClients: 0,
    HybridAPIGW: 0, HybridCognito: 1,
    HybridLambda: 0,
    HybridDynamo: 0, HybridRDS: 1,
    HybridSQS: 0, HybridSNS: 1, HybridEB: 2,
    HybridKMS: 0, HybridSecrets: 1, HybridCW: 2, HybridS3: 3,
  },
};

const GROUP_CONFIGS: Record<GroupMode, GroupConfig> = {
  service: SERVICE_CONFIG,
  domain: DOMAIN_CONFIG,
  flow: FLOW_CONFIG,
  hybrid: HYBRID_CONFIG,
};

const TAB_LABELS: Record<GroupMode, string> = {
  service: 'By Service',
  domain: 'By Domain',
  flow: 'By Flow',
  hybrid: 'Hybrid',
};

// --- Layout engine ---

function layoutGraph(
  data: TopoData,
  config: GroupConfig,
): {
  positionedNodes: PositionedNode[];
  positionedGroups: PositionedGroup[];
  activeGroups: TopoGroup[];
} {
  // Re-assign nodes to groups using the config's nodeToGroup function
  const remappedNodes: TopoNode[] = data.nodes.map(n => ({
    ...n,
    group: config.nodeToGroup(n),
  }));

  // Group nodes by their new group assignment
  const nodesByGroup = new Map<string, TopoNode[]>();
  for (const n of remappedNodes) {
    const arr = nodesByGroup.get(n.group) || [];
    arr.push(n);
    nodesByGroup.set(n.group, arr);
  }

  // Build group info lookup from config groups
  const groupInfo = new Map<string, TopoGroup>();
  for (const g of config.groups) {
    groupInfo.set(g.id, g);
  }

  // Build active groups list (only groups that have nodes)
  const activeGroups: TopoGroup[] = [];
  for (const g of config.groups) {
    if (nodesByGroup.has(g.id)) {
      activeGroups.push(g);
    }
  }

  // Organise groups by layer
  const layerGroups = new Map<number, { group: TopoGroup; nodes: TopoNode[] }[]>();
  for (const [gid, nodes] of nodesByGroup) {
    const layer = config.layers[gid] ?? 3;
    const arr = layerGroups.get(layer) || [];
    const info = groupInfo.get(gid) || { id: gid, label: gid, color: '#94A3B8' };
    arr.push({ group: info, nodes });
    layerGroups.set(layer, arr);
  }

  // Sort within each layer by order
  for (const [, arr] of layerGroups) {
    arr.sort((a, b) => (config.order[a.group.id] ?? 99) - (config.order[b.group.id] ?? 99));
  }

  const positionedNodes: PositionedNode[] = [];
  const positionedGroups: PositionedGroup[] = [];

  // First pass: compute cluster sizes
  interface ClusterSize { width: number; height: number; nodeCount: number; }
  const clusterSizes = new Map<string, ClusterSize>();
  for (const [, arr] of layerGroups) {
    for (const { group, nodes } of arr) {
      const cols = Math.min(COLS, nodes.length);
      const rows = Math.ceil(nodes.length / COLS);
      const cw = Math.max(180, cols * RES_W + (cols - 1) * RES_GAP_X + CLUSTER_PAD_X * 2);
      const ch = CLUSTER_HEADER + rows * RES_H + (rows - 1) * RES_GAP_Y + CLUSTER_PAD_Y * 2;
      clusterSizes.set(group.id, { width: cw, height: ch, nodeCount: nodes.length });
    }
  }

  // Second pass: compute layer x offsets
  const layers = Array.from(layerGroups.keys()).sort((a, b) => a - b);
  const layerWidths = new Map<number, number>();
  for (const layer of layers) {
    let maxW = 0;
    for (const { group } of layerGroups.get(layer)!) {
      const sz = clusterSizes.get(group.id)!;
      maxW = Math.max(maxW, sz.width);
    }
    layerWidths.set(layer, maxW);
  }

  // X positions per layer
  const layerX = new Map<number, number>();
  let currentX = 40;
  for (const layer of layers) {
    layerX.set(layer, currentX);
    currentX += (layerWidths.get(layer) || 200) + LAYER_GAP;
  }

  // Third pass: position clusters and nodes
  for (const layer of layers) {
    const groups = layerGroups.get(layer) || [];
    let currentY = 40;

    for (const { group, nodes } of groups) {
      const x = layerX.get(layer)!;
      const sz = clusterSizes.get(group.id)!;

      positionedGroups.push({
        id: group.id,
        label: group.label,
        color: group.color,
        x,
        y: currentY,
        width: sz.width,
        height: sz.height,
        nodeCount: sz.nodeCount,
      });

      // Position nodes in single-column grid inside the cluster
      for (let i = 0; i < nodes.length; i++) {
        const col = i % COLS;
        const row = Math.floor(i / COLS);
        const nx = x + CLUSTER_PAD_X + col * (RES_W + RES_GAP_X);
        const ny = currentY + CLUSTER_HEADER + CLUSTER_PAD_Y + row * (RES_H + RES_GAP_Y);

        positionedNodes.push({
          id: nodes[i].id,
          label: nodes[i].label,
          service: nodes[i].service,
          type: nodes[i].type,
          group: nodes[i].group,
          x: nx,
          y: ny,
        });
      }

      currentY += sz.height + GROUP_GAP_Y;
    }
  }

  return { positionedNodes, positionedGroups, activeGroups };
}

// --- SVG helpers ---

function bezierPath(x1: number, y1: number, x2: number, y2: number): string {
  const dx = Math.abs(x2 - x1) * 0.45;
  return `M ${x1} ${y1} C ${x1 + dx} ${y1}, ${x2 - dx} ${y2}, ${x2} ${y2}`;
}

// --- Component ---

export function TopologyPage({ sse }: TopologyPageProps) {
  const [topoData, setTopoData] = useState<TopoData | null>(null);
  const [groupMode, setGroupMode] = useState<GroupMode>('service');
  const [showAll, setShowAll] = useState(true);
  const [hoveredNode, setHoveredNode] = useState<string | null>(null);
  const [selectedNode, setSelectedNode] = useState<TopoNode | null>(null);
  const [hoveredEdge, setHoveredEdge] = useState<number | null>(null);
  const [hoveredCluster, setHoveredCluster] = useState<string | null>(null);
  const [transform, setTransform] = useState({ x: 0, y: 0, scale: 1 });
  const [pulsingNodes, setPulsingNodes] = useState<Map<string, number>>(new Map());
  const [packets, setPackets] = useState<{id: number; edgeIdx: number; timestamp: number}[]>([]);
  const packetId = useRef(0);
  const svgRef = useRef<SVGSVGElement>(null);
  const dragging = useRef<{ startX: number; startY: number; origX: number; origY: number } | null>(null);

  // Live SSE: pulse nodes when requests arrive
  useEffect(() => {
    if (!sse?.events?.length) return;
    const latest = sse.events[0];
    if (!latest?.data?.service) return;

    // Find all nodes matching this service
    const svcName = latest.data.service;
    setPulsingNodes(prev => {
      const next = new Map(prev);
      // Pulse any node whose service matches
      if (topoData) {
        for (const n of topoData.nodes) {
          if (n.service === svcName || n.id.startsWith(svcName + ':')) {
            next.set(n.id, Date.now());
          }
        }
      }
      return next;
    });

    // Spawn packets on edges that involve this service
    if (topoData) {
      const now = Date.now();
      const newPackets: {id: number; edgeIdx: number; timestamp: number}[] = [];
      topoData.edges.forEach((e, idx) => {
        if (e.target.startsWith(svcName + ':') || e.source.startsWith(svcName + ':')) {
          newPackets.push({ id: ++packetId.current, edgeIdx: idx, timestamp: now });
        }
      });
      if (newPackets.length > 0) {
        setPackets(prev => [...prev, ...newPackets].slice(-30)); // keep max 30 active
      }
    }

    // Clear pulse + packets after 1.5s
    const timer = setTimeout(() => {
      setPulsingNodes(prev => {
        const next = new Map(prev);
        const cutoff = Date.now() - 1400;
        for (const [k, v] of next) {
          if (v < cutoff) next.delete(k);
        }
        return next;
      });
      setPackets(prev => prev.filter(p => Date.now() - p.timestamp < 2000));
    }, 2000);
    return () => clearTimeout(timer);
  }, [sse?.events?.length, topoData]);

  // Fetch topology data
  useEffect(() => {
    api('/api/topology').then(data => {
      setTopoData(data);
    }).catch(() => {});
  }, []);

  // Layout (depends on topoData AND groupMode)
  const layout = useMemo(() => {
    if (!topoData) return null;
    const config = GROUP_CONFIGS[groupMode];
    return layoutGraph(topoData, config);
  }, [topoData, groupMode]);

  // Node position lookup
  const nodePos = useMemo(() => {
    if (!layout) return new Map<string, { cx: number; cy: number }>();
    const m = new Map<string, { cx: number; cy: number }>();
    for (const n of layout.positionedNodes) {
      m.set(n.id, { cx: n.x + RES_W / 2, cy: n.y + RES_H / 2 });
    }
    return m;
  }, [layout]);

  // Connected edges and nodes for hover highlight
  const connectedEdges = useMemo(() => {
    if (!hoveredNode || !topoData) return new Set<number>();
    const set = new Set<number>();
    topoData.edges.forEach((e, i) => {
      if (e.source === hoveredNode || e.target === hoveredNode) set.add(i);
    });
    return set;
  }, [hoveredNode, topoData]);

  const connectedNodes = useMemo(() => {
    if (!hoveredNode || !topoData) return new Set<string>();
    const set = new Set<string>([hoveredNode]);
    topoData.edges.forEach(e => {
      if (e.source === hoveredNode) set.add(e.target);
      if (e.target === hoveredNode) set.add(e.source);
    });
    return set;
  }, [hoveredNode, topoData]);

  // SVG dimensions
  const { svgW, svgH } = useMemo(() => {
    if (!layout) return { svgW: 1600, svgH: 900 };
    let maxX = 0;
    let maxY = 0;
    for (const g of layout.positionedGroups) {
      maxX = Math.max(maxX, g.x + g.width);
      maxY = Math.max(maxY, g.y + g.height);
    }
    return { svgW: Math.max(1600, maxX + 80), svgH: Math.max(900, maxY + 80) };
  }, [layout]);

  // Zoom handler
  const onWheel = useCallback((e: WheelEvent) => {
    e.preventDefault();
    const delta = e.deltaY > 0 ? 0.9 : 1.1;
    setTransform(t => {
      const newScale = Math.max(0.2, Math.min(3, t.scale * delta));
      return { ...t, scale: newScale };
    });
  }, []);

  // Pan handlers
  const onMouseDown = useCallback((e: MouseEvent) => {
    if (e.button !== 0) return;
    dragging.current = { startX: e.clientX, startY: e.clientY, origX: transform.x, origY: transform.y };
  }, [transform]);

  const onMouseMove = useCallback((e: MouseEvent) => {
    if (!dragging.current) return;
    const dx = e.clientX - dragging.current.startX;
    const dy = e.clientY - dragging.current.startY;
    setTransform(t => ({ ...t, x: dragging.current!.origX + dx, y: dragging.current!.origY + dy }));
  }, []);

  const onMouseUp = useCallback(() => { dragging.current = null; }, []);

  function navigateToResource(service: string, resourceName: string) {
    location.hash = `/resources?service=${encodeURIComponent(service)}&resource=${encodeURIComponent(resourceName)}`;
  }

  // Reset pan/zoom when switching tabs
  const handleTabChange = useCallback((mode: GroupMode) => {
    setGroupMode(mode);
    setTransform({ x: 0, y: 0, scale: 1 });
    setHoveredNode(null);
    setHoveredEdge(null);
    setHoveredCluster(null);
  }, []);

  // Minimap
  const minimapW = 180;
  const minimapH = 110;
  const minimapScale = useMemo(() => {
    return Math.min(minimapW / svgW, minimapH / svgH);
  }, [svgW, svgH]);

  // Node lookup by id for group membership (uses layout groups, not server groups)
  const nodeGroupMap = useMemo(() => {
    if (!layout) return new Map<string, string>();
    const m = new Map<string, string>();
    for (const n of layout.positionedNodes) {
      m.set(n.id, n.group);
    }
    return m;
  }, [layout]);

  // Group color lookup for current layout
  const groupColorMap = useMemo(() => {
    if (!layout) return new Map<string, string>();
    const m = new Map<string, string>();
    for (const g of layout.positionedGroups) {
      m.set(g.id, g.color);
    }
    return m;
  }, [layout]);

  if (!topoData || !layout) {
    return (
      <div>
        <div class="mb-6">
          <h1 class="page-title">Service Topology</h1>
          <p class="page-desc">Loading topology...</p>
        </div>
      </div>
    );
  }

  const edges = topoData.edges;
  const activeGroups = layout.activeGroups;

  return (
    <div>
      <div class="mb-6 flex items-center justify-between">
        <div>
          <h1 class="page-title">Service Topology</h1>
          <p class="page-desc">Unified resource graph with {topoData.nodes.length} resources across {activeGroups.length} groups</p>
        </div>
        <div style="display:flex;gap:8px;align-items:center;flex-wrap:wrap;max-width:780px">
          <label
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: '4px',
              padding: '3px 10px',
              borderRadius: '12px',
              border: '1.5px solid #64748B',
              background: showAll ? '#64748B20' : 'transparent',
              color: showAll ? '#64748B' : '#94A3B8',
              fontSize: '11px',
              fontWeight: 600,
              cursor: 'pointer',
              fontFamily: 'var(--font-sans)',
              userSelect: 'none',
            }}
          >
            <input
              type="checkbox"
              checked={showAll}
              onChange={() => setShowAll(v => !v)}
              style={{ width: '12px', height: '12px', margin: 0, cursor: 'pointer' }}
            />
            Show all
          </label>
          {/* Group legend chips */}
          {activeGroups.map(g => {
            const count = layout.positionedNodes.filter(n => n.group === g.id).length;
            return (
              <span
                key={g.id}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: '4px',
                  padding: '3px 10px',
                  borderRadius: '12px',
                  border: `1.5px solid ${g.color}`,
                  background: `${g.color}20`,
                  color: g.color,
                  fontSize: '11px',
                  fontWeight: 600,
                  fontFamily: 'var(--font-sans)',
                }}
              >
                <span style={{ width: '8px', height: '8px', borderRadius: '50%', background: g.color }} />
                {g.label}
                <span style={{ fontSize: '9px', opacity: 0.7 }}>({count})</span>
              </span>
            );
          })}
        </div>
      </div>

      {/* Tab bar */}
      <div class="topo-tabs">
        {(['service', 'domain', 'flow', 'hybrid'] as const).map(mode => (
          <button
            key={mode}
            class={`topo-tab ${groupMode === mode ? 'active' : ''}`}
            onClick={() => handleTabChange(mode)}
          >
            {TAB_LABELS[mode]}
          </button>
        ))}
      </div>

      <div class="card topology-container" style="position:relative;overflow:hidden">
        {/* biome-ignore lint: internal dashboard SVG */}
        <svg
          ref={svgRef}
          viewBox={`0 0 ${svgW} ${svgH}`}
          style="width:100%;height:100%;cursor:grab;user-select:none"
          onWheel={onWheel as any}
          onMouseDown={onMouseDown as any}
          onMouseMove={onMouseMove as any}
          onMouseUp={onMouseUp}
          onMouseLeave={onMouseUp}
        >
          <defs>
            <pattern id="topo-grid" width="30" height="30" patternUnits="userSpaceOnUse">
              <path d="M 30 0 L 0 0 0 30" fill="none" stroke="#E2E8F0" stroke-width="0.5" />
            </pattern>
            <marker id="topo-arrow" markerWidth="8" markerHeight="6" refX="8" refY="3" orient="auto">
              <polygon points="0 0, 8 3, 0 6" fill="#CBD5E1" />
            </marker>
            <marker id="topo-arrow-active" markerWidth="8" markerHeight="6" refX="8" refY="3" orient="auto">
              <polygon points="0 0, 8 3, 0 6" fill="#3B82F6" />
            </marker>
          </defs>

          <rect width={svgW} height={svgH} fill="url(#topo-grid)" />

          <g transform={`translate(${transform.x},${transform.y}) scale(${transform.scale})`}>

            {/* Group cluster rectangles */}
            {layout.positionedGroups.map(g => {
              const dimmed = hoveredCluster && hoveredCluster !== g.id;
              return (
                <g
                  key={`cluster-${g.id}`}
                  style={{ opacity: dimmed ? 0.2 : 1, transition: 'opacity 0.2s' }}
                  onMouseEnter={() => setHoveredCluster(g.id)}
                  onMouseLeave={() => setHoveredCluster(null)}
                >
                  <rect
                    x={g.x}
                    y={g.y}
                    width={g.width}
                    height={g.height}
                    rx={14}
                    fill={`${g.color}0A`}
                    stroke={`${g.color}30`}
                    stroke-width="1"
                  />
                  {/* Colored top accent border */}
                  <rect
                    x={g.x}
                    y={g.y}
                    width={g.width}
                    height={4}
                    rx={2}
                    fill={g.color}
                    opacity="0.6"
                  />
                  {/* Group label */}
                  <text
                    x={g.x + 10}
                    y={g.y + 18}
                    font-size="10.5"
                    font-weight="700"
                    font-family="var(--font-sans)"
                    fill={g.color}
                    style={{ pointerEvents: 'none' }}
                  >
                    {g.label}
                  </text>
                  {/* Resource count badge */}
                  <g>
                    <rect
                      x={g.x + g.width - 32}
                      y={g.y + 8}
                      width={22}
                      height={16}
                      rx={8}
                      fill={g.color}
                      opacity="0.2"
                    />
                    <text
                      x={g.x + g.width - 21}
                      y={g.y + 16}
                      text-anchor="middle"
                      dominant-baseline="central"
                      font-size="9"
                      font-weight="700"
                      font-family="var(--font-mono)"
                      fill={g.color}
                      style={{ pointerEvents: 'none' }}
                    >
                      {g.nodeCount}
                    </text>
                  </g>
                </g>
              );
            })}

            {/* Edges (arrows) */}
            {edges.map((edge, i) => {
              const from = nodePos.get(edge.source);
              const to = nodePos.get(edge.target);
              if (!from || !to) return null;

              const highlighted = connectedEdges.has(i) || hoveredEdge === i;
              const dimmedByNode = hoveredNode && !highlighted;
              const dimmedByCluster = hoveredCluster && !(
                nodeGroupMap.get(edge.source) === hoveredCluster ||
                nodeGroupMap.get(edge.target) === hoveredCluster
              );
              const dimmed = dimmedByNode || dimmedByCluster;

              const isDashed = edge.discovered === 'traffic';
              const edgeColor = highlighted ? '#3B82F6' : '#CBD5E1';

              const dx = to.cx - from.cx;
              const dy = to.cy - from.cy;
              const dist = Math.sqrt(dx * dx + dy * dy) || 1;
              const startX = from.cx + (dx / dist) * (RES_W / 2 + 3);
              const startY = from.cy + (dy / dist) * (RES_H / 2 + 2);
              const endX = to.cx - (dx / dist) * (RES_W / 2 + 3);
              const endY = to.cy - (dy / dist) * (RES_H / 2 + 2);

              const path = bezierPath(startX, startY, endX, endY);
              const midX = (startX + endX) / 2;
              const midY = (startY + endY) / 2 - 6;

              // Build label with latency info
              const avgMs = (edge as any).avgLatencyMs || 0;
              const callCount = (edge as any).callCount || 0;
              const latencyStr = avgMs > 0
                ? avgMs < 1 ? '<1ms' : avgMs < 1000 ? `${Math.round(avgMs)}ms` : `${(avgMs/1000).toFixed(1)}s`
                : '';
              const labelText = latencyStr
                ? `${edge.label || ''} ${latencyStr}`
                : (edge.label || '');
              const labelW = labelText.length * 5 + 12;

              // Thicker lines for more traffic
              const baseWidth = callCount > 50 ? 2.5 : callCount > 10 ? 1.8 : callCount > 0 ? 1.2 : 0.8;

              return (
                <g
                  key={`e-${i}`}
                  style={{ opacity: dimmed ? 0.08 : 1, transition: 'opacity 0.2s', cursor: 'default' }}
                  onMouseEnter={() => setHoveredEdge(i)}
                  onMouseLeave={() => setHoveredEdge(null)}
                >
                  <path
                    d={path}
                    fill="none"
                    stroke={edgeColor}
                    stroke-width={highlighted ? baseWidth + 1 : baseWidth}
                    stroke-dasharray={isDashed ? '5 3' : 'none'}
                    marker-end={highlighted ? 'url(#topo-arrow-active)' : 'url(#topo-arrow)'}
                  />
                  {labelText && (
                    <>
                      <rect
                        x={midX - labelW / 2}
                        y={midY - 7}
                        width={labelW}
                        height={14}
                        rx={3}
                        fill="white"
                        stroke={highlighted ? '#3B82F6' : '#E2E8F0'}
                        stroke-width="0.5"
                      />
                      <text
                        x={midX}
                        y={midY + 1}
                        text-anchor="middle"
                        dominant-baseline="central"
                        font-size="8"
                        font-family="var(--font-sans)"
                        fill={highlighted ? '#3B82F6' : '#94A3B8'}
                        style={{ pointerEvents: 'none' }}
                      >
                        {labelText}
                      </text>
                    </>
                  )}
                  {/* Edge tooltip on hover */}
                  {hoveredEdge === i && (
                    <g style={{ pointerEvents: 'none' }}>
                      <rect
                        x={midX - 60}
                        y={midY - 30}
                        width={120}
                        height={22}
                        rx={4}
                        fill="#0F172A"
                        opacity="0.92"
                      />
                      <text
                        x={midX}
                        y={midY - 19}
                        text-anchor="middle"
                        dominant-baseline="central"
                        font-size="9"
                        font-family="var(--font-sans)"
                        fill="white"
                      >
                        {edge.label || edge.type}{callCount > 0 ? ` \u00B7 ${callCount} calls \u00B7 avg ${latencyStr}` : ''} ({edge.discovered})
                      </text>
                    </g>
                  )}
                </g>
              );
            })}

            {/* Animated packets (Packet Tracer style) */}
            {packets.map(pkt => {
              const edge = edges[pkt.edgeIdx];
              if (!edge) return null;
              const from = nodePos.get(edge.source);
              const to = nodePos.get(edge.target);
              if (!from || !to) return null;
              const dx = to.cx - from.cx;
              const dy = to.cy - from.cy;
              const dist = Math.sqrt(dx * dx + dy * dy) || 1;
              const sx = from.cx + (dx / dist) * (RES_W / 2 + 3);
              const sy = from.cy + (dy / dist) * (RES_H / 2 + 2);
              const ex = to.cx - (dx / dist) * (RES_W / 2 + 3);
              const ey = to.cy - (dy / dist) * (RES_H / 2 + 2);
              const pth = bezierPath(sx, sy, ex, ey);
              const groupColor = groupColorMap.get(edge.source.split(':')[0] === 'external' ? 'API' : (topoData?.nodes.find(n => n.id === edge.source)?.group || '')) || '#3B82F6';
              return (
                <circle
                  key={`pkt-${pkt.id}`}
                  r={5}
                  fill={groupColor}
                  opacity={0.9}
                  style={{ filter: `drop-shadow(0 0 4px ${groupColor})` }}
                >
                  <animateMotion
                    dur="1.2s"
                    repeatCount="1"
                    fill="freeze"
                    {...{ path: pth.replace('M ', 'M').replace(' C ', 'C') } as any}
                  />
                </circle>
              );
            })}

            {/* Resource nodes */}
            {layout.positionedNodes.map(n => {
              const groupColor = groupColorMap.get(n.group) || '#94A3B8';
              const isHovered = hoveredNode === n.id;
              const isSelected = selectedNode?.id === n.id;
              const dimmedByNode = hoveredNode && !connectedNodes.has(n.id);
              const dimmedByCluster = hoveredCluster && n.group !== hoveredCluster;
              const dimmed = dimmedByNode || dimmedByCluster;
              const truncated = n.label.length > 18 ? n.label.slice(0, 17) + '\u2026' : n.label;

              const isPulsing = pulsingNodes.has(n.id);

              return (
                <g
                  key={n.id}
                  style={{
                    cursor: 'pointer',
                    opacity: dimmed ? 0.15 : 1,
                    transition: 'opacity 0.2s',
                  }}
                  onMouseEnter={() => { setHoveredNode(n.id); setHoveredCluster(null); }}
                  onMouseLeave={() => setHoveredNode(null)}
                  onClick={(e: MouseEvent) => {
                    e.stopPropagation();
                    const topoNode: TopoNode = { id: n.id, label: n.label, service: n.service, type: n.type, group: n.group };
                    setSelectedNode(topoNode);
                  }}
                >
                  {/* Pulse glow on live traffic */}
                  {isPulsing && (
                    <rect
                      x={n.x - 4}
                      y={n.y - 4}
                      width={RES_W + 8}
                      height={RES_H + 8}
                      rx={14}
                      fill="none"
                      stroke={groupColor}
                      stroke-width="3"
                      opacity="0.6"
                      style={{ animation: 'topo-pulse 1.5s ease-out' }}
                    />
                  )}
                  {/* Shadow */}
                  <rect
                    x={n.x + 1}
                    y={n.y + 2}
                    width={RES_W}
                    height={RES_H}
                    rx={10}
                    fill="rgba(0,0,0,0.06)"
                  />
                  {/* Node body */}
                  <rect
                    x={n.x}
                    y={n.y}
                    width={RES_W}
                    height={RES_H}
                    rx={10}
                    fill={isSelected ? `${groupColor}25` : isPulsing ? `${groupColor}18` : isHovered ? `${groupColor}20` : 'white'}
                    stroke={isSelected ? groupColor : isPulsing ? groupColor : isHovered ? groupColor : `${groupColor}50`}
                    stroke-width={isSelected ? 3 : isPulsing ? 2.5 : isHovered ? 2.5 : 1.5}
                    style={{ transition: 'all 0.15s ease' }}
                  />
                  {/* Left color accent */}
                  <rect
                    x={n.x}
                    y={n.y + 8}
                    width={4}
                    height={RES_H - 16}
                    rx={2}
                    fill={groupColor}
                    opacity={isHovered ? 1 : 0.6}
                  />
                  {/* Resource name */}
                  <text
                    x={n.x + 14}
                    y={n.y + RES_H / 2 - 4}
                    dominant-baseline="central"
                    font-size="11.5"
                    font-weight="600"
                    font-family="var(--font-sans)"
                    fill="#1E293B"
                    style={{ pointerEvents: 'none' }}
                  >
                    {truncated}
                  </text>
                  {/* Service type label */}
                  <text
                    x={n.x + 14}
                    y={n.y + RES_H / 2 + 9}
                    dominant-baseline="central"
                    font-size="8.5"
                    font-family="var(--font-sans)"
                    fill="#94A3B8"
                    style={{ pointerEvents: 'none' }}
                  >
                    {n.type}
                  </text>

                  {/* Tooltip on hover */}
                  {isHovered && (
                    <g style={{ pointerEvents: 'none' }}>
                      <rect
                        x={n.x + RES_W / 2 - 75}
                        y={n.y - 36}
                        width={150}
                        height={30}
                        rx={5}
                        fill="#0F172A"
                        opacity="0.92"
                      />
                      <text
                        x={n.x + RES_W / 2}
                        y={n.y - 25}
                        text-anchor="middle"
                        font-size="9.5"
                        font-weight="600"
                        font-family="var(--font-sans)"
                        fill="white"
                      >
                        {n.label}
                      </text>
                      <text
                        x={n.x + RES_W / 2}
                        y={n.y - 12}
                        text-anchor="middle"
                        font-size="8.5"
                        font-family="var(--font-sans)"
                        fill="#94A3B8"
                      >
                        {n.service} | {n.type}
                      </text>
                    </g>
                  )}
                </g>
              );
            })}
          </g>

          {/* Minimap */}
          <g transform={`translate(${svgW - minimapW - 16}, ${svgH - minimapH - 16})`}>
            <rect
              width={minimapW}
              height={minimapH}
              rx={4}
              fill="white"
              stroke="#E2E8F0"
              stroke-width="1"
              opacity="0.92"
            />
            <g transform={`scale(${minimapScale})`}>
              {/* Minimap clusters */}
              {layout.positionedGroups.map(g => (
                <rect
                  key={`mm-c-${g.id}`}
                  x={g.x}
                  y={g.y}
                  width={g.width}
                  height={g.height}
                  rx={4}
                  fill={g.color}
                  opacity={0.15}
                />
              ))}
              {/* Minimap nodes */}
              {layout.positionedNodes.map(n => {
                const gc = groupColorMap.get(n.group) || '#94A3B8';
                return (
                  <rect
                    key={`mm-${n.id}`}
                    x={n.x}
                    y={n.y}
                    width={RES_W}
                    height={RES_H}
                    rx={2}
                    fill={gc}
                    opacity={0.5}
                  />
                );
              })}
              {/* Minimap edges */}
              {edges.map((edge, i) => {
                const from = nodePos.get(edge.source);
                const to = nodePos.get(edge.target);
                if (!from || !to) return null;
                return (
                  <line
                    key={`mme-${i}`}
                    x1={from.cx}
                    y1={from.cy}
                    x2={to.cx}
                    y2={to.cy}
                    stroke="#CBD5E1"
                    stroke-width="1"
                    opacity="0.3"
                  />
                );
              })}
            </g>
            {/* Viewport indicator */}
            <rect
              x={(-transform.x / transform.scale) * minimapScale}
              y={(-transform.y / transform.scale) * minimapScale}
              width={(svgW / transform.scale) * minimapScale}
              height={(svgH / transform.scale) * minimapScale}
              rx={2}
              fill="none"
              stroke="#3B82F6"
              stroke-width="1.5"
              opacity="0.6"
            />
          </g>
        </svg>
      </div>

      {selectedNode && (
        <NodeDetailDrawer
          node={selectedNode}
          edges={edges}
          nodes={topoData ? topoData.nodes : []}
          onClose={() => setSelectedNode(null)}
          onSelectNode={(n) => setSelectedNode(n)}
        />
      )}
    </div>
  );
}
