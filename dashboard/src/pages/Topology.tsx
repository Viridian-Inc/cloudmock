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
  expandedHeight?: number;
  resources?: any[];
}

interface TopoEdge {
  from: string;
  to: string;
  label?: string;
  animated: boolean;
}

// Expanded mode: each resource is its own node
interface ResourceNode {
  id: string;           // e.g. "Lambda::attendance-handler"
  resourceName: string; // e.g. "attendance-handler"
  service: string;      // e.g. "Lambda"
  category: string;
  layer: number;
  color: string;
  x: number;
  y: number;
}

interface ResourceEdge {
  from: string;  // ResourceNode id
  to: string;    // ResourceNode id
  label?: string;
}

interface ResourceCluster {
  service: string;
  category: string;
  color: string;
  layer: number;
  x: number;
  y: number;
  width: number;
  height: number;
  resourceCount: number;
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

// --- Expanded mode constants ---
const RES_NODE_W = 140;
const RES_NODE_H = 32;
const RES_V_GAP = 40;  // vertical gap between resource nodes within a cluster
const CLUSTER_PAD = 12;
const CLUSTER_LABEL_H = 18;

// Infer resource-level edges from naming conventions
function inferResourceEdges(
  resourcesByService: Record<string, any[]>,
  serviceEdges: TopoEdge[],
  activeServices: Set<string>,
): ResourceEdge[] {
  const edges: ResourceEdge[] = [];
  const seen = new Set<string>();

  function addEdge(from: string, to: string, label?: string) {
    const key = `${from}->${to}`;
    if (!seen.has(key)) {
      seen.add(key);
      edges.push({ from, to, label });
    }
  }

  // Build lookup: resource name -> resource node id
  const resourceLookup = new Map<string, { id: string; service: string }>();
  for (const [svc, resources] of Object.entries(resourcesByService)) {
    for (const r of resources) {
      const name = r.name || r.id || (typeof r === 'string' ? r : '');
      if (name) {
        resourceLookup.set(`${svc}::${name}`, { id: `${svc}::${name}`, service: svc });
      }
    }
  }

  // For each service-level edge, try to resolve to resource-level
  for (const sEdge of serviceEdges) {
    const fromResources = resourcesByService[sEdge.from] || [];
    const toResources = resourcesByService[sEdge.to] || [];

    if (fromResources.length === 0 && toResources.length === 0) continue;

    // Lambda -> DynamoDB: match by naming convention (handler prefix -> table name)
    if (sEdge.from === 'Lambda' && sEdge.to === 'DynamoDB' && fromResources.length > 0 && toResources.length > 0) {
      let matched = false;
      for (const fn of fromResources) {
        const fnName: string = fn.name || fn.id || '';
        // Extract prefix: "attendance-handler" -> "attendance"
        const prefix = fnName.replace(/[-_]?(handler|function|fn|processor|worker)$/i, '');
        for (const table of toResources) {
          const tableName: string = table.name || table.id || '';
          if (tableName === prefix || tableName.startsWith(prefix + '-') || prefix.startsWith(tableName)) {
            addEdge(`Lambda::${fnName}`, `DynamoDB::${tableName}`, 'read/write');
            matched = true;
          }
        }
      }
      // If some lambdas had no match, check for shared tables (like "enterprise")
      if (matched) continue;
    }

    // DynamoDB -> Lambda (streams): match by naming convention
    if (sEdge.from === 'DynamoDB' && sEdge.to === 'Lambda' && fromResources.length > 0 && toResources.length > 0) {
      let matched = false;
      for (const fn of toResources) {
        const fnName: string = fn.name || fn.id || '';
        if (fnName.includes('stream') || fnName.includes('sync')) {
          // Stream processor connects to all DynamoDB tables
          for (const table of fromResources) {
            const tableName: string = table.name || table.id || '';
            addEdge(`DynamoDB::${tableName}`, `Lambda::${fnName}`, 'stream');
            matched = true;
          }
        }
      }
      if (matched) continue;
    }

    // Lambda -> SQS: match by naming convention
    if (sEdge.from === 'Lambda' && sEdge.to === 'SQS' && fromResources.length > 0 && toResources.length > 0) {
      let matched = false;
      for (const fn of fromResources) {
        const fnName: string = fn.name || fn.id || '';
        const prefix = fnName.replace(/[-_]?(handler|function|fn|processor|worker)$/i, '');
        for (const q of toResources) {
          const qName: string = q.name || q.id || '';
          if (qName.includes(prefix) || prefix.includes(qName.replace(/-queue$/i, ''))) {
            addEdge(`Lambda::${fnName}`, `SQS::${qName}`, 'send');
            matched = true;
          }
        }
      }
      if (matched) continue;
    }

    // S3 -> SQS: connect all S3 buckets to SQS queues (event notifications)
    if (sEdge.from === 'S3' && sEdge.to === 'SQS') {
      for (const bucket of fromResources) {
        const bName: string = bucket.name || bucket.id || '';
        for (const q of toResources) {
          const qName: string = q.name || q.id || '';
          addEdge(`S3::${bName}`, `SQS::${qName}`, 'notification');
        }
      }
      continue;
    }

    // Fallback: fan out from all source resources to all target resources
    if (fromResources.length > 0 && toResources.length > 0) {
      for (const fr of fromResources) {
        const frName: string = fr.name || fr.id || '';
        for (const tr of toResources) {
          const trName: string = tr.name || tr.id || '';
          addEdge(`${sEdge.from}::${frName}`, `${sEdge.to}::${trName}`, sEdge.label);
        }
      }
    } else if (fromResources.length > 0) {
      // Target service has no resources, draw from all source resources to the service placeholder
      for (const fr of fromResources) {
        const frName: string = fr.name || fr.id || '';
        addEdge(`${sEdge.from}::${frName}`, `${sEdge.to}::__service__`, sEdge.label);
      }
    } else if (toResources.length > 0) {
      // Source service has no resources, draw from service placeholder to all target resources
      for (const tr of toResources) {
        const trName: string = tr.name || tr.id || '';
        addEdge(`${sEdge.from}::__service__`, `${sEdge.to}::${trName}`, sEdge.label);
      }
    }
  }

  return edges;
}

// Build expanded layout: each resource becomes its own node
function buildExpandedLayout(
  nodes: TopoNode[],
  edges: TopoEdge[],
  resourcesByService: Record<string, any[]>,
): { resourceNodes: ResourceNode[]; resourceEdges: ResourceEdge[]; clusters: ResourceCluster[] } {
  const resourceNodes: ResourceNode[] = [];
  const clusters: ResourceCluster[] = [];

  // Group nodes by layer
  const layerGroups = new Map<number, TopoNode[]>();
  for (const n of nodes) {
    const arr = layerGroups.get(n.layer) || [];
    arr.push(n);
    layerGroups.set(n.layer, arr);
  }

  // For each service, create resource nodes
  for (const [_layer, group] of layerGroups) {
    const sorted = [...group].sort((a, b) => a.y - b.y);
    let currentY = sorted[0]?.y ?? 60;

    for (const node of sorted) {
      const resources = resourcesByService[node.id] || [];
      const x = nodeX(node.layer);

      if (resources.length === 0) {
        // Service with no resources: create a single placeholder node
        resourceNodes.push({
          id: `${node.id}::__service__`,
          resourceName: node.label,
          service: node.id,
          category: node.category,
          layer: node.layer,
          color: node.color,
          x: x + CLUSTER_PAD,
          y: currentY + CLUSTER_LABEL_H + CLUSTER_PAD,
        });
        const clusterH = CLUSTER_LABEL_H + CLUSTER_PAD * 2 + RES_NODE_H;
        clusters.push({
          service: node.id,
          category: node.category,
          color: node.color,
          layer: node.layer,
          x,
          y: currentY,
          width: RES_NODE_W + CLUSTER_PAD * 2,
          height: clusterH,
          resourceCount: 0,
        });
        currentY += clusterH + 20;
      } else {
        const clusterContentH = resources.length * RES_NODE_H + (resources.length - 1) * (RES_V_GAP - RES_NODE_H);
        const clusterH = CLUSTER_LABEL_H + CLUSTER_PAD * 2 + clusterContentH;

        clusters.push({
          service: node.id,
          category: node.category,
          color: node.color,
          layer: node.layer,
          x,
          y: currentY,
          width: RES_NODE_W + CLUSTER_PAD * 2,
          height: clusterH,
          resourceCount: resources.length,
        });

        for (let i = 0; i < resources.length; i++) {
          const r = resources[i];
          const rName = r.name || r.id || (typeof r === 'string' ? r : JSON.stringify(r));
          resourceNodes.push({
            id: `${node.id}::${rName}`,
            resourceName: rName,
            service: node.id,
            category: node.category,
            layer: node.layer,
            color: node.color,
            x: x + CLUSTER_PAD,
            y: currentY + CLUSTER_LABEL_H + CLUSTER_PAD + i * RES_V_GAP,
          });
        }
        currentY += clusterH + 20;
      }
    }
  }

  // Build resource-level edges
  const resourceEdges = inferResourceEdges(resourcesByService, edges, new Set(nodes.map(n => n.id)));

  // Filter edges: only keep edges where both endpoints exist
  const nodeIds = new Set(resourceNodes.map(n => n.id));
  const validEdges = resourceEdges.filter(e => nodeIds.has(e.from) && nodeIds.has(e.to));

  return { resourceNodes, resourceEdges: validEdges, clusters };
}

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
  const [viewMode, setViewMode] = useState<'collapsed' | 'expanded'>('collapsed');
  const [expandedNode, setExpandedNode] = useState<string | null>(null);
  const [nodeResources, setNodeResources] = useState<Record<string, any[]>>({});
  const [loadingResources, setLoadingResources] = useState<Set<string>>(new Set());
  const [allResourcesLoaded, setAllResourcesLoaded] = useState(false);
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

