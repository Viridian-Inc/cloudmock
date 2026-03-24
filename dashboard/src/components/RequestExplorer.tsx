import { useState, useEffect, useMemo, useRef, useCallback } from 'preact/hooks';
import { api, getViews, createView, deleteView } from '../api';
import { StatusBadge } from './StatusBadge';
import { fmtTime, fmtDuration } from '../utils';
import type { SSEState } from '../hooks/useSSE';
import type { RequestFilters } from '../hooks/useFilters';

interface RequestExplorerProps {
  sse: SSEState;
  filters: RequestFilters;
  setFilter: <K extends keyof RequestFilters>(key: K, value: RequestFilters[K]) => void;
  clearFilters: () => void;
  hasActiveFilters: boolean;
  onSelectRequest: (req: any) => void;
  selectedRequestId?: string;
}

const METHOD_COLORS: Record<string, string> = {
  GET: '#538eff', POST: '#36d982', PUT: '#fad065', DELETE: '#ff4e5e',
  PATCH: '#8B5CF6', HEAD: '#5a6577', OPTIONS: '#5a6577',
};

export function RequestExplorer({
  sse, filters, setFilter, clearFilters, hasActiveFilters,
  onSelectRequest, selectedRequestId,
}: RequestExplorerProps) {
  const [requests, setRequests] = useState<any[]>([]);
  const [services, setServices] = useState<string[]>([]);
  const [paused, setPaused] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [showFilters, setShowFilters] = useState(false);
  const [searchText, setSearchText] = useState('');
  const [views, setViews] = useState<any[]>([]);
  const [viewName, setViewName] = useState('');
  const [showViewMenu, setShowViewMenu] = useState(false);
  const bufferRef = useRef<any[]>([]);
  const listRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    api('/api/requests?limit=100&level=app').then(setRequests).catch(() => {});
    api('/api/services').then((s: any[]) => setServices(s.map((x: any) => x.name).sort())).catch(() => {});
    getViews().then(setViews).catch(() => {});
  }, []);

  // SSE live tail - filter infra events
  useEffect(() => {
    if (sse.events.length === 0) return;
    const latest = sse.events[0];
    if (latest?.type === 'request' && latest.data) {
      if (latest.data.level === 'infra') return;
      if (paused) {
        bufferRef.current = [latest.data, ...bufferRef.current].slice(0, 200);
      } else {
        setRequests(prev => [latest.data, ...prev].slice(0, 200));
      }
    }
  }, [sse.events, paused]);

  const handleResume = useCallback(() => {
    setPaused(false);
    if (bufferRef.current.length > 0) {
      setRequests(prev => [...bufferRef.current, ...prev].slice(0, 200));
      bufferRef.current = [];
    }
  }, []);

  const handleSaveView = useCallback(() => {
    if (!viewName.trim()) return;
    createView({ name: viewName, filters }).then(() => {
      getViews().then(setViews).catch(() => {});
      setViewName('');
      setShowViewMenu(false);
    }).catch(() => {});
  }, [viewName, filters]);

  const handleLoadView = useCallback((view: any) => {
    if (view.filters) {
      const f = view.filters;
      Object.entries(f).forEach(([k, v]) => {
        setFilter(k as keyof RequestFilters, v as any);
      });
    }
    setShowViewMenu(false);
  }, [setFilter]);

  const handleDeleteView = useCallback((id: string) => {
    deleteView(id).then(() => {
      getViews().then(setViews).catch(() => {});
    }).catch(() => {});
  }, []);

  const activeFilterCount = [
    filters.service, filters.method, filters.error ? 'x' : '',
    filters.tenant_id, filters.org_id, filters.user_id,
    filters.path, filters.min_latency_ms, filters.max_latency_ms,
    filters.from, filters.to,
  ].filter(Boolean).length;

  const filtered = useMemo(() => {
    return requests.filter((r: any) => {
      if (filters.service && r.service !== filters.service) return false;
      if (filters.method && r.method !== filters.method) return false;
      if (filters.path && !(r.path || '').startsWith(filters.path)) return false;
      if (filters.caller_id && r.caller_id !== filters.caller_id) return false;
      if (filters.error && (r.status || r.status_code) < 400) return false;
      if (filters.tenant_id && r.tenant_id !== filters.tenant_id) return false;
      if (filters.org_id && r.org_id !== filters.org_id) return false;
      if (filters.user_id && r.user_id !== filters.user_id) return false;
      if (filters.min_latency_ms && (r.latency_ms || 0) < filters.min_latency_ms) return false;
      if (filters.max_latency_ms && (r.latency_ms || 0) > filters.max_latency_ms) return false;
      if (searchText) {
        const q = searchText.toLowerCase();
        const hay = `${r.service} ${r.action} ${r.method} ${r.path} ${r.id || ''}`.toLowerCase();
        if (!hay.includes(q)) return false;
      }
      return true;
    });
  }, [requests, filters, searchText]);

  return (
    <div style={S.panel}>
      {/* Header */}
      <div style={S.header}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={S.title}>Explorer</span>
          <span style={S.count}>{filtered.length}</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <button
            onClick={() => setShowViewMenu(!showViewMenu)}
            style={{ ...S.iconBtn, color: showViewMenu ? 'var(--brand-teal)' : 'var(--text-tertiary)' }}
            title="Saved views"
          >
            {'\u2606'}
          </button>
          <button
            onClick={() => setShowFilters(!showFilters)}
            style={{ ...S.iconBtn, color: showFilters || hasActiveFilters ? 'var(--brand-teal)' : 'var(--text-tertiary)' }}
            title="Filters"
          >
            {'\u2630'}
            {activeFilterCount > 0 && <span style={S.filterBadge}>{activeFilterCount}</span>}
          </button>
        </div>
      </div>

      {/* Saved views dropdown */}
      {showViewMenu && (
        <div style={S.viewMenu}>
          <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-secondary)', marginBottom: 6 }}>Saved Views</div>
          {views.length === 0 && (
            <div style={{ fontSize: 11, color: 'var(--text-tertiary)', padding: '4px 0' }}>No saved views</div>
          )}
          {views.map((v: any) => (
            <div key={v.id} style={S.viewRow}>
              <span
                onClick={() => handleLoadView(v)}
                style={{ flex: 1, cursor: 'pointer', fontSize: 11, fontWeight: 500 }}
              >
                {v.name}
              </span>
              <button
                onClick={() => handleDeleteView(v.id)}
                style={{ ...S.iconBtn, fontSize: 10, padding: 2 }}
              >
                {'\u2715'}
              </button>
            </div>
          ))}
          <div style={{ display: 'flex', gap: 4, marginTop: 6, borderTop: '1px solid var(--border-subtle)', paddingTop: 6 }}>
            <input
              type="text"
              placeholder="View name..."
              value={viewName}
              onInput={(e) => setViewName((e.target as HTMLInputElement).value)}
              style={{ ...S.filterInput, flex: 1 }}
            />
            <button onClick={handleSaveView} style={S.saveBtn}>Save</button>
          </div>
        </div>
      )}

      {/* Search */}
      <div style={S.searchWrap}>
        <input
          type="text"
          placeholder="Search requests..."
          value={searchText}
          onInput={(e) => setSearchText((e.target as HTMLInputElement).value)}
          style={S.searchInput}
        />
      </div>

      {/* Quick filters */}
      {showFilters && (
        <div style={S.filterSection}>
          <FilterSelect label="Service" value={filters.service || ''} onChange={(v) => setFilter('service', v || undefined)}
            options={[{ value: '', label: 'All' }, ...services.map(s => ({ value: s, label: s }))]} />
          <FilterSelect label="Method" value={filters.method || ''} onChange={(v) => setFilter('method', v || undefined)}
            options={[{ value: '', label: 'All' }, ...['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD'].map(m => ({ value: m, label: m }))]} />
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '4px 0' }}>
            <label style={{ ...S.filterLabel, display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer' }}>
              <input type="checkbox" checked={!!filters.error} onChange={(e) => setFilter('error', (e.target as HTMLInputElement).checked || undefined)} />
              Errors only
            </label>
            <button
              onClick={() => setShowAdvanced(!showAdvanced)}
              style={{ ...S.clearBtn, color: showAdvanced ? 'var(--text-accent)' : 'var(--text-tertiary)' }}
            >
              {showAdvanced ? 'Hide advanced' : 'Advanced'}
            </button>
          </div>

          {/* Advanced filters */}
          {showAdvanced && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 4, borderTop: '1px solid var(--border-subtle)', paddingTop: 6 }}>
              <FilterInput label="Tenant" value={filters.tenant_id || ''} placeholder="tenant_id"
                onChange={(v) => setFilter('tenant_id', v || undefined)} />
              <FilterInput label="Org" value={filters.org_id || ''} placeholder="org_id"
                onChange={(v) => setFilter('org_id', v || undefined)} />
              <FilterInput label="User" value={filters.user_id || ''} placeholder="user_id"
                onChange={(v) => setFilter('user_id', v || undefined)} />
              <FilterInput label="Route" value={filters.path || ''} placeholder="/bff/"
                onChange={(v) => setFilter('path', v || undefined)} />
              <div style={S.filterRow}>
                <label style={S.filterLabel}>Latency</label>
                <input
                  type="number"
                  placeholder="min ms"
                  value={filters.min_latency_ms ?? ''}
                  onInput={(e) => {
                    const v = parseInt((e.target as HTMLInputElement).value);
                    setFilter('min_latency_ms', isNaN(v) ? undefined : v);
                  }}
                  style={{ ...S.filterInput, width: '50%' }}
                />
                <input
                  type="number"
                  placeholder="max ms"
                  value={filters.max_latency_ms ?? ''}
                  onInput={(e) => {
                    const v = parseInt((e.target as HTMLInputElement).value);
                    setFilter('max_latency_ms', isNaN(v) ? undefined : v);
                  }}
                  style={{ ...S.filterInput, width: '50%' }}
                />
              </div>
              <div style={S.filterRow}>
                <label style={S.filterLabel}>From</label>
                <input
                  type="datetime-local"
                  value={filters.from || ''}
                  onInput={(e) => setFilter('from', (e.target as HTMLInputElement).value || undefined)}
                  style={S.filterInput}
                />
              </div>
              <div style={S.filterRow}>
                <label style={S.filterLabel}>To</label>
                <input
                  type="datetime-local"
                  value={filters.to || ''}
                  onInput={(e) => setFilter('to', (e.target as HTMLInputElement).value || undefined)}
                  style={S.filterInput}
                />
              </div>
            </div>
          )}

          {hasActiveFilters && (
            <button onClick={clearFilters} style={{ ...S.clearBtn, marginTop: 4, textAlign: 'right' }}>Clear all filters</button>
          )}
        </div>
      )}

      {/* Live indicator */}
      <div style={S.liveBar}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <span style={{ ...S.liveDot, background: paused ? 'var(--text-tertiary)' : 'var(--success)' }} />
          <span style={{ fontSize: 11, fontWeight: 600, color: paused ? 'var(--text-tertiary)' : 'var(--success)' }}>
            {paused ? 'PAUSED' : 'LIVE'}
          </span>
          {paused && bufferRef.current.length > 0 && (
            <span style={{ fontSize: 10, color: 'var(--text-tertiary)' }}>({bufferRef.current.length} buffered)</span>
          )}
        </div>
        <button onClick={paused ? handleResume : () => setPaused(true)} style={S.pauseBtn}>
          {paused ? '\u25B6' : '\u23F8'}
        </button>
      </div>

      {/* Request list */}
      <div ref={listRef} style={S.list}>
        {filtered.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '32px 16px', color: 'var(--text-tertiary)', fontSize: 12 }}>
            {hasActiveFilters || searchText ? 'No matching requests' : 'No requests yet'}
          </div>
        ) : (
          filtered.map((r: any) => (
            <RequestRow
              key={r.id}
              req={r}
              selected={selectedRequestId === r.id}
              onClick={() => onSelectRequest(r)}
            />
          ))
        )}
      </div>
    </div>
  );
}

