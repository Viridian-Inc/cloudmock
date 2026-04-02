import { useEffect, useState, useCallback, useRef, useMemo, useImperativeHandle } from 'preact/hooks';
import { forwardRef } from 'preact/compat';
import type { TopoNode, TopoEdge } from './index';
import type { ServiceMetrics, DeployEvent } from '../../lib/health';
import type { IncidentInfo } from '../../lib/types';
import { computeHealthState, isUserFacing, getBlastRadius, formatEdgeLabel } from '../../lib/health';
import type { SavedLayout } from './layouts';
import ELK from 'elkjs/lib/elk.bundled.js';

/** Compute approximate path length from a list of points */
function pathLength(pts: { x: number; y: number }[]): number {
  let len = 0;
  for (let i = 1; i < pts.length; i++) {
    const dx = pts[i].x - pts[i - 1].x;
    const dy = pts[i].y - pts[i - 1].y;
    len += Math.sqrt(dx * dx + dy * dy);
  }
  return len;
}

/** How many animated packets to show for a given callCount */
function packetCount(callCount: number): number {
  if (callCount > 50) return 3;
  if (callCount > 10) return 2;
  return 1;
}

/** Compute bounding box for a group of layout nodes */
function groupBounds(layoutNodes: LayoutNode[]): { x: number; y: number; w: number; h: number } {
  let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
  for (const ln of layoutNodes) {
    minX = Math.min(minX, ln.x);
    minY = Math.min(minY, ln.y);
    maxX = Math.max(maxX, ln.x + ln.w);
    maxY = Math.max(maxY, ln.y + ln.h);
  }
  return { x: minX, y: minY, w: maxX - minX, h: maxY - minY };
}

export interface CanvasLayoutState {
  pinnedPositions: Record<string, { x: number; y: number }>;
  pan: { x: number; y: number };
  scale: number;
}

export interface TopologyCanvasHandle {
  getLayoutState: () => CanvasLayoutState;
  applyLayout: (state: CanvasLayoutState) => void;
}

interface CanvasProps {
  nodes: TopoNode[];
  edges: TopoEdge[];
  selectedNodeId: string | null;
  onSelectNode: (node: TopoNode | null) => void;
  loading: boolean;
  metrics: ServiceMetrics[];
  deploys: DeployEvent[];
  incidents: IncidentInfo[];
  metricsHistory?: Map<string, number[]>;
  inactiveNodeIds?: Set<string>;
  onApplyLayout?: (layout: SavedLayout) => void;
}

// Assign colors by group — auto-generated from hash if not listed
const PALETTE = [
  '#4AE5F8', '#a78bfa', '#60a5fa', '#fbbf24', '#818cf8',
  '#4ade80', '#f472b6', '#22d3ee', '#f87171', '#fb923c',
  '#34d399', '#c084fc', '#38bdf8', '#facc15', '#a3e635',
];

const groupColorCache = new Map<string, string>();
function getColor(group: string): string {
  if (!groupColorCache.has(group)) {
    let hash = 5381;
    for (let i = 0; i < group.length; i++) {
      hash = ((hash << 5) + hash + group.charCodeAt(i)) >>> 0;
    }
    groupColorCache.set(group, PALETTE[hash % PALETTE.length]);
  }
  return groupColorCache.get(group)!;
}

const HEALTH_COLORS = {
  green: '#22c55e',
  yellow: '#fbbf24',
  red: '#ef4444',
} as const;

interface LayoutNode { id: string; x: number; y: number; w: number; h: number; node: TopoNode }
interface LayoutEdge { id: string; points: { x: number; y: number }[]; edge: TopoEdge }

const NODE_W = 200;
const NODE_H = 44;

/** Tiny inline sparkline SVG for canvas nodes */
function MiniSparkline({ data, color }: { data: number[]; color: string }) {
  const w = 40, h = 12;
  if (data.length < 2) return null;
  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max - min || 1;
  const points = data.map((v, i) => {
    const x = (i / (data.length - 1)) * w;
    const y = h - ((v - min) / range) * (h - 2) - 1;
    return `${x},${y}`;
  }).join(' ');
  return (
    <svg width={w} height={h} viewBox={`0 0 ${w} ${h}`} style={{ display: 'block' }}>
      <polyline points={points} fill="none" stroke={color} stroke-width="1" stroke-linejoin="round" />
    </svg>
  );
}

