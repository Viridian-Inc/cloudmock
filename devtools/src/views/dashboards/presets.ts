import type { Widget } from './types';

export type DashboardLabel = 'infrastructure' | 'application' | 'security' | 'performance';
export type DashboardRole = 'all' | 'admin' | 'developer' | 'viewer';

export interface DashboardPreset {
  id: string;
  name: string;
  description: string;
  label: DashboardLabel;
  role: DashboardRole;
  widgets: Widget[];
}

export const DASHBOARD_LABELS: DashboardLabel[] = [
  'infrastructure',
  'application',
  'performance',
  'security',
];

export const PRESET_DASHBOARDS: DashboardPreset[] = [
  {
    id: 'preset-service-overview',
    name: 'Service Overview',
    description: 'Request rates, latency, and error rates across all services',
    label: 'infrastructure',
    role: 'all',
    widgets: [
      {
        id: 'ps-ov-1',
        title: 'Request Volume',
        type: 'timeseries',
        query: { metric: 'http.request.count', aggregation: 'sum', filters: {} },
        col: 0, row: 0, colSpan: 6, rowSpan: 1,
        unit: 'req/s',
      },
      {
        id: 'ps-ov-2',
        title: 'P99 Latency',
        type: 'timeseries',
        query: { metric: 'http.request.duration', aggregation: 'p99', filters: {} },
        col: 6, row: 0, colSpan: 6, rowSpan: 1,
        unit: 'ms',
      },
      {
        id: 'ps-ov-3',
        title: 'Error Rate',
        type: 'single-stat',
        query: { metric: 'http.error.rate', aggregation: 'avg', filters: {} },
        col: 0, row: 1, colSpan: 3, rowSpan: 1,
        unit: '%',
        thresholds: { warning: 1, critical: 5 },
      },
      {
        id: 'ps-ov-4',
        title: 'CPU Usage',
        type: 'gauge',
        query: { metric: 'system.cpu.usage', aggregation: 'avg', filters: {} },
        col: 3, row: 1, colSpan: 3, rowSpan: 1,
        unit: '%',
        thresholds: { warning: 70, critical: 90 },
      },
      {
        id: 'ps-ov-5',
        title: 'Avg Latency',
        type: 'single-stat',
        query: { metric: 'http.request.duration', aggregation: 'avg', filters: {} },
        col: 6, row: 1, colSpan: 3, rowSpan: 1,
        unit: 'ms',
        thresholds: { warning: 200, critical: 500 },
      },
      {
        id: 'ps-ov-6',
        title: 'Memory',
        type: 'gauge',
        query: { metric: 'system.memory.usage', aggregation: 'avg', filters: {} },
        col: 9, row: 1, colSpan: 3, rowSpan: 1,
        unit: '%',
        thresholds: { warning: 75, critical: 90 },
      },
    ],
  },
  {
    id: 'preset-error-tracking',
    name: 'Error Tracking',
    description: 'Error rates, top failing services, status code distribution',
    label: 'application',
    role: 'developer',
    widgets: [
      {
        id: 'ps-err-1',
        title: 'Error Rate Trend',
        type: 'timeseries',
        query: { metric: 'http.error.rate', aggregation: 'avg', filters: {}, groupBy: 'service' },
        col: 0, row: 0, colSpan: 8, rowSpan: 1,
        unit: '%',
      },
      {
        id: 'ps-err-2',
        title: 'Total Errors',
        type: 'single-stat',
        query: { metric: 'http.error.rate', aggregation: 'sum', filters: {} },
        col: 8, row: 0, colSpan: 4, rowSpan: 1,
        unit: '%',
        thresholds: { warning: 1, critical: 5 },
      },
      {
        id: 'ps-err-3',
        title: '5xx Count',
        type: 'timeseries',
        query: { metric: 'http.request.count', aggregation: 'count', filters: { status: '5xx' } },
        col: 0, row: 1, colSpan: 6, rowSpan: 1,
        unit: 'req/s',
      },
      {
        id: 'ps-err-4',
        title: '4xx Count',
        type: 'timeseries',
        query: { metric: 'http.request.count', aggregation: 'count', filters: { status: '4xx' } },
        col: 6, row: 1, colSpan: 6, rowSpan: 1,
        unit: 'req/s',
      },
    ],
  },
  {
    id: 'preset-latency-deep-dive',
    name: 'Latency Deep Dive',
    description: 'P50/P95/P99 latency by service with trend analysis',
    label: 'performance',
    role: 'all',
    widgets: [
      {
        id: 'ps-lat-1',
        title: 'P50 Latency',
        type: 'timeseries',
        query: { metric: 'http.request.duration', aggregation: 'p50', filters: {} },
        col: 0, row: 0, colSpan: 4, rowSpan: 1,
        unit: 'ms',
      },
      {
        id: 'ps-lat-2',
        title: 'P95 Latency',
        type: 'timeseries',
        query: { metric: 'http.request.duration', aggregation: 'p95', filters: {} },
        col: 4, row: 0, colSpan: 4, rowSpan: 1,
        unit: 'ms',
      },
      {
        id: 'ps-lat-3',
        title: 'P99 Latency',
        type: 'timeseries',
        query: { metric: 'http.request.duration', aggregation: 'p99', filters: {} },
        col: 8, row: 0, colSpan: 4, rowSpan: 1,
        unit: 'ms',
      },
      {
        id: 'ps-lat-4',
        title: 'Avg Latency by Service',
        type: 'timeseries',
        query: { metric: 'http.request.duration', aggregation: 'avg', filters: {}, groupBy: 'service' },
        col: 0, row: 1, colSpan: 8, rowSpan: 1,
        unit: 'ms',
      },
      {
        id: 'ps-lat-5',
        title: 'Max Latency',
        type: 'single-stat',
        query: { metric: 'http.request.duration', aggregation: 'max', filters: {} },
        col: 8, row: 1, colSpan: 4, rowSpan: 1,
        unit: 'ms',
        thresholds: { warning: 500, critical: 2000 },
      },
    ],
  },
  {
    id: 'preset-traffic-overview',
    name: 'Traffic Overview',
    description: 'Inbound/outbound request volume, top routes, throughput',
    label: 'infrastructure',
    role: 'all',
    widgets: [
      {
        id: 'ps-trf-1',
        title: 'Total Requests',
        type: 'timeseries',
        query: { metric: 'http.request.count', aggregation: 'sum', filters: {} },
        col: 0, row: 0, colSpan: 8, rowSpan: 1,
        unit: 'req/s',
      },
      {
        id: 'ps-trf-2',
        title: 'Request Rate',
        type: 'single-stat',
        query: { metric: 'http.request.count', aggregation: 'sum', filters: {} },
        col: 8, row: 0, colSpan: 4, rowSpan: 1,
        unit: 'req/s',
      },
      {
        id: 'ps-trf-3',
        title: 'Requests by Service',
        type: 'timeseries',
        query: { metric: 'http.request.count', aggregation: 'sum', filters: {}, groupBy: 'service' },
        col: 0, row: 1, colSpan: 6, rowSpan: 1,
        unit: 'req/s',
      },
      {
        id: 'ps-trf-4',
        title: 'Queue Depth',
        type: 'timeseries',
        query: { metric: 'queue.message.count', aggregation: 'avg', filters: {} },
        col: 6, row: 1, colSpan: 6, rowSpan: 1,
      },
    ],
  },
  {
    id: 'preset-security-audit',
    name: 'Security & IAM',
    description: 'Auth failures, IAM policy denials, rate limit hits',
    label: 'security',
    role: 'admin',
    widgets: [
      {
        id: 'ps-sec-1',
        title: 'Auth Failures',
        type: 'timeseries',
        query: { metric: 'http.request.count', aggregation: 'count', filters: { status: '401' } },
        col: 0, row: 0, colSpan: 6, rowSpan: 1,
        unit: 'req/s',
      },
      {
        id: 'ps-sec-2',
        title: 'Forbidden Requests',
        type: 'timeseries',
        query: { metric: 'http.request.count', aggregation: 'count', filters: { status: '403' } },
        col: 6, row: 0, colSpan: 6, rowSpan: 1,
        unit: 'req/s',
      },
      {
        id: 'ps-sec-3',
        title: 'Rate Limited',
        type: 'single-stat',
        query: { metric: 'http.request.count', aggregation: 'count', filters: { status: '429' } },
        col: 0, row: 1, colSpan: 4, rowSpan: 1,
        unit: 'req/s',
        thresholds: { warning: 10, critical: 50 },
      },
      {
        id: 'ps-sec-4',
        title: 'Auth Failure Rate',
        type: 'gauge',
        query: { metric: 'http.error.rate', aggregation: 'avg', filters: { status: '401' } },
        col: 4, row: 1, colSpan: 4, rowSpan: 1,
        unit: '%',
        thresholds: { warning: 5, critical: 15 },
      },
      {
        id: 'ps-sec-5',
        title: 'Error Latency (4xx)',
        type: 'timeseries',
        query: { metric: 'http.request.duration', aggregation: 'avg', filters: { status: '4xx' } },
        col: 8, row: 1, colSpan: 4, rowSpan: 1,
        unit: 'ms',
      },
    ],
  },
];
