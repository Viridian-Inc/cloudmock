import { useState, useEffect, useCallback, useMemo } from 'preact/hooks';
import { api } from '../../lib/api';

interface AuditEntry {
  id?: string;
  timestamp: string;
  actor: string;
  action: string;
  resource: string;
  details?: string;
}

const PAGE_SIZE = 50;

function fmtTime(ts: string): string {
  try {
    const d = new Date(ts);
    return d.toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  } catch {
    return ts;
  }
}

export function Audit() {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionFilter, setActionFilter] = useState('');
  const [page, setPage] = useState(0);

  const load = useCallback(() => {
    setLoading(true);
    api<AuditEntry[]>('/api/audit?limit=500')
      .then((data) => {
        setEntries(data);
        setError(null);
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : String(err));
        setEntries([]);
      })
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  // Derive unique actions for filter dropdown
  const uniqueActions = useMemo(() => {
    const actions = new Set<string>();
    for (const entry of entries) {
      if (entry.action) actions.add(entry.action);
    }
    return Array.from(actions).sort();
  }, [entries]);

  // Filtered entries
  const filtered = useMemo(() => {
    if (!actionFilter) return entries;
    return entries.filter((e) => e.action === actionFilter);
  }, [entries, actionFilter]);

  // Paginated entries
  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const paginated = useMemo(() => {
    const start = page * PAGE_SIZE;
    return filtered.slice(start, start + PAGE_SIZE);
  }, [filtered, page]);

  // Reset page when filter changes
  useEffect(() => {
    setPage(0);
  }, [actionFilter]);

  return (
    <div class="settings-section" style="max-width: 900px;">
      <h3 class="settings-section-title">Audit Log</h3>
      <p class="settings-section-desc">
        All administrative actions and API operations recorded by cloudmock.
      </p>

      {error && <div class="settings-error">{error}</div>}

      <div class="audit-controls">
        <select
          class="input audit-filter-select"
          value={actionFilter}
          onChange={(e) => setActionFilter((e.target as HTMLSelectElement).value)}
        >
          <option value="">All actions</option>
          {uniqueActions.map((a) => (
            <option key={a} value={a}>
              {a}
            </option>
          ))}
        </select>
        <span class="audit-count">{filtered.length} entries</span>
        <button class="btn btn-ghost" onClick={load} disabled={loading}>
          {loading ? 'Loading...' : 'Refresh'}
        </button>
      </div>

      {loading && entries.length === 0 ? (
        <div class="settings-placeholder">Loading audit log...</div>
      ) : (
        <div class="audit-table-wrapper">
          <table class="audit-table">
            <thead>
              <tr>
                <th>Timestamp</th>
                <th>User</th>
                <th>Action</th>
                <th>Resource</th>
                <th>Details</th>
              </tr>
            </thead>
            <tbody>
              {paginated.map((entry, i) => (
                <tr key={entry.id || `${entry.timestamp}-${i}`}>
                  <td>
                    <span class="audit-timestamp">{fmtTime(entry.timestamp)}</span>
                  </td>
                  <td>{entry.actor || '-'}</td>
                  <td>
                    <code class="audit-action-code">{entry.action}</code>
                  </td>
                  <td>{entry.resource || '-'}</td>
                  <td>
                    <span class="audit-details">
                      {entry.details || '-'}
                    </span>
                  </td>
                </tr>
              ))}
              {paginated.length === 0 && (
                <tr>
                  <td colSpan={5} class="audit-empty">
                    No audit entries found.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      {totalPages > 1 && (
        <div class="audit-pagination">
          <button
            class="btn btn-ghost"
            onClick={() => setPage((p) => Math.max(0, p - 1))}
            disabled={page === 0}
          >
            Previous
          </button>
          <span class="audit-page-info">
            Page {page + 1} of {totalPages}
          </span>
          <button
            class="btn btn-ghost"
            onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
            disabled={page >= totalPages - 1}
          >
            Next
          </button>
        </div>
      )}
    </div>
  );
}
