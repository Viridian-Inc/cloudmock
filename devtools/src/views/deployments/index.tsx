import { useState, useEffect, useMemo } from 'preact/hooks';
import { api } from '../../lib/api';
import type { ServiceMetrics } from '../../lib/health';
import './deployments.css';

interface DeployEvent {
  id: string;
  timestamp: string;
  service: string;
  commit: string;
  author: string;
  message: string;
  branch?: string;
  // PascalCase variants from cloudmock
  ID?: string;
  Service?: string;
  CommitSHA?: string;
  Author?: string;
  Description?: string;
  DeployedAt?: string;
}

interface Regression {
  id: string;
  service: string;
  algorithm: string;
  severity: string;
  status: string;
  detected_at?: string;
  change_percent?: number;
}

function normalizeDeploy(raw: DeployEvent): DeployEvent {
  return {
    id: raw.ID || raw.id || '',
    service: raw.Service || raw.service || '',
    commit: raw.CommitSHA || raw.commit || '',
    author: raw.Author || raw.author || '',
    message: raw.Description || raw.message || '',
    timestamp: raw.DeployedAt || raw.timestamp || '',
    branch: raw.branch || '',
  };
}

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  if (diff < 0) return 'just now';
  const secs = Math.floor(diff / 1000);
  if (secs < 60) return `${secs}s ago`;
  const mins = Math.floor(secs / 60);
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  return `${days}d ago`;
}

function formatTimestamp(iso: string): string {
  try {
    const d = new Date(iso);
    return d.toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  } catch {
    return iso;
  }
}

/**
 * Check if a regression was detected close in time to a deploy (within 30 minutes after).
 */
function isCorrelatedRegression(deploy: DeployEvent, regression: Regression): boolean {
  if (!deploy.timestamp || !regression.detected_at) return false;
  const deployTs = new Date(deploy.timestamp).getTime();
  const regressionTs = new Date(regression.detected_at).getTime();
  // Regression detected within 30 minutes after deploy
  return regressionTs >= deployTs && regressionTs - deployTs <= 30 * 60 * 1000;
}

