interface SparklineProps {
  data: number[];
  color: string;
  label: string;
  value: string;
  height?: number;
}

const WIDTH = 120;

export function Sparkline({ data, color, label, value, height = 60 }: SparklineProps) {
  if (data.length === 0) {
    return (
      <div class="sparkline-container">
        <div class="sparkline-header">
          <span class="sparkline-label">{label}</span>
          <span class="sparkline-value" style={{ color }}>--</span>
        </div>
        <svg class="sparkline-svg" width={WIDTH} height={height} viewBox={`0 0 ${WIDTH} ${height}`}>
          <line x1="0" y1={height / 2} x2={WIDTH} y2={height / 2}
            stroke="rgba(255,255,255,0.05)" stroke-width="1" stroke-dasharray="4,4" />
        </svg>
      </div>
    );
  }

  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max - min || 1;
  const padTop = 4;
  const padBottom = 4;
  const drawH = height - padTop - padBottom;

  const points = data.map((v, i) => {
    const x = (i / Math.max(data.length - 1, 1)) * WIDTH;
    const y = padTop + drawH - ((v - min) / range) * drawH;
    return `${x},${y}`;
  });

  const polyline = points.join(' ');
  // Build filled area polygon: line + bottom edge
  const firstX = (0 / Math.max(data.length - 1, 1)) * WIDTH;
  const lastX = ((data.length - 1) / Math.max(data.length - 1, 1)) * WIDTH;
  const areaPath = `${polyline} ${lastX},${height} ${firstX},${height}`;

  return (
    <div class="sparkline-container">
      <div class="sparkline-header">
        <span class="sparkline-label">{label}</span>
        <span class="sparkline-value" style={{ color }}>{value}</span>
      </div>
      <svg class="sparkline-svg" width={WIDTH} height={height} viewBox={`0 0 ${WIDTH} ${height}`}>
        <polygon points={areaPath} fill={color} opacity="0.1" />
        <polyline points={polyline} fill="none" stroke={color} stroke-width="1.5"
          stroke-linejoin="round" stroke-linecap="round" />
      </svg>
    </div>
  );
}
