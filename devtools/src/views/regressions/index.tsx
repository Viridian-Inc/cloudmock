import { useState, useEffect, useMemo } from 'preact/hooks';
import { api } from '../../lib/api';
import './regressions.css';

interface Regression {
  id: string;
  algorithm: string;
  severity: string;
  service: string;
  status: string;
  change_percent?: number;
  confidence?: number;
  deploy_id?: string;
  detected_at?: string;
  before_value?: number;
  after_value?: number;
  sample_size?: number;
  time_window?: string;
  tenant?: string;
}

const ALGORITHM_TYPES = ['latency', 'error', 'tenant', 'cache', 'fanout', 'payload'] as const;

function formatRelativeTime(ts: string | undefined | null): string {
  if (!ts) return '';
  const diff = Date.now() - new Date(ts).getTime();
  if (diff < 0) return 'just now';
  const seconds = Math.floor(diff / 1000);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

function formatChange(pct: number | undefined | null): { text: string; color: string } {
  if (pct === undefined || pct === null) return { text: '--', color: 'var(--text-secondary)' };
  const sign = pct >= 0 ? '+' : '';
  const color = pct > 0 ? 'var(--error)' : pct < 0 ? 'var(--success)' : 'var(--text-secondary)';
  return { text: `${sign}${pct.toFixed(1)}%`, color };
}

function truncateDeploy(id: string | undefined | null): string {
  if (!id) return '--';
  return id.length > 12 ? id.slice(0, 12) + '...' : id;
}

function severityClass(sev: string): string {
  switch (sev) {
    case 'critical': return 'regressions-sev-critical';
    case 'warning': return 'regressions-sev-warning';
    case 'info': return 'regressions-sev-info';
    default: return '';
  }
}

function algorithmClass(algo: string): string {
  return `regressions-algo-${algo}` || '';
}

export function RegressionsView() {
  const [regressions, setRegressions] = useState<Regression[]>([]);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState('');
  const [severityFilter, setSeverityFilter] = useState('');
  const [algorithmFilter, setAlgorithmFilter] = useState('');
  const [loading, setLoading] = useState(true);

  function loadRegressions() {
    setLoading(true);
    const params = new URLSearchParams();
    if (statusFilter) params.set('status', statusFilter);
    if (severityFilter) params.set('severity', severityFilter);
    const qs = params.toString();
    api<Regression[]>(`/api/regressions${qs ? `?${qs}` : ''}`)
      .then(setRegressions)
      .catch(() => setRegressions([]))
      .finally(() => setLoading(false));
  }

  useEffect(() => {
    loadRegressions();
  }, [statusFilter, severityFilter]);

  const filtered = useMemo(() => {
    let data = [...regressions];
    if (algorithmFilter) {
      data = data.filter((r) => r.algorithm === algorithmFilter);
    }
    return data.sort((a, b) => {
      const ta = new Date(a.detected_at || 0).getTime();
      const tb = new Date(b.detected_at || 0).getTime();
      return tb - ta;
    });
  }, [regressions, algorithmFilter]);

  // Summary counts
  const activeCount = regressions.filter((r) => r.status === 'active').length;
  const criticalCount = regressions.filter((r) => r.severity === 'critical').length;
  const warningCount = regressions.filter((r) => r.severity === 'warning').length;
  const infoCount = regressions.filter((r) => r.severity === 'info').length;

  function toggleExpand(id: string) {
    setExpanded((prev) => (prev === id ? null : id));
  }

  function handleDismiss(e: Event, id: string) {
    e.stopPropagation();
    api<void>(`/api/regressions/${id}/dismiss`, { method: 'POST' })
      .then(() => loadRegressions())
      .catch(() => {});
  }

  return (
    <div class="regressions-view">
      <div class="regressions-header">
        <div>
          <h2 class="regressions-title">Regressions</h2>
          <p class="regressions-desc">
            Detected performance and reliability regressions
          </p>
        </div>
        <button class="regressions-refresh-btn" onClick={loadRegressions}>
          Refresh
        </button>
      </div>

      {/* Summary cards */}
      <div class="regressions-cards">
        <div class="regressions-stat">
          <div
            class="regressions-stat-value"
            style={{ color: activeCount > 0 ? 'var(--error)' : undefined }}
          >
            {activeCount}
          </div>
          <div class="regressions-stat-label">Active</div>
        </div>
        <div class="regressions-stat">
          <div
            class="regressions-stat-value"
            style={{ color: criticalCount > 0 ? 'var(--error)' : undefined }}
          >
            {criticalCount}
          </div>
          <div class="regressions-stat-label">Critical</div>
        </div>
        <div class="regressions-stat">
          <div
            class="regressions-stat-value"
            style={{ color: warningCount > 0 ? 'var(--warning)' : undefined }}
          >
            {warningCount}
          </div>
          <div class="regressions-stat-label">Warning</div>
        </div>
        <div class="regressions-stat">
          <div class="regressions-stat-value">{infoCount}</div>
          <div class="regressions-stat-label">Info</div>
        </div>
      </div>

      {/* Filters */}
      <div class="regressions-filters">
        <select
          class="regressions-select"
          value={statusFilter}
          onChange={(e) =>
            setStatusFilter((e.target as HTMLSelectElement).value)
          }
        >
          <option value="">All Status</option>
          <option value="active">Active</option>
          <option value="dismissed">Dismissed</option>
        </select>
        <select
          class="regressions-select"
          value={severityFilter}
          onChange={(e) =>
            setSeverityFilter((e.target as HTMLSelectElement).value)
          }
        >
          <option value="">All Severity</option>
          <option value="critical">Critical</option>
          <option value="warning">Warning</option>
          <option value="info">Info</option>
        </select>
        <select
          class="regressions-select"
          value={algorithmFilter}
          onChange={(e) =>
            setAlgorithmFilter((e.target as HTMLSelectElement).value)
          }
        >
          <option value="">All Algorithms</option>
          {ALGORITHM_TYPES.map((t) => (
            <option key={t} value={t}>
              {t.charAt(0).toUpperCase() + t.slice(1)}
            </option>
          ))}
        </select>
        <span class="regressions-count">
          {filtered.length} regression{filtered.length !== 1 ? 's' : ''}
        </span>
      </div>

      {/* Table */}
      <div class="regressions-table-wrap">
        <table class="regressions-table">
          <thead>
            <tr>
              <th style={{ width: '80px' }}>Severity</th>
              <th style={{ width: '90px' }}>Algorithm</th>
              <th>Service</th>
              <th style={{ width: '85px' }}>Change</th>
              <th style={{ width: '90px' }}>Confidence</th>
              <th style={{ width: '100px' }}>Deploy</th>
              <th style={{ width: '90px' }}>Detected</th>
            </tr>
          </thead>
          <tbody>
            {loading && filtered.length === 0 ? (
              <tr>
                <td colSpan={7}>
                  <div class="regressions-empty">Loading...</div>
                </td>
              </tr>
            ) : filtered.length === 0 ? (
              <tr>
                <td colSpan={7}>
                  <div class="regressions-empty">No regressions found</div>
                </td>
              </tr>
            ) : (
              filtered.map((reg) => {
                const change = formatChange(reg.change_percent);
                const isExpanded = expanded === reg.id;
                return (
                  <>
                    <tr
                      key={reg.id}
                      class={`regressions-row ${isExpanded ? 'regressions-row-expanded' : ''}`}
                      onClick={() => toggleExpand(reg.id)}
                    >
                      <td>
                        <span
                          class={`regressions-badge ${severityClass(reg.severity)}`}
                        >
                          {reg.severity}
                        </span>
                      </td>
                      <td>
                        <span
                          class={`regressions-badge ${algorithmClass(reg.algorithm)}`}
                        >
                          {reg.algorithm}
                        </span>
                      </td>
                      <td class="regressions-service">{reg.service}</td>
                      <td
                        class="regressions-mono"
                        style={{ color: change.color, fontWeight: 600 }}
                      >
                        {change.text}
                      </td>
                      <td class="regressions-mono">
                        {reg.confidence != null
                          ? `${(reg.confidence * 100).toFixed(0)}%`
                          : '--'}
                      </td>
                      <td
                        class="regressions-mono"
                        title={reg.deploy_id || ''}
                      >
                        {truncateDeploy(reg.deploy_id)}
                      </td>
                      <td class="regressions-mono">
                        {formatRelativeTime(reg.detected_at)}
                      </td>
                    </tr>
                    {isExpanded && (
                      <tr key={`${reg.id}-detail`}>
                        <td colSpan={7} class="regressions-detail-cell">
                          <div class="regressions-detail">
                            <table class="regressions-detail-table">
                              <tbody>
                                <tr>
                                  <td class="regressions-detail-label">
                                    Before Value
                                  </td>
                                  <td class="regressions-mono">
                                    {reg.before_value ?? '--'}
                                  </td>
                                </tr>
                                <tr>
                                  <td class="regressions-detail-label">
                                    After Value
                                  </td>
                                  <td class="regressions-mono">
                                    {reg.after_value ?? '--'}
                                  </td>
                                </tr>
                                <tr>
                                  <td class="regressions-detail-label">
                                    Sample Size
                                  </td>
                                  <td class="regressions-mono">
                                    {reg.sample_size ?? '--'}
                                  </td>
                                </tr>
                                {reg.time_window && (
                                  <tr>
                                    <td class="regressions-detail-label">
                                      Time Window
                                    </td>
                                    <td class="regressions-mono">
                                      {reg.time_window}
                                    </td>
                                  </tr>
                                )}
                                {reg.tenant && (
                                  <tr>
                                    <td class="regressions-detail-label">
                                      Tenant
                                    </td>
                                    <td>{reg.tenant}</td>
                                  </tr>
                                )}
                              </tbody>
                            </table>
                            {reg.status !== 'dismissed' && (
                              <button
                                class="regressions-dismiss-btn"
                                onClick={(e) => handleDismiss(e, reg.id)}
                              >
                                Dismiss
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                    )}
                  </>
                );
              })
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
