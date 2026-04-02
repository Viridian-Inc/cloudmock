import { useState, useMemo } from 'preact/hooks';
import type { QueryResult } from '../types';

interface HeatmapWidgetProps {
  title: string;
  data: QueryResult | null;
  unit?: string;
  height?: number;
}

/** Interpolate between two colors via RGB */
function interpolateColor(
  lowR: number, lowG: number, lowB: number,
  highR: number, highG: number, highB: number,
  t: number,
): string {
  const r = Math.round(lowR + (highR - lowR) * t);
  const g = Math.round(lowG + (highG - lowG) * t);
  const b = Math.round(lowB + (highB - lowB) * t);
  return `rgb(${r}, ${g}, ${b})`;
}

// Low color: --bg-tertiary approximation (dark grey)
const LOW_R = 40, LOW_G = 40, LOW_B = 46;
// High color: --brand-teal (#4AE5F8)
const HIGH_R = 74, HIGH_G = 229, HIGH_B = 248;

function formatValue(value: number, unit?: string): string {
  if (unit === 'ms') {
    if (value < 1) return `${(value * 1000).toFixed(0)}us`;
    if (value < 1000) return `${value.toFixed(1)}ms`;
    return `${(value / 1000).toFixed(2)}s`;
  }
  if (unit === '%') return `${value.toFixed(2)}%`;
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
  if (value >= 1000) return `${(value / 1000).toFixed(1)}k`;
  return value.toFixed(1);
}

interface TooltipInfo {
  x: number;
  y: number;
  label: string;
  timeBucket: string;
  value: number;
}