function RequestRow({ req, selected, onClick }: { req: any; selected: boolean; onClick: () => void }) {
  const status = req.status || req.status_code;
  const isError = status >= 400;
  const methodColor = METHOD_COLORS[req.method] || '#6B7280';

  return (
    <div
      onClick={onClick}
      style={{
        ...S.row,
        background: selected ? 'var(--bg-active)' : isError ? 'var(--error-50)' : 'transparent',
        borderLeft: selected ? '3px solid var(--brand-teal)' : '3px solid transparent',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 2 }}>
        <span style={{ ...S.methodBadge, background: `${methodColor}18`, color: methodColor }}>{req.method}</span>
        <span style={S.pathText}>{req.path || req.action}</span>
        <span style={{ marginLeft: 'auto', fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--text-tertiary)' }}>
          {fmtTime(req.timestamp)}
        </span>
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        <span style={S.serviceChip}>{req.service}</span>
        {req.action && req.action !== req.path && (
          <span style={{ fontSize: 10, color: 'var(--text-tertiary)' }}>{req.action}</span>
        )}
        <span style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 6 }}>
          <StatusBadge code={status} />
          <span style={{
            fontFamily: 'var(--font-mono)', fontSize: 10,
            color: isError ? 'var(--error)' : 'var(--text-tertiary)',
          }}>{fmtDuration(req.latency_ms)}</span>
        </span>
      </div>
    </div>
  );
}

