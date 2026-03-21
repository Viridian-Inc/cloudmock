import { useState, useEffect, useRef, useCallback, useMemo } from 'preact/hooks';
import { api } from '../api';
import type { SSEState } from '../hooks/useSSE';

// Internal developer dashboard -- SVG content is generated programmatically
// from our own service API data, not from user input.

// --- Types ---

interface TopoNode {
  id: string;
  label: string;
  category: string;
  layer: number;
  y: number;
  color: string;
  requests: number;
  active: boolean;
}

interface TopoEdge {
  from: string;
  to: string;
  label?: string;
  animated: boolean;
}

interface TopologyPageProps {
  sse: SSEState;
}

// --- Constants ---

const CATEGORY_COLORS: Record<string, string> = {
  Client:     '#6366F1',
  Compute:    '#3B82F6',
  Auth:       '#8B5CF6',
  Database:   '#10B981',
  Storage:    '#F59E0B',
  Messaging:  '#F97316',
  API:        '#06B6D4',
  Monitoring: '#EC4899',
  Config:     '#6366F1',
  Network:    '#14B8A6',
  Infra:      '#64748B',
  Email:      '#EF4444',
  Streaming:  '#A855F7',
  Other:      '#94A3B8',
};

// Map service names to their category and layer
const SERVICE_DEFS: Record<string, { category: string; layer: number }> = {
  'Client Apps':       { category: 'Client',     layer: 0 },
  'API Gateway':       { category: 'API',        layer: 1 },
  'Cognito':           { category: 'Auth',       layer: 1 },
  'Lambda':            { category: 'Compute',    layer: 2 },
  'IAM':               { category: 'Auth',       layer: 2 },
  'STS':               { category: 'Auth',       layer: 2 },
  'DynamoDB':          { category: 'Database',   layer: 3 },
  'SQS':               { category: 'Messaging',  layer: 3 },
  'SNS':               { category: 'Messaging',  layer: 3 },
  'EventBridge':       { category: 'Messaging',  layer: 3 },
  'S3':                { category: 'Storage',    layer: 3 },
  'SES':               { category: 'Email',      layer: 4 },
  'Secrets Manager':   { category: 'Config',     layer: 4 },
  'KMS':               { category: 'Config',     layer: 4 },
  'SSM':               { category: 'Config',     layer: 4 },
  'CloudWatch':        { category: 'Monitoring', layer: 4 },
  'CloudWatch Logs':   { category: 'Monitoring', layer: 4 },
  'RDS':               { category: 'Database',   layer: 3 },
  'VPC (EC2)':         { category: 'Network',    layer: 4 },
  'Route 53':          { category: 'Network',    layer: 4 },
  'CloudFormation':    { category: 'Infra',      layer: 4 },
  'ECS':               { category: 'Infra',      layer: 4 },
  'ECR':               { category: 'Infra',      layer: 4 },
  'Kinesis':           { category: 'Streaming',  layer: 4 },
  'Firehose':          { category: 'Streaming',  layer: 4 },
  'Step Functions':    { category: 'Other',      layer: 4 },
};

// Canonical name mapping: API service name -> topology display name
const NAME_MAP: Record<string, string> = {
  'lambda':            'Lambda',
  'cognito':           'Cognito',
  'cognito-idp':       'Cognito',
  'iam':               'IAM',
  'sts':               'STS',
  'dynamodb':          'DynamoDB',
  'rds':               'RDS',
  's3':                'S3',
  'sqs':               'SQS',
  'sns':               'SNS',
  'events':            'EventBridge',
  'eventbridge':       'EventBridge',
  'monitoring':        'CloudWatch',
  'apigateway':        'API Gateway',
  'apigatewayv2':      'API Gateway',
  'execute-api':       'API Gateway',
  'cloudwatch':        'CloudWatch',
  'logs':              'CloudWatch Logs',
  'secretsmanager':    'Secrets Manager',
  'ssm':               'SSM',
  'kms':               'KMS',
  'ses':               'SES',
  'sesv2':             'SES',
  'ec2':               'VPC (EC2)',
  'route53':           'Route 53',
  'cloudformation':    'CloudFormation',
  'ecs':               'ECS',
  'ecr':               'ECR',
  'kinesis':           'Kinesis',
  'firehose':          'Firehose',
  'states':            'Step Functions',
  'stepfunctions':     'Step Functions',
};

