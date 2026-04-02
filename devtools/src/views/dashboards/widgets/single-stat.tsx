import type { QueryResult } from '../types';

interface SingleStatWidgetProps {
  title: string;
  data: QueryResult | null;
  unit?: string;
  thresholds?: { warning: number; critical: number };
}

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

function computeTrend(points: { time: number; value: number }[]): {
  direction: 'up' | 'down' | 'flat';
  percent: number;
} {
  if (points.length < 2) return { direction: 'flat', percent: 0 };

  // Compare the average of the last quarter to the first quarter
  const quarter = Math.max(1, Math.floor(points.length / 4));
  const firstSlice = points.slice(0, quarter);
  const lastSlice = points.slice(-quarter);

  const firstAvg = firstSlice.reduce((s, p) => s + p.value, 0) / firstSlice.length;
  const lastAvg = lastSlice.reduce((s, p) => s + p.value, 0) / lastSlice.length;

  if (firstAvg === 0) return { direction: lastAvg > 0 ? 'up' : 'flat', percent: 0 };

  const change = ((lastAvg - firstAvg) / firstAvg) * 100;
  const direction = Math.abs(change) < 1 ? 'flat' : change > 0 ? 'up' : 'down';
  return { direction, percent: Math.abs(change) };
}

function colorForValue(
  value: number,
  thresholds?: { warning: number; critical: number },
): string {
  if (!thresholds) return 'var(--text-primary)';
  if (value >= thresholds.critical) return 'var(--error)';
  if (value >= thresholds.warning) return 'var(--warning)';
  return 'var(--success)';
}

export function SingleStatWidget({
  title,
  data,
  unit,
  thresholds,
}: SingleStatWidgetProps) {
  const points = data?.series?.[0]?.points ?? [];
  const currentValue = points.length > 0 ? points[points.length - 1].value : 0;
  const trend = computeTrend(points);
  const valueColor = colorForValue(currentValue, thresholds);

  const trendArrow =
    trend.direction === 'up' ? '\u2191' :
    trend.direction === 'down' ? '\u2193' : '';

  const trendColor =
    trend.direction === 'up' ? 'var(--error)' :
    trend.direction === 'down' ? 'var(--success)' :
    'var(--text-tertiary)';

  // Sparkline from the last N points
  const sparkPoints = points.slice(-20);
  const sparkW = 120;
  const sparkH = 32;

  let sparkPath = '';
  if (sparkPoints.length >= 2) {
    const vals = sparkPoints.map((p) => p.value);
    const min = Math.min(...vals);
    const max = Math.max(...vals);
    const range = max - min || 1;
    sparkPath = sparkPoints
      .map((p, i) => {
        const x = (i / (sparkPoints.length - 1)) * sparkW;
        const y = sparkH - ((p.value - min) / range) * (sparkH - 4) - 2;
        return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`;
      })
      .join(' ');
  }

  return (
    <div class="dashboard-widget dashboard-widget-single-stat">
      <div class="dashboard-widget-header">
        <span class="dashboard-widget-title">{title}</span>
      </div>
      <div class="single-stat-body">
        <div class="single-stat-value" style={{ color: valueColor }}>
          {points.length > 0 ? formatValue(currentValue, unit) : '--'}
          {unit && <span class="single-stat-unit">{unit}</span>}
        </div>
        <div class="single-stat-trend" style={{ color: trendColor }}>
          {trendArrow && (
            <span class="single-stat-trend-arrow">{trendArrow}</span>
          )}
          {trend.percent > 0 && (
            <span class="single-stat-trend-pct">{trend.percent.toFixed(1)}%</span>
          )}
        </div>
        {sparkPath && (
          <svg
            class="single-stat-sparkline"
            width={sparkW}
            height={sparkH}
            viewBox={`0 0 ${sparkW} ${sparkH}`}
          >
            <path
              d={sparkPath}
              fill="none"
              stroke={valueColor}
              stroke-width="1.5"
              stroke-linecap="round"
              stroke-linejoin="round"
              opacity="0.5"
            />
          </svg>
        )}
      </div>
    </div>
  );
}