async function runLayout(nodes: TopoNode[], edges: TopoEdge[]) {
  const elk = new ELK();
  // Only include edges whose source+target both exist
  const nodeIds = new Set(nodes.map((n) => n.id));
  const validEdges = edges.filter((e) => nodeIds.has(e.source) && nodeIds.has(e.target) && e.source !== e.target);

  const result = await elk.layout({
    id: 'root',
    layoutOptions: {
      'elk.algorithm': 'layered',
      'elk.direction': 'DOWN',
      'elk.spacing.nodeNode': '50',
      'elk.layered.spacing.nodeNodeBetweenLayers': '80',
      'elk.spacing.edgeEdge': '20',
      'elk.layered.nodePlacement.strategy': 'BRANDES_KOEPF',
      'elk.layered.crossingMinimization.strategy': 'LAYER_SWEEP',
      'elk.padding': '[top=50,left=50,bottom=50,right=50]',
      'elk.layered.considerModelOrder.strategy': 'NODES_AND_EDGES',
    },
    children: nodes.map((n) => ({ id: n.id, width: NODE_W, height: NODE_H })),
    edges: validEdges.map((e, i) => ({ id: `e${i}`, sources: [e.source], targets: [e.target] })),
  } as any);

  const nodeMap = new Map(nodes.map((n) => [n.id, n]));
  return {
    nodes: (result.children || []).map((c: any) => ({
      id: c.id, x: c.x, y: c.y, w: c.width, h: c.height, node: nodeMap.get(c.id)!,
    })),
    edges: (result.edges || []).map((e: any, i: number) => {
      const pts: { x: number; y: number }[] = [];
      for (const sec of e.sections || []) {
        pts.push(sec.startPoint);
        if (sec.bendPoints) pts.push(...sec.bendPoints);
        pts.push(sec.endPoint);
      }
      return { id: e.id, points: pts, edge: validEdges[i] };
    }),
    w: result.width || 800,
    h: result.height || 600,
  };
}

function smoothPath(pts: { x: number; y: number }[]): string {
  if (pts.length < 2) return '';
  const [start, ...rest] = pts;
  let d = `M ${start.x} ${start.y}`;
  for (const p of rest) {
    d += ` L ${p.x} ${p.y}`;
  }
  return d;
}

function edgeMidpoint(pts: { x: number; y: number }[]): { x: number; y: number } {
  if (pts.length === 0) return { x: 0, y: 0 };
  const mid = Math.floor(pts.length / 2);
  return pts[mid];
}

function getServiceKey(node: TopoNode): string {
  return node.service || node.id.replace(/^svc:/, '');
}

/** Check if a deploy happened within the last hour */
function hasRecentDeploy(svcKey: string, deploys: DeployEvent[]): boolean {
  const oneHourAgo = Date.now() - 60 * 60 * 1000;
  return deploys.some(
    (d) =>
      (d.service === svcKey || d.service === `svc:${svcKey}`) &&
      new Date(d.timestamp).getTime() > oneHourAgo,
  );
}

function getIncidentCount(svcKey: string, incidents: IncidentInfo[]): number {
  return incidents.filter((i) => i.affected_services.includes(svcKey)).length;
}

/** Recompute edge points when a node has been dragged to a new position.
 *  Shifts all points on the edge that originated from source/target by the delta. */
function adjustEdgePoints(
  le: LayoutEdge,
  nodePositions: Map<string, { x: number; y: number }>,
  elkNodes: LayoutNode[],
): { x: number; y: number }[] {
  const srcPos = nodePositions.get(le.edge.source);
  const tgtPos = nodePositions.get(le.edge.target);
  const srcElk = elkNodes.find((n) => n.id === le.edge.source);
  const tgtElk = elkNodes.find((n) => n.id === le.edge.target);
  if (!srcPos || !tgtPos || !srcElk || !tgtElk) return le.points;
  const srcDx = srcPos.x - srcElk.x;
  const srcDy = srcPos.y - srcElk.y;
  const tgtDx = tgtPos.x - tgtElk.x;
  const tgtDy = tgtPos.y - tgtElk.y;
  if (srcDx === 0 && srcDy === 0 && tgtDx === 0 && tgtDy === 0) return le.points;
  const pts = le.points;
  if (pts.length < 2) return pts;
  // Interpolate: first point gets source delta, last gets target delta, middle interpolates
  return pts.map((p, i) => {
    const t = pts.length > 1 ? i / (pts.length - 1) : 0;
    return { x: p.x + srcDx * (1 - t) + tgtDx * t, y: p.y + srcDy * (1 - t) + tgtDy * t };
  });
}

