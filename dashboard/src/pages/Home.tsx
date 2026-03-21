import { useState, useEffect, useMemo, useCallback } from 'preact/hooks';
import { api } from '../api';
import { SSEState } from '../hooks/useSSE';
import { statusClass } from '../components/StatusBadge';
import { fmtTime, fmtDuration } from '../utils';

interface HomePageProps {
  sse: SSEState;
  showToast: (msg: string) => void;
}

const SERVICE_COLORS = ['#097FF5', '#7CCEF2', '#029662', '#FEC307', '#FF9A4B', '#94A3B8'];

function useHomeData() {
  const [services, setServices] = useState<any[]>([]);
  const [stats, setStats] = useState<any>({});
  const [health, setHealth] = useState<any>(null);
  const [requests, setRequests] = useState<any[]>([]);
  const [lambdaLogs, setLambdaLogs] = useState<any[]>([]);
  const [emails, setEmails] = useState<any[]>([]);

  const load = useCallback(() => {
    api('/api/services').then(setServices).catch(() => {});
    api('/api/stats').then(setStats).catch(() => {});
    api('/api/health').then(setHealth).catch(() => {});
    api('/api/requests?limit=10').then(setRequests).catch(() => {});
    api('/api/lambda/logs?limit=5').then((r: any) => setLambdaLogs(r || [])).catch(() => {});
    api('/api/ses/emails').then((e: any) => setEmails(Array.isArray(e) ? e.slice(0, 5) : [])).catch(() => {});
  }, []);

  useEffect(() => { load(); }, [load]);

  return { services, stats, health, requests, setRequests, lambdaLogs, setLambdaLogs, emails };
}

