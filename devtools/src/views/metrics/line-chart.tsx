import { useState, useRef, useCallback } from 'preact/hooks';

export interface LineChartProps {
  data: { time: number; value: number }[];
  color: string;
  label: string;
  unit: string; // "req/s", "ms", "%"
  height?: number;
}

const PADDING = { top: 20, right: 16, bottom: 32, left: 52 };

function formatTime(ts: number): string {
  const d = new Date(ts);
  const h = d.getHours().toString().padStart(2, '0');
  const m = d.getMinutes().toString().padStart(2, '0');
  return `${h}:${m}`;
}

function formatValue(v: number, unit: string): string {
  if (unit === 'ms') {
    if (v < 1) return `${(v * 1000).toFixed(0)}us`;
    if (v < 1000) return `${v.toFixed(1)}ms`;
    return `${(v / 1000).toFixed(2)}s`;
  }
  if (unit === '%') return `${v.toFixed(2)}%`;
  if (v >= 1000) return `${(v / 1000).toFixed(1)}k`;
  return v.toFixed(0);
}

/** Generate nice Y-axis tick values */
function yTicks(min: number, max: number, count: number): number[] {
  if (max === min) return [min];
  const range = max - min;
  const rawStep = range / (count - 1);
  const magnitude = Math.pow(10, Math.floor(Math.log10(rawStep)));
  const residual = rawStep / magnitude;
  let step: number;
  if (residual <= 1.5) step = 1 * magnitude;
  else if (residual <= 3) step = 2 * magnitude;
  else if (residual <= 7) step = 5 * magnitude;
  else step = 10 * magnitude;

  const tickMin = Math.floor(min / step) * step;
  const ticks: number[] = [];
  for (let v = tickMin; v <= max + step * 0.01; v += step) {
    ticks.push(v);
    if (ticks.length > count + 2) break;
  }
  return ticks;
}

/** Generate X-axis time labels roughly every 5 minutes */
function xLabels(startTs: number, endTs: number): { ts: number; label: string }[] {
  const range = endTs - startTs;
  if (range <= 0) return [];

  // Target ~5 minute intervals, but adapt to the data range
  const fiveMinMs = 5 * 60 * 1000;
  let interval = fiveMinMs;
  if (range < 10 * 60 * 1000) interval = 2 * 60 * 1000; // 2m for ranges < 10m
  if (range > 60 * 60 * 1000) interval = 15 * 60 * 1000; // 15m for ranges > 1h
  if (range > 6 * 60 * 60 * 1000) interval = 60 * 60 * 1000; // 1h for ranges > 6h

  const firstTick = Math.ceil(startTs / interval) * interval;
  const labels: { ts: number; label: string }[] = [];
  for (let ts = firstTick; ts <= endTs; ts += interval) {
    labels.push({ ts, label: formatTime(ts) });
  }
  return labels;
}