const KNOWN_EDGES: { from: string; to: string; label: string }[] = [
  { from: 'Client Apps',   to: 'API Gateway',      label: 'REST API' },
  { from: 'API Gateway',   to: 'Lambda',            label: 'proxy' },
  { from: 'API Gateway',   to: 'Cognito',           label: 'authorizer' },
  { from: 'Lambda',        to: 'DynamoDB',          label: 'read/write' },
  { from: 'Lambda',        to: 'SQS',               label: 'send messages' },
  { from: 'Lambda',        to: 'SNS',               label: 'publish' },
  { from: 'Lambda',        to: 'SES',               label: 'send email' },
  { from: 'Lambda',        to: 'Secrets Manager',   label: 'get secrets' },
  { from: 'Lambda',        to: 'KMS',               label: 'encrypt/decrypt' },
  { from: 'Lambda',        to: 'S3',                label: 'read/write' },
  { from: 'Lambda',        to: 'EventBridge',       label: 'put events' },
  { from: 'Lambda',        to: 'IAM',               label: 'assume role' },
  { from: 'DynamoDB',      to: 'Lambda',            label: 'streams trigger' },
  { from: 'SQS',           to: 'Lambda',            label: 'event source' },
  { from: 'SNS',           to: 'SQS',               label: 'fan-out' },
  { from: 'EventBridge',   to: 'SQS',               label: 'rule target' },
  { from: 'EventBridge',   to: 'Lambda',            label: 'rule target' },
  { from: 'S3',            to: 'SQS',               label: 'event notification' },
  { from: 'CloudWatch',    to: 'SNS',               label: 'alarm actions' },
];

// --- Service Icons (simple SVG paths centered at 0,0) ---

