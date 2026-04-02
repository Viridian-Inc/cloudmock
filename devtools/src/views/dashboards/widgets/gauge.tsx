import type { QueryResult } from '../types';

interface GaugeWidgetProps {
  title: string;
  data: QueryResult | null;
  unit?: string;
  thresholds?: { warning: number; critical: number };
  min?: number;
  max?: number;
}

function formatValue(value: number, unit?: string): string {
  if (unit === 'ms') {
    if (value < 1) return `${(value * 1000).toFixed(0)}us`;
    if (value < 1000) return `${value.toFixed(1)}ms`;
    return `${(value / 1000).toFixed(2)}s`;
  }
  if (unit === '%') return `${value.toFixed(1)}%`;
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
  if (value >= 1000) return `${(value / 1000).toFixed(1)}k`;
  return value.toFixed(1);
}

/**
 * SVG arc gauge with green/yellow/red zones.
 *
 * Draws a 240-degree arc from -120 to +120 degrees.
 * The arc is divided into three color zones based on thresholds.
 */
export function GaugeWidget({
  title,
  data,
  unit,
  thresholds,
  min = 0,
  max = 100,
}: GaugeWidgetProps) {
  const points = data?.series?.[0]?.points ?? [];
  const currentValue = points.length > 0 ? points[points.length - 1].value : 0;

  const SIZE = 160;
  const CX = SIZE / 2;
  const CY = SIZE / 2 + 10;
  const R = 58;
  const STROKE = 10;

  // Arc from -120deg to +120deg (240-degree sweep)
  const START_ANGLE = -210; // degrees (measured from 3-o'clock, CCW)
  const END_ANGLE = 30;
  const SWEEP = END_ANGLE - START_ANGLE; // 240 degrees

  function angleToXY(angleDeg: number): { x: number; y: number } {
    const rad = (angleDeg * Math.PI) / 180;
    return { x: CX + R * Math.cos(rad), y: CY + R * Math.sin(rad) };
  }

  function arcPath(startDeg: number, endDeg: number): string {
    const s = angleToXY(startDeg);
    const e = angleToXY(endDeg);
    const sweep = endDeg - startDeg;
    const largeArc = Math.abs(sweep) > 180 ? 1 : 0;
    return `M ${s.x} ${s.y} A ${R} ${R} 0 ${largeArc} 1 ${e.x} ${e.y}`;
  }

  // Threshold fractions
  const range = max - min || 1;
  const warnFrac = thresholds ? Math.min(1, (thresholds.warning - min) / range) : 0.6;
  const critFrac = thresholds ? Math.min(1, (thresholds.critical - min) / range) : 0.85;
  const valueFrac = Math.min(1, Math.max(0, (currentValue - min) / range));

  // Zone angles
  const greenEnd = START_ANGLE + warnFrac * SWEEP;
  const yellowEnd = START_ANGLE + critFrac * SWEEP;

  // Needle angle
  const needleAngle = START_ANGLE + valueFrac * SWEEP;
  const needleTip = angleToXY(needleAngle);

  // Value color
  let valueColor = 'var(--success)';
  if (thresholds) {
    if (currentValue >= thresholds.critical) valueColor = 'var(--error)';
    else if (currentValue >= thresholds.warning) valueColor = 'var(--warning)';
  }

  return (
    <div class="dashboard-widget dashboard-widget-gauge">
      <div class="dashboard-widget-header">
        <span class="dashboard-widget-title">{title}</span>
      </div>
      <div class="gauge-body">
        <svg
          width={SIZE}
          height={SIZE * 0.72}
          viewBox={`0 0 ${SIZE} ${SIZE * 0.72}`}
          class="gauge-svg"
        >
          {/* Background track */}
          <path
            d={arcPath(START_ANGLE, END_ANGLE)}
            fill="none"
            stroke="rgba(255,255,255,0.06)"
            stroke-width={STROKE}
            stroke-linecap="round"
          />

          {/* Green zone */}
          <path
            d={arcPath(START_ANGLE, greenEnd)}
            fill="none"
            stroke="rgba(54, 217, 130, 0.35)"
            stroke-width={STROKE}
            stroke-linecap="round"
          />

          {/* Yellow zone */}
          <path
            d={arcPath(greenEnd, yellowEnd)}
            fill="none"
            stroke="rgba(250, 208, 101, 0.35)"
            stroke-width={STROKE}
          />

          {/* Red zone */}
          <path
            d={arcPath(yellowEnd, END_ANGLE)}
            fill="none"
            stroke="rgba(255, 78, 94, 0.35)"
            stroke-width={STROKE}
            stroke-linecap="round"
          />

          {/* Needle */}
          <line
            x1={CX}
            y1={CY}
            x2={needleTip.x}
            y2={needleTip.y}
            stroke={valueColor}
            stroke-width="2"
            stroke-linecap="round"
          />
          <circle cx={CX} cy={CY} r="4" fill={valueColor} />

          {/* Value text */}
          <text
            x={CX}
            y={CY + 22}
            text-anchor="middle"
            fill={valueColor}
            font-size="16"
            font-weight="700"
            font-family="var(--font-mono)"
          >
            {points.length > 0 ? formatValue(currentValue, unit) : '--'}
          </text>
        </svg>
      </div>
    </div>
  );
}
