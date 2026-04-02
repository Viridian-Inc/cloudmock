export type WidgetType = 'timeseries' | 'single-stat' | 'gauge' | 'table' | 'heatmap';

export type AggregationFn = 'avg' | 'sum' | 'min' | 'max' | 'count' | 'p50' | 'p95' | 'p99';

export interface MetricQuery {
  metric: string;          // e.g. "http.request.duration"
  aggregation: AggregationFn;
  filters: Record<string, string>;  // e.g. { service: "api-gateway" }
  groupBy?: string;        // e.g. "service"
}

export interface QueryResult {
  series: {
    labels: Record<string, string>;
    points: { time: number; value: number }[];
  }[];
  meta: {
    query: string;
    executionMs: number;
  };
}

export interface TimeWindowOption {
  label: string;
  value: string;        // e.g. "15m", "1h", "6h", "24h", "7d"
  seconds: number;
}

export interface Widget {
  id: string;
  title: string;
  type: WidgetType;
  query: MetricQuery;
  /** Grid position — column start (0-based) */
  col: number;
  /** Grid position — row start (0-based) */
  row: number;
  /** Column span (1-12) */
  colSpan: number;
  /** Row span (each row is ~160px) */
  rowSpan: number;
  /** Thresholds for gauge/single-stat coloring */
  thresholds?: {
    warning: number;
    critical: number;
  };
  /** Unit label */
  unit?: string;
}

export interface Dashboard {
  id: string;
  name: string;
  description?: string;
  widgets: Widget[];
  timeWindow: string;     // references TimeWindowOption.value
  refreshInterval: number; // seconds, 0 = manual
  createdAt: string;
  updatedAt: string;
}