  // Fetch all resources when switching to expanded mode
  useEffect(() => {
    if (viewMode !== 'expanded') return;
    if (allResourcesLoaded) return;

    const activeNodeNames = nodes.filter(n => n.active && n.id !== 'Client Apps').map(n => n.id);
    let cancelled = false;

    async function fetchAll() {
      for (const name of activeNodeNames) {
        if (cancelled) break;
        if (name in nodeResources) continue;
        setLoadingResources(prev => new Set([...prev, name]));
        try {
          const res = await api(`/api/resources/${encodeURIComponent(name)}`);
          if (!cancelled) {
            setNodeResources(prev => ({ ...prev, [name]: res.resources || [] }));
          }
        } catch {
          if (!cancelled) {
            setNodeResources(prev => ({ ...prev, [name]: [] }));
          }
        } finally {
          if (!cancelled) {
            setLoadingResources(prev => {
              const next = new Set(prev);
              next.delete(name);
              return next;
            });
          }
        }
      }
      if (!cancelled) setAllResourcesLoaded(true);
    }

    fetchAll();
    return () => { cancelled = true; };
  }, [viewMode, nodes]);

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

  // Expanded mode: build resource-level layout
  const expandedLayout = useMemo(() => {
    if (viewMode !== 'expanded') return null;
    return buildExpandedLayout(filteredNodes, filteredEdges, nodeResources);
  }, [viewMode, filteredNodes, filteredEdges, nodeResources]);