function ServiceIcon({ service, x, y, color }: { service: string; x: number; y: number; color: string }) {
  const s = 7; // half-size
  const iconColor = color;

  switch (service) {
    case 'Lambda':
      // Lambda symbol
      return (
        <g transform={`translate(${x},${y})`}>
          <path d={`M${-s} ${s} L0 ${-s} L${s} ${s} Z`} fill="none" stroke={iconColor} stroke-width="1.5" />
          <text x="0" y="2" text-anchor="middle" font-size="8" font-weight="700" fill={iconColor} style={{ pointerEvents: 'none' }}>{'λ'}</text>
        </g>
      );
    case 'DynamoDB':
    case 'RDS':
      // Database cylinder
      return (
        <g transform={`translate(${x},${y})`}>
          <ellipse cx="0" cy={-s + 2} rx={s} ry="3" fill="none" stroke={iconColor} stroke-width="1.3" />
          <line x1={-s} y1={-s + 2} x2={-s} y2={s - 2} stroke={iconColor} stroke-width="1.3" />
          <line x1={s} y1={-s + 2} x2={s} y2={s - 2} stroke={iconColor} stroke-width="1.3" />
          <ellipse cx="0" cy={s - 2} rx={s} ry="3" fill="none" stroke={iconColor} stroke-width="1.3" />
        </g>
      );
    case 'S3':
      // Bucket
      return (
        <g transform={`translate(${x},${y})`}>
          <path d={`M${-s} ${-s} L${-s + 2} ${s} L${s - 2} ${s} L${s} ${-s} Z`} fill="none" stroke={iconColor} stroke-width="1.3" />
          <line x1={-s} y1={-s + 3} x2={s} y2={-s + 3} stroke={iconColor} stroke-width="1.3" />
        </g>
      );
    case 'SQS':
      // Queue (stacked lines)
      return (
        <g transform={`translate(${x},${y})`}>
          <rect x={-s} y={-s} width={s * 2} height={s * 2} rx="2" fill="none" stroke={iconColor} stroke-width="1.3" />
          <line x1={-s + 2} y1={-2} x2={s - 2} y2={-2} stroke={iconColor} stroke-width="1.2" />
          <line x1={-s + 2} y1="2" x2={s - 2} y2="2" stroke={iconColor} stroke-width="1.2" />
        </g>
      );
    case 'SNS':
      // Bell / notification
      return (
        <g transform={`translate(${x},${y})`}>
          <circle cx="0" cy="0" r={s} fill="none" stroke={iconColor} stroke-width="1.3" />
          <circle cx="0" cy="0" r="2" fill={iconColor} />
          <line x1="0" y1={-s} x2={s - 1} y2={-s - 3} stroke={iconColor} stroke-width="1.2" />
          <line x1="0" y1={-s} x2={-s + 1} y2={-s - 3} stroke={iconColor} stroke-width="1.2" />
        </g>
      );
    case 'API Gateway':
      // Gateway arrows
      return (
        <g transform={`translate(${x},${y})`}>
          <path d={`M${-s} 0 L0 ${-s} L${s} 0 L0 ${s} Z`} fill="none" stroke={iconColor} stroke-width="1.3" />
          <line x1={-3} y1="0" x2="3" y2="0" stroke={iconColor} stroke-width="1.3" />
        </g>
      );
    case 'Cognito':
    case 'IAM':
    case 'STS':
      // Shield / lock
      return (
        <g transform={`translate(${x},${y})`}>
          <path d={`M0 ${-s} L${s} ${-s + 3} L${s} ${s - 3} L0 ${s} L${-s} ${s - 3} L${-s} ${-s + 3} Z`} fill="none" stroke={iconColor} stroke-width="1.3" />
          <circle cx="0" cy="-1" r="2" fill="none" stroke={iconColor} stroke-width="1.2" />
          <line x1="0" y1="1" x2="0" y2="4" stroke={iconColor} stroke-width="1.2" />
        </g>
      );
    case 'CloudWatch':
    case 'CloudWatch Logs':
      // Chart / graph
      return (
        <g transform={`translate(${x},${y})`}>
          <rect x={-s} y={-s} width={s * 2} height={s * 2} rx="1" fill="none" stroke={iconColor} stroke-width="1.3" />
          <polyline points={`${-s + 2},${s - 3} ${-2},0 ${2},${s - 5} ${s - 2},${-s + 3}`} fill="none" stroke={iconColor} stroke-width="1.3" />
        </g>
      );
    case 'EventBridge':
      // Event bus
      return (
        <g transform={`translate(${x},${y})`}>
          <circle cx="0" cy="0" r={s} fill="none" stroke={iconColor} stroke-width="1.3" />
          <line x1={-s} y1="0" x2={s} y2="0" stroke={iconColor} stroke-width="1" />
          <line x1="0" y1={-s} x2="0" y2={s} stroke={iconColor} stroke-width="1" />
        </g>
      );
    case 'SES':
      // Envelope
      return (
        <g transform={`translate(${x},${y})`}>
          <rect x={-s} y={-s + 2} width={s * 2} height={s * 2 - 4} rx="1" fill="none" stroke={iconColor} stroke-width="1.3" />
          <polyline points={`${-s},${-s + 2} 0,2 ${s},${-s + 2}`} fill="none" stroke={iconColor} stroke-width="1.3" />
        </g>
      );
    case 'Secrets Manager':
    case 'KMS':
      // Key
      return (
        <g transform={`translate(${x},${y})`}>
          <circle cx={-2} cy={-2} r="3" fill="none" stroke={iconColor} stroke-width="1.3" />
          <line x1="1" y1="1" x2={s} y2={s} stroke={iconColor} stroke-width="1.3" />
          <line x1={s - 2} y1={s} x2={s} y2={s - 2} stroke={iconColor} stroke-width="1.2" />
        </g>
      );
    case 'SSM':
      // Settings gear (simplified)
      return (
        <g transform={`translate(${x},${y})`}>
          <circle cx="0" cy="0" r={s - 2} fill="none" stroke={iconColor} stroke-width="1.3" />
          <circle cx="0" cy="0" r="2" fill={iconColor} />
        </g>
      );
    case 'Step Functions':
      // Flow nodes
      return (
        <g transform={`translate(${x},${y})`}>
          <circle cx={-3} cy={-3} r="2.5" fill="none" stroke={iconColor} stroke-width="1.2" />
          <circle cx="3" cy="3" r="2.5" fill="none" stroke={iconColor} stroke-width="1.2" />
          <line x1={-1} y1={-1} x2="1" y2="1" stroke={iconColor} stroke-width="1.2" />
        </g>
      );
    case 'Client Apps':
      // Browser window
      return (
        <g transform={`translate(${x},${y})`}>
          <rect x={-s} y={-s} width={s * 2} height={s * 2} rx="2" fill="none" stroke={iconColor} stroke-width="1.3" />
          <line x1={-s} y1={-s + 4} x2={s} y2={-s + 4} stroke={iconColor} stroke-width="1" />
          <circle cx={-s + 3} cy={-s + 2} r="1" fill={iconColor} />
        </g>
      );
    default:
      // Generic box
      return (
        <g transform={`translate(${x},${y})`}>
          <rect x={-s} y={-s} width={s * 2} height={s * 2} rx="3" fill="none" stroke={iconColor} stroke-width="1.3" />
        </g>
      );
  }
}

// --- Layout ---

const LAYER_X = [80, 280, 480, 720, 960];
const NODE_W = 160;
const NODE_H = 48;
const NODE_RX = 10;
const V_GAP = 80; // vertical gap between nodes

// Layer labels for category headers
const LAYER_HEADERS: Record<number, string> = {
  0: 'Client',
  1: 'API',
  2: 'Compute',
  3: 'Data & Messaging',
  4: 'Config & Monitoring',
};

