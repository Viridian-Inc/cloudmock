import { useState, useEffect, useRef, useCallback, useMemo } from 'preact/hooks';
import { SplitPanel } from '../../components/panels/split-panel';
import { api, cachedApi } from '../../lib/api';
import { useTopologyMetrics, type TimeWindow } from '../../hooks/use-topology-metrics';
import { loadDomainConfig, type DomainConfig } from '../../lib/domains';
import type { ServiceMetrics, DeployEvent } from '../../lib/health';
import { TopologyCanvas, type TopologyCanvasHandle } from './topology-canvas';
import { NodeInspector } from './node-inspector';
import { Timeline, type TimelineEvent } from './timeline';
import { TimeRangeSelector } from '../../components/time-range-selector/time-range-selector';
import { ServiceBrowser } from './service-browser';
import { DeployDetail } from './deploy-detail';
import {
  loadLayouts, saveLayout, deleteLayout, setDefaultLayout, getDefaultLayout,
  type SavedLayout,
} from './layouts';
import type { ManifestService } from './endpoints-tab';
import './topology.css';

export interface TopoNode {
  id: string;
  label: string;
  service: string;
  type: string;
  group: string;
  resourceCount?: number;
  requestService?: string;
}

export interface TopoEdge {
  source: string;
  target: string;
  label?: string;
  type?: string;
  discovered?: string;
  callCount?: number;
  avgLatencyMs?: number;
}

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
/**
 * Domain groups for microservices. Each group becomes a single expandable node.
 * Services not listed go into "Other Services".
 */
const DOMAIN_GROUPS: Record<string, { label: string; members: string[] }> = {
  'core': {
    label: 'Core',
    members: ['enterprise', 'membership', 'resource', 'session', 'order', 'event'],
  },
  'attendance': {
    label: 'Attendance',
    members: ['attendance', 'attendancePolicy', 'attendance_policy'],
  },
  'billing': {
    label: 'Billing & Payments',
    members: ['billing', 'stripeWebhook', 'stripe_webhook'],
  },
  'auth': {
    label: 'Auth & Identity',
    members: ['sso', 'cognito_enterprise', 'cognito_token', 'membership_authorizer', 'accessControl', 'access_control'],
  },
  'admin': {
    label: 'Admin & Analytics',
    members: ['userAdmin', 'user_admin', 'settings', 'compliance', 'audit', 'analytics', 'release', 'report'],
  },
  'comms': {
    label: 'Communications',
    members: ['notification', 'webhook', 'invitation'],
  },
  'content': {
    label: 'Content & Scheduling',
    members: ['classTemplate', 'userGroup', 'lms', 'feature_flag', 'featureFlag'],
  },
};

/** Nodes that are always shown individually (never grouped). */
const ALWAYS_INDIVIDUAL = new Set([
  'client:', 'apigw:', 'lambda:autotend-bff', 'svc:cognito', 'cognito:',
  'svc:sns', 'sns:', 'svc:sqs', 'sqs:',
  'svc:s3', 's3:', 'eventbridge:', 'svc:events',
]);

function isAlwaysIndividual(id: string): boolean {
  for (const prefix of ALWAYS_INDIVIDUAL) {
    if (id.startsWith(prefix)) return true;
  }
  return false;
}

function findDomainGroup(name: string): string | null {
  for (const [groupId, group] of Object.entries(DOMAIN_GROUPS)) {
    if (group.members.includes(name)) return groupId;
  }
  return null;
}

