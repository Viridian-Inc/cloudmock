import { useState, useEffect, useMemo, useCallback } from 'preact/hooks';
import { api } from '../../lib/api';
import { LineChart } from './line-chart';
import { TimeRangeSelector } from '../../components/time-range-selector/time-range-selector';
import './metrics.css';

type MetricsTab = 'latency' | 'cost';

interface TraceEntry {
  TraceID: string;
  RootService: string;
  Method: string;
  Path: string;
  DurationMs: number;
  StatusCode: number;
  SpanCount: number;
  HasError: boolean;
  StartTime: string;
}

interface SLOWindow {
  Service: string;
  Action: string;
  Total: number;
  Errors: number;
  P50Ms: number;
  P95Ms: number;
  P99Ms: number;
  ErrorRate: number;
  Healthy: boolean;
  Violations: string[];
}

interface SLOResponse {
  healthy: boolean;
  rules: any[];
  windows: SLOWindow[];
  alerts: any[];
}

interface ServiceStats {
  service: string;
  total_requests: number;
  avg_latency_ms: number;
  error_rate: number;
  p50_ms: number;
  p95_ms: number;
  p99_ms: number;
}

interface Stats {
  total_requests: number;
  avg_latency_ms: number;
  error_rate: number;
  services: ServiceStats[];
}

interface CostRoute {
  service: string;
  method: string;
  path: string;
  requests: number;
  total_cost: number;
  avg_cost: number;
}

interface CostTenant {
  tenant_id: string;
  requests: number;
  total_cost: number;
  avg_cost: number;
}

interface CostTrendBucket {
  timestamp: string;
  requests: number;
  total_cost: number;
}

function fmtCost(v: number): string {
  return v >= 0.01 ? `$${v.toFixed(2)}` : `$${v.toFixed(6)}`;
}

