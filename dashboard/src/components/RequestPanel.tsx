import { useState, useEffect, useMemo, useRef, useCallback } from 'preact/hooks';
import { api } from '../api';
import { StatusBadge } from './StatusBadge';
import { fmtTime, fmtDuration } from '../utils';
import type { SSEState } from '../hooks/useSSE';

interface RequestPanelProps {
  sse: SSEState;
  onSelectRequest: (req: any) => void;
  selectedRequestId?: string;
}

interface Filters {
  text: string;
  service: string;
  status: string;
  method: string;
  path: string;
  callerId: string;
  errorOnly: boolean;
  showInfra: boolean;
}

const EMPTY_FILTERS: Filters = {
  text: '', service: '', status: '', method: '', path: '', callerId: '', errorOnly: false, showInfra: true,
};

const METHOD_COLORS: Record<string, string> = {
  GET: '#3B82F6', POST: '#10B981', PUT: '#F59E0B', DELETE: '#EF4444',
  PATCH: '#8B5CF6', HEAD: '#6B7280', OPTIONS: '#6B7280',
};

export function RequestPanel({ sse, onSelectRequest, selectedRequestId }: RequestPanelProps) {
  const [requests, setRequests] = useState<any[]>([]);
  const [services, setServices] = useState<string[]>([]);
  const [filters, setFilters] = useState<Filters>(EMPTY_FILTERS);
  const [paused, setPaused] = useState(false);
  const [showFilters, setShowFilters] = useState(false);
  const bufferRef = useRef<any[]>([]);
  const listRef = useRef<HTMLDivElement>(null);

  // Load initial data
  useEffect(() => {
    api('/api/requests?limit=100').then(setRequests).catch(() => {});
    api('/api/services').then((s: any[]) => setServices(s.map((x: any) => x.name).sort())).catch(() => {});
  }, []);

  // SSE live tail
  useEffect(() => {
    if (sse.events.length === 0) return;
    const latest = sse.events[0];
    if (latest?.type === 'request' && latest.data) {
      if (paused) {
        bufferRef.current = [latest.data, ...bufferRef.current].slice(0, 200);
      } else {
        setRequests(prev => [latest.data, ...prev].slice(0, 200));
      }
    }
  }, [sse.events, paused]);

  // Resume: flush buffer
  const handleResume = useCallback(() => {
    setPaused(false);
    if (bufferRef.current.length > 0) {
      setRequests(prev => [...bufferRef.current, ...prev].slice(0, 200));
      bufferRef.current = [];
    }
  }, []);

  const hasActiveFilters = filters.text || filters.service || filters.status || filters.method || filters.path || filters.callerId || filters.errorOnly;
  const activeFilterCount = [filters.service, filters.status, filters.method, filters.path, filters.callerId, filters.errorOnly ? 'x' : ''].filter(Boolean).length;

  const filtered = useMemo(() => {
    return requests.filter((r: any) => {
      if (!filters.showInfra && r.level === 'infra') return false;
      if (filters.service && r.service !== filters.service) return false;
      if (filters.status) {
        const s = String(r.status || r.status_code);
        if (filters.status === '2xx' && !s.startsWith('2')) return false;
        if (filters.status === '4xx' && !s.startsWith('4')) return false;
        if (filters.status === '5xx' && !s.startsWith('5')) return false;
      }
      if (filters.method && r.method !== filters.method) return false;
      if (filters.path && !(r.path || '').startsWith(filters.path)) return false;
      if (filters.callerId && r.caller_id !== filters.callerId) return false;
      if (filters.errorOnly && (r.status || r.status_code) < 400) return false;
      if (filters.text) {
        const q = filters.text.toLowerCase();
        const hay = `${r.service} ${r.action} ${r.method} ${r.path} ${r.id || ''}`.toLowerCase();
        if (!hay.includes(q)) return false;
      }
      return true;
    });
  }, [requests, filters]);

  const setFilter = (key: keyof Filters, value: any) => {
    setFilters(prev => ({ ...prev, [key]: value }));
  };

  return (
    <div style={S.panel}>
      {/* Header */}
      <div style={S.header}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={S.title}>Requests</span>
          <span style={S.count}>{filtered.length}</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <button
            onClick={() => setShowFilters(!showFilters)}
            style={{
              ...S.iconBtn,
              color: showFilters || hasActiveFilters ? 'var(--brand-blue, #097FF5)' : 'var(--text-tertiary)',
            }}
            title="Filters"
          >
            {'\u2630'}
            {activeFilterCount > 0 && <span style={S.filterBadge}>{activeFilterCount}</span>}
          </button>
        </div>
      </div>

      {/* Search */}
      <div style={S.searchWrap}>
        <input
          type="text"
          placeholder="Search requests..."
          value={filters.text}
          onInput={(e) => setFilter('text', (e.target as HTMLInputElement).value)}
          style={S.searchInput}
        />
      </div>

      {/* Filters */}
      {showFilters && (
        <div style={S.filterSection}>
          <Select label="Service" value={filters.service} onChange={(v) => setFilter('service', v)}
            options={[{ value: '', label: 'All' }, ...services.map(s => ({ value: s, label: s }))]} />
          <Select label="Status" value={filters.status} onChange={(v) => setFilter('status', v)}
            options={[{ value: '', label: 'All' }, { value: '2xx', label: '2xx' }, { value: '4xx', label: '4xx' }, { value: '5xx', label: '5xx' }]} />
          <Select label="Method" value={filters.method} onChange={(v) => setFilter('method', v)}
            options={[{ value: '', label: 'All' }, ...['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD'].map(m => ({ value: m, label: m }))]} />
          <div style={S.filterRow}>
            <label style={S.filterLabel}>Route</label>
            <input type="text" placeholder="/bff/" value={filters.path} onInput={(e) => setFilter('path', (e.target as HTMLInputElement).value)} style={S.filterInput} />
          </div>
          <div style={S.filterRow}>
            <label style={S.filterLabel}>Caller</label>
            <input type="text" placeholder="caller ID" value={filters.callerId} onInput={(e) => setFilter('callerId', (e.target as HTMLInputElement).value)} style={S.filterInput} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '4px 0' }}>
            <label style={{ ...S.filterLabel, display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer' }}>
              <input type="checkbox" checked={filters.errorOnly} onChange={(e) => setFilter('errorOnly', (e.target as HTMLInputElement).checked)} />
              Errors only
            </label>
            {hasActiveFilters && (
              <button onClick={() => setFilters(EMPTY_FILTERS)} style={S.clearBtn}>Clear all</button>
            )}
          </div>
          <div style={{ display: 'flex', alignItems: 'center', padding: '4px 0' }}>
            <label style={{ ...S.filterLabel, display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer', width: 'auto' }}>
              <input type="checkbox" checked={filters.showInfra} onChange={(e) => setFilter('showInfra', (e.target as HTMLInputElement).checked)} />
              Show infrastructure traffic
            </label>
          </div>
        </div>
      )}

      {/* Live indicator */}
      <div style={S.liveBar}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <span style={{ ...S.liveDot, background: paused ? 'var(--border-default)' : '#10B981' }} />
          <span style={{ fontSize: 11, fontWeight: 600, color: paused ? 'var(--text-tertiary)' : '#10B981' }}>
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
            {hasActiveFilters ? 'No matching requests' : 'No requests yet'}
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
  const isInfra = req.level === 'infra';
  const methodColor = METHOD_COLORS[req.method] || '#6B7280';

  return (
    <div
      onClick={onClick}
      style={{
        ...S.row,
        background: selected ? 'var(--bg-active)' : isError ? '#FEF2F230' : 'transparent',
        borderLeft: selected ? '3px solid var(--brand-blue, #097FF5)' : '3px solid transparent',
        opacity: isInfra ? 0.6 : 1,
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 2 }}>
        <span style={{ ...S.methodBadge, background: `${methodColor}18`, color: methodColor }}>{req.method}</span>
        {isInfra && <span style={S.infraBadge}>infra</span>}
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
            color: isError ? '#EF4444' : 'var(--text-tertiary)',
          }}>{fmtDuration(req.latency_ms)}</span>
        </span>
      </div>
    </div>
  );
}

function Select({ label, value, onChange, options }: {
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

const S = {
  panel: {
    width: 300, flexShrink: 0, borderRight: '1px solid var(--border-default)',
    display: 'flex', flexDirection: 'column' as const, background: 'var(--bg-secondary)',
    height: '100%', overflow: 'hidden',
  },
  header: {
    display: 'flex', alignItems: 'center', justifyContent: 'space-between',
    padding: '12px 14px 8px', flexShrink: 0,
  },
  title: { fontSize: 14, fontWeight: 700, color: 'var(--text-primary)' },
  count: {
    fontSize: 11, fontWeight: 600, color: 'var(--text-tertiary)',
    background: 'var(--bg-secondary)', borderRadius: 10, padding: '1px 7px',
  },
  searchWrap: { padding: '0 14px 8px', flexShrink: 0 },
  searchInput: {
    width: '100%', padding: '6px 10px', fontSize: 12, border: '1px solid var(--border-default)',
    borderRadius: 6, outline: 'none', background: 'var(--bg-primary)',
    boxSizing: 'border-box' as const,
  },
  filterSection: {
    padding: '0 14px 8px', borderBottom: '1px solid var(--bg-secondary)',
    flexShrink: 0, display: 'flex', flexDirection: 'column' as const, gap: 4,
  },
  filterRow: { display: 'flex', alignItems: 'center', gap: 8 },
  filterLabel: { fontSize: 11, fontWeight: 500, color: 'var(--text-tertiary)', width: 50, flexShrink: 0 },
  filterSelect: {
    flex: 1, padding: '3px 6px', fontSize: 11, border: '1px solid var(--border-default)',
    borderRadius: 4, background: 'var(--bg-tertiary)', outline: 'none',
  },
  filterInput: {
    flex: 1, padding: '3px 6px', fontSize: 11, border: '1px solid var(--border-default)',
    borderRadius: 4, outline: 'none', fontFamily: 'var(--font-mono)',
  },
  clearBtn: {
    fontSize: 10, color: 'var(--brand-blue, #097FF5)', background: 'none',
    border: 'none', cursor: 'pointer', fontWeight: 600,
  },
  liveBar: {
    display: 'flex', alignItems: 'center', justifyContent: 'space-between',
    padding: '6px 14px', borderBottom: '1px solid var(--bg-secondary)', flexShrink: 0,
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
    padding: '8px 14px', cursor: 'pointer', borderBottom: '1px solid var(--bg-secondary)',
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
    fontSize: 10, fontWeight: 500, color: 'var(--text-secondary)', background: 'var(--bg-secondary)',
    padding: '1px 5px', borderRadius: 3,
  },
  iconBtn: {
    background: 'none', border: 'none', cursor: 'pointer', fontSize: 14,
    padding: 4, position: 'relative' as const,
  },
  filterBadge: {
    position: 'absolute' as const, top: 0, right: -2,
    fontSize: 8, fontWeight: 700, color: 'white', background: 'var(--brand-blue, #097FF5)',
    width: 14, height: 14, borderRadius: '50%', display: 'flex',
    alignItems: 'center', justifyContent: 'center',
  },
  infraBadge: {
    fontSize: 8, fontWeight: 600, color: 'var(--text-tertiary)', background: 'var(--bg-tertiary)',
    padding: '1px 4px', borderRadius: 3, border: '1px solid var(--border-default)',
    letterSpacing: 0.3, textTransform: 'uppercase' as const, flexShrink: 0,
  },
};