function collapseTopology(
  rawNodes: RawNode[],
  rawEdges: RawEdge[],
  expandedGroups?: Set<string>,
): { nodes: TopoNode[]; edges: TopoEdge[] } {
  const nodes: TopoNode[] = [];
  const nodeIds = new Set<string>();
  const collapseMap = new Map<string, string>();
  const expanded = expandedGroups || new Set<string>();

  // Infrastructure to hide entirely (plumbing, not app-relevant)
  const HIDDEN = new Set(['iam', 'sts', 'kms', 'logs', 'monitoring', 'secretsmanager',
    'ssm', 'cloudformation', 'rds', 'ec2', 'ecr', 'ecs', 'route53', 'ses']);

  // Track domain group membership counts
  const groupCounts = new Map<string, number>();
  const groupMembers = new Map<string, RawNode[]>();

  for (const n of rawNodes) {
    const prefix = n.id.split(':')[0];
    const svc = n.service || prefix;

    // Hide infra plumbing
    if (HIDDEN.has(prefix) || HIDDEN.has(svc)) {
      continue;
    }

    // Always-individual nodes pass through
    if (isAlwaysIndividual(n.id)) {
      nodes.push({ ...n });
      nodeIds.add(n.id);
      collapseMap.set(n.id, n.id);
      continue;
    }

    // Microservice/function nodes → check domain grouping
    if (prefix === 'microservice' || prefix === 'lambda') {
      const msName = n.id.split(':').slice(1).join(':');
      const groupId = findDomainGroup(msName);

      if (groupId && !expanded.has(groupId)) {
        // Collapse into domain group
        const groupNodeId = `domain:${groupId}`;
        collapseMap.set(n.id, groupNodeId);
        groupCounts.set(groupId, (groupCounts.get(groupId) || 0) + 1);
        if (!groupMembers.has(groupId)) groupMembers.set(groupId, []);
        groupMembers.get(groupId)!.push(n);
        continue;
      } else if (groupId && expanded.has(groupId)) {
        // Group is expanded — show individual node + keep group header for collapsing
        nodes.push({ ...n });
        nodeIds.add(n.id);
        collapseMap.set(n.id, n.id);
        // Ensure the group header node exists (click it to collapse back)
        const groupNodeId = `domain:${groupId}`;
        if (!nodeIds.has(groupNodeId)) {
          const group = DOMAIN_GROUPS[groupId];
          nodes.push({
            id: groupNodeId,
            label: `\u25B4 ${group.label}`,  // ▴ collapse arrow
            service: groupId,
            type: 'domain-group',
            group: 'Compute',
          });
          nodeIds.add(groupNodeId);
        }
        continue;
      }
    }

    // Collapse remaining AWS resource nodes by service (DynamoDB tables → 1 node, etc.)
    const collapsedId = `svc:${svc}`;
    collapseMap.set(n.id, collapsedId);
    if (!nodeIds.has(collapsedId)) {
      const svcLabels: Record<string, string> = {
        dynamodb: 'DynamoDB', sns: 'SNS', sqs: 'SQS', s3: 'S3', events: 'EventBridge',
        'cognito-idp': 'Cognito', apigateway: 'API Gateway', lambda: 'Lambda',
      };
      nodes.push({
        id: collapsedId,
        label: svcLabels[svc] || svc,
        service: svc,
        type: 'aws-service',
        group: n.group,
      });
      nodeIds.add(collapsedId);
    }
  }

  // Create domain group nodes
  for (const [groupId, count] of groupCounts) {
    const group = DOMAIN_GROUPS[groupId];
    const nodeId = `domain:${groupId}`;
    if (!nodeIds.has(nodeId)) {
      nodes.push({
        id: nodeId,
        label: `${group.label} (${count})`,
        service: groupId,
        type: 'domain-group',
        group: 'Compute',
        resourceCount: count,
      });
      nodeIds.add(nodeId);
    }
  }

  // Remap + dedup edges
  function resolveId(id: string): string {
    return collapseMap.get(id) || id;
  }

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

/** How long ago a deploy happened, as human-readable text */
function deployTimeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

const MAX_HISTORY = 20;

/** Time mode state for live vs historical viewing */
interface TimeMode {
  mode: 'live' | 'historical';
  rangeMinutes: number;
  start?: number;
  end?: number;
}

const TIME_PRESETS = [
  { label: '15m', minutes: 15 },
  { label: '1h', minutes: 60 },
  { label: '6h', minutes: 360 },
  { label: '24h', minutes: 1440 },
] as const;

/** Format a time-ago string for the paused indicator */
function formatViewingTime(playheadTime: number | null, rangeMinutes: number): string {
  if (playheadTime) {
    const diff = Date.now() - playheadTime;
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return 'just now';
    if (mins < 60) return `${mins}m ago`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return `${hrs}h ${mins % 60}m ago`;
    return `${Math.floor(hrs / 24)}d ago`;
  }
  if (rangeMinutes <= 60) return `last ${rangeMinutes}m`;
  if (rangeMinutes <= 1440) return `last ${rangeMinutes / 60}h`;
  return `last ${Math.floor(rangeMinutes / 1440)}d`;
}

/** Super-collapse: merge ALL AWS services into category-level nodes */
function superCollapse(
  nodes: TopoNode[],
  edges: TopoEdge[],
): { nodes: TopoNode[]; edges: TopoEdge[] } {
  const AWS_CATEGORIES: Record<string, string> = {
    dynamodb: 'Data Layer', rds: 'Data Layer', s3: 'Data Layer',
    sqs: 'Messaging', sns: 'Messaging', ses: 'Messaging', events: 'Messaging',
    'cognito-idp': 'Auth & Security', iam: 'Auth & Security', sts: 'Auth & Security',
    kms: 'Auth & Security', secretsmanager: 'Auth & Security', ssm: 'Auth & Security',
    lambda: 'Compute', apigateway: 'Compute',
    logs: 'Monitoring', monitoring: 'Monitoring', cloudformation: 'Monitoring',
  };

  const outNodes: TopoNode[] = [];
  const nodeIds = new Set<string>();
  const collapseMap = new Map<string, string>();

  for (const n of nodes) {
    if (n.type === 'aws-service') {
      const category = AWS_CATEGORIES[n.service] || 'AWS';
      const catId = `cat:${category}`;
      collapseMap.set(n.id, catId);
      if (!nodeIds.has(catId)) {
        outNodes.push({
          id: catId,
          label: category,
          service: category,
          type: 'aws-category',
          group: category,
          resourceCount: 1,
        });
        nodeIds.add(catId);
      } else {
        const cat = outNodes.find((o) => o.id === catId);
        if (cat) cat.resourceCount = (cat.resourceCount || 0) + 1;
      }
    } else {
      outNodes.push(n);
      nodeIds.add(n.id);
      collapseMap.set(n.id, n.id);
    }
  }

  const outEdges: TopoEdge[] = [];
  const edgeSeen = new Set<string>();
  for (const e of edges) {
    const source = collapseMap.get(e.source) || e.source;
    const target = collapseMap.get(e.target) || e.target;
    if (source === target || !nodeIds.has(source) || !nodeIds.has(target)) continue;
    const key = `${source}→${target}`;
    if (edgeSeen.has(key)) continue;
    edgeSeen.add(key);
    outEdges.push({ ...e, source, target });
  }

  return { nodes: outNodes, edges: outEdges };
}

export function TopologyView() {
  const [rawNodes, setRawNodes] = useState<TopoNode[]>([]);
  const [rawEdges, setRawEdges] = useState<TopoEdge[]>([]);
  const [selectedNode, setSelectedNode] = useState<TopoNode | null>(null);
  const [loading, setLoading] = useState(true);
  const [domainConfig, setDomainConfig] = useState<DomainConfig | null>(null);
  const [manifest, setManifest] = useState<ManifestService[] | null>(null);
  const [showServiceBrowser, setShowServiceBrowser] = useState(false);
  const [collapseAWS, setCollapseAWS] = useState(false);
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());
  const [timelineDeployDetail, setTimelineDeployDetail] = useState<DeployEvent | null>(null);

  // --- Layouts state ---
  const canvasRef = useRef<TopologyCanvasHandle>(null);
  const [layouts, setLayouts] = useState<SavedLayout[]>(loadLayouts);
  const [layoutDropdownOpen, setLayoutDropdownOpen] = useState(false);
  const [newLayoutName, setNewLayoutName] = useState('');
  const layoutDropdownRef = useRef<HTMLDivElement>(null);

  // --- Time travel state ---
  const [timeMode, setTimeMode] = useState<TimeMode>({ mode: 'live', rangeMinutes: 15 });
  const [playheadTime, setPlayheadTime] = useState<number | null>(null);
  const [showCustomPicker, setShowCustomPicker] = useState(false);
  const [customStart, setCustomStart] = useState('');
  const [customEnd, setCustomEnd] = useState('');

  const isLive = timeMode.mode === 'live';
  const isPaused = !isLive;

  // Compute the visible time range from the time mode
  const timeRange = useMemo(() => {
    if (timeMode.mode === 'historical' && timeMode.start != null && timeMode.end != null) {
      return { start: timeMode.start, end: timeMode.end };
    }
    // Live mode or preset: window ending at now
    const now = Date.now();
    return { start: now - timeMode.rangeMinutes * 60 * 1000, end: now };
  }, [timeMode]);

  // Time window for filtering metrics (when historical, use playhead or end of range)
  const metricsTimeWindow: TimeWindow | undefined = useMemo(() => {
    if (isLive && playheadTime == null) return undefined;
    // If playhead is set, filter up to playhead time
    const effectiveEnd = playheadTime ?? timeRange.end;
    return { start: timeRange.start, end: effectiveEnd };
  }, [isLive, playheadTime, timeRange]);

  // Metrics history ring buffers: keyed by "svcKey:metric"
  const [metricsHistory, setMetricsHistory] = useState<Map<string, number[]>>(new Map());
  const prevMetricsRef = useRef<ServiceMetrics[]>([]);

  const { metrics, deploys, incidents, loading: metricsLoading } = useTopologyMetrics(
    isPaused,
    metricsTimeWindow,
  );

  // --- Time range selector handlers ---
  const handlePresetClick = useCallback((minutes: number) => {
    const now = Date.now();
    setTimeMode({
      mode: 'historical',
      rangeMinutes: minutes,
      start: now - minutes * 60 * 1000,
      end: now,
    });
    setPlayheadTime(null);
    setShowCustomPicker(false);
  }, []);

  const handleLiveClick = useCallback(() => {
    setTimeMode({ mode: 'live', rangeMinutes: 15 });
    setPlayheadTime(null);
    setShowCustomPicker(false);
  }, []);

  const handleCustomApply = useCallback(() => {
    if (!customStart || !customEnd) return;
    const start = new Date(customStart).getTime();
    const end = new Date(customEnd).getTime();
    if (isNaN(start) || isNaN(end) || start >= end) return;
    setTimeMode({
      mode: 'historical',
      rangeMinutes: Math.round((end - start) / 60000),
      start,
      end,
    });
    setPlayheadTime(null);
    setShowCustomPicker(false);
  }, [customStart, customEnd]);

  const handlePlayheadChange = useCallback((timestamp: number | null) => {
    setPlayheadTime(timestamp);
    // If we are in live mode and the user drags the playhead, switch to historical
    if (timestamp != null && isLive) {
      const now = Date.now();
      setTimeMode({
        mode: 'historical',
        rangeMinutes: 15,
        start: now - 15 * 60 * 1000,
        end: now,
      });
    }
  }, [isLive]);

  const handleNowClick = useCallback(() => {
    handleLiveClick();
  }, [handleLiveClick]);

  /** Handle zoom/pan from the TimeRangeSelector */
  const handleTimeRangeSelectorChange = useCallback((range: { start: number; end: number }) => {
    const durationMin = Math.round((range.end - range.start) / 60000);
    // If the selection is close to "now", stay in live mode
    const isNearNow = Math.abs(range.end - Date.now()) < 10000;
    if (isNearNow && durationMin <= 15) {
      setTimeMode({ mode: 'live', rangeMinutes: Math.max(1, durationMin) });
    } else {
      setTimeMode({
        mode: 'historical',
        rangeMinutes: durationMin,
        start: range.start,
        end: range.end,
      });
    }
    setPlayheadTime(null);
  }, []);

  // --- Layout handlers ---
  const handleSaveLayout = useCallback(() => {
    const name = newLayoutName.trim();
    if (!name || !canvasRef.current) return;
    const state = canvasRef.current.getLayoutState();
    const layout: SavedLayout = {
      name,
      createdAt: new Date().toISOString(),
      pinnedPositions: state.pinnedPositions,
      pan: state.pan,
      scale: state.scale,
      collapsed: collapseAWS,
      isDefault: false,
    };
    saveLayout(layout);
    setLayouts(loadLayouts());
    setNewLayoutName('');
  }, [newLayoutName, collapseAWS]);

  const handleApplyLayout = useCallback((layout: SavedLayout) => {
    if (!canvasRef.current) return;
    canvasRef.current.applyLayout({
      pinnedPositions: layout.pinnedPositions,
      pan: layout.pan,
      scale: layout.scale,
    });
    setCollapseAWS(layout.collapsed);
    setLayoutDropdownOpen(false);
  }, []);

  const handleDeleteLayout = useCallback((name: string) => {
    deleteLayout(name);
    setLayouts(loadLayouts());
  }, []);

  const handleSetDefault = useCallback((name: string) => {
    setDefaultLayout(name);
    setLayouts(loadLayouts());
  }, []);

  const handleResetLayout = useCallback(() => {
    if (!canvasRef.current) return;
    canvasRef.current.applyLayout({
      pinnedPositions: {},
      pan: { x: 20, y: 20 },
      scale: 0.85,
    });
    setLayoutDropdownOpen(false);
  }, []);

  // Close layout dropdown when clicking outside
  useEffect(() => {
    if (!layoutDropdownOpen) return;
    const handler = (e: MouseEvent) => {
      if (layoutDropdownRef.current && !layoutDropdownRef.current.contains(e.target as Node)) {
        setLayoutDropdownOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [layoutDropdownOpen]);

  // Load topology
  useEffect(() => {
    setLoading(true);
    cachedApi<{ nodes: RawNode[]; edges: RawEdge[] }>('/api/topology', 'topology:graph')
      .then((data) => {
        setRawNodes(data.nodes || []);
        setRawEdges(data.edges || []);
      })
      .catch(() => { setRawNodes([]); setRawEdges([]); })
      .finally(() => setLoading(false));
  }, []);

  // Apply default layout on initial topology load
  const defaultLayoutAppliedRef = useRef(false);
  useEffect(() => {
    if (defaultLayoutAppliedRef.current || rawNodes.length === 0 || !canvasRef.current) return;
    const defaultLayout = getDefaultLayout();
    if (defaultLayout) {
      // Delay slightly so the ELK layout is ready before overriding positions
      const timer = setTimeout(() => {
        if (canvasRef.current) {
          canvasRef.current.applyLayout({
            pinnedPositions: defaultLayout.pinnedPositions,
            pan: defaultLayout.pan,
            scale: defaultLayout.scale,
          });
          setCollapseAWS(defaultLayout.collapsed);
        }
      }, 300);
      defaultLayoutAppliedRef.current = true;
      return () => clearTimeout(timer);
    }
    defaultLayoutAppliedRef.current = true;
  }, [rawNodes]);

  // Memoize collapse + enrichment to prevent infinite re-renders
  // (enrichment creates new arrays which would re-trigger ELK layout otherwise)
  const { nodes, edges } = useMemo(() => {
    const collapsed = collapseAWS
      ? superCollapse(rawNodes, rawEdges)
      : collapseTopology(rawNodes, rawEdges, expandedGroups);

    // Enrich edges with traffic data from metrics (computed from traces)
    const enrichedEdges = collapsed.edges.map((e) => {
      if ((e.callCount ?? 0) > 0) return e;
      const targetService = e.target.replace(/^svc:|^ms:|^cat:/, '').toLowerCase();
      const m = metrics.find((sm) => sm.service.toLowerCase() === targetService);
      if (m && m.totalCalls > 0) {
        return { ...e, callCount: m.totalCalls, avgLatencyMs: m.avgMs };
      }
      return e;
    });

    // Add traffic-discovered nodes AND edges
    const existingEdgeTargets = new Set(enrichedEdges.map((e) => e.target));
    const nodeIdSet = new Set(collapsed.nodes.map((n) => n.id));
    const mutableNodes = [...collapsed.nodes];

    for (const m of metrics) {
      if (m.totalCalls === 0) continue;
      const targetId = `svc:${m.service}`;
      if (!nodeIdSet.has(targetId)) {
        mutableNodes.push({
          id: targetId,
          label: m.service.charAt(0).toUpperCase() + m.service.slice(1),
          service: m.service,
          type: 'aws-service',
          group: 'AWS',
          resourceCount: undefined,
        });
        nodeIdSet.add(targetId);
      }
      if (!existingEdgeTargets.has(targetId)) {
        const bff = mutableNodes.find((n) => n.id === 'external:bff-service');
        if (bff) {
          enrichedEdges.push({
            source: bff.id,
            target: targetId,
            label: 'traffic',
            type: 'invoke',
            discovered: 'traffic',
            callCount: m.totalCalls,
            avgLatencyMs: m.avgMs,
          });
          existingEdgeTargets.add(targetId);
        }
      }
    }

    // Enrich inbound edges (Client → API) based on API outbound traffic
    // If BFF has 231 outbound calls, Client→BFF should also show traffic
    const apiNodes = mutableNodes.filter((n) =>
      n.group === 'API' || n.type === 'server',
    );
    for (const apiNode of apiNodes) {
      const outboundCalls = enrichedEdges
        .filter((e) => e.source === apiNode.id)
        .reduce((sum, e) => sum + (e.callCount ?? 0), 0);

      if (outboundCalls > 0) {
        // Find inbound edges to this API node and give them traffic
        for (const e of enrichedEdges) {
          if (e.target === apiNode.id && (e.callCount ?? 0) === 0) {
            e.callCount = outboundCalls;
            e.label = e.label || 'inferred';
            e.discovered = 'inferred';
          }
        }
      }
    }

    return { nodes: mutableNodes, edges: enrichedEdges };
  }, [rawNodes, rawEdges, collapseAWS, expandedGroups, metrics]);

  // In historical mode, compute which nodes have no traffic in the time window
  const inactiveNodeIds = useMemo(() => {
    if (isLive && playheadTime == null) return undefined;
    // Build set of services that have traffic in the current metrics
    const activeServices = new Set(
      metrics.filter((m) => m.totalCalls > 0).map((m) => m.service.toLowerCase()),
    );
    // Also consider edge traffic: nodes that are source or target of active edges
    const edgeActiveIds = new Set<string>();
    for (const e of edges) {
      if ((e.callCount ?? 0) > 0) {
        edgeActiveIds.add(e.source);
        edgeActiveIds.add(e.target);
      }
    }
    const inactive = new Set<string>();
    for (const n of nodes) {
      const svcKey = n.service || n.id.replace(/^svc:|^ms:/, '');
      const hasMetrics = activeServices.has(svcKey.toLowerCase());
      const hasEdgeTraffic = edgeActiveIds.has(n.id);
      if (!hasMetrics && !hasEdgeTraffic) {
        inactive.add(n.id);
      }
    }
    return inactive.size > 0 ? inactive : undefined;
  }, [isLive, playheadTime, nodes, edges, metrics]);

  // Load domain config
  useEffect(() => {
    loadDomainConfig().then(setDomainConfig).catch((e) => { console.warn('[Topology] Failed to load domain config:', e); });
  }, []);

  // Load IaC topology config and merge with dynamic topology
  useEffect(() => {
    api<{ nodes: RawNode[]; edges: RawEdge[]; services?: ManifestService[] }>('/api/topology/config')
      .then((data) => {
        // Merge IaC nodes/edges into the raw topology
        if (data.nodes && data.nodes.length > 0) {
          const iacNodes = data.nodes || [];
          const iacEdges = data.edges || [];
          setRawNodes((prev) => {
            const existingIds = new Set(prev.map((n) => n.id));
            const newNodes = iacNodes.filter((n: any) => !existingIds.has(n.id));
            return newNodes.length > 0 ? [...prev, ...newNodes] : prev;
          });
          setRawEdges((prev) => {
            const existingKeys = new Set(prev.map((e) => `${e.source}->${e.target}`));
            const newEdges = iacEdges.filter(
              (e: any) => !existingKeys.has(`${e.source}->${e.target}`),
            );
            return newEdges.length > 0 ? [...prev, ...newEdges] : prev;
          });
        }
        // Use services array from config as manifest if available
        if (data.services && data.services.length > 0) {
          setManifest(data.services);
        } else {
          // Fallback: build manifest from /api/services registry
          api<{ name: string; actions: any[] }[]>('/api/services')
            .then((svcs) => {
              const m = svcs.map((s) => ({
                name: s.name,
                tables: [],
                sdkClients: [],
                routes: (s.actions || []).map((a: any) => ({
                  method: a.Method || 'POST',
                  path: `/${s.name}/${a.Name || a.name || ''}`,
                })),
                schemas: [],
              }));
              setManifest(m);
            })
            .catch(() => { setManifest([]); });
        }
      })
      .catch(() => { setManifest([]); });
  }, []);

  // Accumulate metrics history for sparklines
  useEffect(() => {
    if (metrics.length === 0) return;
    // Only append if metrics actually changed (different reference)
    if (metrics === prevMetricsRef.current) return;
    prevMetricsRef.current = metrics;

    setMetricsHistory((prev) => {
      const next = new Map(prev);
      for (const m of metrics) {
        const reqKey = `${m.service}:req`;
        const p99Key = `${m.service}:p99`;
        const errKey = `${m.service}:err`;

        const reqArr = [...(next.get(reqKey) ?? []), m.totalCalls];
        const p99Arr = [...(next.get(p99Key) ?? []), m.p99ms];
        const errArr = [...(next.get(errKey) ?? []), m.errorRate * 100];

        next.set(reqKey, reqArr.slice(-MAX_HISTORY));
        next.set(p99Key, p99Arr.slice(-MAX_HISTORY));
        next.set(errKey, errArr.slice(-MAX_HISTORY));
      }
      return next;
    });
  }, [metrics]);

  // Compute toolbar indicators
  const activeIncidentCount = incidents.length;
  const lastDeploy = deploys.length > 0
    ? deploys.reduce((a, b) => new Date(a.timestamp) > new Date(b.timestamp) ? a : b)
    : null;

  const handleTimelineEvent = (event: TimelineEvent) => {
    if (event.type === 'deploy') {
      const deploy = event.data as DeployEvent;
      // Select the corresponding node
      const matchNode = nodes.find(
        (n) => n.service === deploy.service || n.id === deploy.service || n.id === `svc:${deploy.service}`,
      );
      if (matchNode) setSelectedNode(matchNode);
      // Open deploy detail
      setTimelineDeployDetail(deploy);
    }
  };

  return (
    <div class="topology-view">
      <div style={{ display: 'flex', height: '100%' }}>
        {/* Service Browser sidebar */}
        {showServiceBrowser && (
          <ServiceBrowser
            nodes={nodes}
            edges={edges}
            metrics={metrics}
            incidents={incidents}
            domainConfig={domainConfig}
            selectedNodeId={selectedNode?.id ?? null}
            onSelectNode={setSelectedNode}
            manifest={manifest}
          />
        )}

        {/* Main content area */}
        <div style={{ flex: 1, minWidth: 0, height: '100%' }}>
          <SplitPanel
            initialSplit={72}
            direction="horizontal"
            minSize={250}
            left={
              <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
                <div style={{ flex: 1, minHeight: 0 }}>
                  <TopologyCanvas
                    ref={canvasRef}
                    nodes={nodes}
                    edges={edges}
                    selectedNodeId={selectedNode?.id ?? null}
                    onSelectNode={(node) => {
                      // Click on domain group → toggle expand/collapse
                      if (node && node.id.startsWith('domain:')) {
                        const groupId = node.id.replace('domain:', '');
                        setExpandedGroups((prev) => {
                          const next = new Set(prev);
                          if (next.has(groupId)) next.delete(groupId);
                          else next.add(groupId);
                          return next;
                        });
                        return;
                      }
                      setSelectedNode(node);
                    }}
                    loading={loading}
                    metrics={metrics}
                    deploys={deploys}
                    incidents={incidents}
                    metricsHistory={metricsHistory}
                    inactiveNodeIds={inactiveNodeIds}
                  />
                </div>
                <Timeline
                  deploys={deploys}
                  incidents={incidents}
                  onSelectEvent={handleTimelineEvent}
                  timeRange={timeRange}
                  playheadTime={playheadTime}
                  onPlayheadChange={handlePlayheadChange}
                  isLive={isLive}
                />
                <TimeRangeSelector
                  dataRange={{
                    start: timeRange.start - (timeRange.end - timeRange.start),
                    end: Math.max(timeRange.end, Date.now()),
                  }}
                  selectedRange={timeRange}
                  onRangeChange={handleTimeRangeSelectorChange}
                  live={isLive && playheadTime == null}
                  height={40}
                />
              </div>
            }
            right={
              <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
                {/* Live status toolbar */}
                <div class="topology-toolbar" style={{ justifyContent: 'flex-start', gap: '14px', flexWrap: 'wrap' }}>
                  <button
                    class={`btn btn-ghost toolbar-services-btn ${showServiceBrowser ? 'active' : ''}`}
                    onClick={() => setShowServiceBrowser((v) => !v)}
                    title="Toggle service browser"
                  >
                    {'\u{1F4CB}'} Services
                  </button>
                  <button
                    class={`btn btn-ghost ${collapseAWS ? 'active' : ''}`}
                    onClick={() => setCollapseAWS((v) => !v)}
                    title="Collapse AWS services into categories"
                  >
                    {collapseAWS ? '\uD83D\uDD0D Expand' : '\uD83D\uDCE6 Collapse'}
                  </button>

                  {/* Layouts dropdown */}
                  <div ref={layoutDropdownRef} style={{ position: 'relative' }}>
                    <button
                      class={`btn btn-ghost ${layoutDropdownOpen ? 'active' : ''}`}
                      onClick={() => setLayoutDropdownOpen((v) => !v)}
                      title="Saved layouts"
                    >
                      {'\uD83D\uDCBE'} Layouts
                    </button>
                    {layoutDropdownOpen && (
                      <div class="layout-dropdown">
                        <div class="layout-dropdown-header">Saved Layouts</div>
                        {layouts.length === 0 && (
                          <div class="layout-dropdown-empty">No saved layouts</div>
                        )}
                        {layouts.map((l) => (
                          <div key={l.name} class="layout-dropdown-item">
                            <button
                              class="layout-dropdown-name"
                              onClick={() => handleApplyLayout(l)}
                              title={`Apply "${l.name}"`}
                            >
                              {l.isDefault && <span class="layout-default-star">{'\u2605'}</span>}
                              {l.name}
                            </button>
                            <button
                              class="layout-dropdown-action"
                              onClick={() => handleSetDefault(l.name)}
                              title={l.isDefault ? 'Default layout' : 'Set as default'}
                            >
                              {l.isDefault ? '\u2605' : '\u2606'}
                            </button>
                            <button
                              class="layout-dropdown-action layout-dropdown-delete"
                              onClick={() => handleDeleteLayout(l.name)}
                              title="Delete layout"
                            >
                              {'\u2715'}
                            </button>
                          </div>
                        ))}
                        <div class="layout-dropdown-save">
                          <input
                            class="input layout-dropdown-input"
                            type="text"
                            placeholder="Layout name..."
                            value={newLayoutName}
                            onInput={(e) => setNewLayoutName((e.target as HTMLInputElement).value)}
                            onKeyDown={(e) => { if (e.key === 'Enter') handleSaveLayout(); }}
                          />
                          <button
                            class="btn btn-sm layout-dropdown-save-btn"
                            onClick={handleSaveLayout}
                            disabled={!newLayoutName.trim()}
                          >
                            Save
                          </button>
                        </div>
                        <button
                          class="btn btn-ghost layout-dropdown-reset"
                          onClick={handleResetLayout}
                        >
                          Reset to auto
                        </button>
                      </div>
                    )}
                  </div>

                  {/* Divider */}
                  <div style={{ width: '1px', height: '18px', background: 'var(--border-subtle, rgba(74,229,248,0.06))' }} />

                  {/* Time range selector */}
                  <div class="time-range-bar">
                    <button
                      class={`time-range-btn ${isLive ? 'active live' : ''}`}
                      onClick={handleLiveClick}
                      title="Switch to live mode"
                    >
                      {isLive && <span class="toolbar-live-dot" />}
                      Live
                    </button>
                    {TIME_PRESETS.map((preset) => (
                      <button
                        key={preset.minutes}
                        class={`time-range-btn ${!isLive && timeMode.rangeMinutes === preset.minutes && !showCustomPicker ? 'active' : ''}`}
                        onClick={() => handlePresetClick(preset.minutes)}
                        title={`View last ${preset.label}`}
                      >
                        {preset.label}
                      </button>
                    ))}
                    <div style={{ position: 'relative' }}>
                      <button
                        class={`time-range-btn ${showCustomPicker ? 'active' : ''}`}
                        onClick={() => setShowCustomPicker((v) => !v)}
                        title="Custom time range"
                      >
                        Custom {'\u25BE'}
                      </button>
                      {showCustomPicker && (
                        <div class="time-range-custom-picker">
                          <label class="time-range-custom-label">
                            From
                            <input
                              type="datetime-local"
                              class="time-range-custom-input"
                              value={customStart}
                              onInput={(e) => setCustomStart((e.target as HTMLInputElement).value)}
                            />
                          </label>
                          <label class="time-range-custom-label">
                            To
                            <input
                              type="datetime-local"
                              class="time-range-custom-input"
                              value={customEnd}
                              onInput={(e) => setCustomEnd((e.target as HTMLInputElement).value)}
                            />
                          </label>
                          <button class="btn btn-sm time-range-custom-apply" onClick={handleCustomApply}>
                            Apply
                          </button>
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Live indicator or Paused indicator */}
                  {isLive && !playheadTime ? (
                    <div class="toolbar-live-indicator">
                      <span class="toolbar-live-dot" />
                      <span>Live</span>
                    </div>
                  ) : (
                    <div class="toolbar-paused-indicator">
                      <span class="toolbar-paused-icon">{'\u23F8'}</span>
                      <span>Paused — viewing {formatViewingTime(playheadTime, timeMode.rangeMinutes)}</span>
                      <button class="btn btn-ghost btn-xs toolbar-now-btn" onClick={handleNowClick}>
                        Now
                      </button>
                    </div>
                  )}

                  {activeIncidentCount > 0 && (
                    <div class="toolbar-incident-badge">
                      <span class="toolbar-incident-dot" />
                      <span>{activeIncidentCount} incident{activeIncidentCount !== 1 ? 's' : ''}</span>
                    </div>
                  )}
                  {lastDeploy && (
                    <div class="toolbar-deploy-badge">
                      <span class="toolbar-deploy-dot" />
                      <span>{lastDeploy.service} {deployTimeAgo(lastDeploy.timestamp)}</span>
                    </div>
                  )}
                </div>
                {/* Right panel: inspector when node selected, service browser when not */}
                <div style={{ flex: 1, overflow: 'hidden' }}>
                  {selectedNode ? (
                    <NodeInspector
                      node={selectedNode}
                      edges={edges}
                      allNodes={nodes}
                      metrics={metrics}
                      deploys={deploys}
                      incidents={incidents}
                      domainConfig={domainConfig}
                      metricsHistory={metricsHistory}
                      manifest={manifest}
                    />
                  ) : (
                    <ServiceBrowser
                      nodes={nodes}
                      edges={edges}
                      metrics={metrics}
                      incidents={incidents}
                      domainConfig={domainConfig}
                      selectedNodeId={null}
                      onSelectNode={setSelectedNode}
                      manifest={manifest}
                    />
                  )}
                </div>
              </div>
            }
          />
        </div>
      </div>

      {timelineDeployDetail && (
        <DeployDetail
          deploy={timelineDeployDetail}
          onClose={() => setTimelineDeployDetail(null)}
        />
      )}
    </div>
  );
}