export function LineChart({ data, color, label, unit, height = 160 }: LineChartProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [hover, setHover] = useState<{ x: number; y: number; time: number; value: number } | null>(null);

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!svgRef.current || data.length === 0) return;
      const rect = svgRef.current.getBoundingClientRect();
      const mouseX = e.clientX - rect.left;
      const chartWidth = rect.width - PADDING.left - PADDING.right;
      const ratio = (mouseX - PADDING.left) / chartWidth;
      if (ratio < 0 || ratio > 1) {
        setHover(null);
        return;
      }
      const idx = Math.round(ratio * (data.length - 1));
      const clamped = Math.max(0, Math.min(data.length - 1, idx));
      const point = data[clamped];
      const px = PADDING.left + (clamped / Math.max(data.length - 1, 1)) * chartWidth;
      setHover({ x: px, y: e.clientY - rect.top, time: point.time, value: point.value });
    },
    [data],
  );

  const handleMouseLeave = useCallback(() => setHover(null), []);

  if (data.length === 0) {
    return (
      <div class="line-chart">
        <div class="line-chart-header">
          <span class="line-chart-label">{label}</span>
          <span class="line-chart-current" style={{ color }}>--</span>
        </div>
        <div class="line-chart-empty" style={{ height }}>
          No data
        </div>
      </div>
    );
  }

  const values = data.map((d) => d.value);
  const minVal = Math.min(...values);
  const maxVal = Math.max(...values);
  const yMin = Math.min(0, minVal);
  const yMax = maxVal === yMin ? yMin + 1 : maxVal * 1.1;
  const yRange = yMax - yMin || 1;

  const startTs = data[0].time;
  const endTs = data[data.length - 1].time;

  const ticks = yTicks(yMin, yMax, 4);
  const xLbls = xLabels(startTs, endTs);
  const tsRange = endTs - startTs || 1;

  const currentValue = data[data.length - 1].value;

  return (
    <div class="line-chart">
      <div class="line-chart-header">
        <span class="line-chart-label">{label}</span>
        <span class="line-chart-current" style={{ color }}>
          {formatValue(currentValue, unit)}
          <span class="line-chart-unit">{unit}</span>
        </span>
      </div>
      <svg
        ref={svgRef}
        class="line-chart-svg"
        width="100%"
        height={height}
        onMouseMove={handleMouseMove}
        onMouseLeave={handleMouseLeave}
      >
        {/* Chart is rendered in viewBox-independent coords using percentages */}
        {(() => {
          // We render with a fixed internal coordinate system
          const W = 800;
          const H = height;
          const chartW = W - PADDING.left - PADDING.right;
          const chartH = H - PADDING.top - PADDING.bottom;

          // Build line path
          const points = data.map((d, i) => {
            const x = PADDING.left + (i / Math.max(data.length - 1, 1)) * chartW;
            const y = PADDING.top + chartH - ((d.value - yMin) / yRange) * chartH;
            return { x, y };
          });

          const linePath = points.map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x},${p.y}`).join(' ');
          const areaPath = `${linePath} L${points[points.length - 1].x},${PADDING.top + chartH} L${points[0].x},${PADDING.top + chartH} Z`;

          // Hover line position (scaled to internal coords)
          let hoverLine: { x: number; point: { x: number; y: number } } | null = null;
          if (hover) {
            const ratio = (hover.x - PADDING.left) / (svgRef.current ? svgRef.current.getBoundingClientRect().width - PADDING.left - PADDING.right : chartW);
            const idx = Math.round(ratio * (data.length - 1));
            const clamped = Math.max(0, Math.min(data.length - 1, idx));
            hoverLine = { x: points[clamped].x, point: points[clamped] };
          }

          return (
            <svg viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" width="100%" height={H}>
              {/* Grid lines */}
              {ticks.map((t) => {
                const y = PADDING.top + chartH - ((t - yMin) / yRange) * chartH;
                return (
                  <g key={t}>
                    <line
                      x1={PADDING.left} y1={y} x2={W - PADDING.right} y2={y}
                      stroke="rgba(255,255,255,0.06)" stroke-width="1"
                    />
                    <text x={PADDING.left - 6} y={y + 3} text-anchor="end"
                      fill="rgba(255,255,255,0.35)" font-size="10" font-family="var(--font-mono)">
                      {formatValue(t, unit)}
                    </text>
                  </g>
                );
              })}

              {/* X-axis labels */}
              {xLbls.map((l) => {
                const x = PADDING.left + ((l.ts - startTs) / tsRange) * chartW;
                if (x < PADDING.left || x > W - PADDING.right) return null;
                return (
                  <text key={l.ts} x={x} y={H - 6} text-anchor="middle"
                    fill="rgba(255,255,255,0.35)" font-size="10" font-family="var(--font-mono)">
                    {l.label}
                  </text>
                );
              })}

              {/* Area fill */}
              <path d={areaPath} fill={color} opacity="0.08" />

              {/* Line */}
              <path d={linePath} fill="none" stroke={color} stroke-width="2"
                stroke-linejoin="round" stroke-linecap="round" vector-effect="non-scaling-stroke" />

              {/* Hover line + dot */}
              {hoverLine && (
                <g>
                  <line
                    x1={hoverLine.x} y1={PADDING.top}
                    x2={hoverLine.x} y2={PADDING.top + chartH}
                    stroke="rgba(255,255,255,0.2)" stroke-width="1" stroke-dasharray="4,4"
                  />
                  <circle cx={hoverLine.point.x} cy={hoverLine.point.y}
                    r="4" fill={color} stroke="var(--bg-primary)" stroke-width="2" />
                </g>
              )}
            </svg>
          );
        })()}
      </svg>

      {/* Tooltip */}
      {hover && (
        <div class="line-chart-tooltip" style={{ left: `${hover.x}px`, top: `${hover.y - 40}px` }}>
          <div class="line-chart-tooltip-time">{formatTime(hover.time)}</div>
          <div class="line-chart-tooltip-value" style={{ color }}>
            {formatValue(hover.value, unit)} {unit}
          </div>
        </div>
      )}
    </div>
  );
}