function CostTab() {
  const [costRoutes, setCostRoutes] = useState<CostRoute[]>([]);
  const [costTenants, setCostTenants] = useState<CostTenant[]>([]);
  const [costTrend, setCostTrend] = useState<CostTrendBucket[]>([]);

  const fetchCostData = useCallback(() => {
    api<CostRoute[]>('/api/cost/routes?limit=20').then(setCostRoutes).catch(() => {});
    api<CostTenant[]>('/api/cost/tenants?limit=20').then(setCostTenants).catch(() => {});
    api<CostTrendBucket[]>('/api/cost/trend?period=24h&resolution=1h').then(setCostTrend).catch(() => {});
  }, []);

  useEffect(() => {
    fetchCostData();
  }, [fetchCostData]);

  const maxTrendCost = Math.max(1e-9, ...costTrend.map((b) => b.total_cost));

  // Build chart data from cost trend buckets
  const trendChartData = useMemo(() => {
    return costTrend.map((b) => ({
      time: new Date(b.timestamp).getTime(),
      value: b.total_cost,
    }));
  }, [costTrend]);

  return (
    <div>
      {/* Cost trend chart */}
      {trendChartData.length > 0 && (
        <div class="metrics-section">
          <h3 class="metrics-section-title">Cost Trend (24h)</h3>
          <LineChart
            data={trendChartData}
            color="#fbbf24"
            label="Total Cost"
            unit="$"
            height={180}
          />
        </div>
      )}

      {/* Top routes by cost */}
      <div class="metrics-section">
        <h3 class="metrics-section-title">Top Routes by Cost</h3>
        <div class="metrics-table-wrap">
          <table class="metrics-table">
            <thead>
              <tr>
                <th>Service</th>
                <th>Method</th>
                <th>Path</th>
                <th style={{ textAlign: 'right' }}>Requests</th>
                <th style={{ textAlign: 'right' }}>Total Cost</th>
                <th style={{ textAlign: 'right' }}>Avg Cost</th>
              </tr>
            </thead>
            <tbody>
              {costRoutes.length === 0 ? (
                <tr>
                  <td colSpan={6} style={{ textAlign: 'center', padding: '24px', color: 'var(--text-tertiary)' }}>
                    No cost data yet
                  </td>
                </tr>
              ) : (
                costRoutes.map((r, i) => (
                  <tr key={i}>
                    <td class="metrics-service-name">{r.service}</td>
                    <td class="metrics-mono" style={{ color: 'var(--brand-blue, #60a5fa)', fontWeight: 600 }}>
                      {r.method}
                    </td>
                    <td class="metrics-mono" style={{ maxWidth: '200px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {r.path}
                    </td>
                    <td class="metrics-mono" style={{ textAlign: 'right' }}>
                      {r.requests.toLocaleString()}
                    </td>
                    <td class="metrics-mono" style={{ textAlign: 'right', fontWeight: 600 }}>
                      {fmtCost(r.total_cost)}
                    </td>
                    <td class="metrics-mono" style={{ textAlign: 'right' }}>
                      {fmtCost(r.avg_cost)}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Top tenants by cost */}
      <div class="metrics-section">
        <h3 class="metrics-section-title">Top Tenants by Cost</h3>
        <div class="metrics-table-wrap">
          <table class="metrics-table">
            <thead>
              <tr>
                <th>Tenant ID</th>
                <th style={{ textAlign: 'right' }}>Requests</th>
                <th style={{ textAlign: 'right' }}>Total Cost</th>
                <th style={{ textAlign: 'right' }}>Avg Cost</th>
              </tr>
            </thead>
            <tbody>
              {costTenants.length === 0 ? (
                <tr>
                  <td colSpan={4} style={{ textAlign: 'center', padding: '24px', color: 'var(--text-tertiary)' }}>
                    No tenant cost data yet
                  </td>
                </tr>
              ) : (
                costTenants.map((t, i) => (
                  <tr key={i}>
                    <td class="metrics-mono" style={{ fontWeight: 500 }}>
                      {t.tenant_id}
                    </td>
                    <td class="metrics-mono" style={{ textAlign: 'right' }}>
                      {t.requests.toLocaleString()}
                    </td>
                    <td class="metrics-mono" style={{ textAlign: 'right', fontWeight: 600 }}>
                      {fmtCost(t.total_cost)}
                    </td>
                    <td class="metrics-mono" style={{ textAlign: 'right' }}>
                      {fmtCost(t.avg_cost)}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Detailed trend table */}
      {costTrend.length > 0 && (
        <div class="metrics-section">
          <h3 class="metrics-section-title">Hourly Breakdown</h3>
          <div class="metrics-table-wrap">
            <table class="metrics-table">
              <thead>
                <tr>
                  <th style={{ width: '140px' }}>Time</th>
                  <th style={{ textAlign: 'right', width: '80px' }}>Requests</th>
                  <th style={{ textAlign: 'right', width: '100px' }}>Total Cost</th>
                  <th>Distribution</th>
                </tr>
              </thead>
              <tbody>
                {costTrend.map((b, i) => {
                  const barPct = (b.total_cost / maxTrendCost) * 100;
                  const ts = new Date(b.timestamp);
                  const barColor =
                    barPct > 75 ? 'var(--error)' : barPct > 40 ? 'var(--warning)' : 'var(--brand-blue, #60a5fa)';
                  return (
                    <tr key={i}>
                      <td class="metrics-mono">
                        {ts.toLocaleDateString([], { month: 'short', day: 'numeric' })}{' '}
                        {ts.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                      </td>
                      <td class="metrics-mono" style={{ textAlign: 'right' }}>
                        {b.requests.toLocaleString()}
                      </td>
                      <td class="metrics-mono" style={{ textAlign: 'right', fontWeight: 600 }}>
                        {fmtCost(b.total_cost)}
                      </td>
                      <td>
                        <div
                          style={{
                            height: '12px',
                            background: 'var(--bg-tertiary)',
                            borderRadius: '2px',
                            overflow: 'hidden',
                          }}
                        >
                          <div
                            style={{
                              height: '100%',
                              width: `${barPct}%`,
                              background: barColor,
                              borderRadius: '2px',
                              transition: 'width 0.3s ease',
                            }}
                          />
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}

function percentile(sorted: number[], p: number): number {
  if (sorted.length === 0) return 0;
  const idx = Math.ceil((p / 100) * sorted.length) - 1;
  return sorted[Math.max(0, idx)];
}

function computeStatsFromTraces(traces: TraceEntry[], sloWindows: SLOWindow[]): Stats {
  // Build SLO window lookup for per-service percentiles
  const sloByService = new Map<string, SLOWindow>();
  for (const w of sloWindows) {
    sloByService.set(w.Service, w);
  }

  // Group traces by RootService
  const byService = new Map<string, TraceEntry[]>();
  for (const t of traces) {
    const svc = t.RootService || 'unknown';
    if (!byService.has(svc)) byService.set(svc, []);
    byService.get(svc)!.push(t);
  }

  const services: ServiceStats[] = [];
  let totalLatency = 0;
  let totalErrors = 0;

  for (const [svc, svcTraces] of byService) {
    const count = svcTraces.length;
    const errors = svcTraces.filter((t) => t.HasError || t.StatusCode >= 500).length;
    const durations = svcTraces.map((t) => t.DurationMs).sort((a, b) => a - b);
    const avgLatency = durations.reduce((s, d) => s + d, 0) / count;

    totalLatency += durations.reduce((s, d) => s + d, 0);
    totalErrors += errors;

    // Use SLO window percentiles if available, else compute from trace durations
    const sloWindow = sloByService.get(svc);
    const p50 = sloWindow ? sloWindow.P50Ms : percentile(durations, 50);
    const p95 = sloWindow ? sloWindow.P95Ms : percentile(durations, 95);
    const p99 = sloWindow ? sloWindow.P99Ms : percentile(durations, 99);

    services.push({
      service: svc,
      total_requests: count,
      avg_latency_ms: avgLatency,
      error_rate: count > 0 ? errors / count : 0,
      p50_ms: p50,
      p95_ms: p95,
      p99_ms: p99,
    });
  }

  // Sort by request count descending
  services.sort((a, b) => b.total_requests - a.total_requests);

  const totalRequests = traces.length;

  return {
    total_requests: totalRequests,
    avg_latency_ms: totalRequests > 0 ? totalLatency / totalRequests : 0,
    error_rate: totalRequests > 0 ? totalErrors / totalRequests : 0,
    services,
  };
}

interface TimeBucket {
  time: number; // minute timestamp
  count: number;
  totalLatency: number;
  errorCount: number;
  latencies: number[];
}

/** Bucket traces by minute for time-series charts */
function bucketByMinute(traces: TraceEntry[]): TimeBucket[] {
  const buckets = new Map<number, TimeBucket>();

  for (const t of traces) {
    const ts = new Date(t.StartTime).getTime();
    if (isNaN(ts)) continue;
    // Truncate to minute
    const minuteTs = Math.floor(ts / 60000) * 60000;

    let bucket = buckets.get(minuteTs);
    if (!bucket) {
      bucket = { time: minuteTs, count: 0, totalLatency: 0, errorCount: 0, latencies: [] };
      buckets.set(minuteTs, bucket);
    }

    bucket.count++;
    bucket.totalLatency += t.DurationMs;
    bucket.latencies.push(t.DurationMs);
    if (t.HasError || t.StatusCode >= 500) {
      bucket.errorCount++;
    }
  }

  // Sort by time
  return Array.from(buckets.values()).sort((a, b) => a.time - b.time);
}

/** Compute P99 from a sorted array of latencies */
function p99(sorted: number[]): number {
  if (sorted.length === 0) return 0;
  const idx = Math.ceil(0.99 * sorted.length) - 1;
  return sorted[Math.max(0, idx)];
}

function formatLatency(ms: number): string {
  if (ms < 1) return `${(ms * 1000).toFixed(0)}us`;
  if (ms < 1000) return `${ms.toFixed(1)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

function formatRate(rate: number): string {
  return `${(rate * 100).toFixed(2)}%`;
}

function StatCard({
  label,
  value,
  sub,
  variant,
}: {
  label: string;
  value: string;
  sub?: string;
  variant?: 'default' | 'success' | 'warning' | 'error';
}) {
  return (
    <div class={`metric-card metric-card-${variant || 'default'}`}>
      <div class="metric-card-label">{label}</div>
      <div class="metric-card-value">{value}</div>
      {sub && <div class="metric-card-sub">{sub}</div>}
    </div>
  );
}

const COMPARE_COLORS = [
  'var(--brand-teal, #4AE5F8)',
  '#a78bfa',
  '#f472b6',
  '#fbbf24',
  '#60a5fa',
  '#4ade80',
  '#fb923c',
  '#f87171',
];

/** Bucket traces for a single service by minute */
function bucketByMinuteForService(
  traces: TraceEntry[],
  service: string,
): TimeBucket[] {
  const filtered = traces.filter((t) => t.RootService === service);
  return bucketByMinute(filtered);
}

function ServiceCompareView({
  selectedServices,
  traces,
  stats,
  onClose,
}: {
  selectedServices: string[];
  traces: TraceEntry[];
  stats: Stats;
  onClose: () => void;
}) {
  // Build per-service time-series data
  const perServiceCharts = useMemo(() => {
    const result: Record<string, {
      volume: { time: number; value: number }[];
      latency: { time: number; value: number }[];
      errorRate: { time: number; value: number }[];
    }> = {};

    for (const svc of selectedServices) {
      const buckets = bucketByMinuteForService(traces, svc);
      result[svc] = {
        volume: buckets.map((b) => ({ time: b.time, value: b.count })),
        latency: buckets.map((b) => {
          const sorted = b.latencies.slice().sort((a, c) => a - c);
          return { time: b.time, value: p99(sorted) };
        }),
        errorRate: buckets.map((b) => ({
          time: b.time,
          value: b.count > 0 ? (b.errorCount / b.count) * 100 : 0,
        })),
      };
    }

    return result;
  }, [selectedServices, traces]);

  // Get ServiceStats for selected services
  const selectedStats = stats.services.filter((s) => selectedServices.includes(s.service));

  return (
    <div class="metrics-compare">
      <div class="metrics-compare-header">
        <h3 class="metrics-section-title">
          Comparing {selectedServices.length} Services
        </h3>
        <button class="metrics-compare-close" onClick={onClose}>
          Exit Comparison
        </button>
      </div>

      {/* Side-by-side stat cards */}
      <div class="metrics-compare-cards">
        {selectedStats.map((svc, i) => (
          <div key={svc.service} class="metrics-compare-card">
            <div
              class="metrics-compare-card-bar"
              style={{ background: COMPARE_COLORS[i % COMPARE_COLORS.length] }}
            />
            <div class="metrics-compare-card-name">{svc.service}</div>
            <div class="metrics-compare-card-grid">
              <div class="metrics-compare-stat">
                <span class="metrics-compare-stat-label">Requests</span>
                <span class="metrics-compare-stat-value">{svc.total_requests}</span>
              </div>
              <div class="metrics-compare-stat">
                <span class="metrics-compare-stat-label">Avg Latency</span>
                <span class="metrics-compare-stat-value">{formatLatency(svc.avg_latency_ms)}</span>
              </div>
              <div class="metrics-compare-stat">
                <span class="metrics-compare-stat-label">P99</span>
                <span class="metrics-compare-stat-value">{formatLatency(svc.p99_ms)}</span>
              </div>
              <div class="metrics-compare-stat">
                <span class="metrics-compare-stat-label">Error Rate</span>
                <span class={`metrics-compare-stat-value ${svc.error_rate > 0.05 ? 'metrics-error' : svc.error_rate > 0.01 ? 'metrics-warning' : ''}`}>
                  {formatRate(svc.error_rate)}
                </span>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Overlaid line charts */}
      <div class="metrics-compare-charts">
        <div class="metrics-compare-chart-section">
          <h4 class="metrics-compare-chart-title">Request Volume</h4>
          <div class="metrics-compare-chart-overlay">
            {selectedServices.map((svc, i) => {
              const data = perServiceCharts[svc]?.volume || [];
              if (data.length === 0) return null;
              return (
                <div key={svc} class="metrics-compare-chart-layer">
                  <LineChart
                    data={data}
                    color={COMPARE_COLORS[i % COMPARE_COLORS.length]}
                    label={svc}
                    unit="req/s"
                    height={160}
                  />
                </div>
              );
            })}
          </div>
          <div class="metrics-compare-legend">
            {selectedServices.map((svc, i) => (
              <span key={svc} class="metrics-compare-legend-item">
                <span
                  class="metrics-compare-legend-dot"
                  style={{ background: COMPARE_COLORS[i % COMPARE_COLORS.length] }}
                />
                {svc}
              </span>
            ))}
          </div>
        </div>

        <div class="metrics-compare-chart-section">
          <h4 class="metrics-compare-chart-title">P99 Latency</h4>
          <div class="metrics-compare-chart-overlay">
            {selectedServices.map((svc, i) => {
              const data = perServiceCharts[svc]?.latency || [];
              if (data.length === 0) return null;
              return (
                <div key={svc} class="metrics-compare-chart-layer">
                  <LineChart
                    data={data}
                    color={COMPARE_COLORS[i % COMPARE_COLORS.length]}
                    label={svc}
                    unit="ms"
                    height={160}
                  />
                </div>
              );
            })}
          </div>
        </div>

        <div class="metrics-compare-chart-section">
          <h4 class="metrics-compare-chart-title">Error Rate</h4>
          <div class="metrics-compare-chart-overlay">
            {selectedServices.map((svc, i) => {
              const data = perServiceCharts[svc]?.errorRate || [];
              if (data.length === 0) return null;
              return (
                <div key={svc} class="metrics-compare-chart-layer">
                  <LineChart
                    data={data}
                    color={COMPARE_COLORS[i % COMPARE_COLORS.length]}
                    label={svc}
                    unit="%"
                    height={160}
                  />
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}

export function MetricsView() {
  const [activeTab, setActiveTab] = useState<MetricsTab>('latency');
  const [stats, setStats] = useState<Stats | null>(null);
  const [traces, setTraces] = useState<TraceEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedServices, setSelectedServices] = useState<Set<string>>(new Set());
  const [showCompare, setShowCompare] = useState(false);
  const [metricsTimeRange, setMetricsTimeRange] = useState<{ start: number; end: number } | null>(null);

  useEffect(() => {
    async function load() {
      try {
        // Fetch traces and SLO data in parallel
        const [rawTraces, sloData] = await Promise.all([
          api<TraceEntry[]>('/api/traces').catch(() => []),
          api<SLOResponse>('/api/slo').catch(() => ({ healthy: true, rules: [], windows: [], alerts: [] })),
        ]);

        const traceList = Array.isArray(rawTraces) ? rawTraces : [];
        const sloWindows = Array.isArray(sloData?.windows) ? sloData.windows : [];

        setTraces(traceList);

        if (traceList.length === 0 && sloWindows.length === 0) {
          setStats(null);
        } else {
          setStats(computeStatsFromTraces(traceList, sloWindows));
        }
      } catch {
        setStats(null);
      } finally {
        setLoading(false);
      }
    }

    load();
  }, []);

  // Compute full data range from traces
  const dataRange = useMemo(() => {
    if (traces.length === 0) return null;
    const times = traces
      .map((t) => new Date(t.StartTime).getTime())
      .filter((ts) => !isNaN(ts));
    if (times.length === 0) return null;
    return { start: Math.min(...times), end: Math.max(...times) };
  }, [traces]);

  // Initialize metricsTimeRange to full data range when data loads
  useEffect(() => {
    if (dataRange && !metricsTimeRange) {
      setMetricsTimeRange(dataRange);
    }
  }, [dataRange, metricsTimeRange]);

  // Filter traces by selected time range
  const filteredTraces = useMemo(() => {
    if (!metricsTimeRange) return traces;
    return traces.filter((t) => {
      const ts = new Date(t.StartTime).getTime();
      return ts >= metricsTimeRange.start && ts <= metricsTimeRange.end;
    });
  }, [traces, metricsTimeRange]);

  // Compute time-series chart data from trace buckets
  const chartData = useMemo(() => {
    if (filteredTraces.length === 0) return null;

    const buckets = bucketByMinute(filteredTraces);
    if (buckets.length === 0) return null;

    const volume = buckets.map((b) => ({ time: b.time, value: b.count }));
    const latency = buckets.map((b) => {
      const sorted = b.latencies.slice().sort((a, c) => a - c);
      return { time: b.time, value: p99(sorted) };
    });
    const errorRate = buckets.map((b) => ({
      time: b.time,
      value: b.count > 0 ? (b.errorCount / b.count) * 100 : 0,
    }));

    return { volume, latency, errorRate };
  }, [filteredTraces]);

  // Overview data for the time range selector mini chart
  const overviewData = useMemo(() => {
    if (traces.length === 0) return undefined;
    const buckets = bucketByMinute(traces);
    return buckets.map((b) => ({ timestamp: b.time, value: b.count }));
  }, [traces]);

  const toggleServiceSelection = (service: string) => {
    setSelectedServices((prev) => {
      const next = new Set(prev);
      if (next.has(service)) {
        next.delete(service);
      } else {
        next.add(service);
      }
      return next;
    });
  };

  const handleCompare = () => {
    setShowCompare(true);
  };

  const handleCloseCompare = () => {
    setShowCompare(false);
    setSelectedServices(new Set());
  };

  if (loading) {
    return (
      <div class="metrics-view metrics-view-empty">
        <div class="metrics-placeholder">Loading metrics...</div>
      </div>
    );
  }

  const isEmpty = !stats || stats.total_requests === 0;

  if (isEmpty) {
    return (
      <div class="metrics-view metrics-view-empty">
        <div class="metrics-placeholder">Generate traffic to see metrics</div>
      </div>
    );
  }

  // Show comparison view
  if (showCompare && selectedServices.size >= 2) {
    return (
      <div class="metrics-view">
        <div class="metrics-header">
          <h2 class="metrics-title">Metrics</h2>
          <span class="metrics-subtitle">{stats.total_requests} traces</span>
        </div>
        <ServiceCompareView
          selectedServices={[...selectedServices]}
          traces={traces}
          stats={stats}
          onClose={handleCloseCompare}
        />
      </div>
    );
  }

  const errorVariant =
    stats.error_rate > 0.05
      ? 'error'
      : stats.error_rate > 0.01
        ? 'warning'
        : 'success';

  return (
    <div class="metrics-view">
      <div class="metrics-header-bar">
        <div class="metrics-header">
          <h2 class="metrics-title">Metrics</h2>
          <span class="metrics-subtitle">{stats.total_requests} traces</span>
        </div>
        <div class="metrics-tabs">
          {(['latency', 'cost'] as MetricsTab[]).map((t) => (
            <button
              key={t}
              class={`metrics-tab ${activeTab === t ? 'metrics-tab-active' : ''}`}
              onClick={() => setActiveTab(t)}
            >
              {t === 'latency' ? 'Latency' : 'Cost'}
            </button>
          ))}
        </div>
      </div>

      {activeTab === 'cost' && <CostTab />}

      {activeTab === 'latency' && <>
      <div class="metrics-cards">
        <StatCard
          label="Total Requests"
          value={String(stats.total_requests)}
        />
        <StatCard
          label="Avg Latency"
          value={formatLatency(stats.avg_latency_ms)}
        />
        <StatCard
          label="Error Rate"
          value={formatRate(stats.error_rate)}
          variant={errorVariant}
        />
      </div>

      {stats.services.length > 0 && (
        <div class="metrics-section">
          <div class="metrics-section-header">
            <h3 class="metrics-section-title">Per-Service Breakdown</h3>
            {selectedServices.size >= 2 && (
              <button class="metrics-compare-btn" onClick={handleCompare}>
                Compare ({selectedServices.size})
              </button>
            )}
          </div>
          <div class="metrics-table-wrap">
            <table class="metrics-table">
              <thead>
                <tr>
                  <th class="metrics-th-checkbox" />
                  <th>Service</th>
                  <th>Requests</th>
                  <th>Avg Latency</th>
                  <th>P50</th>
                  <th>P95</th>
                  <th>P99</th>
                  <th>Error Rate</th>
                </tr>
              </thead>
              <tbody>
                {stats.services.map((svc) => (
                  <tr key={svc.service} class={selectedServices.has(svc.service) ? 'metrics-row-selected' : ''}>
                    <td class="metrics-td-checkbox">
                      <input
                        type="checkbox"
                        class="metrics-checkbox"
                        checked={selectedServices.has(svc.service)}
                        onChange={() => toggleServiceSelection(svc.service)}
                      />
                    </td>
                    <td class="metrics-service-name">{svc.service}</td>
                    <td class="metrics-mono">{svc.total_requests}</td>
                    <td class="metrics-mono">{formatLatency(svc.avg_latency_ms)}</td>
                    <td class="metrics-mono">{formatLatency(svc.p50_ms)}</td>
                    <td class="metrics-mono">{formatLatency(svc.p95_ms)}</td>
                    <td class="metrics-mono">{formatLatency(svc.p99_ms)}</td>
                    <td class={`metrics-mono ${svc.error_rate > 0.05 ? 'metrics-error' : svc.error_rate > 0.01 ? 'metrics-warning' : ''}`}>
                      {formatRate(svc.error_rate)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {chartData && (
        <div class="metrics-section">
          <h3 class="metrics-section-title">Time Series</h3>
          <div class="metrics-charts">
            <LineChart
              data={chartData.volume}
              color="var(--brand-teal)"
              label="Request Volume"
              unit="req/s"
              height={160}
            />
            <LineChart
              data={chartData.latency}
              color="var(--brand-blue)"
              label="P99 Latency"
              unit="ms"
              height={160}
            />
            <LineChart
              data={chartData.errorRate}
              color="var(--error)"
              label="Error Rate"
              unit="%"
              height={160}
            />
          </div>
        </div>
      )}

      {dataRange && metricsTimeRange && (
        <TimeRangeSelector
          dataRange={{
            start: dataRange.start - (dataRange.end - dataRange.start) * 0.5,
            end: Math.max(dataRange.end, Date.now()),
          }}
          selectedRange={metricsTimeRange}
          onRangeChange={setMetricsTimeRange}
          overviewData={overviewData}
          height={48}
        />
      )}
      </>}
    </div>
  );
}
