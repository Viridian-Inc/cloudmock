import { useState, useEffect, useRef } from 'preact/hooks';
import { api } from '../api';

// Internal developer dashboard -- SVG content is generated programmatically
// from our own topology API data, not from user input.

function renderTopologySVG(svgEl: SVGSVGElement, data: any) {
  if (!data || !data.nodes || data.nodes.length === 0) {
    svgEl.textContent = '';
    const text = document.createElementNS('http://www.w3.org/2000/svg', 'text');
    text.setAttribute('x', '50%');
    text.setAttribute('y', '50%');
    text.setAttribute('text-anchor', 'middle');
    text.setAttribute('fill', '#94A3B8');
    text.setAttribute('font-family', 'Figtree');
    text.setAttribute('font-size', '16');
    text.textContent = 'No topology data';
    svgEl.appendChild(text);
    return;
  }

  const nodes = data.nodes;
  const edges = data.edges || [];
  const W = svgEl.clientWidth || 900;
  const H = svgEl.clientHeight || 600;
  const ns = 'http://www.w3.org/2000/svg';

  const cols = Math.ceil(Math.sqrt(nodes.length));
  const cellW = W / (cols + 1);
  const cellH = H / (Math.ceil(nodes.length / cols) + 1);

  const positions: Record<string, { x: number; y: number }> = {};
  nodes.forEach((node: any, i: number) => {
    const row = Math.floor(i / cols);
    const col = i % cols;
    positions[node.id] = {
      x: cellW * (col + 1),
      y: cellH * (row + 1),
    };
  });

  // Clear and rebuild using DOM APIs
  svgEl.textContent = '';

  // Defs for arrowhead marker
  const defs = document.createElementNS(ns, 'defs');
  const marker = document.createElementNS(ns, 'marker');
  marker.setAttribute('id', 'arrowhead');
  marker.setAttribute('markerWidth', '10');
  marker.setAttribute('markerHeight', '7');
  marker.setAttribute('refX', '10');
  marker.setAttribute('refY', '3.5');
  marker.setAttribute('orient', 'auto');
  const polygon = document.createElementNS(ns, 'polygon');
  polygon.setAttribute('points', '0 0, 10 3.5, 0 7');
  polygon.setAttribute('fill', '#94A3B8');
  marker.appendChild(polygon);
  defs.appendChild(marker);
  svgEl.appendChild(defs);

  // Edges
  edges.forEach((edge: any) => {
    const s = positions[edge.source];
    const t = positions[edge.target];
    if (!s || !t) return;
    const line = document.createElementNS(ns, 'line');
    line.setAttribute('x1', String(s.x));
    line.setAttribute('y1', String(s.y));
    line.setAttribute('x2', String(t.x));
    line.setAttribute('y2', String(t.y));
    line.setAttribute('stroke', '#CBD5E1');
    line.setAttribute('stroke-width', '1.5');
    line.setAttribute('marker-end', 'url(#arrowhead)');
    svgEl.appendChild(line);

    if (edge.label) {
      const mx = (s.x + t.x) / 2;
      const my = (s.y + t.y) / 2;
      const text = document.createElementNS(ns, 'text');
      text.setAttribute('x', String(mx));
      text.setAttribute('y', String(my - 6));
      text.setAttribute('text-anchor', 'middle');
      text.setAttribute('font-size', '10');
      text.setAttribute('fill', '#94A3B8');
      text.setAttribute('font-family', 'Figtree');
      text.textContent = edge.label;
      svgEl.appendChild(text);
    }
  });

  // Nodes
  nodes.forEach((node: any) => {
    const p = positions[node.id];
    const g = document.createElementNS(ns, 'g');
    const rect = document.createElementNS(ns, 'rect');
    rect.setAttribute('x', String(p.x - 50));
    rect.setAttribute('y', String(p.y - 18));
    rect.setAttribute('width', '100');
    rect.setAttribute('height', '36');
    rect.setAttribute('rx', '8');
    rect.setAttribute('fill', '#0A1F44');
    rect.setAttribute('stroke', '#097FF5');
    rect.setAttribute('stroke-width', '1.5');
    g.appendChild(rect);

    const text = document.createElementNS(ns, 'text');
    text.setAttribute('x', String(p.x));
    text.setAttribute('y', String(p.y + 5));
    text.setAttribute('text-anchor', 'middle');
    text.setAttribute('font-size', '12');
    text.setAttribute('fill', 'white');
    text.setAttribute('font-family', 'Figtree');
    text.setAttribute('font-weight', '600');
    text.textContent = node.name;
    g.appendChild(text);

    svgEl.appendChild(g);
  });
}

export function TopologyPage() {
  const [topology, setTopology] = useState<any>(null);
  const svgRef = useRef<SVGSVGElement>(null);

  useEffect(() => {
    api('/api/topology').then(setTopology).catch(() => {});
  }, []);

  useEffect(() => {
    if (!topology || !svgRef.current) return;
    renderTopologySVG(svgRef.current, topology);
  }, [topology]);

  return (
    <div>
      <div class="mb-6">
        <h1 class="page-title">Service Topology</h1>
        <p class="page-desc">Connections between AWS service mocks</p>
      </div>
      <div class="card topology-container">
        <svg ref={svgRef} />
      </div>
    </div>
  );
}
