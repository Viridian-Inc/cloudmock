import { useState, useEffect } from 'preact/hooks';
import { api } from '../../lib/api';
import './platform-audit.css';

interface AuditEntry {
  id: string;
  created_at: string;
  actor: string;
  actor_type: string;
  action: string;
  resource: string;
  resource_id: string;
  ip_address: string;
}

interface AuditResponse {
  entries: AuditEntry[];
  total: number;
  offset: number;
  limit: number;
}

const PAGE_SIZE = 5;

const ACTION_TYPES = [
  'all',
  'app.create',
  'app.update',
  'app.delete',
  'key.create',
  'key.revoke',
  'aws.request',
  'org.settings.update',
];

function formatTimestamp(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });
}

function actionBadgeClass(action: string): string {
  if (action.startsWith('app.')) return 'audit-badge audit-badge-blue';
  if (action.startsWith('key.')) return 'audit-badge audit-badge-yellow';
  if (action.startsWith('aws.')) return 'audit-badge audit-badge-gray';
  if (action.startsWith('org.')) return 'audit-badge audit-badge-green';
  return 'audit-badge audit-badge-gray';
}

export function PlatformAuditView() {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [actionFilter, setActionFilter] = useState('all');
  const [page, setPage] = useState(0);

  useEffect(() => {
    const params = new URLSearchParams();
    if (actionFilter !== 'all') params.set('action', actionFilter);
    params.set('offset', String(page * PAGE_SIZE));
    params.set('limit', String(PAGE_SIZE));
    api<AuditResponse>(`/api/platform/audit?${params.toString()}`)
      .then((resp) => {
        setEntries(resp.entries ?? []);
        setTotal(resp.total ?? 0);
      })
      .catch(console.error);
  }, [actionFilter, page]);

  const totalPages = Math.ceil(total / PAGE_SIZE);

  function handleFilterChange(val: string) {
    setActionFilter(val);
    setPage(0);
  }

  function handleExport() {
    const params = new URLSearchParams();
    if (actionFilter !== 'all') params.set('action', actionFilter);
    // Trigger download via direct navigation
    const base = (window as any).__adminBase ?? '';
    window.location.href = `${base}/api/platform/audit/export?${params.toString()}`;
  }

  return (
    <div class="platform-view">
      <div class="platform-header">
        <div class="platform-header-left">
          <h2 class="platform-title">Audit Log</h2>
          <p class="platform-subtitle">Track all actions taken across your organization</p>
        </div>
        <button class="btn" onClick={handleExport}>
          Export CSV
        </button>
      </div>

      {/* Filters */}
      <div class="audit-filters">
        <label class="platform-label">Action Type</label>
        <select
          class="input"
          value={actionFilter}
          onChange={(e) => handleFilterChange((e.target as HTMLSelectElement).value)}
        >
          {ACTION_TYPES.map((a) => (
            <option key={a} value={a}>
              {a === 'all' ? 'All Actions' : a}
            </option>
          ))}
        </select>
      </div>

      <div class="platform-table-wrap">
        <table class="platform-table">
          <thead>
            <tr>
              <th>Timestamp</th>
              <th>Actor</th>
              <th>Action</th>
              <th>Resource</th>
              <th>IP Address</th>
            </tr>
          </thead>
          <tbody>
            {entries.length === 0 && (
              <tr>
                <td colspan={5} class="platform-table-empty">No audit entries match your filter</td>
              </tr>
            )}
            {entries.map((entry) => (
              <tr key={entry.id}>
                <td class="audit-timestamp">{formatTimestamp(entry.created_at)}</td>
                <td>
                  <code class="audit-actor">{entry.actor}</code>
                </td>
                <td>
                  <span class={actionBadgeClass(entry.action)}>{entry.action}</span>
                </td>
                <td class="audit-resource">{entry.resource}</td>
                <td class="audit-ip">{entry.ip_address}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div class="audit-pagination">
          <button
            class="btn"
            disabled={page === 0}
            onClick={() => setPage((p) => p - 1)}
          >
            ← Prev
          </button>
          <span class="audit-page-info">
            Page {page + 1} of {totalPages}
          </span>
          <button
            class="btn"
            disabled={page >= totalPages - 1}
            onClick={() => setPage((p) => p + 1)}
          >
            Next →
          </button>
        </div>
      )}
    </div>
  );
}