function FilterSelect({ label, value, onChange, options }: {
  label: string; value: string; onChange: (v: string) => void;
  options: { value: string; label: string }[];
}) {
  return (
    <div style={S.filterRow}>
      <label style={S.filterLabel}>{label}</label>
      <select value={value} onChange={(e) => onChange((e.target as HTMLSelectElement).value)} style={S.filterSelect}>
        {options.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
      </select>
    </div>
  );
}

function FilterInput({ label, value, placeholder, onChange }: {
  label: string; value: string; placeholder: string; onChange: (v: string) => void;
}) {
  return (
    <div style={S.filterRow}>
      <label style={S.filterLabel}>{label}</label>
      <input
        type="text"
        placeholder={placeholder}
        value={value}
        onInput={(e) => onChange((e.target as HTMLInputElement).value)}
        style={S.filterInput}
      />
    </div>
  );
}

const S = {
  panel: {
    display: 'flex', flexDirection: 'column' as const, height: '100%', overflow: 'hidden',
  },
  header: {
    display: 'flex', alignItems: 'center', justifyContent: 'space-between',
    padding: '12px 14px 8px', flexShrink: 0,
  },
  title: { fontSize: 14, fontWeight: 700, color: 'var(--text-primary)' },
  count: {
    fontSize: 11, fontWeight: 600, color: 'var(--text-tertiary)',
    background: 'var(--bg-tertiary)', borderRadius: 10, padding: '1px 7px',
  },
  searchWrap: { padding: '0 14px 8px', flexShrink: 0 },
  searchInput: {
    width: '100%', padding: '6px 10px', fontSize: 12, border: '1px solid var(--border-default)',
    borderRadius: 6, outline: 'none', background: 'var(--bg-tertiary)', color: 'var(--text-primary)',
    boxSizing: 'border-box' as const,
  },
  filterSection: {
    padding: '0 14px 8px', borderBottom: '1px solid var(--border-subtle)',
    flexShrink: 0, display: 'flex', flexDirection: 'column' as const, gap: 4,
  },
  filterRow: { display: 'flex', alignItems: 'center', gap: 8 },
  filterLabel: { fontSize: 11, fontWeight: 500, color: 'var(--text-tertiary)', width: 50, flexShrink: 0 },
  filterSelect: {
    flex: 1, padding: '3px 6px', fontSize: 11, border: '1px solid var(--border-default)',
    borderRadius: 4, background: 'var(--bg-tertiary)', color: 'var(--text-primary)', outline: 'none',
  },
  filterInput: {
    flex: 1, padding: '3px 6px', fontSize: 11, border: '1px solid var(--border-default)',
    borderRadius: 4, outline: 'none', fontFamily: 'var(--font-mono)',
    background: 'var(--bg-tertiary)', color: 'var(--text-primary)',
  },
  clearBtn: {
    fontSize: 10, color: 'var(--text-accent)', background: 'none',
    border: 'none', cursor: 'pointer', fontWeight: 600,
  },
  saveBtn: {
    fontSize: 10, fontWeight: 600, color: 'var(--bg-primary)', background: 'var(--brand-teal)',
    border: 'none', borderRadius: 4, padding: '3px 8px', cursor: 'pointer',
  },
  viewMenu: {
    padding: '8px 14px', borderBottom: '1px solid var(--border-subtle)',
    flexShrink: 0, background: 'var(--bg-tertiary)',
  },
  viewRow: {
    display: 'flex', alignItems: 'center', gap: 6, padding: '3px 0',
  },
  liveBar: {
    display: 'flex', alignItems: 'center', justifyContent: 'space-between',
    padding: '6px 14px', borderBottom: '1px solid var(--border-subtle)', flexShrink: 0,
  },
  liveDot: {
    width: 7, height: 7, borderRadius: '50%', flexShrink: 0,
    animation: 'pulse 2s ease-in-out infinite',
  },
  pauseBtn: {
    width: 24, height: 24, display: 'flex', alignItems: 'center', justifyContent: 'center',
    background: 'none', border: '1px solid var(--border-default)', borderRadius: 4,
    cursor: 'pointer', fontSize: 10, color: 'var(--text-secondary)',
  },
  list: {
    flex: 1, overflowY: 'auto' as const, overflowX: 'hidden' as const,
  },
  row: {
    padding: '8px 14px', cursor: 'pointer', borderBottom: '1px solid var(--border-subtle)',
    transition: 'background 0.1s', fontSize: 12,
  },
  methodBadge: {
    fontSize: 9, fontWeight: 700, padding: '1px 5px', borderRadius: 3,
    fontFamily: 'var(--font-mono)', letterSpacing: 0.3,
  },
  pathText: {
    fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--text-primary)',
    overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' as const, flex: 1,
  },
  serviceChip: {
    fontSize: 10, fontWeight: 500, color: 'var(--text-secondary)', background: 'var(--bg-tertiary)',
    padding: '1px 5px', borderRadius: 3,
  },
  iconBtn: {
    background: 'none', border: 'none', cursor: 'pointer', fontSize: 14,
    padding: 4, position: 'relative' as const, color: 'var(--text-tertiary)',
  },
  filterBadge: {
    position: 'absolute' as const, top: 0, right: -2,
    fontSize: 8, fontWeight: 700, color: 'var(--bg-primary)', background: 'var(--brand-teal)',
    width: 14, height: 14, borderRadius: '50%', display: 'flex',
    alignItems: 'center', justifyContent: 'center',
  },
};
