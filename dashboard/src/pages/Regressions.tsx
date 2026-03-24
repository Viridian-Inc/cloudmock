import { useState, useEffect, useMemo } from 'preact/hooks';
import { fetchRegressions, dismissRegression } from '../api';
import { SummaryCards } from '../components/SummaryCards';

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

function severityColor(sev: string): string {
  switch (sev) {
    case 'critical': return 'var(--error)';
    case 'warning': return 'var(--warning)';
    case 'info': return 'var(--brand-blue)';
    default: return 'var(--text-secondary)';
  }
}

function severityBg(sev: string): string {
  switch (sev) {
    case 'critical': return 'rgba(255,78,94,0.1)';
    case 'warning': return 'rgba(255,154,75,0.15)';
    case 'info': return 'rgba(9,127,245,0.1)';
    default: return 'rgba(100,116,139,0.1)';
  }
}

const ALGORITHM_COLORS: Record<string, { bg: string; fg: string }> = {
  latency:  { bg: 'rgba(255,78,94,0.1)',   fg: 'var(--error)' },
  error:    { bg: 'rgba(255,154,75,0.15)',  fg: 'var(--warning)' },
  tenant:   { bg: 'rgba(167,139,250,0.12)', fg: '#7C3AED' },
  cache:    { bg: 'rgba(9,127,245,0.1)',    fg: 'var(--brand-blue)' },
  fanout:   { bg: 'rgba(2,150,98,0.1)',     fg: 'var(--success)' },
  payload:  { bg: 'rgba(254,195,7,0.15)',   fg: '#B8860B' },
};

function algorithmStyle(algo: string) {
  const c = ALGORITHM_COLORS[algo] || { bg: 'rgba(100,116,139,0.1)', fg: 'var(--text-secondary)' };
  return { background: c.bg, color: c.fg };
}

const ALGORITHM_TYPES = ['latency', 'error', 'tenant', 'cache', 'fanout', 'payload'];