const MINIMAP_W = 150;
const MINIMAP_H = 100;

export const TopologyCanvas = forwardRef<TopologyCanvasHandle, CanvasProps>(function TopologyCanvas({
  nodes: rawNodes, edges, selectedNodeId, onSelectNode, loading,
  metrics, deploys, incidents, metricsHistory, inactiveNodeIds, onApplyLayout,
}: CanvasProps, ref) {
  const [elkLayout, setElkLayout] = useState<{ nodes: LayoutNode[]; edges: LayoutEdge[]; w: number; h: number } | null>(null);
  const [pan, setPan] = useState({ x: 20, y: 20 });
  const [scale, setScale] = useState(0.85);
  const [hoveredNodeId, setHoveredNodeId] = useState<string | null>(null);
  const [minimapVisible, setMinimapVisible] = useState(true);
  const containerRef = useRef<HTMLDivElement>(null);

  // --- Drag-to-reposition state ---
  const [pinnedPositions, setPinnedPositions] = useState<Map<string, { x: number; y: number }>>(new Map());
  const dragStateRef = useRef<{ nodeId: string; startX: number; startY: number; origX: number; origY: number; moved: boolean } | null>(null);

  // Expose layout state and apply method to parent via ref
  useImperativeHandle(ref, () => ({
    getLayoutState: (): CanvasLayoutState => {
      const positions: Record<string, { x: number; y: number }> = {};
      pinnedPositions.forEach((pos, id) => { positions[id] = pos; });
      return { pinnedPositions: positions, pan, scale };
    },
    applyLayout: (state: CanvasLayoutState) => {
      const newPinned = new Map<string, { x: number; y: number }>();
      for (const [id, pos] of Object.entries(state.pinnedPositions)) {
        newPinned.set(id, pos);
      }
      setPinnedPositions(newPinned);
      setPan(state.pan);
      setScale(state.scale);
    },
  }), [pinnedPositions, pan, scale]);

  useEffect(() => {
    if (rawNodes.length === 0) { setElkLayout(null); return; }
    runLayout(rawNodes, edges).then(setElkLayout).catch(() => setElkLayout(null));
  }, [rawNodes, edges]);

  // Compute effective layout: ELK positions with pinned overrides
  const layout = useMemo(() => {
    if (!elkLayout) return null;
    if (pinnedPositions.size === 0) return elkLayout;
    const nodes = elkLayout.nodes.map((ln) => {
      const pinned = pinnedPositions.get(ln.id);
      return pinned ? { ...ln, x: pinned.x, y: pinned.y } : ln;
    });
    // Recompute edges with adjusted points
    const posMap = new Map(nodes.map((n) => [n.id, { x: n.x, y: n.y }]));
    const adjustedEdges = elkLayout.edges.map((le) => ({
      ...le,
      points: adjustEdgePoints(le, posMap, elkLayout.nodes),
    }));
    return { ...elkLayout, nodes, edges: adjustedEdges };
  }, [elkLayout, pinnedPositions]);

  // Build metrics lookup
  const metricsMap = useMemo(() => {
    const m = new Map<string, ServiceMetrics>();
    for (const sm of metrics) m.set(sm.service, sm);
    return m;
  }, [metrics]);

  // Compute health per node
  const healthMap = useMemo(() => {
    const h = new Map<string, 'green' | 'yellow' | 'red'>();
    for (const n of rawNodes) {
      const svcKey = getServiceKey(n);
      const m = metricsMap.get(svcKey);
      const hasIncident = incidents.some((i) => i.affected_services.includes(svcKey));
      h.set(n.id, computeHealthState(m, undefined, hasIncident));
    }
    return h;
  }, [rawNodes, metricsMap, incidents]);

  // Compute blast radius for selected node
  const blastRadiusSet = useMemo(() => {
    if (!selectedNodeId) return null;
    const br = getBlastRadius(selectedNodeId, edges);
    br.add(selectedNodeId); // include self
    return br;
  }, [selectedNodeId, edges]);

  // Compute user-facing impacted nodes
  const userFacingImpacts = useMemo(() => {
    const impacted: { service: string; health: 'yellow' | 'red' }[] = [];
    for (const n of rawNodes) {
      const health = healthMap.get(n.id);
      if (health !== 'yellow' && health !== 'red') continue;
      if (isUserFacing(n.id, rawNodes, edges)) {
        impacted.push({ service: n.label, health });
      }
    }
    return impacted;
  }, [rawNodes, edges, healthMap]);

  // The "active" node for edge label visibility: hovered takes precedence, then selected
  const activeNodeId = hoveredNodeId ?? selectedNodeId;

  // Group layout nodes by their group field for bounding boxes
  const nodeGroups = useMemo(() => {
    if (!layout) return new Map<string, { nodes: LayoutNode[]; color: string }>();
    const groups = new Map<string, { nodes: LayoutNode[]; color: string }>();
    for (const ln of layout.nodes) {
      const g = ln.node.group;
      if (!groups.has(g)) groups.set(g, { nodes: [], color: getColor(g) });
      groups.get(g)!.nodes.push(ln);
    }
    return groups;
  }, [layout]);

  const handleWheel = useCallback((e: WheelEvent) => {
    e.preventDefault();
    setScale((s) => Math.max(0.15, Math.min(2.5, s * (e.deltaY > 0 ? 0.92 : 1.08))));
  }, []);

  const handleMouseDown = useCallback((e: MouseEvent) => {
    if (e.button !== 0) return;
    // Don't start pan if a node drag is active
    if (dragStateRef.current) return;
    const sx = e.clientX - pan.x;
    const sy = e.clientY - pan.y;
    const move = (me: MouseEvent) => setPan({ x: me.clientX - sx, y: me.clientY - sy });
    const up = () => { window.removeEventListener('mousemove', move); window.removeEventListener('mouseup', up); };
    window.addEventListener('mousemove', move);
    window.addEventListener('mouseup', up);
  }, [pan]);

  // --- Node drag handlers ---
  const handleNodeMouseDown = useCallback((e: MouseEvent, nodeId: string, nodeX: number, nodeY: number) => {
    if (e.button !== 0) return;
    e.stopPropagation(); // prevent pan
    dragStateRef.current = { nodeId, startX: e.clientX, startY: e.clientY, origX: nodeX, origY: nodeY, moved: false };

    const move = (me: MouseEvent) => {
      const ds = dragStateRef.current;
      if (!ds) return;
      const dx = me.clientX - ds.startX;
      const dy = me.clientY - ds.startY;
      if (!ds.moved && Math.abs(dx) + Math.abs(dy) > 4) {
        ds.moved = true;
      }
      if (ds.moved) {
        const newX = ds.origX + dx / scale;
        const newY = ds.origY + dy / scale;
        setPinnedPositions((prev) => {
          const next = new Map(prev);
          next.set(ds.nodeId, { x: newX, y: newY });
          return next;
        });
      }
    };

    const up = () => {
      window.removeEventListener('mousemove', move);
      window.removeEventListener('mouseup', up);
      dragStateRef.current = null;
    };

    window.addEventListener('mousemove', move);
    window.addEventListener('mouseup', up);
  }, [scale]);

  const handleNodeClick = useCallback((e: MouseEvent, node: TopoNode) => {
    e.stopPropagation();
    // Only select if no drag occurred
    if (!dragStateRef.current || !dragStateRef.current.moved) {
      onSelectNode(node);
    }
  }, [onSelectNode]);

  const handleNodeDblClick = useCallback((e: MouseEvent, nodeId: string) => {
    e.stopPropagation();
    // Unpin a pinned node on double-click
    if (pinnedPositions.has(nodeId)) {
      setPinnedPositions((prev) => {
        const next = new Map(prev);
        next.delete(nodeId);
        return next;
      });
    }
  }, [pinnedPositions]);

  // --- Minimap click-to-pan ---
  const handleMinimapClick = useCallback((e: MouseEvent) => {
    if (!layout || !containerRef.current) return;
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const clickX = e.clientX - rect.left;
    const clickY = e.clientY - rect.top;
    // Map minimap coords to graph coords
    const pad = 8;
    const innerW = MINIMAP_W - pad * 2;
    const innerH = MINIMAP_H - pad * 2;
    const scaleX = innerW / layout.w;
    const scaleY = innerH / layout.h;
    const miniScale = Math.min(scaleX, scaleY);
    const offsetX = pad + (innerW - layout.w * miniScale) / 2;
    const offsetY = pad + (innerH - layout.h * miniScale) / 2;
    const graphX = (clickX - offsetX) / miniScale;
    const graphY = (clickY - offsetY) / miniScale;
    // Center the viewport on that point
    const cw = containerRef.current.clientWidth;
    const ch = containerRef.current.clientHeight;
    setPan({ x: cw / 2 - graphX * scale, y: ch / 2 - graphY * scale });
  }, [layout, scale]);

  if (loading) return <div class="topology-canvas-placeholder"><span>Loading topology...</span></div>;
  if (!layout) return <div class="topology-canvas-placeholder"><span>Computing layout...</span></div>;

  // --- Minimap calculations ---
  const miniPad = 8;
  const miniInnerW = MINIMAP_W - miniPad * 2;
  const miniInnerH = MINIMAP_H - miniPad * 2;
  const miniScaleX = miniInnerW / layout.w;
  const miniScaleY = miniInnerH / layout.h;
  const miniScale = Math.min(miniScaleX, miniScaleY);
  const miniOffX = miniPad + (miniInnerW - layout.w * miniScale) / 2;
  const miniOffY = miniPad + (miniInnerH - layout.h * miniScale) / 2;

  // Viewport rectangle in minimap coordinates
  const containerEl = containerRef.current;
  const vpW = containerEl ? containerEl.clientWidth : 800;
  const vpH = containerEl ? containerEl.clientHeight : 600;
  const vpGraphX = -pan.x / scale;
  const vpGraphY = -pan.y / scale;
  const vpGraphW = vpW / scale;
  const vpGraphH = vpH / scale;
  const vpMiniX = miniOffX + vpGraphX * miniScale;
  const vpMiniY = miniOffY + vpGraphY * miniScale;
  const vpMiniW = vpGraphW * miniScale;
  const vpMiniH = vpGraphH * miniScale;

  return (
    <div class="topology-canvas-container">
      <div class="topology-toolbar">
        <span class="topology-stats">{rawNodes.length} services · {edges.length} connections</span>
        <div style={{ display: 'flex', gap: '6px', alignItems: 'center' }}>
          {pinnedPositions.size > 0 && (
            <button class="btn btn-ghost" style={{ fontSize: '11px' }}
              onClick={() => setPinnedPositions(new Map())}>
              Unpin all ({pinnedPositions.size})
            </button>
          )}
          <button class="btn btn-ghost" style={{ fontSize: '11px' }}
            onClick={() => setMinimapVisible((v) => !v)}>
            {minimapVisible ? 'Hide' : 'Show'} minimap
          </button>
          <button class="btn btn-ghost" onClick={() => { setPan({ x: 20, y: 20 }); setScale(0.85); }}>Reset</button>
        </div>
      </div>
      <div class="topology-graph" ref={containerRef} onWheel={handleWheel} onMouseDown={handleMouseDown}
        onClick={(e) => {
          if (e.target === containerRef.current || (e.target as HTMLElement).classList.contains('topology-graph'))
            onSelectNode(null);
        }}>
        <div class="topology-graph-inner" style={{
          transform: `translate(${pan.x}px, ${pan.y}px) scale(${scale})`,
          transformOrigin: '0 0',
          width: `${layout.w}px`,
          height: `${layout.h}px`,
        }}>
          {/* Edges as SVG */}
          <svg class="topology-edges-svg" style={{
            position: 'absolute', top: 0, left: 0,
            width: layout.w, height: layout.h,
            pointerEvents: 'none', overflow: 'visible',
          }}>
            <defs>
              <marker id="arrow" markerWidth="6" markerHeight="5" refX="6" refY="2.5" orient="auto">
                <path d="M0,0 L6,2.5 L0,5" fill="none" stroke="rgba(74,229,248,0.4)" stroke-width="1" />
              </marker>
              {/* Glow filter for animated packets */}
              <filter id="packet-glow" x="-50%" y="-50%" width="200%" height="200%">
                <feGaussianBlur stdDeviation="2" result="blur" />
                <feMerge>
                  <feMergeNode in="blur" />
                  <feMergeNode in="SourceGraphic" />
                </feMerge>
              </filter>
            </defs>

            {/* Domain group bounding boxes */}
            {[...nodeGroups.entries()].map(([group, { nodes: groupNodes, color }]) => {
              if (groupNodes.length < 2) return null;
              const pad = 20;
              const bounds = groupBounds(groupNodes);
              return (
                <g key={`group-${group}`}>
                  <rect
                    x={bounds.x - pad} y={bounds.y - pad}
                    width={bounds.w + pad * 2} height={bounds.h + pad * 2}
                    rx={12} ry={12}
                    fill={color} fill-opacity={0.05}
                    stroke={color} stroke-opacity={0.1} stroke-width={1}
                  />
                  <text
                    x={bounds.x - pad + 8} y={bounds.y - pad + 14}
                    class="topology-group-label"
                    fill={color} fill-opacity={0.35}
                  >
                    {group}
                  </text>
                </g>
              );
            })}

            {layout.edges.map((le) => {
              const isSel = le.edge.source === selectedNodeId || le.edge.target === selectedNodeId;
              const isActive = le.edge.source === activeNodeId || le.edge.target === activeNodeId;
              const label = formatEdgeLabel(le.edge.callCount ?? 0, le.edge.avgLatencyMs ?? 0);
              const mid = edgeMidpoint(le.points);
              const d = smoothPath(le.points);
              const cc = le.edge.callCount ?? 0;
              const isBLE = le.edge.type === 'ble';
              const pktColor = isBLE ? '#a78bfa' : '#4AE5F8';
              const len = pathLength(le.points);
              // Duration: shorter edges = faster. Scale 1-4s based on length
              const dur = Math.max(1, Math.min(4, len / 100));
              const count = cc > 0 ? packetCount(cc) : 0;

              const pathId = `edge-path-${le.id}`;

              return (
                <g key={le.id}>
                  <path id={pathId} d={d} fill="none"
                    stroke={isSel ? '#4AE5F8' : 'rgba(74,229,248,0.18)'}
                    stroke-width={isSel ? 2 : 1}
                    marker-end="url(#arrow)" />

                  {/* Animated packets for edges with traffic */}
                  {count > 0 && Array.from({ length: count }, (_, i) => (
                    <circle key={`${le.id}-pkt-${i}`}
                      r={3} fill={pktColor} opacity={0.9}
                      filter="url(#packet-glow)">
                      <animateMotion
                        dur={`${dur}s`}
                        repeatCount="indefinite"
                        begin={`${(i * dur) / count}s`}>
                        <mpath href={`#${pathId}`} />
                      </animateMotion>
                    </circle>
                  ))}

                  {/* Edge label: visible only when connected node is hovered/selected */}
                  {label && (
                    <text x={mid.x} y={mid.y - 6} text-anchor="middle"
                      class={`topology-edge-label ${isActive ? 'topology-edge-label-visible' : ''}`}>
                      {label}
                    </text>
                  )}
                </g>
              );
            })}
          </svg>

          {/* Nodes as HTML pills */}
          {layout.nodes.map((ln) => {
            const n = ln.node;
            const color = getColor(n.group);
            const isSel = n.id === selectedNodeId;
            const health = healthMap.get(n.id) ?? 'green';
            const healthColor = HEALTH_COLORS[health];
            const svcKey = getServiceKey(n);
            const incidentCount = getIncidentCount(svcKey, incidents);
            const recentDeploy = hasRecentDeploy(svcKey, deploys);
            const isUFImpact = (health === 'yellow' || health === 'red') && isUserFacing(n.id, rawNodes, edges);
            const isPinned = pinnedPositions.has(n.id);

            // Blast radius dimming: if a node is selected and this node is NOT in the radius
            const dimmed = blastRadiusSet !== null && !blastRadiusSet.has(n.id);
            // Time travel dimming: node had no traffic in the current time window
            const inactive = inactiveNodeIds != null && inactiveNodeIds.has(n.id);

            const classes = [
              'gh-node',
              isSel ? 'gh-node-selected' : '',
              `health-${health}`,
              isUFImpact ? 'user-impact-pulse' : '',
              dimmed ? 'node-dimmed' : '',
              inactive ? 'gh-node-inactive' : '',
            ].filter(Boolean).join(' ');

            return (
              <div key={n.id}
                class={classes}
                style={{
                  position: 'absolute',
                  left: `${ln.x}px`,
                  top: `${ln.y}px`,
                  width: `${ln.w}px`,
                  height: `${ln.h}px`,
                  '--node-color': color,
                  borderColor: isSel ? color : healthColor,
                  boxShadow: isSel ? `0 0 12px ${color}44` : undefined,
                } as any}
                onMouseDown={(e) => handleNodeMouseDown(e as unknown as MouseEvent, n.id, ln.x, ln.y)}
                onClick={(e) => handleNodeClick(e as unknown as MouseEvent, n)}
                onDblClick={(e) => handleNodeDblClick(e as unknown as MouseEvent, n.id)}
                onMouseEnter={() => setHoveredNodeId(n.id)}
                onMouseLeave={() => setHoveredNodeId(null)}>
                <span class="gh-node-dot" style={{ background: healthColor }} />
                <div class="gh-node-text">
                  <span class="gh-node-label">{n.label}</span>
                  <span class="gh-node-sub">
                    {n.group} · {n.type}
                    {(() => {
                      const m = metricsMap.get(svcKey);
                      if (m && m.totalCalls > 0) return ` · ${m.totalCalls} req`;
                      return null;
                    })()}
                  </span>
                </div>
                {isPinned && <span class="gh-node-pin-indicator" title="Pinned (double-click to unpin)" />}
                {incidentCount > 0 && (
                  <span class="node-incident-badge">{incidentCount}</span>
                )}
                {recentDeploy && <span class="node-deploy-marker" />}
                {metricsHistory?.get(`${svcKey}:req`) && (metricsHistory.get(`${svcKey}:req`)!).length >= 2 && (
                  <div class="gh-node-sparkline">
                    <MiniSparkline
                      data={metricsHistory.get(`${svcKey}:req`)!}
                      color={healthColor}
                    />
                  </div>
                )}
                {(() => {
                  const m = metricsMap.get(svcKey);
                  if (m && m.totalCalls > 0) {
                    return (
                      <span class="node-req-badge" style={{ color }}>
                        {m.totalCalls} req
                      </span>
                    );
                  }
                  return null;
                })()}
              </div>
            );
          })}
        </div>

        {/* User-facing impact banner */}
        {userFacingImpacts.length > 0 && (
          <div class="user-impact-banner">
            <span class="user-impact-banner-icon">&#9888;</span>
            <span>
              User-Facing Impact:{' '}
              {userFacingImpacts.map((imp, i) => (
                <span key={i}>
                  {i > 0 ? ', ' : ''}
                  <strong>{imp.service}</strong>
                  {' '}({imp.health === 'red' ? 'degraded' : 'elevated latency'})
                </span>
              ))}
            </span>
          </div>
        )}

        {/* Minimap */}
        {minimapVisible && (
          <svg class="topology-minimap" width={MINIMAP_W} height={MINIMAP_H}
            onClick={handleMinimapClick as any}>
            {/* Node dots */}
            {layout.nodes.map((ln) => {
              const health = healthMap.get(ln.id) ?? 'green';
              const cx = miniOffX + (ln.x + ln.w / 2) * miniScale;
              const cy = miniOffY + (ln.y + ln.h / 2) * miniScale;
              return (
                <circle key={`mini-${ln.id}`}
                  cx={cx} cy={cy} r={2.5}
                  fill={HEALTH_COLORS[health]} opacity={0.9} />
              );
            })}
            {/* Viewport rectangle */}
            <rect
              x={vpMiniX} y={vpMiniY} width={vpMiniW} height={vpMiniH}
              fill="rgba(74, 229, 248, 0.08)" stroke="rgba(74, 229, 248, 0.5)"
              stroke-width={1} rx={2} ry={2}
            />
          </svg>
        )}
      </div>
      <div class="topology-legend">
        {[...new Set(rawNodes.map((n) => n.group))].map((g) => (
          <span key={g} class="topology-legend-item">
            <span class="topology-legend-dot" style={{ background: getColor(g) }} />
            {g}
          </span>
        ))}
      </div>
    </div>
  );
});