export function HomePage({ sse, showToast }: HomePageProps) {
  const { services, stats, health, requests, setRequests, lambdaLogs, setLambdaLogs, emails } = useHomeData();
  const [rateHistory, setRateHistory] = useState<{ minute: string; counts: Record<string, number> }[]>([]);
  const [liveRequestCount, setLiveRequestCount] = useState(0);
  const [resetting, setResetting] = useState(false);
  const [fadeIn, setFadeIn] = useState(false);

  useEffect(() => { requestAnimationFrame(() => setFadeIn(true)); }, []);

  // SSE: track new requests
  useEffect(() => {
    if (sse.events.length === 0) return;
    const latest = sse.events[0];
    if (latest && latest.type === 'request' && latest.data) {
      setLiveRequestCount(c => c + 1);
      setRequests((prev: any[]) => [latest.data, ...prev].slice(0, 10));
      // Update rate history
      const now = new Date();
      const minute = `${now.getHours()}:${String(now.getMinutes()).padStart(2, '0')}`;
      const svc = latest.data.service || 'unknown';
      setRateHistory(prev => {
        const copy = [...prev];
        const last = copy[copy.length - 1];
        if (last && last.minute === minute) {
          last.counts[svc] = (last.counts[svc] || 0) + 1;
        } else {
          copy.push({ minute, counts: { [svc]: 1 } });
          if (copy.length > 15) copy.shift();
        }
        return copy;
      });
    }
    if (latest && latest.type === 'lambda_log' && latest.data) {
      setLambdaLogs((prev: any[]) => [latest.data, ...prev].slice(0, 5));
    }
  }, [sse.events]);

  const totalRequests = useMemo(() => {
    if (!stats || !stats.services) return 0;
    return Object.values(stats.services).reduce((sum: number, s: any) => sum + (s.total || 0), 0) + liveRequestCount;
  }, [stats, liveRequestCount]);

  const requestsPerMin = useMemo(() => {
    if (!stats || !stats.services) return 0;
    return Object.values(stats.services).reduce((sum: number, s: any) => sum + (s.rpm || 0), 0);
  }, [stats]);

  const healthyCount = useMemo(() => {
    if (!health || !health.services) return 0;
    return Object.values(health.services).filter(Boolean).length;
  }, [health]);

  const degradedCount = useMemo(() => services.length - healthyCount, [services, healthyCount]);

  const activeResources = useMemo(() => {
    if (!stats || !stats.services) return 0;
    return Object.values(stats.services).reduce((sum: number, s: any) => sum + (s.resources || 0), 0);
  }, [stats]);

  const uptime = useMemo(() => {
    if (!health || !health.uptime) return '---';
    const secs = health.uptime;
    if (secs < 60) return `${secs}s`;
    if (secs < 3600) return `${Math.floor(secs / 60)}m`;
    if (secs < 86400) return `${Math.floor(secs / 3600)}h ${Math.floor((secs % 3600) / 60)}m`;
    return `${Math.floor(secs / 86400)}d ${Math.floor((secs % 86400) / 3600)}h`;
  }, [health]);

  // Top services by request count
  const topServices = useMemo(() => {
    if (!stats || !stats.services) return [];
    const entries = Object.entries(stats.services).map(([name, s]: [string, any]) => ({
      name,
      count: s.total || 0,
    }));
    entries.sort((a, b) => b.count - a.count);
    return entries.slice(0, 10);
  }, [stats]);

  const topServicesMax = useMemo(() => {
    if (topServices.length === 0) return 1;
    return topServices[0].count || 1;
  }, [topServices]);

  // Service health map
  const serviceHealthMap = useMemo(() => {
    const map: Record<string, boolean> = {};
    if (health && health.services) {
      Object.entries(health.services).forEach(([k, v]) => { map[k] = v as boolean; });
    }
    return map;
  }, [health]);

  const serviceRequestMap = useMemo(() => {
    const map: Record<string, number> = {};
    if (stats && stats.services) {
      Object.entries(stats.services).forEach(([k, v]: [string, any]) => { map[k] = v.total || 0; });
    }
    return map;
  }, [stats]);

  // Chart: top service names for coloring
  const chartServiceNames = useMemo(() => {
    const names = [...topServices.slice(0, 5).map(s => s.name)];
    return names;
  }, [topServices]);

  // Rate chart max
  const rateMax = useMemo(() => {
    if (rateHistory.length === 0) return 1;
    return Math.max(1, ...rateHistory.map(r => Object.values(r.counts).reduce((a, b) => a + b, 0)));
  }, [rateHistory]);

  async function resetAll() {
    setResetting(true);
    try {
      await api('/api/reset', { method: 'POST' });
      showToast('All services reset');
    } catch {
      showToast('Reset failed');
    }
    setResetting(false);
  }

  return (
    <div class={`home-page ${fadeIn ? 'home-fade-in' : ''}`}>
      {/* Top Row: Stat Cards */}
      <div class="home-stats-row">
        <div class="home-stat-card">
          <div class="home-stat-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20">
              <rect x="2" y="3" width="20" height="14" rx="2" />
              <line x1="8" y1="21" x2="16" y2="21" /><line x1="12" y1="17" x2="12" y2="21" />
            </svg>
          </div>
          <div class="home-stat-value">{services.length}</div>
          <div class="home-stat-label">Total Services</div>
          <div class="home-stat-sub">
            <span class="home-stat-green">{healthyCount} healthy</span>
            {degradedCount > 0 && <span class="home-stat-red"> / {degradedCount} degraded</span>}
          </div>
        </div>

        <div class="home-stat-card">
          <div class="home-stat-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20">
              <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
            </svg>
          </div>
          <div class="home-stat-value">{totalRequests.toLocaleString()}</div>
          <div class="home-stat-label">Total Requests</div>
          <div class="home-stat-sub">{requestsPerMin} req/min</div>
        </div>

        <div class="home-stat-card">
          <div class="home-stat-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20">
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
            </svg>
          </div>
          <div class="home-stat-value">{activeResources.toLocaleString()}</div>
          <div class="home-stat-label">Active Resources</div>
          <div class="home-stat-sub">across all services</div>
        </div>

        <div class="home-stat-card">
          <div class="home-stat-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20">
              <circle cx="12" cy="12" r="10" /><polyline points="12 6 12 12 16 14" />
            </svg>
          </div>
          <div class="home-stat-value">{uptime}</div>
          <div class="home-stat-label">Uptime</div>
          <div class="home-stat-sub">since gateway started</div>
        </div>
      </div>

      {/* Second Row: Request Rate + Service Health */}
      <div class="home-row-2">
        <div class="home-panel home-panel-60">
          <div class="home-panel-header">
            <h3>Request Rate</h3>
            <span class="home-panel-sub">Last 15 minutes (live via SSE)</span>
          </div>
          <div class="home-chart-area">
            {rateHistory.length === 0 ? (
              <div class="home-chart-empty">Waiting for requests...</div>
            ) : (
              <svg class="home-rate-chart" viewBox={`0 0 ${rateHistory.length * 48} 120`} preserveAspectRatio="none">
                {rateHistory.map((bucket, i) => {
                  const total = Object.values(bucket.counts).reduce((a, b) => a + b, 0);
                  const barHeight = (total / rateMax) * 100;
                  // Stack by service
                  let yOffset = 120 - barHeight;
                  const segments: preact.JSX.Element[] = [];
                  const entries = Object.entries(bucket.counts).sort((a, b) => b[1] - a[1]);
                  entries.forEach(([svc, count], si) => {
                    const segH = (count / rateMax) * 100;
                    const colorIdx = chartServiceNames.indexOf(svc);
                    const color = colorIdx >= 0 ? SERVICE_COLORS[colorIdx] : SERVICE_COLORS[5];
                    segments.push(
                      <rect x={i * 48 + 4} y={yOffset} width="40" height={segH} rx="3" fill={color} opacity="0.85">
                        <title>{svc}: {count}</title>
                      </rect>
                    );
                    yOffset += segH;
                  });
                  return (
                    <g>
                      {segments}
                      <text x={i * 48 + 24} y="118" text-anchor="middle" font-size="8" fill="#94A3B8">{bucket.minute}</text>
                    </g>
                  );
                })}
              </svg>
            )}
          </div>
          {chartServiceNames.length > 0 && (
            <div class="home-chart-legend">
              {chartServiceNames.map((name, i) => (
                <span class="home-legend-item">
                  <span class="home-legend-dot" style={`background:${SERVICE_COLORS[i]}`} />
                  {name}
                </span>
              ))}
              <span class="home-legend-item">
                <span class="home-legend-dot" style={`background:${SERVICE_COLORS[5]}`} />
                other
              </span>
            </div>
          )}
        </div>

        <div class="home-panel home-panel-40">
          <div class="home-panel-header">
            <h3>Service Health</h3>
            <span class="home-panel-sub">{services.length} services</span>
          </div>
          <div class="home-health-grid">
            {services.filter((s: any) => s.action_count > 5).map((svc: any) => {
              const isHealthy = serviceHealthMap[svc.name] !== false;
              const hasRequests = (serviceRequestMap[svc.name] || 0) > 0;
              const color = !isHealthy ? 'home-tile-red' : hasRequests ? 'home-tile-green' : 'home-tile-blue';
              return (
                <div
                  class={`home-health-tile ${color}`}
                  title={`${svc.name} — ${serviceRequestMap[svc.name] || 0} requests`}
                  onClick={() => (location.hash = `/resources?service=${svc.name}`)}
                >
                  <span class="home-tile-label">{svc.name}</span>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      {/* Third Row: Recent Requests, Top Services, Quick Actions */}
      <div class="home-row-3">
        <div class="home-panel home-panel-40">
          <div class="home-panel-header">
            <h3>Recent Requests</h3>
            <a class="home-view-all" href="#/requests">View all &rarr;</a>
          </div>
          <div class="home-compact-table">
            {requests.length === 0 ? (
              <div class="home-empty-small">No requests yet</div>
            ) : (
              <table>
                <tbody>
                  {requests.slice(0, 10).map((req: any) => (
                    <tr class="clickable" onClick={() => (location.hash = `/requests/${req.id}`)}>
                      <td><span class="home-svc-badge">{req.service}</span></td>
                      <td class="font-mono text-sm">{req.action}</td>
                      <td><span class={`status-pill ${statusClass(req.status)}`}>{req.status}</span></td>
                      <td class="font-mono text-sm">{fmtDuration(req.latency_ms || req.duration_ms)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>

        <div class="home-panel home-panel-30">
          <div class="home-panel-header">
            <h3>Top Services</h3>
          </div>
          <div class="home-top-services">
            {topServices.map((s, i) => (
              <div class="home-top-svc-row clickable" onClick={() => (location.hash = `/resources?service=${s.name}`)}>
                <span class="home-top-svc-rank">{i + 1}</span>
                <span class="home-top-svc-name">{s.name}</span>
                <span class="home-top-svc-count">{s.count}</span>
                <div class="home-top-svc-bar">
                  <div class="home-top-svc-fill" style={`width:${(s.count / topServicesMax) * 100}%`} />
                </div>
              </div>
            ))}
            {topServices.length === 0 && <div class="home-empty-small">No data yet</div>}
          </div>
        </div>

        <div class="home-panel home-panel-30">
          <div class="home-panel-header">
            <h3>Quick Actions</h3>
          </div>
          <div class="home-actions">
            <button class="home-action-btn home-action-danger" onClick={resetAll} disabled={resetting}>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                <polyline points="23 4 23 10 17 10" /><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
              </svg>
              {resetting ? 'Resetting...' : 'Reset All Services'}
            </button>
            <a class="home-action-btn" href="#/dynamodb">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                <ellipse cx="12" cy="5" rx="9" ry="3" /><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3" /><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5" />
              </svg>
              Open DynamoDB
            </a>
            <a class="home-action-btn" href="#/s3">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                <path d="M22 5L2 5" /><path d="M4 5l1 14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2l1-14" /><path d="M9 9v8" /><path d="M15 9v8" /><path d="M2 5l2-2h16l2 2" />
              </svg>
              Open S3
            </a>
            <a class="home-action-btn" href="#/topology">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                <circle cx="18" cy="18" r="3" /><circle cx="6" cy="6" r="3" /><circle cx="18" cy="6" r="3" /><line x1="6" y1="9" x2="6" y2="21" /><path d="M9 6h6" /><path d="M6 21a3 3 0 0 0 3-3V9" />
              </svg>
              View Topology
            </a>
            <a class="home-action-btn" href="#/services">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                <rect x="2" y="3" width="20" height="14" rx="2" /><line x1="8" y1="21" x2="16" y2="21" /><line x1="12" y1="17" x2="12" y2="21" />
              </svg>
              All Services
            </a>
          </div>
        </div>
      </div>

      {/* Bottom Row: Lambda Activity + SES Mailbox */}
      <div class="home-row-bottom">
        <div class="home-panel home-panel-50">
          <div class="home-panel-header">
            <h3>Lambda Activity</h3>
            <a class="home-view-all" href="#/lambda">View all &rarr;</a>
          </div>
          <div class="home-compact-table">
            {lambdaLogs.length === 0 ? (
              <div class="home-empty-small">No Lambda invocations</div>
            ) : (
              <table>
                <thead>
                  <tr><th>Function</th><th>Time</th><th>Message</th></tr>
                </thead>
                <tbody>
                  {lambdaLogs.slice(0, 5).map((l: any) => (
                    <tr>
                      <td class="font-mono text-sm truncate" style="max-width:150px">{l.function_name || ''}</td>
                      <td class="font-mono text-sm">{fmtTime(l.timestamp || l.time)}</td>
                      <td class="text-sm truncate" style="max-width:250px">{l.message || ''}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>

        <div class="home-panel home-panel-50">
          <div class="home-panel-header">
            <h3>SES Mailbox</h3>
            <a class="home-view-all" href="#/mail">View all &rarr;</a>
          </div>
          <div class="home-compact-table">
            {emails.length === 0 ? (
              <div class="home-empty-small">No emails captured</div>
            ) : (
              <table>
                <thead>
                  <tr><th>From</th><th>To</th><th>Subject</th></tr>
                </thead>
                <tbody>
                  {emails.slice(0, 5).map((e: any) => (
                    <tr>
                      <td class="text-sm truncate" style="max-width:130px">{e.source || ''}</td>
                      <td class="text-sm truncate" style="max-width:130px">{(e.to_addresses || []).join(', ')}</td>
                      <td class="text-sm truncate" style="max-width:200px">{e.subject || '(no subject)'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