function buildLayout(
  activeNames: Set<string>,
  requestCounts: Map<string, number>,
  showAll: boolean,
): { nodes: TopoNode[]; edges: TopoEdge[] } {
  // Collect service names referenced by edges
  const edgeServices = new Set<string>();
  for (const e of KNOWN_EDGES) {
    edgeServices.add(e.from);
    edgeServices.add(e.to);
  }

  // Always include Client Apps as entry point
  const included = new Set<string>(['Client Apps']);

  // Determine which services to include
  for (const [name] of Object.entries(SERVICE_DEFS)) {
    const hasRequests = (requestCounts.get(name) || 0) > 0;
    const isEdgeMember = edgeServices.has(name);
    const isActive = activeNames.has(name);

    if (showAll) {
      included.add(name);
    } else if (hasRequests || (isActive && isEdgeMember)) {
      included.add(name);
    }
  }

  // If a service is included via requests, also include services connected by known edges
  // so the dependency map makes sense
  for (const e of KNOWN_EDGES) {
    if (included.has(e.from) && edgeServices.has(e.to) && SERVICE_DEFS[e.to]) {
      // only add the other end if it has requests or showAll
      if (showAll || included.has(e.to)) {
        // already handled
      }
    }
  }

  // Gather nodes per layer
  const layerNodes: Map<number, string[]> = new Map();
  for (const name of included) {
    const def = SERVICE_DEFS[name];
    if (!def) continue;
    const arr = layerNodes.get(def.layer) || [];
    arr.push(name);
    layerNodes.set(def.layer, arr);
  }

  // Sort nodes within each layer for consistent ordering
  const layerOrder: Record<number, string[]> = {
    0: ['Client Apps'],
    1: ['API Gateway', 'Cognito'],
    2: ['Lambda', 'IAM', 'STS'],
    3: ['DynamoDB', 'SQS', 'SNS', 'EventBridge', 'S3', 'RDS'],
    4: ['SES', 'Secrets Manager', 'KMS', 'SSM', 'CloudWatch', 'CloudWatch Logs', 'VPC (EC2)', 'Route 53', 'CloudFormation', 'ECS', 'ECR', 'Kinesis', 'Firehose', 'Step Functions'],
  };

  const nodes: TopoNode[] = [];

  for (const [layer, names] of layerNodes) {
    const order = layerOrder[layer] || names;
    const sorted = order.filter(n => names.includes(n));
    const count = sorted.length;
    const totalH = count * (NODE_H + V_GAP) - V_GAP;
    const startY = Math.max(60, 350 - totalH / 2);

    sorted.forEach((name, i) => {
      const def = SERVICE_DEFS[name]!;
      nodes.push({
        id: name,
        label: name,
        category: def.category,
        layer: def.layer,
        y: startY + i * (NODE_H + V_GAP),
        color: CATEGORY_COLORS[def.category] || '#94A3B8',
        requests: requestCounts.get(name) || 0,
        active: activeNames.has(name),
      });
    });
  }

  const nodeIds = new Set(nodes.map(n => n.id));

  // Only include edges where both endpoints exist
  const edges: TopoEdge[] = KNOWN_EDGES
    .filter(e => nodeIds.has(e.from) && nodeIds.has(e.to))
    .map(e => ({
      from: e.from,
      to: e.to,
      label: e.label,
      animated: false,
    }));

  return { nodes, edges };
}

// --- SVG helpers ---

function nodeX(layer: number) {
  return LAYER_X[layer] || (80 + layer * 200);
}

function bezierPath(x1: number, y1: number, x2: number, y2: number): string {
  const dx = Math.abs(x2 - x1) * 0.5;
  return `M ${x1} ${y1} C ${x1 + dx} ${y1}, ${x2 - dx} ${y2}, ${x2} ${y2}`;
}

// --- Component ---