  // Resource node position lookup for expanded mode edges
  const resNodePos = useMemo(() => {
    if (!expandedLayout) return {};
    const map: Record<string, { cx: number; cy: number }> = {};
    for (const rn of expandedLayout.resourceNodes) {
      map[rn.id] = { cx: rn.x + RES_NODE_W / 2, cy: rn.y + RES_NODE_H / 2 };
    }
    return map;
  }, [expandedLayout]);

  // SVG dimensions
  const svgW = 1200;
  const maxNodeBottom = useMemo(() => {
    let max = 0;
    if (viewMode === 'expanded' && expandedLayout) {
      for (const c of expandedLayout.clusters) {
        max = Math.max(max, c.y + c.height + 60);
      }
    } else {
      for (const n of filteredNodes) {
        max = Math.max(max, n.y + NODE_H + 60);
      }
    }
    return max;
  }, [filteredNodes, viewMode, expandedLayout]);
  const svgH = Math.max(750, maxNodeBottom);

  // Lookup positions for collapsed mode
  const renderNodes = filteredNodes;
  const nodePos = useMemo(() => {
    const map: Record<string, { cx: number; cy: number }> = {};
    for (const n of renderNodes) {
      map[n.id] = { cx: nodeX(n.layer) + NODE_W / 2, cy: n.y + NODE_H / 2 };
    }
    return map;
  }, [renderNodes]);

  // Unique categories present
  const presentCategories = useMemo(() => {
    const cats = new Set<string>();
    nodes.forEach(n => cats.add(n.category));
    return Array.from(cats);
  }, [nodes]);

