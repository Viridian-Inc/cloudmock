import { useState, useEffect, useRef, useCallback } from 'preact/hooks';
import { api, fetchCostRoutes, fetchCostTenants, fetchCostTrend } from '../api';

interface ServiceMetric {
  service: string;
  p50ms: number;
  p95ms: number;
  p99ms: number;
  avgMs: number;
  errorRate: number;
  totalCalls: number;
  errorCalls: number;
}

interface BucketMetric {
  calls: number;
  avgMs: number;
  errors: number;
  p50ms: number;
  p95ms: number;
  p99ms: number;
}

interface TimeBucket {
  timestamp: string;
  services: Record<string, BucketMetric>;
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

type SortField = 'service' | 'p50ms' | 'p95ms' | 'p99ms' | 'avgMs' | 'errorRate' | 'totalCalls';
type MetricsTab = 'latency' | 'cost';

export function MetricsPage() {
  const [activeTab, setActiveTab] = useState<MetricsTab>('latency');
  const [metrics, setMetrics] = useState<ServiceMetric[]>([]);
  const [timeline, setTimeline] = useState<TimeBucket[]>([]);
  const [selectedService, setSelectedService] = useState<string>('_all');
  const [sortField, setSortField] = useState<SortField>('totalCalls');
  const [sortAsc, setSortAsc] = useState(false);
  const [tooltip, setTooltip] = useState<{ x: number; y: number; bucket: TimeBucket } | null>(null);
  const chartRef = useRef<SVGSVGElement>(null);

  // Cost tab state
  const [costRoutes, setCostRoutes] = useState<CostRoute[]>([]);
  const [costTenants, setCostTenants] = useState<CostTenant[]>([]);
  const [costTrend, setCostTrend] = useState<CostTrendBucket[]>([]);

  const fetchData = useCallback(() => {
    api('/api/metrics').then(setMetrics).catch(() => {});
    api('/api/metrics/timeline?minutes=15&bucket=1m').then(setTimeline).catch(() => {});
  }, []);

  const fetchCostData = useCallback(() => {
    fetchCostRoutes(20).then(setCostRoutes).catch(() => {});
    fetchCostTenants(20).then(setCostTenants).catch(() => {});
    fetchCostTrend('24h', '1h').then(setCostTrend).catch(() => {});
  }, []);

  useEffect(() => {
    fetchData();
    const iv = setInterval(fetchData, 5000);
    return () => clearInterval(iv);
  }, [fetchData]);

  useEffect(() => {
    if (activeTab === 'cost') {
      fetchCostData();
    }
  }, [activeTab, fetchCostData]);

  // Compute summary stats.
  const totalRequests = metrics.reduce((s, m) => s + m.totalCalls, 0);
  const avgLatency = totalRequests > 0
    ? (metrics.reduce((s, m) => s + m.avgMs * m.totalCalls, 0) / totalRequests).toFixed(1)
    : '0.0';
  const totalErrors = metrics.reduce((s, m) => s + m.errorCalls, 0);
  const errorRatePct = totalRequests > 0 ? ((totalErrors / totalRequests) * 100).toFixed(2) : '0.00';
  const activeServices = metrics.length;

  const services = metrics.map(m => m.service);

  // Chart data: aggregate P50/P95/P99 per bucket.
  const chartData = timeline.map(bucket => {
    const ts = new Date(bucket.timestamp);
    let p50 = 0, p95 = 0, p99 = 0, calls = 0;
    for (const [svc, bm] of Object.entries(bucket.services)) {
      if (selectedService !== '_all' && svc !== selectedService) continue;
      p50 += bm.p50ms * bm.calls;
      p95 += bm.p95ms * bm.calls;
      p99 += bm.p99ms * bm.calls;
      calls += bm.calls;
    }
    return {
      ts,
      p50: calls > 0 ? p50 / calls : 0,
      p95: calls > 0 ? p95 / calls : 0,
      p99: calls > 0 ? p99 / calls : 0,
      calls,
    };
  });

  const maxY = Math.max(1, ...chartData.map(d => Math.max(d.p50, d.p95, d.p99)));

  // SVG chart dimensions.
  const W = 800, H = 300, padL = 60, padR = 20, padT = 20, padB = 40;
  const plotW = W - padL - padR, plotH = H - padT - padB;

  const toX = (i: number) => padL + (i / Math.max(1, chartData.length - 1)) * plotW;
  const toY = (v: number) => padT + plotH - (v / maxY) * plotH;

  const makeLine = (key: 'p50' | 'p95' | 'p99') =>
    chartData.map((d, i) => `${i === 0 ? 'M' : 'L'}${toX(i).toFixed(1)},${toY(d[key]).toFixed(1)}`).join(' ');

  const handleChartMouseMove = (e: MouseEvent) => {
    if (!chartRef.current || chartData.length === 0) return;
    const rect = chartRef.current.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const idx = Math.round(((x - padL) / plotW) * (chartData.length - 1));
    if (idx >= 0 && idx < chartData.length && idx < timeline.length) {
      setTooltip({ x: e.clientX - rect.left, y: e.clientY - rect.top, bucket: timeline[idx] });
    }
  };

  const handleChartMouseLeave = () => setTooltip(null);

  // Sort handler
  const handleSort = (field: SortField) => {
    if (sortField === field) setSortAsc(!sortAsc);
    else { setSortField(field); setSortAsc(field === 'service'); }
  };

  const sorted = [...metrics].sort((a, b) => {
    const av = a[sortField], bv = b[sortField];
    if (typeof av === 'string' && typeof bv === 'string') return sortAsc ? av.localeCompare(bv) : bv.localeCompare(av);
    return sortAsc ? (av as number) - (bv as number) : (bv as number) - (av as number);
  });

  const errorColor = (rate: number) => {
    if (rate > 0.05) return '#ef4444';
    if (rate > 0.01) return '#f59e0b';
    return '#22c55e';
  };

  const sortArrow = (field: SortField) => sortField === field ? (sortAsc ? ' \u25B2' : ' \u25BC') : '';

  const fmtCost = (v: number) => v >= 0.01 ? `$${v.toFixed(2)}` : `$${v.toFixed(6)}`;

  const maxTrendCost = Math.max(1e-9, ...costTrend.map(b => b.total_cost));

  // Tooltip content
  const tooltipContent = tooltip ? (() => {
    const b = tooltip.bucket;
    const svcEntries = Object.entries(b.services);
    const filteredEntries = selectedService === '_all' ? svcEntries : svcEntries.filter(([s]) => s === selectedService);
    return filteredEntries;
  })() : null;

  return (
    <div style={{ padding: '24px' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '20px' }}>
        <h2 style={{ margin: 0, fontSize: '20px', fontWeight: 600 }}>Metrics</h2>
        <div style={{ display: 'flex', gap: 0, borderBottom: '2px solid var(--border)' }}>
          {(['latency', 'cost'] as MetricsTab[]).map(t => (
            <button
              key={t}
              onClick={() => setActiveTab(t)}
              style={{
                padding: '6px 16px', background: 'none', border: 'none', cursor: 'pointer',
                fontSize: '13px', fontWeight: activeTab === t ? 600 : 400,
                color: activeTab === t ? 'var(--brand-blue, #097FF5)' : 'var(--text-secondary)',
                borderBottom: activeTab === t ? '2px solid var(--brand-blue, #097FF5)' : '2px solid transparent',
                marginBottom: '-2px', textTransform: 'capitalize' as const,
              }}
            >
              {t === 'latency' ? 'Latency' : 'Cost'}
            </button>
          ))}
        </div>
      </div>

      {activeTab === 'cost' && (
        <div>
          {/* Top routes by cost */}
          <div class="card" style={{ padding: '16px', marginBottom: '24px' }}>
            <h3 style={{ margin: '0 0 12px', fontSize: '14px', fontWeight: 600 }}>Top Routes by Cost</h3>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid var(--border)' }}>
                  {['Service', 'Method', 'Path', 'Requests', 'Total Cost ($)', 'Avg Cost ($)'].map(h => (
                    <th key={h} style={{ padding: '8px', textAlign: h === 'Service' || h === 'Method' || h === 'Path' ? 'left' : 'right', fontWeight: 500, opacity: 0.8 }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {costRoutes.map((r, i) => (
                  <tr key={i} style={{ borderBottom: '1px solid var(--border)' }}>
                    <td style={{ padding: '8px', fontWeight: 500 }}>{r.service}</td>
                    <td style={{ padding: '8px', fontFamily: 'monospace', fontSize: '11px', fontWeight: 600, color: '#3B82F6' }}>{r.method}</td>
                    <td style={{ padding: '8px', fontFamily: 'monospace', fontSize: '11px', maxWidth: '200px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' as const }}>{r.path}</td>
                    <td style={{ padding: '8px', textAlign: 'right' }}>{r.requests.toLocaleString()}</td>
                    <td style={{ padding: '8px', textAlign: 'right', fontFamily: 'monospace', fontWeight: 600 }}>{fmtCost(r.total_cost)}</td>
                    <td style={{ padding: '8px', textAlign: 'right', fontFamily: 'monospace' }}>{fmtCost(r.avg_cost)}</td>
                  </tr>
                ))}
                {costRoutes.length === 0 && (
                  <tr><td colSpan={6} style={{ padding: '24px', textAlign: 'center', opacity: 0.5 }}>No cost data yet.</td></tr>
                )}
              </tbody>
            </table>
          </div>

          {/* Top tenants by cost */}
          <div class="card" style={{ padding: '16px', marginBottom: '24px' }}>
            <h3 style={{ margin: '0 0 12px', fontSize: '14px', fontWeight: 600 }}>Top Tenants by Cost</h3>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid var(--border)' }}>
                  {['Tenant ID', 'Requests', 'Total Cost ($)', 'Avg Cost ($)'].map(h => (
                    <th key={h} style={{ padding: '8px', textAlign: h === 'Tenant ID' ? 'left' : 'right', fontWeight: 500, opacity: 0.8 }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {costTenants.map((t, i) => (
                  <tr key={i} style={{ borderBottom: '1px solid var(--border)' }}>
                    <td style={{ padding: '8px', fontFamily: 'monospace', fontWeight: 500 }}>{t.tenant_id}</td>
                    <td style={{ padding: '8px', textAlign: 'right' }}>{t.requests.toLocaleString()}</td>
                    <td style={{ padding: '8px', textAlign: 'right', fontFamily: 'monospace', fontWeight: 600 }}>{fmtCost(t.total_cost)}</td>
                    <td style={{ padding: '8px', textAlign: 'right', fontFamily: 'monospace' }}>{fmtCost(t.avg_cost)}</td>
                  </tr>
                ))}
                {costTenants.length === 0 && (
                  <tr><td colSpan={4} style={{ padding: '24px', textAlign: 'center', opacity: 0.5 }}>No tenant cost data yet.</td></tr>
                )}
              </tbody>
            </table>
          </div>

          {/* Cost trend */}
          <div class="card" style={{ padding: '16px' }}>
            <h3 style={{ margin: '0 0 12px', fontSize: '14px', fontWeight: 600 }}>Cost Trend (24h, 1h buckets)</h3>
            {costTrend.length === 0 ? (
              <div style={{ padding: '24px', textAlign: 'center', opacity: 0.5 }}>No trend data yet.</div>
            ) : (
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '12px' }}>
                <thead>
                  <tr style={{ borderBottom: '1px solid var(--border)' }}>
                    <th style={{ padding: '6px 8px', textAlign: 'left', fontWeight: 500, opacity: 0.8, width: '140px' }}>Time</th>
                    <th style={{ padding: '6px 8px', textAlign: 'right', fontWeight: 500, opacity: 0.8, width: '80px' }}>Requests</th>
                    <th style={{ padding: '6px 8px', textAlign: 'right', fontWeight: 500, opacity: 0.8, width: '100px' }}>Total Cost</th>
                    <th style={{ padding: '6px 8px', fontWeight: 500, opacity: 0.8 }}>Distribution</th>
                  </tr>
                </thead>
                <tbody>
                  {costTrend.map((b, i) => {
                    const barPct = (b.total_cost / maxTrendCost) * 100;
                    const ts = new Date(b.timestamp);
                    return (
                      <tr key={i} style={{ borderBottom: '1px solid var(--border)' }}>
                        <td style={{ padding: '6px 8px', fontFamily: 'monospace', fontSize: '11px' }}>
                          {ts.toLocaleDateString([], { month: 'short', day: 'numeric' })} {ts.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                        </td>
                        <td style={{ padding: '6px 8px', textAlign: 'right' }}>{b.requests.toLocaleString()}</td>
                        <td style={{ padding: '6px 8px', textAlign: 'right', fontFamily: 'monospace', fontWeight: 600 }}>{fmtCost(b.total_cost)}</td>
                        <td style={{ padding: '6px 8px' }}>
                          <div style={{ height: '12px', background: 'var(--bg-secondary)', borderRadius: '2px', overflow: 'hidden' }}>
                            <div style={{
                              height: '100%', width: `${barPct}%`,
                              background: barPct > 75 ? '#ef4444' : barPct > 40 ? '#f59e0b' : '#3B82F6',
                              borderRadius: '2px', transition: 'width 0.3s ease',
                            }} />
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}

      {activeTab === 'latency' && <>
      {/* Summary cards */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px', marginBottom: '24px' }}>
        <div class="card" style={{ padding: '16px' }}>
          <div style={{ fontSize: '12px', opacity: 0.6, marginBottom: '4px' }}>Total Requests (15 min)</div>
          <div style={{ fontSize: '28px', fontWeight: 700 }}>{totalRequests.toLocaleString()}</div>
        </div>
        <div class="card" style={{ padding: '16px' }}>
          <div style={{ fontSize: '12px', opacity: 0.6, marginBottom: '4px' }}>Avg Latency</div>
          <div style={{ fontSize: '28px', fontWeight: 700 }}>{avgLatency} ms</div>
        </div>
        <div class="card" style={{ padding: '16px' }}>
          <div style={{ fontSize: '12px', opacity: 0.6, marginBottom: '4px' }}>Error Rate</div>
          <div style={{ fontSize: '28px', fontWeight: 700, color: parseFloat(errorRatePct) > 5 ? '#ef4444' : parseFloat(errorRatePct) > 1 ? '#f59e0b' : '#22c55e' }}>{errorRatePct}%</div>
        </div>
        <div class="card" style={{ padding: '16px' }}>
          <div style={{ fontSize: '12px', opacity: 0.6, marginBottom: '4px' }}>Active Services</div>
          <div style={{ fontSize: '28px', fontWeight: 700 }}>{activeServices}</div>
        </div>
      </div>

      {/* Chart section */}
      <div class="card" style={{ padding: '16px', marginBottom: '24px' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
          <h3 style={{ margin: 0, fontSize: '14px', fontWeight: 600 }}>Latency Over Time (P50/P95/P99)</h3>
          <select
            value={selectedService}
            onChange={(e: any) => setSelectedService(e.target.value)}
            style={{ padding: '4px 8px', borderRadius: '4px', border: '1px solid var(--border)', background: 'var(--bg-secondary)', color: 'var(--text-primary)', fontSize: '12px' }}
          >
            <option value="_all">All Services</option>
            {services.map(s => <option value={s}>{s}</option>)}
          </select>
        </div>

        <div style={{ position: 'relative' }}>
          <svg
            ref={chartRef}
            viewBox={`0 0 ${W} ${H}`}
            style={{ width: '100%', maxHeight: '300px' }}
            onMouseMove={handleChartMouseMove as any}
            onMouseLeave={handleChartMouseLeave}
          >
            {/* Grid lines */}
            {[0, 0.25, 0.5, 0.75, 1].map(pct => (
              <g key={pct}>
                <line x1={padL} y1={padT + plotH * (1 - pct)} x2={padL + plotW} y2={padT + plotH * (1 - pct)} stroke="var(--border)" stroke-width="0.5" stroke-dasharray="4,4" />
                <text x={padL - 8} y={padT + plotH * (1 - pct) + 4} text-anchor="end" font-size="10" fill="var(--text-secondary)">{(maxY * pct).toFixed(0)}</text>
              </g>
            ))}

            {/* X axis labels */}
            {chartData.filter((_, i) => i % Math.max(1, Math.floor(chartData.length / 5)) === 0).map((d, i) => {
              const idx = chartData.indexOf(d);
              return <text key={i} x={toX(idx)} y={H - 8} text-anchor="middle" font-size="10" fill="var(--text-secondary)">{d.ts.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</text>;
            })}

            {/* Axis labels */}
            <text x={padL - 40} y={padT + plotH / 2} text-anchor="middle" font-size="10" fill="var(--text-secondary)" transform={`rotate(-90, ${padL - 40}, ${padT + plotH / 2})`}>Latency (ms)</text>

            {/* Lines */}
            {chartData.length > 1 && (
              <>
                <path d={makeLine('p50')} fill="none" stroke="#22c55e" stroke-width="2" />
                <path d={makeLine('p95')} fill="none" stroke="#f59e0b" stroke-width="2" />
                <path d={makeLine('p99')} fill="none" stroke="#ef4444" stroke-width="2" />
              </>
            )}

            {/* Legend */}
            <circle cx={padL + 10} cy={12} r={4} fill="#22c55e" />
            <text x={padL + 18} y={15} font-size="10" fill="var(--text-secondary)">P50</text>
            <circle cx={padL + 52} cy={12} r={4} fill="#f59e0b" />
            <text x={padL + 60} y={15} font-size="10" fill="var(--text-secondary)">P95</text>
            <circle cx={padL + 94} cy={12} r={4} fill="#ef4444" />
            <text x={padL + 102} y={15} font-size="10" fill="var(--text-secondary)">P99</text>
          </svg>

          {/* Tooltip */}
          {tooltip && tooltipContent && tooltipContent.length > 0 && (
            <div style={{
              position: 'absolute', left: Math.min(tooltip.x + 10, W - 200), top: tooltip.y - 10,
              background: 'var(--bg-primary)', border: '1px solid var(--border)', borderRadius: '6px',
              padding: '8px 12px', fontSize: '11px', pointerEvents: 'none', zIndex: 10,
              boxShadow: '0 2px 8px rgba(0,0,0,0.15)', minWidth: '150px',
            }}>
              <div style={{ fontWeight: 600, marginBottom: '4px' }}>{new Date(tooltip.bucket.timestamp).toLocaleTimeString()}</div>
              {tooltipContent.map(([svc, bm]) => (
                <div key={svc} style={{ marginBottom: '2px' }}>
                  <span style={{ fontWeight: 500 }}>{svc}</span>: P50={bm.p50ms.toFixed(1)} P95={bm.p95ms.toFixed(1)} P99={bm.p99ms.toFixed(1)} ({bm.calls} calls)
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Service table */}
      <div class="card" style={{ padding: '16px' }}>
        <h3 style={{ margin: '0 0 12px', fontSize: '14px', fontWeight: 600 }}>Service Breakdown</h3>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
          <thead>
            <tr style={{ borderBottom: '1px solid var(--border)' }}>
              {([
                ['service', 'Service'],
                ['p50ms', 'P50'],
                ['p95ms', 'P95'],
                ['p99ms', 'P99'],
                ['avgMs', 'Avg'],
                ['errorRate', 'Error Rate'],
                ['totalCalls', 'Total Calls'],
              ] as [SortField, string][]).map(([field, label]) => (
                <th
                  key={field}
                  onClick={() => handleSort(field)}
                  style={{ padding: '8px', textAlign: field === 'service' ? 'left' : 'right', cursor: 'pointer', userSelect: 'none', fontWeight: 500, opacity: 0.8 }}
                >
                  {label}{sortArrow(field)}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {sorted.map(m => (
              <tr
                key={m.service}
                onClick={() => setSelectedService(m.service === selectedService ? '_all' : m.service)}
                style={{ borderBottom: '1px solid var(--border)', cursor: 'pointer', background: selectedService === m.service ? 'var(--bg-secondary)' : 'transparent' }}
              >
                <td style={{ padding: '8px', fontWeight: 500 }}>{m.service}</td>
                <td style={{ padding: '8px', textAlign: 'right', fontFamily: 'monospace' }}>{m.p50ms.toFixed(1)}</td>
                <td style={{ padding: '8px', textAlign: 'right', fontFamily: 'monospace' }}>{m.p95ms.toFixed(1)}</td>
                <td style={{ padding: '8px', textAlign: 'right', fontFamily: 'monospace' }}>{m.p99ms.toFixed(1)}</td>
                <td style={{ padding: '8px', textAlign: 'right', fontFamily: 'monospace' }}>{m.avgMs.toFixed(1)}</td>
                <td style={{ padding: '8px', textAlign: 'right', color: errorColor(m.errorRate), fontWeight: 600 }}>{(m.errorRate * 100).toFixed(2)}%</td>
                <td style={{ padding: '8px', textAlign: 'right' }}>{m.totalCalls.toLocaleString()}</td>
              </tr>
            ))}
            {sorted.length === 0 && (
              <tr><td colSpan={7} style={{ padding: '24px', textAlign: 'center', opacity: 0.5 }}>No metrics data yet. Make some API requests to see latency data.</td></tr>
            )}
          </tbody>
        </table>
      </div>
      </>}
    </div>
  );
}
