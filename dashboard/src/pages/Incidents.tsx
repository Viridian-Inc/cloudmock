import { useState, useEffect, useMemo } from 'preact/hooks';
import { fetchIncidents, acknowledgeIncident, resolveIncident, fetchIncidentReport } from '../api';
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

function formatTimestamp(ts: string | undefined | null): string {
  if (!ts) return '—';
  return new Date(ts).toLocaleString('en-US', {
    month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false,
  });
}

function severityColor(sev: string): string {
  switch (sev) {
    case 'critical': return 'var(--error)';
    case 'warning': return 'var(--warning)';
    case 'info': return 'var(--brand-blue)';
    default: return 'var(--n500)';
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

function statusColor(status: string): string {
  switch (status) {
    case 'active': return 'var(--error)';
    case 'acknowledged': return 'var(--warning)';
    case 'resolved': return 'var(--primary-green)';
    default: return 'var(--n500)';
  }
}

function statusBg(status: string): string {
  switch (status) {
    case 'active': return 'rgba(255,78,94,0.1)';
    case 'acknowledged': return 'rgba(255,154,75,0.15)';
    case 'resolved': return 'rgba(2,150,98,0.1)';
    default: return 'rgba(100,116,139,0.1)';
  }
}

export function IncidentsPage() {
  const [incidents, setIncidents] = useState<any[]>([]);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState('');
  const [severityFilter, setSeverityFilter] = useState('');
  const [serviceFilter, setServiceFilter] = useState('');
  const [loading, setLoading] = useState(true);

  function loadIncidents() {
    setLoading(true);
    const params: Record<string, string> = {};
    if (statusFilter) params.status = statusFilter;
    if (severityFilter) params.severity = severityFilter;
    if (serviceFilter) params.service = serviceFilter;
    fetchIncidents(params)
      .then(setIncidents)
      .catch(() => setIncidents([]))
      .finally(() => setLoading(false));
  }

  useEffect(() => { loadIncidents(); }, [statusFilter, severityFilter, serviceFilter]);

  const sorted = useMemo(() => {
    return [...incidents].sort((a, b) => {
      const ta = new Date(a.first_seen || 0).getTime();
      const tb = new Date(b.first_seen || 0).getTime();
      return tb - ta;
    });
  }, [incidents]);

  const activeCount = incidents.filter(i => i.status === 'active').length;
  const criticalCount = incidents.filter(i => i.severity === 'critical').length;
  const warningCount = incidents.filter(i => i.severity === 'warning').length;
  const resolvedTodayCount = useMemo(() => {
    const todayStart = new Date();
    todayStart.setHours(0, 0, 0, 0);
    return incidents.filter(i => i.status === 'resolved' && new Date(i.resolved_at || 0) >= todayStart).length;
  }, [incidents]);

  const summaryCards = [
    { label: 'Active', value: activeCount, color: activeCount > 0 ? 'var(--error)' : undefined },
    { label: 'Critical', value: criticalCount, color: criticalCount > 0 ? 'var(--error)' : undefined },
    { label: 'Warning', value: warningCount, color: warningCount > 0 ? 'var(--warning)' : undefined },
    { label: 'Resolved Today', value: resolvedTodayCount },
  ];

  function toggleExpand(id: string) {
    setExpanded(prev => prev === id ? null : id);
  }

  function handleAcknowledge(e: Event, id: string) {
    e.stopPropagation();
    const owner = prompt('Enter owner name:');
    if (!owner) return;
    acknowledgeIncident(id, owner).then(() => loadIncidents()).catch(() => {});
  }

  function handleResolve(e: Event, id: string) {
    e.stopPropagation();
    resolveIncident(id).then(() => loadIncidents()).catch(() => {});
  }

  function handleExportReport(e: Event, id: string) {
    e.stopPropagation();
    fetchIncidentReport(id, 'html').then((html: string) => {
      const blob = new Blob([html], { type: 'text/html' });
      const url = URL.createObjectURL(blob);
      window.open(url, '_blank');
    }).catch(() => {});
  }

  return (
    <div>
      <div class="flex items-center justify-between mb-6">
        <div>
          <h1 class="page-title">Incidents</h1>
          <p class="page-desc">Active and historical incident tracking</p>
        </div>
        <button class="btn btn-ghost btn-sm" onClick={loadIncidents}>
          Refresh
        </button>
      </div>

      <SummaryCards cards={summaryCards} />

      <div class="filters-bar">
        <select class="select" value={statusFilter} onChange={(e) => setStatusFilter((e.target as HTMLSelectElement).value)}>
          <option value="">All Status</option>
          <option value="active">Active</option>
          <option value="acknowledged">Acknowledged</option>
          <option value="resolved">Resolved</option>
        </select>
        <select class="select" value={severityFilter} onChange={(e) => setSeverityFilter((e.target as HTMLSelectElement).value)}>
          <option value="">All Severity</option>
          <option value="critical">Critical</option>
          <option value="warning">Warning</option>
          <option value="info">Info</option>
        </select>
        <input
          class="input input-search"
          placeholder="Filter by service..."
          value={serviceFilter}
          onInput={(e) => setServiceFilter((e.target as HTMLInputElement).value)}
        />
        <span class="text-sm text-muted ml-auto">{sorted.length} incidents</span>
      </div>

      <div class="card">
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th style="width:90px">Severity</th>
                <th>Title</th>
                <th>Affected Services</th>
                <th style="width:80px">Alerts</th>
                <th style="width:100px">First Seen</th>
                <th style="width:110px">Status</th>
              </tr>
            </thead>
            <tbody>
              {loading && sorted.length === 0 ? (
                <tr><td colSpan={6} class="empty-state">Loading...</td></tr>
              ) : sorted.length === 0 ? (
                <tr><td colSpan={6} class="empty-state">No incidents found</td></tr>
              ) : sorted.map((inc: any) => (
                <>
                  <tr
                    class={`clickable ${expanded === inc.id ? 'expanded' : ''}`}
                    onClick={() => toggleExpand(inc.id)}
                    key={inc.id}
                  >
                    <td>
                      <span class="status-pill" style={{ background: severityBg(inc.severity), color: severityColor(inc.severity) }}>
                        {inc.severity}
                      </span>
                    </td>
                    <td style="font-weight:600">{inc.title}</td>
                    <td class="text-sm">{(inc.affected_services || []).join(', ') || '—'}</td>
                    <td class="font-mono text-sm">{inc.alert_count ?? '—'}</td>
                    <td class="font-mono text-sm">{formatRelativeTime(inc.first_seen)}</td>
                    <td>
                      <span class="status-pill" style={{ background: statusBg(inc.status), color: statusColor(inc.status) }}>
                        {inc.status}
                      </span>
                    </td>
                  </tr>
                  {expanded === inc.id && (
                    <tr>
                      <td colSpan={6} style="padding:0">
                        <div class="req-expand">
                          <div class="req-expand-inner">
                            <div class="req-expand-body">
                              <table>
                                <tbody>
                                  {inc.root_cause && (
                                    <tr><td style="font-weight:600;width:150px">Root Cause</td><td>{inc.root_cause}</td></tr>
                                  )}
                                  {inc.related_deploy_id && (
                                    <tr><td style="font-weight:600">Related Deploy</td><td class="font-mono text-sm">{inc.related_deploy_id}</td></tr>
                                  )}
                                  {inc.owner && (
                                    <tr><td style="font-weight:600">Owner</td><td>{inc.owner}</td></tr>
                                  )}
                                  <tr><td style="font-weight:600">First Seen</td><td class="font-mono text-sm">{formatTimestamp(inc.first_seen)}</td></tr>
                                  <tr><td style="font-weight:600">Last Seen</td><td class="font-mono text-sm">{formatTimestamp(inc.last_seen)}</td></tr>
                                  {inc.resolved_at && (
                                    <tr><td style="font-weight:600">Resolved At</td><td class="font-mono text-sm">{formatTimestamp(inc.resolved_at)}</td></tr>
                                  )}
                                </tbody>
                              </table>
                              <div style="margin-top:16px;display:flex;gap:8px">
                                {inc.status === 'active' && (
                                  <button class="btn btn-sm btn-ghost" onClick={(e) => handleAcknowledge(e, inc.id)}>
                                    Acknowledge
                                  </button>
                                )}
                                {inc.status !== 'resolved' && (
                                  <button class="btn btn-sm btn-primary" onClick={(e) => handleResolve(e, inc.id)}>
                                    Resolve
                                  </button>
                                )}
                                <button class="btn btn-sm btn-secondary" onClick={(e) => handleExportReport(e, inc.id)}>
                                  Export Report
                                </button>
                              </div>
                            </div>
                          </div>
                        </div>
                      </td>
                    </tr>
                  )}
                </>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
