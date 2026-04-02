import { LineChart } from '../../metrics/line-chart';
import type { QueryResult } from '../types';

interface TimeseriesWidgetProps {
  title: string;
  data: QueryResult | null;
  unit?: string;
  color?: string;
  height?: number;
}

export function TimeseriesWidget({
  title,
  data,
  unit = '',
  color = 'var(--brand-teal)',
  height = 160,
}: TimeseriesWidgetProps) {
  const points = data?.series?.[0]?.points ?? [];

  return (
    <div class="dashboard-widget dashboard-widget-timeseries">
      <div class="dashboard-widget-header">
        <span class="dashboard-widget-title">{title}</span>
      </div>
      <div class="dashboard-widget-body">
        <LineChart
          data={points}
          color={color}
          label={title}
          unit={unit}
          height={height}
        />
      </div>
    </div>
  );
}