export function RegressionsPage() {
  const [regressions, setRegressions] = useState<any[]>([]);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState('');
  const [severityFilter, setSeverityFilter] = useState('');
  const [algorithmFilter, setAlgorithmFilter] = useState('');
  const [serviceFilter, setServiceFilter] = useState('');
  const [loading, setLoading] = useState(true);

  function loadRegressions() {
    setLoading(true);
    const params: Record<string, string> = {};
    if (statusFilter) params.status = statusFilter;
    if (severityFilter) params.severity = severityFilter;
    if (serviceFilter) params.service = serviceFilter;
    fetchRegressions(params)
      .then(setRegressions)
      .catch(() => setRegressions([]))
      .finally(() => setLoading(false));
  }

  useEffect(() => { loadRegressions(); }, [statusFilter, severityFilter, serviceFilter]);

  const filtered = useMemo(() => {
    let data = [...regressions];
    if (algorithmFilter) {
      data = data.filter(r => r.algorithm === algorithmFilter);
    }
    return data.sort((a, b) => {
      const ta = new Date(a.detected_at || 0).getTime();
      const tb = new Date(b.detected_at || 0).getTime();
      return tb - ta;
    });
  }, [regressions, algorithmFilter]);

  const activeCount = regressions.filter(r => r.status === 'active').length;
  const criticalCount = regressions.filter(r => r.severity === 'critical').length;
  const warningCount = regressions.filter(r => r.severity === 'warning').length;
  const infoCount = regressions.filter(r => r.severity === 'info').length;

  const byAlgorithm = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const t of ALGORITHM_TYPES) counts[t] = 0;
    for (const r of regressions) {
      if (r.algorithm && counts[r.algorithm] !== undefined) counts[r.algorithm]++;
    }
    return counts;
  }, [regressions]);

  const summaryCards = [
    { label: 'Active', value: activeCount, color: activeCount > 0 ? 'var(--error)' : undefined },
    { label: 'Critical', value: criticalCount, color: criticalCount > 0 ? 'var(--error)' : undefined },
    { label: 'Warning', value: warningCount, color: warningCount > 0 ? 'var(--warning)' : undefined },
    { label: 'Info', value: infoCount },
  ];

  const algorithmCards = ALGORITHM_TYPES.map(t => ({
    label: t.charAt(0).toUpperCase() + t.slice(1),
    value: byAlgorithm[t] || 0,
  }));

  function toggleExpand(id: string) {
    setExpanded(prev => prev === id ? null : id);
  }

  function handleDismiss(e: Event, id: string) {
    e.stopPropagation();
    dismissRegression(id).then(() => loadRegressions()).catch(() => {});
  }

  function formatChange(pct: number | undefined | null): { text: string; color: string } {
    if (pct === undefined || pct === null) return { text: '—', color: 'var(--text-secondary)' };
    const sign = pct >= 0 ? '+' : '';
    const color = pct > 0 ? 'var(--error)' : pct < 0 ? 'var(--success)' : 'var(--text-secondary)';
    return { text: `${sign}${pct.toFixed(1)}%`, color };
  }

  function truncateDeploy(id: string | undefined | null): string {
    if (!id) return '—';
    return id.length > 12 ? id.slice(0, 12) + '...' : id;
  }

  return (
    <div>
      <div class="flex items-center justify-between mb-6">
        <div>
          <h1 class="page-title">Regressions</h1>
          <p class="page-desc">Detected performance and reliability regressions</p>
        </div>
        <button class="btn btn-ghost btn-sm" onClick={loadRegressions}>
          Refresh
        </button>
      </div>

      <SummaryCards cards={summaryCards} />
      <SummaryCards cards={algorithmCards} />

      <div class="filters-bar">
        <select class="select" value={statusFilter} onChange={(e) => setStatusFilter((e.target as HTMLSelectElement).value)}>
          <option value="">All Status</option>
          <option value="active">Active</option>
          <option value="dismissed">Dismissed</option>
        </select>
        <select class="select" value={severityFilter} onChange={(e) => setSeverityFilter((e.target as HTMLSelectElement).value)}>
          <option value="">All Severity</option>
          <option value="critical">Critical</option>
          <option value="warning">Warning</option>
          <option value="info">Info</option>
        </select>
        <select class="select" value={algorithmFilter} onChange={(e) => setAlgorithmFilter((e.target as HTMLSelectElement).value)}>
          <option value="">All Algorithms</option>
          {ALGORITHM_TYPES.map(t => <option value={t}>{t.charAt(0).toUpperCase() + t.slice(1)}</option>)}
        </select>
        <input
          class="input input-search"
          placeholder="Filter by service..."
          value={serviceFilter}
          onInput={(e) => setServiceFilter((e.target as HTMLInputElement).value)}
        />
        <span class="text-sm text-muted ml-auto">{filtered.length} regressions</span>
      </div>

      <div class="card">
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th style="width:90px">Severity</th>
                <th style="width:100px">Algorithm</th>
                <th>Service</th>
                <th style="width:90px">Change %</th>
                <th style="width:100px">Confidence</th>
                <th style="width:110px">Deploy</th>
                <th style="width:100px">Detected</th>
              </tr>
            </thead>
            <tbody>
              {loading && filtered.length === 0 ? (
                <tr><td colSpan={7} class="empty-state">Loading...</td></tr>
              ) : filtered.length === 0 ? (
                <tr><td colSpan={7} class="empty-state">No regressions found</td></tr>
              ) : filtered.map((reg: any) => {
                const change = formatChange(reg.change_percent);
                return (
                  <>
                    <tr
                      class={`clickable ${expanded === reg.id ? 'expanded' : ''}`}
                      onClick={() => toggleExpand(reg.id)}
                      key={reg.id}
                    >
                      <td>
                        <span class="status-pill" style={{ background: severityBg(reg.severity), color: severityColor(reg.severity) }}>
                          {reg.severity}
                        </span>
                      </td>
                      <td>
                        <span class="status-pill" style={algorithmStyle(reg.algorithm)}>
                          {reg.algorithm}
                        </span>
                      </td>
                      <td style="font-weight:600">{reg.service}</td>
                      <td class="font-mono text-sm" style={{ color: change.color, fontWeight: 600 }}>
                        {change.text}
                      </td>
                      <td class="font-mono text-sm">
                        {reg.confidence != null ? `${(reg.confidence * 100).toFixed(0)}%` : '—'}
                      </td>
                      <td class="font-mono text-sm" title={reg.deploy_id || ''}>
                        {truncateDeploy(reg.deploy_id)}
                      </td>
                      <td class="font-mono text-sm">{formatRelativeTime(reg.detected_at)}</td>
                    </tr>
                    {expanded === reg.id && (
                      <tr>
                        <td colSpan={7} style="padding:0">
                          <div class="req-expand">
                            <div class="req-expand-inner">
                              <div class="req-expand-body">
                                <table>
                                  <tbody>
                                    <tr><td style="font-weight:600;width:150px">Before Value</td><td class="font-mono text-sm">{reg.before_value ?? '—'}</td></tr>
                                    <tr><td style="font-weight:600">After Value</td><td class="font-mono text-sm">{reg.after_value ?? '—'}</td></tr>
                                    <tr><td style="font-weight:600">Sample Size</td><td class="font-mono text-sm">{reg.sample_size ?? '—'}</td></tr>
                                    {reg.time_window && (
                                      <tr><td style="font-weight:600">Time Window</td><td class="font-mono text-sm">{reg.time_window}</td></tr>
                                    )}
                                    {reg.tenant && (
                                      <tr><td style="font-weight:600">Tenant</td><td>{reg.tenant}</td></tr>
                                    )}
                                  </tbody>
                                </table>
                                <div style="margin-top:16px;display:flex;gap:8px">
                                  {reg.status !== 'dismissed' && (
                                    <button class="btn btn-sm btn-ghost" onClick={(e) => handleDismiss(e, reg.id)}>
                                      Dismiss
                                    </button>
                                  )}
                                </div>
                              </div>
                            </div>
                          </div>
                        </td>
                      </tr>
                    )}
                  </>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
