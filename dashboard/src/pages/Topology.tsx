import { useState, useEffect, useRef, useCallback, useMemo } from 'preact/hooks';
import { api } from '../api';
import type { SSEState } from '../hooks/useSSE';

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

const RES_W = 120;
const RES_H = 28;
const RES_GAP_X = 10;
const RES_GAP_Y = 8;
const COLS = 2;
const CLUSTER_PAD_X = 14;
const CLUSTER_PAD_Y = 14;
const CLUSTER_HEADER = 28;
const GROUP_GAP_X = 60;
const GROUP_GAP_Y = 30;
const LAYER_GAP = 80;

// Group -> layer assignment (left to right)
const GROUP_LAYERS: Record<string, number> = {
  Client:       0,
  Plugins:      0,
  API:          1,
  Auth:         1,
  Compute:      2,
  'Core Data':  3,
  Features:     3,
  Admin:        3,
  Integrations: 3,
  Facilities:   3,
  Messaging:    4,
  Storage:      4,
  Security:     5,
  Monitoring:   5,
};

// Vertical order within each layer
const GROUP_ORDER: Record<string, number> = {
  Client:       0,
  Plugins:      1,
  API:          0,
  Auth:         1,
  Compute:      0,
  'Core Data':  0,
  Features:     1,
  Admin:        2,
  Integrations: 3,
  Facilities:   4,
  Messaging:    0,
  Storage:      1,
  Security:     0,
  Monitoring:   1,
};

// --- Layout engine ---

function layoutGraph(data: TopoData): {
  positionedNodes: PositionedNode[];
  positionedGroups: PositionedGroup[];
} {
  // Group nodes by their group field
  const nodesByGroup = new Map<string, TopoNode[]>();
  for (const n of data.nodes) {
    const arr = nodesByGroup.get(n.group) || [];
    arr.push(n);
    nodesByGroup.set(n.group, arr);
  }

  // Group info lookup
  const groupInfo = new Map<string, TopoGroup>();
  for (const g of data.groups) {
    groupInfo.set(g.id, g);
  }

  // Organise groups by layer
  const layerGroups = new Map<number, { group: TopoGroup; nodes: TopoNode[] }[]>();
  for (const [gid, nodes] of nodesByGroup) {
    const layer = GROUP_LAYERS[gid] ?? 3;
    const arr = layerGroups.get(layer) || [];
    const info = groupInfo.get(gid) || { id: gid, label: gid, color: '#94A3B8' };
    arr.push({ group: info, nodes });
    layerGroups.set(layer, arr);
  }

  // Sort within each layer by GROUP_ORDER
  for (const [, arr] of layerGroups) {
    arr.sort((a, b) => (GROUP_ORDER[a.group.id] ?? 99) - (GROUP_ORDER[b.group.id] ?? 99));
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

      // Position nodes in 2-column grid inside the cluster
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

  return { positionedNodes, positionedGroups };
}

// --- SVG helpers ---

function bezierPath(x1: number, y1: number, x2: number, y2: number): string {
  const dx = Math.abs(x2 - x1) * 0.45;
  return `M ${x1} ${y1} C ${x1 + dx} ${y1}, ${x2 - dx} ${y2}, ${x2} ${y2}`;
}

// --- Component ---

export function TopologyPage({ sse }: TopologyPageProps) {
  const [topoData, setTopoData] = useState<TopoData | null>(null);
  const [showAll, setShowAll] = useState(true);
  const [hoveredNode, setHoveredNode] = useState<string | null>(null);
  const [hoveredEdge, setHoveredEdge] = useState<number | null>(null);
  const [hoveredCluster, setHoveredCluster] = useState<string | null>(null);
  const [transform, setTransform] = useState({ x: 0, y: 0, scale: 1 });
  const svgRef = useRef<SVGSVGElement>(null);
  const dragging = useRef<{ startX: number; startY: number; origX: number; origY: number } | null>(null);

  // Fetch topology data
  useEffect(() => {
    api('/api/topology').then(data => {
      setTopoData(data);
    }).catch(() => {});
  }, []);

  // Layout
  const layout = useMemo(() => {
    if (!topoData) return null;
    return layoutGraph(topoData);
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

  // Minimap
  const minimapW = 180;
  const minimapH = 110;
  const minimapScale = useMemo(() => {
    return Math.min(minimapW / svgW, minimapH / svgH);
  }, [svgW, svgH]);

  // Node lookup by id for group membership
  const nodeGroupMap = useMemo(() => {
    if (!layout) return new Map<string, string>();
    const m = new Map<string, string>();
    for (const n of layout.positionedNodes) {
      m.set(n.id, n.group);
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

  return (
    <div>
      <div class="mb-6 flex items-center justify-between">
        <div>
          <h1 class="page-title">Service Topology</h1>
          <p class="page-desc">Unified resource graph with {topoData.nodes.length} resources across {topoData.groups.length} groups</p>
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
          {topoData.groups.map(g => (
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
              <span style={{ fontSize: '9px', opacity: 0.7 }}>
                ({(topoData.nodes.filter(n => n.group === g.id)).length})
              </span>
            </span>
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
                    rx={12}
                    fill={`${g.color}08`}
                    stroke={`${g.color}4D`}
                    stroke-width="1.5"
                    stroke-dasharray="6 3"
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

              const labelText = edge.label || '';
              const labelW = labelText.length * 5 + 10;

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
                    stroke-width={highlighted ? 2 : 1}
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
                        {edge.label || edge.type} ({edge.discovered})
                      </text>
                    </g>
                  )}
                </g>
              );
            })}

            {/* Resource nodes */}
            {layout.positionedNodes.map(n => {
              const groupColor = topoData.groups.find(g => g.id === n.group)?.color || '#94A3B8';
              const isHovered = hoveredNode === n.id;
              const dimmedByNode = hoveredNode && !connectedNodes.has(n.id);
              const dimmedByCluster = hoveredCluster && n.group !== hoveredCluster;
              const dimmed = dimmedByNode || dimmedByCluster;
              const truncated = n.label.length > 14 ? n.label.slice(0, 13) + '\u2026' : n.label;

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
                    navigateToResource(n.service, n.label);
                  }}
                >
                  <rect
                    x={n.x}
                    y={n.y}
                    width={RES_W}
                    height={RES_H}
                    rx={6}
                    fill={isHovered ? `${groupColor}30` : `${groupColor}1A`}
                    stroke={isHovered ? groupColor : `${groupColor}66`}
                    stroke-width={isHovered ? 2 : 1}
                    style={{ transition: 'all 0.15s ease' }}
                  />
                  <text
                    x={n.x + 8}
                    y={n.y + RES_H / 2 + 1}
                    dominant-baseline="central"
                    font-size="10"
                    font-family="var(--font-mono)"
                    fill="#334155"
                    style={{ pointerEvents: 'none' }}
                  >
                    {truncated}
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
                const groupColor = topoData.groups.find(g => g.id === n.group)?.color || '#94A3B8';
                return (
                  <rect
                    key={`mm-${n.id}`}
                    x={n.x}
                    y={n.y}
                    width={RES_W}
                    height={RES_H}
                    rx={2}
                    fill={groupColor}
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
    </div>
  );
}