export function DeploymentsView() {
  const [deploys, setDeploys] = useState<DeployEvent[]>([]);
  const [regressions, setRegressions] = useState<Regression[]>([]);
  const [metrics, setMetrics] = useState<ServiceMetrics[]>([]);
  const [serviceFilter, setServiceFilter] = useState('');
  const [loading, setLoading] = useState(true);

  function loadData() {
    setLoading(true);
    Promise.all([
      api<DeployEvent[]>('/api/deploys').catch(() => []),
      api<Regression[]>('/api/regressions?status=active').catch(() => []),
      api<ServiceMetrics[]>('/api/metrics').catch(() => []),
    ])
      .then(([rawDeploys, regs, m]) => {
        const normalized = (rawDeploys || []).map(normalizeDeploy);
        setDeploys(normalized);
        setRegressions(regs || []);
        setMetrics(m || []);
      })
      .finally(() => setLoading(false));
  }

  useEffect(() => {
    loadData();
  }, []);

  // Unique service names for filter
  const serviceNames = useMemo(() => {
    const names = new Set(deploys.map((d) => d.service).filter(Boolean));
    return [...names].sort();
  }, [deploys]);

  // Filter and sort
  const filtered = useMemo(() => {
    let data = [...deploys];
    if (serviceFilter) {
      data = data.filter((d) => d.service === serviceFilter);
    }
    return data.sort((a, b) => {
      const ta = new Date(a.timestamp || 0).getTime();
      const tb = new Date(b.timestamp || 0).getTime();
      return tb - ta;
    });
  }, [deploys, serviceFilter]);

  // Build correlation map: deploy id -> correlated regressions
  const correlationMap = useMemo(() => {
    const map = new Map<string, Regression[]>();
    for (const deploy of deploys) {
      const correlated = regressions.filter((r) => {
        // Match by service name (flexible)
        const ds = deploy.service.toLowerCase();
        const rs = r.service.toLowerCase();
        const serviceMatch = ds === rs || ds.includes(rs) || rs.includes(ds);
        return serviceMatch && isCorrelatedRegression(deploy, r);
      });
      if (correlated.length > 0) {
        map.set(deploy.id, correlated);
      }
    }
    return map;
  }, [deploys, regressions]);

  // Summary stats
  const totalDeploys = filtered.length;
  const servicesDeployed = new Set(filtered.map((d) => d.service)).size;
  const correlatedCount = [...correlationMap.values()].reduce((sum, regs) => sum + regs.length, 0);

  return (
    <div class="deployments-view">
      <div class="deployments-header">
        <div>
          <h2 class="deployments-title">Deployments</h2>
          <p class="deployments-desc">Deploy timeline with regression correlation</p>
        </div>
        <button class="deployments-refresh-btn" onClick={loadData}>
          Refresh
        </button>
      </div>

      {/* Summary cards */}
      <div class="deployments-cards">
        <div class="deployments-stat">
          <div class="deployments-stat-value">{totalDeploys}</div>
          <div class="deployments-stat-label">Deploys</div>
        </div>
        <div class="deployments-stat">
          <div class="deployments-stat-value">{servicesDeployed}</div>
          <div class="deployments-stat-label">Services</div>
        </div>
        <div class="deployments-stat">
          <div
            class="deployments-stat-value"
            style={{ color: correlatedCount > 0 ? 'var(--error)' : undefined }}
          >
            {correlatedCount}
          </div>
          <div class="deployments-stat-label">Regressions</div>
        </div>
      </div>

      {/* Filter */}
      <div class="deployments-filters">
        <select
          class="deployments-select"
          value={serviceFilter}
          onChange={(e) => setServiceFilter((e.target as HTMLSelectElement).value)}
        >
          <option value="">All Services</option>
          {serviceNames.map((s) => (
            <option key={s} value={s}>{s}</option>
          ))}
        </select>
        <span class="deployments-count">
          {filtered.length} deploy{filtered.length !== 1 ? 's' : ''}
        </span>
      </div>

      {/* Timeline */}
      <div class="deployments-timeline">
        {loading && filtered.length === 0 ? (
          <div class="deployments-empty">Loading...</div>
        ) : filtered.length === 0 ? (
          <div class="deployments-empty">No deployments found</div>
        ) : (
          filtered.map((deploy, idx) => {
            const correlated = correlationMap.get(deploy.id) || [];
            const commitShort = deploy.commit ? deploy.commit.slice(0, 7) : '--';
            const hasCorrelation = correlated.length > 0;

            return (
              <div key={deploy.id || idx} class="deploy-timeline-item">
                <div class="deploy-timeline-track">
                  <div
                    class={`deploy-timeline-dot ${hasCorrelation ? 'deploy-timeline-dot-alert' : ''}`}
                  />
                  {idx < filtered.length - 1 && <div class="deploy-timeline-line" />}
                </div>
                <div class={`deploy-timeline-card ${hasCorrelation ? 'deploy-timeline-card-alert' : ''}`}>
                  <div class="deploy-card-header">
                    <span class="deploy-card-service">{deploy.service}</span>
                    <span class="deploy-card-time" title={formatTimestamp(deploy.timestamp)}>
                      {relativeTime(deploy.timestamp)}
                    </span>
                  </div>
                  <div class="deploy-card-message">{deploy.message || 'No description'}</div>
                  <div class="deploy-card-meta">
                    <span class="deploy-card-author">{deploy.author || 'unknown'}</span>
                    <span class="deploy-card-commit" title={deploy.commit}>
                      {commitShort}
                    </span>
                    {deploy.branch && (
                      <span class="deploy-card-branch">{deploy.branch}</span>
                    )}
                  </div>

                  {/* Correlation indicators */}
                  {hasCorrelation && (
                    <div class="deploy-correlation">
                      <div class="deploy-correlation-header">
                        Regressions detected after deploy
                      </div>
                      {correlated.map((reg) => (
                        <div key={reg.id} class="deploy-correlation-item">
                          <span class={`deploy-correlation-badge sev-${reg.severity}`}>
                            {reg.severity}
                          </span>
                          <span class="deploy-correlation-algo">{reg.algorithm}</span>
                          {reg.change_percent != null && (
                            <span class="deploy-correlation-change">
                              {reg.change_percent >= 0 ? '+' : ''}{reg.change_percent.toFixed(1)}%
                            </span>
                          )}
                          {reg.detected_at && (
                            <span class="deploy-correlation-time">
                              {relativeTime(reg.detected_at)}
                            </span>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
