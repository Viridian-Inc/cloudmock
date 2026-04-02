import { useState, useMemo } from 'preact/hooks';
import type { QueryResult } from '../types';

interface TableWidgetProps {
  title: string;
  data: QueryResult | null;
  unit?: string;
}

type SortDir = 'asc' | 'desc';
type SortCol = 'service' | 'value' | 'trend';

interface TableRow {
  service: string;
  value: number;
  trend: number; // percent change
  trendDir: 'up' | 'down' | 'flat';
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

function computeTrendForPoints(points: { time: number; value: number }[]): {
  direction: 'up' | 'down' | 'flat';
  percent: number;
} {
  if (points.length < 2) return { direction: 'flat', percent: 0 };

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

export function TableWidget({ title, data, unit }: TableWidgetProps) {
  const [sortCol, setSortCol] = useState<SortCol>('value');
  const [sortDir, setSortDir] = useState<SortDir>('desc');

  const rows: TableRow[] = useMemo(() => {
    if (!data?.series?.length) return [];

    return data.series.map((s) => {
      const label =
        Object.values(s.labels).filter(Boolean).join(', ') || 'default';
      const points = s.points;
      const lastValue = points.length > 0 ? points[points.length - 1].value : 0;
      const trend = computeTrendForPoints(points);

      return {
        service: label,
        value: lastValue,
        trend: trend.percent,
        trendDir: trend.direction,
      };
    });
  }, [data]);

  const sortedRows = useMemo(() => {
    const sorted = [...rows];
    sorted.sort((a, b) => {
      let cmp = 0;
      if (sortCol === 'service') cmp = a.service.localeCompare(b.service);
      else if (sortCol === 'value') cmp = a.value - b.value;
      else cmp = a.trend - b.trend;
      return sortDir === 'asc' ? cmp : -cmp;
    });
    return sorted;
  }, [rows, sortCol, sortDir]);

  const handleSort = (col: SortCol) => {
    if (sortCol === col) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortCol(col);
      setSortDir('desc');
    }
  };

  const sortIndicator = (col: SortCol) => {
    if (sortCol !== col) return '';
    return sortDir === 'asc' ? ' \u2191' : ' \u2193';
  };

  return (
    <div class="dashboard-widget dashboard-widget-table">
      <div class="dashboard-widget-header">
        <span class="dashboard-widget-title">{title}</span>
      </div>
      <div class="metrics-table-wrap" style={{ flex: 1, minHeight: 0, overflowY: 'auto' }}>
        <table class="metrics-table">
          <thead>
            <tr>
              <th
                style={{ cursor: 'pointer', userSelect: 'none' }}
                onClick={() => handleSort('service')}
              >
                Service{sortIndicator('service')}
              </th>
              <th
                style={{ cursor: 'pointer', userSelect: 'none' }}
                onClick={() => handleSort('value')}
              >
                Value{sortIndicator('value')}
              </th>
              <th
                style={{ cursor: 'pointer', userSelect: 'none' }}
                onClick={() => handleSort('trend')}
              >
                Trend{sortIndicator('trend')}
              </th>
            </tr>
          </thead>
          <tbody>
            {sortedRows.length === 0 && (
              <tr>
                <td colSpan={3} style={{ textAlign: 'center', color: 'var(--text-tertiary)' }}>
                  No data
                </td>
              </tr>
            )}
            {sortedRows.map((row) => {
              const trendColor =
                row.trendDir === 'up'
                  ? 'var(--error)'
                  : row.trendDir === 'down'
                    ? 'var(--success)'
                    : 'var(--text-tertiary)';
              const trendArrow =
                row.trendDir === 'up'
                  ? '\u2191'
                  : row.trendDir === 'down'
                    ? '\u2193'
                    : '\u2014';

              return (
                <tr key={row.service}>
                  <td>{row.service}</td>
                  <td style={{ fontFamily: 'var(--font-mono)' }}>
                    {formatValue(row.value, unit)}
                  </td>
                  <td style={{ color: trendColor, fontFamily: 'var(--font-mono)' }}>
                    {trendArrow} {row.trend > 0 ? `${row.trend.toFixed(1)}%` : '--'}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