  // Unique layers present for headers
  const presentLayers = useMemo(() => {
    const layers = new Set<number>();
    renderNodes.forEach(n => layers.add(n.layer));
    return Array.from(layers).sort();
  }, [renderNodes]);

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
        <div style="display:flex;gap:8px;align-items:center;flex-wrap:wrap;max-width:780px">
          {/* Expanded / Collapsed toggle */}
          <div class="topo-view-toggle">
            <button
              class={viewMode === 'collapsed' ? 'active' : ''}
              onClick={() => { setViewMode('collapsed'); setExpandedNode(null); }}
            >Collapsed</button>
            <button
              class={viewMode === 'expanded' ? 'active' : ''}
              onClick={() => setViewMode('expanded')}
            >Expanded</button>
          </div>
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

            {/* ===== COLLAPSED MODE: service-level edges and nodes ===== */}
            {viewMode === 'collapsed' && filteredEdges.map((edge, i) => {
              const from = nodePos[edge.from];
              const to = nodePos[edge.to];
              if (!from || !to) return null;
              const highlighted = connectedEdges.has(i);
              const dimmed = hoveredNode && !highlighted;
              const edgeColor = highlighted ? '#3B82F6' : '#CBD5E1';

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
                  {(pulsing.has(edge.from) || pulsing.has(edge.to)) && (
                    <circle r="3" fill={CATEGORY_COLORS[filteredNodes.find(n => n.id === edge.from)?.category || 'Other'] || '#3B82F6'}>
                      <animateMotion dur="1s" repeatCount="indefinite" {...{ path } as any} />
                    </circle>
                  )}
                </g>
              );
            })}

            {viewMode === 'collapsed' && renderNodes.map(node => {
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

                  <rect
                    x={x}
                    y={y}
                    width={NODE_W}
                    height={NODE_H}
                    rx={NODE_RX}
                    fill={`${node.color}${Math.round(fillOpacity * 255).toString(16).padStart(2, '0')}`}
                    stroke={borderColor}
                    stroke-width={isHovered ? 2.5 : 1.5}
                    style={{ transition: 'all 0.3s ease' }}
                  />

                  <ServiceIcon
                    service={node.id}
                    x={x + 16}
                    y={y + NODE_H / 2}
                    color={node.color}
                  />

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
                        {node.category} | {node.requests} req | {(nodeResources[node.id] || []).length} resources
                      </text>
                    </g>
                  )}
                </g>
              );
            })}

            {/* Collapsed mode: resource panel on click */}
            {viewMode === 'collapsed' && expandedNode && (() => {
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

            {/* ===== EXPANDED MODE: resource-level nodes and edges ===== */}
            {viewMode === 'expanded' && expandedLayout && (
              <g>
                {/* Cluster group rectangles */}
                {expandedLayout.clusters.map(cluster => (
                  <g key={`cluster-${cluster.service}`}>
                    <rect
                      x={cluster.x}
                      y={cluster.y}
                      width={cluster.width}
                      height={cluster.height}
                      rx={12}
                      fill="none"
                      stroke={`${cluster.color}4D`}
                      stroke-width="1.5"
                      stroke-dasharray="6 3"
                    />
                    {/* Service name label above cluster */}
                    <text
                      x={cluster.x + 8}
                      y={cluster.y + 13}
                      font-size="10"
                      font-weight="600"
                      font-family="var(--font-sans)"
                      fill={cluster.color}
                      opacity="0.8"
                      style={{ pointerEvents: 'none' }}
                    >
                      {cluster.service}
                      {cluster.resourceCount > 0 ? ` (${cluster.resourceCount})` : ''}
                    </text>
                  </g>
                ))}

                {/* Resource-level edges */}
                {expandedLayout.resourceEdges.map((edge, i) => {
                  const from = resNodePos[edge.from];
                  const to = resNodePos[edge.to];
                  if (!from || !to) return null;

                  const dx = to.cx - from.cx;
                  const dy = to.cy - from.cy;
                  const dist = Math.sqrt(dx * dx + dy * dy) || 1;
                  const endX = to.cx - (dx / dist) * (RES_NODE_W / 2 + 3);
                  const endY = to.cy - (dy / dist) * (RES_NODE_H / 2 + 2);
                  const startX = from.cx + (dx / dist) * (RES_NODE_W / 2 + 3);
                  const startY = from.cy + (dy / dist) * (RES_NODE_H / 2 + 2);

                  const path = bezierPath(startX, startY, endX, endY);

                  return (
                    <g key={`re-${i}`} style={{ opacity: 0.6 }}>
                      <path
                        d={path}
                        fill="none"
                        stroke="#CBD5E1"
                        stroke-width="1"
                        marker-end="url(#topo-arrow)"
                      />
                    </g>
                  );
                })}

                {/* Resource nodes */}
                {expandedLayout.resourceNodes.map(rn => {
                  const isPlaceholder = rn.id.endsWith('::__service__');
                  const isHovered = hoveredNode === rn.id;
                  const displayName = isPlaceholder ? rn.resourceName : rn.resourceName;
                  const truncated = displayName.length > 16 ? displayName.slice(0, 15) + '\u2026' : displayName;

                  return (
                    <g
                      key={rn.id}
                      style={{ cursor: 'pointer' }}
                      onMouseEnter={() => setHoveredNode(rn.id)}
                      onMouseLeave={() => setHoveredNode(null)}
                      onClick={(e: MouseEvent) => {
                        e.stopPropagation();
                        if (!isPlaceholder) {
                          navigateToResource(rn.service, { name: rn.resourceName });
                        }
                      }}
                    >
                      <rect
                        x={rn.x}
                        y={rn.y}
                        width={RES_NODE_W}
                        height={RES_NODE_H}
                        rx={6}
                        fill={`${rn.color}1A`}
                        stroke={isHovered ? rn.color : `${rn.color}66`}
                        stroke-width={isHovered ? 2 : 1}
                        style={{ transition: 'all 0.15s ease' }}
                      />

                      {/* Small service icon badge in top-left */}
                      <g transform={`translate(${rn.x + 10}, ${rn.y + RES_NODE_H / 2}) scale(0.6) translate(${-(rn.x + 10)}, ${-(rn.y + RES_NODE_H / 2)})`}>
                        <ServiceIcon
                          service={rn.service}
                          x={rn.x + 10}
                          y={rn.y + RES_NODE_H / 2}
                          color={rn.color}
                        />
                      </g>

                      {/* Resource name label */}
                      <text
                        x={rn.x + 22}
                        y={rn.y + RES_NODE_H / 2 + 1}
                        dominant-baseline="central"
                        font-size="11"
                        font-family="var(--font-mono)"
                        fill="#334155"
                        style={{ pointerEvents: 'none' }}
                      >
                        {truncated}
                      </text>

                      {/* Tooltip on hover showing full name */}
                      {isHovered && displayName.length > 16 && (
                        <g style={{ pointerEvents: 'none' }}>
                          <rect
                            x={rn.x + RES_NODE_W / 2 - (displayName.length * 3.5 + 8)}
                            y={rn.y - 24}
                            width={displayName.length * 7 + 16}
                            height={20}
                            rx={4}
                            fill="#0F172A"
                            opacity="0.92"
                          />
                          <text
                            x={rn.x + RES_NODE_W / 2}
                            y={rn.y - 14}
                            text-anchor="middle"
                            dominant-baseline="central"
                            font-size="10"
                            font-family="var(--font-mono)"
                            fill="white"
                          >
                            {displayName}
                          </text>
                        </g>
                      )}
                    </g>
                  );
                })}

                {/* Loading indicator */}
                {loadingResources.size > 0 && (
                  <text
                    x={svgW / 2}
                    y={40}
                    text-anchor="middle"
                    font-size="11"
                    font-family="var(--font-sans)"
                    fill="#94A3B8"
                    style={{ pointerEvents: 'none' }}
                  >
                    Loading resources...
                  </text>
                )}
              </g>
            )}
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
              {viewMode === 'collapsed' ? renderNodes.map(node => {
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
              }) : expandedLayout ? expandedLayout.resourceNodes.map(rn => (
                <rect
                  key={`mm-${rn.id}`}
                  x={rn.x}
                  y={rn.y}
                  width={RES_NODE_W}
                  height={RES_NODE_H}
                  rx={1}
                  fill={rn.color}
                  opacity={0.6}
                />
              )) : null}
              {/* Minimap edges */}
              {viewMode === 'collapsed' ? filteredEdges.map((edge, i) => {
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
              }) : expandedLayout ? expandedLayout.resourceEdges.map((edge, i) => {
                const from = resNodePos[edge.from];
                const to = resNodePos[edge.to];
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
                    opacity="0.4"
                  />
                );
              }) : null}
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