export function TopologyPage({ sse }: TopologyPageProps) {
  const [services, setServices] = useState<any[]>([]);
  const [stats, setStats] = useState<any>(null);
  const [hoveredNode, setHoveredNode] = useState<string | null>(null);
  const [enabledCategories, setEnabledCategories] = useState<Set<string>>(new Set(Object.keys(CATEGORY_COLORS)));
  const [pulsing, setPulsing] = useState<Map<string, number>>(new Map());
  const [transform, setTransform] = useState({ x: 0, y: 0, scale: 1 });
  const [showAll, setShowAll] = useState(false);
  const [expandedNode, setExpandedNode] = useState<string | null>(null);
  const [nodeResources, setNodeResources] = useState<Record<string, any[]>>({});
  const [loadingResources, setLoadingResources] = useState<Set<string>>(new Set());
  const svgRef = useRef<SVGSVGElement>(null);
  const dragging = useRef<{ startX: number; startY: number; origX: number; origY: number } | null>(null);

  // Fetch data
  useEffect(() => {
    api('/api/services').then(setServices).catch(() => {});
    api('/api/stats').then(setStats).catch(() => {});
    const iv = setInterval(() => {
      api('/api/stats').then(setStats).catch(() => {});
    }, 5000);
    return () => clearInterval(iv);
  }, []);

  // Process active services -- canonicalize and deduplicate
  const { activeNames, requestCounts } = useMemo(() => {
    const activeNames = new Set<string>();
    const requestCounts = new Map<string, number>();

    for (const svc of services) {
      const key = svc.name?.toLowerCase?.() || svc.name || '';
      const canonical = NAME_MAP[key] || NAME_MAP[svc.name];
      if (canonical && SERVICE_DEFS[canonical]) {
        activeNames.add(canonical);
      }
    }

    if (stats?.services) {
      for (const [key, val] of Object.entries(stats.services)) {
        const canonical = NAME_MAP[key.toLowerCase()] || NAME_MAP[key];
        if (canonical) {
          const prev = requestCounts.get(canonical) || 0;
          requestCounts.set(canonical, prev + ((val as any).total || 0));
        }
      }
    }

    return { activeNames, requestCounts };
  }, [services, stats]);

  // Build layout
  const { nodes, edges } = useMemo(
    () => buildLayout(activeNames, requestCounts, showAll),
    [activeNames, requestCounts, showAll]
  );

  // SSE live traffic pulse
  useEffect(() => {
    if (!sse.events.length) return;
    const latest = sse.events[0];
    if (!latest?.data?.service) return;
    const svcName = NAME_MAP[latest.data.service?.toLowerCase()] || latest.data.service;
    if (!svcName) return;

    setPulsing(prev => {
      const next = new Map(prev);
      next.set(svcName, Date.now());
      return next;
    });

    const timer = setTimeout(() => {
      setPulsing(prev => {
        const next = new Map(prev);
        next.delete(svcName);
        return next;
      });
    }, 1500);

    return () => clearTimeout(timer);
  }, [sse.events.length]);

  // Zoom handler
  const onWheel = useCallback((e: WheelEvent) => {
    e.preventDefault();
    const delta = e.deltaY > 0 ? 0.9 : 1.1;
    setTransform(t => {
      const newScale = Math.max(0.3, Math.min(3, t.scale * delta));
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

  // Filter nodes by category
  const filteredNodes = useMemo(
    () => nodes.filter(n => enabledCategories.has(n.category)),
    [nodes, enabledCategories]
  );
  const filteredNodeIds = useMemo(() => new Set(filteredNodes.map(n => n.id)), [filteredNodes]);

  const filteredEdges = useMemo(
    () => edges.filter(e => filteredNodeIds.has(e.from) && filteredNodeIds.has(e.to)),
    [edges, filteredNodeIds]
  );

  // Connected edges for hover highlight
  const connectedEdges = useMemo(() => {
    if (!hoveredNode) return new Set<number>();
    const set = new Set<number>();
    filteredEdges.forEach((e, i) => {
      if (e.from === hoveredNode || e.to === hoveredNode) set.add(i);
    });
    return set;
  }, [hoveredNode, filteredEdges]);

  const connectedNodes = useMemo(() => {
    if (!hoveredNode) return new Set<string>();
    const set = new Set<string>([hoveredNode]);
    filteredEdges.forEach(e => {
      if (e.from === hoveredNode) set.add(e.to);
      if (e.to === hoveredNode) set.add(e.from);
    });
    return set;
  }, [hoveredNode, filteredEdges]);

  function toggleCategory(cat: string) {
    setEnabledCategories(prev => {
      const next = new Set(prev);
      if (next.has(cat)) next.delete(cat);
      else next.add(cat);
      return next;
    });
  }

  async function handleNodeClick(serviceName: string) {
    if (expandedNode === serviceName) {
      setExpandedNode(null);
      return;
    }

    setExpandedNode(serviceName);

    // Fetch resources if not already cached or loading
    if (!(serviceName in nodeResources) && !loadingResources.has(serviceName)) {
      setLoadingResources(prev => new Set([...prev, serviceName]));
      try {
        const res = await api(`/api/resources/${encodeURIComponent(serviceName)}`);
        setNodeResources(prev => ({ ...prev, [serviceName]: res.resources || [] }));
      } catch {
        setNodeResources(prev => ({ ...prev, [serviceName]: [] }));
      } finally {
        setLoadingResources(prev => {
          const next = new Set(prev);
          next.delete(serviceName);
          return next;
        });
      }
    }
  }

  function navigateToResource(serviceName: string, resource: any) {
    const id = resource.name || resource.id || (typeof resource === 'string' ? resource : '');
    location.hash = `/resources?service=${encodeURIComponent(serviceName)}&resource=${encodeURIComponent(id)}`;
  }

  // SVG dimensions
  const svgW = 1200;
  const svgH = 750;

  // Lookup positions
  const nodePos = useMemo(() => {
    const map: Record<string, { cx: number; cy: number }> = {};
    for (const n of filteredNodes) {
      map[n.id] = { cx: nodeX(n.layer) + NODE_W / 2, cy: n.y + NODE_H / 2 };
    }
    return map;
  }, [filteredNodes]);

  // Unique categories present
  const presentCategories = useMemo(() => {
    const cats = new Set<string>();
    nodes.forEach(n => cats.add(n.category));
    return Array.from(cats);
  }, [nodes]);

  // Unique layers present for headers
  const presentLayers = useMemo(() => {
    const layers = new Set<number>();
    filteredNodes.forEach(n => layers.add(n.layer));
    return Array.from(layers).sort();
  }, [filteredNodes]);

  // Minimap computation
  const minimapW = 180;
  const minimapH = 100;
  const minimapScale = useMemo(() => {
    return Math.min(minimapW / svgW, minimapH / svgH);
  }, []);

  return (
    <div>
      <div class="mb-6 flex items-center justify-between">
        <div>
          <h1 class="page-title">Service Topology</h1>
          <p class="page-desc">Service dependency map with live traffic</p>
        </div>
        <div style="display:flex;gap:8px;align-items:center;flex-wrap:wrap;max-width:700px">
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
          {presentCategories.map(cat => (
            <button
              key={cat}
              onClick={() => toggleCategory(cat)}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: '4px',
                padding: '3px 10px',
                borderRadius: '12px',
                border: `1.5px solid ${CATEGORY_COLORS[cat]}`,
                background: enabledCategories.has(cat) ? CATEGORY_COLORS[cat] + '20' : 'transparent',
                color: enabledCategories.has(cat) ? CATEGORY_COLORS[cat] : '#94A3B8',
                fontSize: '11px',
                fontWeight: 600,
                cursor: 'pointer',
                fontFamily: 'var(--font-sans)',
                opacity: enabledCategories.has(cat) ? 1 : 0.5,
                transition: 'all 0.15s ease',
              }}
            >
              <span style={{
                width: '8px', height: '8px', borderRadius: '50%',
                background: enabledCategories.has(cat) ? CATEGORY_COLORS[cat] : '#94A3B8',
              }} />
              {cat}
            </button>
          ))}
        </div>
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
          {/* Background grid */}
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
            {/* Pulse animation */}
            <filter id="topo-pulse">
              <feGaussianBlur in="SourceGraphic" stdDeviation="3" />
            </filter>
          </defs>
          <rect
            width={svgW}
            height={svgH}
            fill="url(#topo-grid)"
            onClick={() => setExpandedNode(null)}
            style={{ cursor: 'grab' }}
          />

          <g transform={`translate(${transform.x},${transform.y}) scale(${transform.scale})`}>
            {/* Layer / category headers */}
            {presentLayers.map(layer => {
              const lx = nodeX(layer);
              const header = LAYER_HEADERS[layer] || '';
              return header ? (
                <text
                  key={`header-${layer}`}
                  x={lx + NODE_W / 2}
                  y={30}
                  text-anchor="middle"
                  font-size="11"
                  font-weight="700"
                  font-family="var(--font-sans)"
                  fill="#94A3B8"
                  letter-spacing="0.5"
                  style={{ textTransform: 'uppercase', pointerEvents: 'none' } as any}
                >
                  {header}
                </text>
              ) : null;
            })}

            {/* Edges */}
            {filteredEdges.map((edge, i) => {
              const from = nodePos[edge.from];
              const to = nodePos[edge.to];
              if (!from || !to) return null;
              const highlighted = connectedEdges.has(i);
              const dimmed = hoveredNode && !highlighted;
              const edgeColor = highlighted ? '#3B82F6' : '#CBD5E1';

              // Offset arrow to stop at node border
              const dx = to.cx - from.cx;
              const dy = to.cy - from.cy;
              const dist = Math.sqrt(dx * dx + dy * dy) || 1;
              const endX = to.cx - (dx / dist) * (NODE_W / 2 + 4);
              const endY = to.cy - (dy / dist) * (NODE_H / 2 + 2);
              const startX = from.cx + (dx / dist) * (NODE_W / 2 + 4);
              const startY = from.cy + (dy / dist) * (NODE_H / 2 + 2);

              const path = bezierPath(startX, startY, endX, endY);
              const midX = (startX + endX) / 2;
              const midY = (startY + endY) / 2 - 10;

              // Estimate label width
              const labelText = edge.label || '';
              const labelW = labelText.length * 5.5 + 8;
              const labelH = 14;

              return (
                <g key={`e-${i}`} style={{ opacity: dimmed ? 0.15 : 1, transition: 'opacity 0.2s' }}>
                  <path
                    d={path}
                    fill="none"
                    stroke={edgeColor}
                    stroke-width={highlighted ? 2 : 1.2}
                    marker-end={highlighted ? 'url(#topo-arrow-active)' : 'url(#topo-arrow)'}
                  />
                  {labelText && (
                    <>
                      <rect
                        x={midX - labelW / 2}
                        y={midY - labelH / 2}
                        width={labelW}
                        height={labelH}
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
                        font-size="8.5"
                        font-family="var(--font-sans)"
                        fill={highlighted ? '#3B82F6' : '#94A3B8'}
                        style={{ pointerEvents: 'none' }}
                      >
                        {labelText}
                      </text>
                    </>
                  )}
                  {/* Animated dot for pulsing edges */}
                  {(pulsing.has(edge.from) || pulsing.has(edge.to)) && (
                    <circle r="3" fill={CATEGORY_COLORS[filteredNodes.find(n => n.id === edge.from)?.category || 'Other'] || '#3B82F6'}>
                      <animateMotion dur="1s" repeatCount="indefinite" {...{ path } as any} />
                    </circle>
                  )}
                </g>
              );
            })}

            {/* Nodes */}
            {filteredNodes.map(node => {
              const x = nodeX(node.layer);
              const y = node.y;
              const isPulsing = pulsing.has(node.id);
              const isHovered = hoveredNode === node.id;
              const dimmed = hoveredNode && !connectedNodes.has(node.id);
              const isClient = node.id === 'Client Apps';
              const fillOpacity = node.active ? 0.15 : 0.06;
              const borderColor = node.active ? node.color : '#CBD5E1';

              return (
                <g
                  key={node.id}
                  style={{
                    cursor: isClient ? 'default' : 'pointer',
                    opacity: dimmed ? 0.2 : 1,
                    transition: 'opacity 0.2s',
                  }}
                  onMouseEnter={() => setHoveredNode(node.id)}
                  onMouseLeave={() => setHoveredNode(null)}
                  onClick={(e: MouseEvent) => {
                    if (isClient) return;
                    e.stopPropagation();
                    handleNodeClick(node.id);
                  }}
                >
                  {/* Pulse glow */}
                  {isPulsing && (
                    <rect
                      x={x - 4}
                      y={y - 4}
                      width={NODE_W + 8}
                      height={NODE_H + 8}
                      rx={NODE_RX + 2}
                      fill={node.color}
                      opacity="0.25"
                      filter="url(#topo-pulse)"
                    >
                      <animate attributeName="opacity" values="0.25;0.08;0.25" dur="1s" repeatCount="indefinite" />
                    </rect>
                  )}

                  {/* Node rect */}
                  <rect
                    x={x}
                    y={y}
                    width={NODE_W}
                    height={NODE_H}
                    rx={NODE_RX}
                    fill={`${node.color}${Math.round(fillOpacity * 255).toString(16).padStart(2, '0')}`}
                    stroke={borderColor}
                    stroke-width={isHovered ? 2.5 : 1.5}
                    style={{ transition: 'stroke-width 0.15s' }}
                  />

                  {/* Service icon */}
                  <ServiceIcon
                    service={node.id}
                    x={x + 16}
                    y={y + NODE_H / 2}
                    color={node.color}
                  />

                  {/* Label */}
                  <text
                    x={x + 30}
                    y={y + NODE_H / 2 + 1}
                    dominant-baseline="central"
                    font-size="11.5"
                    font-weight="600"
                    font-family="var(--font-sans)"
                    fill="#1E293B"
                    style={{ pointerEvents: 'none' }}
                  >
                    {node.label}
                  </text>

                  {/* Request count badge */}
                  {node.requests > 0 && (
                    <>
                      <rect
                        x={x + NODE_W - 40}
                        y={y + 8}
                        width={34}
                        height={18}
                        rx={9}
                        fill={node.color}
                        opacity="0.18"
                      />
                      <text
                        x={x + NODE_W - 23}
                        y={y + 17}
                        dominant-baseline="central"
                        text-anchor="middle"
                        font-size="9.5"
                        font-weight="700"
                        font-family="var(--font-mono)"
                        fill={node.color}
                        style={{ pointerEvents: 'none' }}
                      >
                        {node.requests > 999 ? `${Math.round(node.requests / 1000)}k` : node.requests}
                      </text>
                    </>
                  )}

                  {/* Tooltip on hover */}
                  {isHovered && (
                    <g style={{ pointerEvents: 'none' }}>
                      <rect
                        x={x + NODE_W / 2 - 80}
                        y={y - 48}
                        width={160}
                        height={40}
                        rx={6}
                        fill="#0F172A"
                        opacity="0.92"
                      />
                      <text
                        x={x + NODE_W / 2}
                        y={y - 33}
                        text-anchor="middle"
                        font-size="10"
                        font-weight="600"
                        font-family="var(--font-sans)"
                        fill="white"
                      >
                        {node.label}
                      </text>
                      <text
                        x={x + NODE_W / 2}
                        y={y - 18}
                        text-anchor="middle"
                        font-size="9"
                        font-family="var(--font-sans)"
                        fill="#94A3B8"
                      >
                        {node.category} | {node.requests} req{node.active ? '' : ' (inactive)'}
                      </text>
                    </g>
                  )}
                </g>
              );
            })}

            {/* Expanded resource panels */}
            {expandedNode && (() => {
              const node = filteredNodes.find(n => n.id === expandedNode);
              if (!node) return null;
              const x = nodeX(node.layer);
              const y = node.y;
              const resources = nodeResources[expandedNode] || [];
              const isLoading = loadingResources.has(expandedNode);
              const panelW = 280;
              const rowH = 28;
              const headerH = 40;
              const footerH = 32;
              const listMaxH = 220;
              const listH = Math.min(listMaxH, resources.length * rowH);
              const panelH = isLoading ? headerH + 48 + footerH : headerH + listH + footerH;
              // Position panel to the right of the node, clamped within SVG
              let panelX = x + NODE_W + 12;
              if (panelX + panelW > 1180) panelX = x - panelW - 12;
              const panelY = Math.max(10, Math.min(y - 20, svgH - panelH - 10));

              return (
                <foreignObject
                  key={`panel-${expandedNode}`}
                  x={panelX}
                  y={panelY}
                  width={panelW}
                  height={panelH + 4}
                  style={{ overflow: 'visible' }}
                  onClick={(e: MouseEvent) => e.stopPropagation()}
                >
                  <div
                    class="topo-resource-panel"
                    style={{ width: `${panelW}px`, height: `${panelH}px` }}
                  >
                    <div class="topo-resource-header">
                      <span>{node.label} Resources</span>
                      <button
                        class="topo-resource-close"
                        onClick={(e: MouseEvent) => { e.stopPropagation(); setExpandedNode(null); }}
                      >×</button>
                    </div>
                    <div class="topo-resource-list">
                      {isLoading ? (
                        <div class="topo-resource-loading">Loading…</div>
                      ) : resources.length === 0 ? (
                        <div class="topo-resource-empty">No resources found</div>
                      ) : (
                        resources.map((r: any, i: number) => {
                          const name = r.name || r.id || (typeof r === 'string' ? r : JSON.stringify(r));
                          return (
                            <div
                              key={i}
                              class="topo-resource-item"
                              onClick={(e: MouseEvent) => { e.stopPropagation(); navigateToResource(expandedNode, r); }}
                            >
                              <span class="topo-resource-dot" />
                              <span class="topo-resource-name">{name}</span>
                            </div>
                          );
                        })
                      )}
                    </div>
                    <a
                      class="topo-resource-footer"
                      href={`#/resources?service=${encodeURIComponent(expandedNode)}`}
                      onClick={(e: MouseEvent) => e.stopPropagation()}
                    >
                      View all in Explorer →
                    </a>
                  </div>
                </foreignObject>
              );
            })()}
          </g>

          {/* Minimap */}
          <g transform={`translate(${svgW - minimapW - 16}, ${svgH - minimapH - 16})`}>
            {/* Minimap background */}
            <rect
              width={minimapW}
              height={minimapH}
              rx={4}
              fill="white"
              stroke="#E2E8F0"
              stroke-width="1"
              opacity="0.92"
            />
            {/* Minimap nodes */}
            <g transform={`scale(${minimapScale})`}>
              {filteredNodes.map(node => {
                const mx = nodeX(node.layer);
                const my = node.y;
                return (
                  <rect
                    key={`mm-${node.id}`}
                    x={mx}
                    y={my}
                    width={NODE_W}
                    height={NODE_H}
                    rx={2}
                    fill={node.color}
                    opacity={node.active ? 0.7 : 0.3}
                  />
                );
              })}
              {/* Minimap edges */}
              {filteredEdges.map((edge, i) => {
                const from = nodePos[edge.from];
                const to = nodePos[edge.to];
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
                    opacity="0.5"
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
    </div>
  );
}