export function HeatmapWidget({ title, data, unit, height = 140 }: HeatmapWidgetProps) {
  const [tooltip, setTooltip] = useState<TooltipInfo | null>(null);

  const { grid, xLabels, minVal, maxVal } = useMemo(() => {
    if (!data?.series?.length) {
      return { grid: [] as number[][], xLabels: [] as string[], minVal: 0, maxVal: 0 };
    }

    const series = data.series;

    // Determine time labels from the first series
    const xLabels: string[] = [];
    if (series[0]?.points.length > 0) {
      const points = series[0].points;
      const step = Math.max(1, Math.floor(points.length / Math.min(points.length, 8)));
      for (let i = 0; i < points.length; i += step) {
        const d = new Date(points[i].time);
        xLabels.push(
          d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' }),
        );
      }
    }

    // Build grid[y][x] = value
    let minVal = Infinity;
    let maxVal = -Infinity;
    const grid: number[][] = [];
    for (const s of series) {
      const row: number[] = [];
      for (const p of s.points) {
        row.push(p.value);
        if (p.value < minVal) minVal = p.value;
        if (p.value > maxVal) maxVal = p.value;
      }
      grid.push(row);
    }

    if (!isFinite(minVal)) minVal = 0;
    if (!isFinite(maxVal)) maxVal = 0;

    return { grid, xLabels, minVal, maxVal };
  }, [data]);

  if (grid.length === 0) {
    return (
      <div class="dashboard-widget dashboard-widget-heatmap">
        <div class="dashboard-widget-header">
          <span class="dashboard-widget-title">{title}</span>
        </div>
        <div class="dashboard-widget-body" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <span style={{ color: 'var(--text-tertiary)', fontSize: '12px' }}>No data</span>
        </div>
      </div>
    );
  }

  const cols = Math.max(...grid.map((r) => r.length));
  const rows = grid.length;
  const LABEL_WIDTH = 60;
  const BOTTOM_LABEL_HEIGHT = 16;
  const svgWidth = 600;
  const svgHeight = height;
  const cellW = (svgWidth - LABEL_WIDTH) / cols;
  const cellH = (svgHeight - BOTTOM_LABEL_HEIGHT) / rows;
  const range = maxVal - minVal || 1;

  const handleMouseEnter = (
    e: MouseEvent,
    rowIdx: number,
    colIdx: number,
    value: number,
  ) => {
    const target = e.currentTarget as SVGRectElement;
    const svg = target.closest('svg');
    if (!svg) return;
    const rect = svg.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    // Determine time label for this column
    const series0 = data?.series?.[0];
    let timeBucket = `col ${colIdx}`;
    if (series0?.points?.[colIdx]) {
      const d = new Date(series0.points[colIdx].time);
      timeBucket = d.toLocaleTimeString(undefined, {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      });
    }

    setTooltip({
      x,
      y,
      label: Object.values(data?.series?.[rowIdx]?.labels ?? {}).filter(Boolean).join(', ') || 'default',
      timeBucket,
      value,
    });
  };

  const handleMouseLeave = () => {
    setTooltip(null);
  };

  return (
    <div class="dashboard-widget dashboard-widget-heatmap">
      <div class="dashboard-widget-header">
        <span class="dashboard-widget-title">{title}</span>
      </div>
      <div class="dashboard-widget-body" style={{ position: 'relative', padding: '4px' }}>
        <svg
          width="100%"
          height={svgHeight}
          viewBox={`0 0 ${svgWidth} ${svgHeight}`}
          preserveAspectRatio="none"
          style={{ display: 'block' }}
        >
          {/* Y-axis labels */}
          {grid.map((_, rowIdx) => (
            <text
              key={`y-${rowIdx}`}
              x={LABEL_WIDTH - 4}
              y={rowIdx * cellH + cellH / 2 + 3}
              text-anchor="end"
              fill="var(--text-tertiary)"
              font-size="9"
              font-family="var(--font-sans)"
            >
              {(
                Object.values(data?.series?.[rowIdx]?.labels ?? {})
                  .filter(Boolean)
                  .join(', ') || 'default'
              ).slice(0, 10)}
            </text>
          ))}

          {/* Heatmap cells */}
          {grid.map((row, rowIdx) =>
            row.map((value, colIdx) => {
              const t = range > 0 ? (value - minVal) / range : 0;
              const color = interpolateColor(
                LOW_R, LOW_G, LOW_B,
                HIGH_R, HIGH_G, HIGH_B,
                t,
              );
              return (
                <rect
                  key={`${rowIdx}-${colIdx}`}
                  x={LABEL_WIDTH + colIdx * cellW}
                  y={rowIdx * cellH}
                  width={Math.max(1, cellW - 1)}
                  height={Math.max(1, cellH - 1)}
                  fill={color}
                  rx="1"
                  style={{ cursor: 'pointer' }}
                  onMouseEnter={(e: any) => handleMouseEnter(e, rowIdx, colIdx, value)}
                  onMouseLeave={handleMouseLeave}
                />
              );
            }),
          )}

          {/* X-axis labels (sparse) */}
          {xLabels.map((label, i) => {
            const step = Math.max(1, Math.floor(cols / Math.min(cols, 8)));
            const xPos = LABEL_WIDTH + i * step * cellW + (step * cellW) / 2;
            return (
              <text
                key={`x-${i}`}
                x={xPos}
                y={svgHeight - 2}
                text-anchor="middle"
                fill="var(--text-tertiary)"
                font-size="8"
                font-family="var(--font-sans)"
              >
                {label}
              </text>
            );
          })}
        </svg>

        {/* Tooltip */}
        {tooltip && (
          <div
            class="heatmap-tooltip"
            style={{
              position: 'absolute',
              left: `${Math.min(tooltip.x + 8, 280)}px`,
              top: `${Math.max(tooltip.y - 36, 0)}px`,
              background: 'var(--bg-elevated, #1e1e24)',
              border: '1px solid var(--border-default)',
              borderRadius: 'var(--radius-sm)',
              padding: '4px 8px',
              fontSize: '10px',
              color: 'var(--text-primary)',
              pointerEvents: 'none',
              whiteSpace: 'nowrap',
              zIndex: 10,
              fontFamily: 'var(--font-mono)',
            }}
          >
            <div style={{ fontWeight: 600 }}>{tooltip.label}</div>
            <div>{tooltip.timeBucket}: {formatValue(tooltip.value, unit)}</div>
          </div>
        )}
      </div>
    </div>
  );
}
