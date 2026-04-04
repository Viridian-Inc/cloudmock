import { useState, useMemo } from 'preact/hooks';
import './platform-audit.css';

interface AuditEntry {
  id: string;
  timestamp: string;
  actor: string;
  action: string;
  resource: string;
  ip: string;
}

// TODO: Replace with API call to GET /api/platform/audit
const MOCK_AUDIT: AuditEntry[] = [
  {
    id: '1',
    timestamp: '2026-04-03T10:45:00Z',
    actor: 'admin@example.com',
    action: 'app.create',
    resource: 'app/staging',
    ip: '203.0.113.10',
  },
  {
    id: '2',
    timestamp: '2026-04-03T10:30:00Z',
    actor: 'cm_live_a1b2',
    action: 'aws.request',
    resource: 's3/my-bucket',
    ip: '198.51.100.5',
  },
  {
    id: '3',
    timestamp: '2026-04-02T18:00:00Z',
    actor: 'admin@example.com',
    action: 'key.revoke',
    resource: 'key/cm_live_e5f6',
    ip: '203.0.113.10',
  },
  {
    id: '4',
    timestamp: '2026-04-01T14:22:00Z',
    actor: 'admin@example.com',
    action: 'key.create',
    resource: 'key/cm_live_c3d4',
    ip: '203.0.113.10',
  },
  {
    id: '5',
    timestamp: '2026-03-30T09:15:00Z',
    actor: 'ci@example.com',
    action: 'app.update',
    resource: 'app/ci-tests',
    ip: '192.0.2.42',
  },
  {
    id: '6',
    timestamp: '2026-03-29T16:00:00Z',
    actor: 'cm_live_a1b2',
    action: 'aws.request',
    resource: 'dynamodb/my-table',
    ip: '198.51.100.5',
  },
  {
    id: '7',
    timestamp: '2026-03-28T11:30:00Z',
    actor: 'admin@example.com',
    action: 'org.settings.update',
    resource: 'org/my-org',
    ip: '203.0.113.10',
  },
];

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

function exportCSV(entries: AuditEntry[]) {
  const header = 'Timestamp,Actor,Action,Resource,IP\n';
  const rows = entries
    .map((e) => `${e.timestamp},"${e.actor}","${e.action}","${e.resource}","${e.ip}"`)
    .join('\n');
  const blob = new Blob([header + rows], { type: 'text/csv' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = 'cloudmock-audit-log.csv';
  a.click();
  URL.revokeObjectURL(url);
}

export function PlatformAuditView() {
  const [actionFilter, setActionFilter] = useState('all');
  const [page, setPage] = useState(0);

  const filtered = useMemo(
    () =>
      actionFilter === 'all'
        ? MOCK_AUDIT
        : MOCK_AUDIT.filter((e) => e.action === actionFilter),
    [actionFilter],
  );

  const totalPages = Math.ceil(filtered.length / PAGE_SIZE);
  const pageEntries = filtered.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);

  function handleFilterChange(val: string) {
    setActionFilter(val);
    setPage(0);
  }

  return (
    <div class="platform-view">
      <div class="platform-header">
        <div class="platform-header-left">
          <h2 class="platform-title">Audit Log</h2>
          <p class="platform-subtitle">Track all actions taken across your organization</p>
        </div>
        <button class="btn" onClick={() => exportCSV(filtered)}>
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
            {pageEntries.length === 0 && (
              <tr>
                <td colspan={5} class="platform-table-empty">No audit entries match your filter</td>
              </tr>
            )}
            {pageEntries.map((entry) => (
              <tr key={entry.id}>
                <td class="audit-timestamp">{formatTimestamp(entry.timestamp)}</td>
                <td>
                  <code class="audit-actor">{entry.actor}</code>
                </td>
                <td>
                  <span class={actionBadgeClass(entry.action)}>{entry.action}</span>
                </td>
                <td class="audit-resource">{entry.resource}</td>
                <td class="audit-ip">{entry.ip}</td>
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
