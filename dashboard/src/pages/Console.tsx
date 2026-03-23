import { useState, useEffect, useRef, useCallback, useMemo } from 'preact/hooks';
import { api, getSLOStatus, getStats } from '../api';
import { useFilters } from '../hooks/useFilters';
import { RequestExplorer } from '../components/RequestExplorer';
import { ServiceInspector } from '../components/ServiceInspector';
import type { SSEState } from '../hooks/useSSE';
import '../styles/console.css';

// --- Types (reused from Topology.tsx) ---

interface TopoNode {
  id: string;
  label: string;
  service: string;
  type: string;
  group: string;
  requestService?: string;
}

interface TopoEdge {
  source: string;
  target: string;
  type: string;
  label: string;
  discovered: string;
  avgLatencyMs?: number;
  callCount?: number;
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

interface PositionedNode {
  id: string;
  label: string;
  service: string;
  type: string;
  group: string;
  x: number;
  y: number;
  requestService?: string;
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

interface ConsolePageProps {
  sse: SSEState;
  showToast: (msg: string) => void;
}

// --- Layout Constants ---

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

// --- Grouping config ---

interface GroupConfig {
  nodeToGroup: (node: TopoNode) => string;
  groups: { id: string; label: string; color: string }[];
  layers: Record<string, number>;
  order: Record<string, number>;
}

const SERVICE_CONFIG: GroupConfig = {
  nodeToGroup: (n: TopoNode) => n.group,
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

// --- Layout engine ---

function layoutGraph(data: TopoData, config: GroupConfig) {
  const remappedNodes: TopoNode[] = data.nodes.map(n => ({
    ...n,
    group: config.nodeToGroup(n),
  }));

  const nodesByGroup = new Map<string, TopoNode[]>();
  for (const n of remappedNodes) {
    const arr = nodesByGroup.get(n.group) || [];
    arr.push(n);
    nodesByGroup.set(n.group, arr);
  }

  const groupInfo = new Map<string, TopoGroup>();
  for (const g of config.groups) {
    groupInfo.set(g.id, g);
  }

  const activeGroups: TopoGroup[] = [];
  for (const g of config.groups) {
    if (nodesByGroup.has(g.id)) activeGroups.push(g);
  }

  const layerGroups = new Map<number, { group: TopoGroup; nodes: TopoNode[] }[]>();
  for (const [gid, nodes] of nodesByGroup) {
    const layer = config.layers[gid] ?? 3;
    const arr = layerGroups.get(layer) || [];
    const info = groupInfo.get(gid) || { id: gid, label: gid, color: '#94A3B8' };
    arr.push({ group: info, nodes });
    layerGroups.set(layer, arr);
  }

  for (const [, arr] of layerGroups) {
    arr.sort((a, b) => (config.order[a.group.id] ?? 99) - (config.order[b.group.id] ?? 99));
  }

  const positionedNodes: PositionedNode[] = [];
  const positionedGroups: PositionedGroup[] = [];

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

  const layerX = new Map<number, number>();
  let currentX = 40;
  for (const layer of layers) {
    layerX.set(layer, currentX);
    currentX += (layerWidths.get(layer) || 200) + LAYER_GAP;
  }

  for (const layer of layers) {
    const groups = layerGroups.get(layer) || [];
    let currentY = 40;

    for (const { group, nodes } of groups) {
      const x = layerX.get(layer)!;
      const sz = clusterSizes.get(group.id)!;

      positionedGroups.push({
        id: group.id, label: group.label, color: group.color,
        x, y: currentY, width: sz.width, height: sz.height, nodeCount: sz.nodeCount,
      });

      for (let i = 0; i < nodes.length; i++) {
        const col = i % COLS;
        const row = Math.floor(i / COLS);
        const nx = x + CLUSTER_PAD_X + col * (RES_W + RES_GAP_X);
        const ny = currentY + CLUSTER_HEADER + CLUSTER_PAD_Y + row * (RES_H + RES_GAP_Y);

        positionedNodes.push({
          id: nodes[i].id, label: nodes[i].label, service: nodes[i].service,
          type: nodes[i].type, group: nodes[i].group, x: nx, y: ny,
          requestService: nodes[i].requestService,
        });
      }

      currentY += sz.height + GROUP_GAP_Y;
    }
  }

  return { positionedNodes, positionedGroups, activeGroups };
}

function bezierPath(x1: number, y1: number, x2: number, y2: number): string {
  const dx = Math.abs(x2 - x1) * 0.45;
  return `M ${x1} ${y1} C ${x1 + dx} ${y1}, ${x2 - dx} ${y2}, ${x2} ${y2}`;
}

// --- Node-request mapping ---

function findNodeForRequest(nodes: TopoNode[], req: any): TopoNode | undefined {
  const svc = req.service;
  let match = nodes.find(n => n.requestService === svc);
  if (match) return match;
  match = nodes.find(n => n.service === svc);
  if (match) return match;
  match = nodes.find(n => n.id.includes(svc) || n.label.toLowerCase().includes(svc));
  if (match) return match;
  if (svc === 'dynamodb' && req.request_body) {
    try {
      const body = typeof req.request_body === 'string' ? JSON.parse(req.request_body) : req.request_body;
      if (body.TableName) {
        match = nodes.find(n => n.id === `dynamodb:${body.TableName}`);
        if (match) return match;
      }
    } catch { /* ignore */ }
  }
  return undefined;
}

// --- Console component ---

export function ConsolePage({ sse, showToast }: ConsolePageProps) {
  const { filters, setFilter, clearFilters, hasActiveFilters } = useFilters();
  const [topoData, setTopoData] = useState<TopoData | null>(null);
  const [selectedNode, setSelectedNode] = useState<TopoNode | null>(null);
  const [leftOpen, setLeftOpen] = useState(true);
  const [rightOpen, setRightOpen] = useState(true);
  const [highlightService, setHighlightService] = useState<string | null>(null);
  const [hoveredNode, setHoveredNode] = useState<string | null>(null);
  const [hoveredEdge, setHoveredEdge] = useState<number | null>(null);
  const [hoveredCluster, setHoveredCluster] = useState<string | null>(null);
  const [transform, setTransform] = useState({ x: 0, y: 0, scale: 1 });
  const [pulsingNodes, setPulsingNodes] = useState<Map<string, number>>(new Map());
  const [sloHealth, setSloHealth] = useState<Record<string, any>>({});
  const [statsCounts, setStatsCounts] = useState<Record<string, number>>({});
  const [selectedRequestId, setSelectedRequestId] = useState<string | undefined>(undefined);
  const svgRef = useRef<SVGSVGElement>(null);
  const dragging = useRef<{ startX: number; startY: number; origX: number; origY: number } | null>(null);

  // Fetch topology
  useEffect(() => {
    api('/api/topology').then(setTopoData).catch(() => {});
  }, []);

  // Fetch SLO + stats for health coloring and sizing
  useEffect(() => {
    getSLOStatus().then(d => {
      if (d?.services) setSloHealth(d.services);
    }).catch(() => {});
    getStats().then(d => {
      if (d) setStatsCounts(d);
    }).catch(() => {});

    const iv = setInterval(() => {
      getSLOStatus().then(d => {
        if (d?.services) setSloHealth(d.services);
      }).catch(() => {});
      getStats().then(d => {
        if (d) setStatsCounts(d);
      }).catch(() => {});
    }, 10000);
    return () => clearInterval(iv);
  }, []);

  // SSE: pulse nodes
  useEffect(() => {
    if (!sse?.events?.length) return;
    const latest = sse.events[0];
    if (!latest?.data?.service) return;
    if (latest.data.level === 'infra') return;

    const svcName = latest.data.service;
    setPulsingNodes(prev => {
      const next = new Map(prev);
      if (topoData) {
        for (const n of topoData.nodes) {
          if (n.service === svcName || n.requestService === svcName || n.id.startsWith(svcName + ':')) {
            next.set(n.id, Date.now());
          }
        }
      }
      return next;
    });

    const timer = setTimeout(() => {
      setPulsingNodes(prev => {
        const next = new Map(prev);
        const cutoff = Date.now() - 1400;
        for (const [k, v] of next) {
          if (v < cutoff) next.delete(k);
        }
        return next;
      });
    }, 2000);
    return () => clearTimeout(timer);
  }, [sse?.events?.length, topoData]);

  // Layout
  const layout = useMemo(() => {
    if (!topoData) return null;
    return layoutGraph(topoData, SERVICE_CONFIG);
  }, [topoData]);

  // Node position lookup
  const nodePos = useMemo(() => {
    if (!layout) return new Map<string, { cx: number; cy: number }>();
    const m = new Map<string, { cx: number; cy: number }>();
    for (const n of layout.positionedNodes) {
      m.set(n.id, { cx: n.x + RES_W / 2, cy: n.y + RES_H / 2 });
    }
    return m;
  }, [layout]);

  // Connected edges/nodes for hover
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

  // Group color lookup
  const groupColorMap = useMemo(() => {
    if (!layout) return new Map<string, string>();
    const m = new Map<string, string>();
    for (const g of layout.positionedGroups) m.set(g.id, g.color);
    return m;
  }, [layout]);

  const nodeGroupMap = useMemo(() => {
    if (!layout) return new Map<string, string>();
    const m = new Map<string, string>();
    for (const n of layout.positionedNodes) m.set(n.id, n.group);
    return m;
  }, [layout]);

  // SVG dimensions
  const { svgW, svgH } = useMemo(() => {
    if (!layout) return { svgW: 1600, svgH: 900 };
    let maxX = 0, maxY = 0;
    for (const g of layout.positionedGroups) {
      maxX = Math.max(maxX, g.x + g.width);
      maxY = Math.max(maxY, g.y + g.height);
    }
    return { svgW: Math.max(1600, maxX + 80), svgH: Math.max(900, maxY + 80) };
  }, [layout]);

  // Minimap
  const minimapW = 160;
  const minimapH = 100;
  const minimapScale = useMemo(() => Math.min(minimapW / svgW, minimapH / svgH), [svgW, svgH]);

  // Health color for a node
  const getNodeHealthColor = useCallback((n: PositionedNode): string | null => {
    const svc = n.requestService || n.service;
    const slo = sloHealth[svc];
    if (!slo) return null;
    if (slo.healthy === false) return '#EF4444';
    if (slo.burn_rate > 0.5) return '#F59E0B';
    return '#10B981';
  }, [sloHealth]);

  // Pan/zoom
  const onWheel = useCallback((e: WheelEvent) => {
    e.preventDefault();
    const delta = e.deltaY > 0 ? 0.9 : 1.1;
    setTransform(t => ({ ...t, scale: Math.max(0.2, Math.min(3, t.scale * delta)) }));
  }, []);

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

  // Handle request click from explorer
  const handleSelectRequest = useCallback((req: any) => {
    setSelectedRequestId(req.id);
    setHighlightService(req.service);
    const matchingNode = findNodeForRequest(topoData?.nodes || [], req);
    if (matchingNode) {
      setSelectedNode(matchingNode);
      if (!rightOpen) setRightOpen(true);
    }
    setTimeout(() => setHighlightService(null), 2000);
  }, [topoData, rightOpen]);

  // Handle node click from canvas
  const handleNodeClick = useCallback((n: PositionedNode) => {
    const topoNode: TopoNode = {
      id: n.id, label: n.label, service: n.service,
      type: n.type, group: n.group, requestService: n.requestService,
    };
    setSelectedNode(topoNode);
    if (!rightOpen) setRightOpen(true);
  }, [rightOpen]);

  if (!topoData || !layout) {
    return (
      <div class="console-layout">
        <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--n400)' }}>
          <div style={{ textAlign: 'center' }}>
            <div style={{ width: 24, height: 24, border: '2px solid var(--n200)', borderTopColor: 'var(--brand-blue)', borderRadius: '50%', animation: 'spin 0.6s linear infinite', margin: '0 auto 12px' }} />
            <div style={{ fontSize: 13 }}>Loading topology...</div>
          </div>
        </div>
      </div>
    );
  }

  const edges = topoData.edges;

  return (
    <div class="console-layout">
      {/* Left rail — Request Explorer */}
      <div class={`console-left ${leftOpen ? '' : 'collapsed'}`}>
        <RequestExplorer
          sse={sse}
          filters={filters}
          setFilter={setFilter}
          clearFilters={clearFilters}
          hasActiveFilters={hasActiveFilters}
          onSelectRequest={handleSelectRequest}
          selectedRequestId={selectedRequestId}
        />
      </div>

      {/* Center — Topology Canvas */}
      <div class="console-center">
        {/* Left toggle */}
        <button
          class="console-panel-toggle"
          style={{ left: leftOpen ? 8 : 8 }}
          onClick={() => setLeftOpen(!leftOpen)}
          title={leftOpen ? 'Hide explorer' : 'Show explorer'}
        >
          {leftOpen ? '\u25C0' : '\u25B6'}
        </button>

        {/* Right toggle */}
        <button
          class="console-panel-toggle"
          style={{ right: 8 }}
          onClick={() => setRightOpen(!rightOpen)}
          title={rightOpen ? 'Hide inspector' : 'Show inspector'}
        >
          {rightOpen ? '\u25B6' : '\u25C0'}
        </button>

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
            <pattern id="console-grid" width="30" height="30" patternUnits="userSpaceOnUse">
              <path d="M 30 0 L 0 0 0 30" fill="none" stroke="#E2E8F0" stroke-width="0.5" />
            </pattern>
            <marker id="console-arrow" markerWidth="8" markerHeight="6" refX="8" refY="3" orient="auto">
              <polygon points="0 0, 8 3, 0 6" fill="#CBD5E1" />
            </marker>
            <marker id="console-arrow-active" markerWidth="8" markerHeight="6" refX="8" refY="3" orient="auto">
              <polygon points="0 0, 8 3, 0 6" fill="#3B82F6" />
            </marker>
          </defs>

          <rect width={svgW} height={svgH} fill="url(#console-grid)" />

          <g transform={`translate(${transform.x},${transform.y}) scale(${transform.scale})`}>
            {/* Group clusters */}
            {layout.positionedGroups.map(g => {
              const dimmed = hoveredCluster != null && hoveredCluster !== g.id;
              return (
                <g
                  key={`cluster-${g.id}`}
                  style={{ opacity: dimmed ? 0.2 : 1, transition: 'opacity 0.2s' }}
                  onMouseEnter={() => setHoveredCluster(g.id)}
                  onMouseLeave={() => setHoveredCluster(null)}
                >
                  <rect x={g.x} y={g.y} width={g.width} height={g.height} rx={14}
                    fill={`${g.color}0A`} stroke={`${g.color}30`} stroke-width="1" />
                  <rect x={g.x} y={g.y} width={g.width} height={4} rx={2} fill={g.color} opacity="0.6" />
                  <text x={g.x + 10} y={g.y + 18} font-size="10.5" font-weight="700"
                    font-family="var(--font-sans)" fill={g.color} style={{ pointerEvents: 'none' }}>
                    {g.label}
                  </text>
                  <g>
                    <rect x={g.x + g.width - 32} y={g.y + 8} width={22} height={16} rx={8} fill={g.color} opacity="0.2" />
                    <text x={g.x + g.width - 21} y={g.y + 16} text-anchor="middle" dominant-baseline="central"
                      font-size="9" font-weight="700" font-family="var(--font-mono)" fill={g.color}
                      style={{ pointerEvents: 'none' }}>
                      {g.nodeCount}
                    </text>
                  </g>
                </g>
              );
            })}

            {/* Edges */}
            {edges.map((edge, i) => {
              const from = nodePos.get(edge.source);
              const to = nodePos.get(edge.target);
              if (!from || !to) return null;

              const highlighted = connectedEdges.has(i) || hoveredEdge === i;
              const dimmedByNode = hoveredNode != null && !highlighted;
              const dimmedByCluster = hoveredCluster != null && !(
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

              const avgMs = (edge as any).avgLatencyMs || 0;
              const callCount = (edge as any).callCount || 0;
              const latencyStr = avgMs > 0
                ? avgMs < 1 ? '<1ms' : avgMs < 1000 ? `${Math.round(avgMs)}ms` : `${(avgMs / 1000).toFixed(1)}s`
                : '';
              const labelText = latencyStr ? `${edge.label || ''} ${latencyStr}` : (edge.label || '');
              const labelW = labelText.length * 5 + 12;
              const baseWidth = callCount > 50 ? 2.5 : callCount > 10 ? 1.8 : callCount > 0 ? 1.2 : 0.8;

              return (
                <g
                  key={`e-${i}`}
                  style={{ opacity: dimmed ? 0.08 : 1, transition: 'opacity 0.2s', cursor: 'default' }}
                  onMouseEnter={() => setHoveredEdge(i)}
                  onMouseLeave={() => setHoveredEdge(null)}
                >
                  <path d={path} fill="none" stroke={edgeColor}
                    stroke-width={highlighted ? baseWidth + 1 : baseWidth}
                    stroke-dasharray={isDashed ? '5 3' : 'none'}
                    marker-end={highlighted ? 'url(#console-arrow-active)' : 'url(#console-arrow)'} />
                  {labelText && (
                    <>
                      <rect x={midX - labelW / 2} y={midY - 7} width={labelW} height={14} rx={3}
                        fill="white" stroke={highlighted ? '#3B82F6' : '#E2E8F0'} stroke-width="0.5" />
                      <text x={midX} y={midY + 1} text-anchor="middle" dominant-baseline="central"
                        font-size="8" font-family="var(--font-sans)"
                        fill={highlighted ? '#3B82F6' : '#94A3B8'} style={{ pointerEvents: 'none' }}>
                        {labelText}
                      </text>
                    </>
                  )}
                </g>
              );
            })}

            {/* Nodes */}
            {layout.positionedNodes.map(n => {
              const groupColor = groupColorMap.get(n.group) || '#94A3B8';
              const healthColor = getNodeHealthColor(n);
              const isHovered = hoveredNode === n.id;
              const isSelected = selectedNode?.id === n.id;
              const isHighlighted = highlightService != null && (
                n.service === highlightService ||
                (n.requestService != null && n.requestService === highlightService) ||
                n.id.includes(highlightService)
              );
              const dimmedByNode = hoveredNode != null && !connectedNodes.has(n.id);
              const dimmedByCluster = hoveredCluster != null && n.group !== hoveredCluster;
              const dimmed = dimmedByNode || dimmedByCluster;
              const isPulsing = pulsingNodes.has(n.id);
              const truncated = n.label.length > 18 ? n.label.slice(0, 17) + '\u2026' : n.label;

              // Traffic-based sizing (scale width slightly based on request count)
              const svc = n.requestService || n.service;
              const count = statsCounts[svc] || 0;
              const maxCount = Math.max(...Object.values(statsCounts), 1);
              const sizeScale = 1 + (count / maxCount) * 0.15;
              const nodeW = RES_W * sizeScale;

              // Determine stroke color: health > highlight > group
              const strokeColor = isHighlighted ? groupColor
                : isSelected ? groupColor
                : healthColor || `${groupColor}50`;

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
                  onClick={(e: MouseEvent) => { e.stopPropagation(); handleNodeClick(n); }}
                >
                  {/* Pulse glow */}
                  {isPulsing && (
                    <rect x={n.x - 4} y={n.y - 4} width={nodeW + 8} height={RES_H + 8} rx={14}
                      fill="none" stroke={groupColor} stroke-width="3" opacity="0.6"
                      style={{ animation: 'topo-pulse 1.5s ease-out' }} />
                  )}
                  {/* Health ring */}
                  {healthColor && (
                    <rect x={n.x - 2} y={n.y - 2} width={nodeW + 4} height={RES_H + 4} rx={12}
                      fill="none" stroke={healthColor} stroke-width="2" opacity="0.4" />
                  )}
                  {/* Shadow */}
                  <rect x={n.x + 1} y={n.y + 2} width={nodeW} height={RES_H} rx={10} fill="rgba(0,0,0,0.06)" />
                  {/* Node body */}
                  <rect x={n.x} y={n.y} width={nodeW} height={RES_H} rx={10}
                    fill={isHighlighted ? `${groupColor}30` : isSelected ? `${groupColor}25` : isPulsing ? `${groupColor}18` : isHovered ? `${groupColor}20` : 'white'}
                    stroke={strokeColor}
                    stroke-width={isHighlighted ? 3.5 : isSelected ? 3 : isPulsing ? 2.5 : isHovered ? 2.5 : 1.5}
                    style={{ transition: 'all 0.15s ease' }} />
                  {/* Left accent */}
                  <rect x={n.x} y={n.y + 8} width={4} height={RES_H - 16} rx={2}
                    fill={groupColor} opacity={isHovered ? 1 : 0.6} />
                  {/* Health dot */}
                  {healthColor && (
                    <circle cx={n.x + nodeW - 12} cy={n.y + 12} r={4} fill={healthColor} />
                  )}
                  {/* Name */}
                  <text x={n.x + 14} y={n.y + RES_H / 2 - 4} dominant-baseline="central"
                    font-size="11.5" font-weight="600" font-family="var(--font-sans)" fill="#1E293B"
                    style={{ pointerEvents: 'none' }}>
                    {truncated}
                  </text>
                  {/* Type label */}
                  <text x={n.x + 14} y={n.y + RES_H / 2 + 9} dominant-baseline="central"
                    font-size="8.5" font-family="var(--font-sans)" fill="#94A3B8"
                    style={{ pointerEvents: 'none' }}>
                    {n.type}
                  </text>
                  {/* Tooltip */}
                  {isHovered && (
                    <g style={{ pointerEvents: 'none' }}>
                      <rect x={n.x + nodeW / 2 - 75} y={n.y - 36} width={150} height={30} rx={5}
                        fill="#0F172A" opacity="0.92" />
                      <text x={n.x + nodeW / 2} y={n.y - 25} text-anchor="middle"
                        font-size="9.5" font-weight="600" font-family="var(--font-sans)" fill="white">
                        {n.label}
                      </text>
                      <text x={n.x + nodeW / 2} y={n.y - 12} text-anchor="middle"
                        font-size="8.5" font-family="var(--font-sans)" fill="#94A3B8">
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
            <rect width={minimapW} height={minimapH} rx={4} fill="white" stroke="#E2E8F0" stroke-width="1" opacity="0.92" />
            <g transform={`scale(${minimapScale})`}>
              {layout.positionedGroups.map(g => (
                <rect key={`mm-c-${g.id}`} x={g.x} y={g.y} width={g.width} height={g.height} rx={4} fill={g.color} opacity={0.15} />
              ))}
              {layout.positionedNodes.map(n => {
                const gc = groupColorMap.get(n.group) || '#94A3B8';
                return <rect key={`mm-${n.id}`} x={n.x} y={n.y} width={RES_W} height={RES_H} rx={2} fill={gc} opacity={0.5} />;
              })}
              {edges.map((edge, i) => {
                const from = nodePos.get(edge.source);
                const to = nodePos.get(edge.target);
                if (!from || !to) return null;
                return <line key={`mme-${i}`} x1={from.cx} y1={from.cy} x2={to.cx} y2={to.cy} stroke="#CBD5E1" stroke-width="1" opacity="0.3" />;
              })}
            </g>
            <rect
              x={(-transform.x / transform.scale) * minimapScale}
              y={(-transform.y / transform.scale) * minimapScale}
              width={(svgW / transform.scale) * minimapScale}
              height={(svgH / transform.scale) * minimapScale}
              rx={2} fill="none" stroke="#3B82F6" stroke-width="1.5" opacity="0.6" />
          </g>
        </svg>
      </div>

      {/* Right panel — Service Inspector */}
      <div class={`console-right ${rightOpen ? '' : 'collapsed'}`}>
        {selectedNode ? (
          <ServiceInspector
            node={selectedNode}
            edges={topoData.edges as any}
            nodes={topoData.nodes}
            onSelectNode={(n) => setSelectedNode(n)}
          />
        ) : (
          <div style={{ padding: 24, textAlign: 'center', color: 'var(--n400)' }}>
            <div style={{ fontSize: 28, marginBottom: 8, opacity: 0.3 }}>{'\u2190'}</div>
            <div style={{ fontSize: 13, fontWeight: 500 }}>Select a node</div>
            <div style={{ fontSize: 11, marginTop: 4 }}>Click a node on the topology graph or a request in the explorer to inspect it.</div>
          </div>
        )}
      </div>
    </div>
  );
}
