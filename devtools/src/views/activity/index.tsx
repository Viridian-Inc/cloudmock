import { useState, useEffect, useCallback, useRef, useMemo } from 'preact/hooks';
import { SplitPanel } from '../../components/panels/split-panel';
import { api, getAdminBase } from '../../lib/api';
import type { RequestEvent } from '../../lib/types';
import { EventList } from './event-list';
import { EventDetail } from './event-detail';
import './activity.css';

interface Filters {
  service: string;
  status: string;
}

/** Read initial service filter from URL hash: #service=dynamodb */
function getInitialServiceFilter(): string {
  const hash = window.location.hash;
  const match = hash.match(/[#&]service=([^&]*)/);
  return match ? decodeURIComponent(match[1]) : '';
}

export function ActivityView() {
  const [requests, setRequests] = useState<RequestEvent[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [paused, setPaused] = useState(false);
  const [search, setSearch] = useState('');
  const [filters, setFilters] = useState<Filters>(() => ({
    service: getInitialServiceFilter(),
    status: '',
  }));
  const [connected, setConnected] = useState(false);
  const [dataSource, setDataSource] = useState<'loading' | 'sse' | 'polling'>('loading');
  const [pinned, setPinned] = useState(false);

  // Refs to avoid recreating EventSource when paused/pinned change
  const pausedRef = useRef(paused);
  useEffect(() => { pausedRef.current = paused; }, [paused]);
  const pinnedRef = useRef(pinned);
  useEffect(() => { pinnedRef.current = pinned; }, [pinned]);

  // === DATA SOURCE 1: SSE stream (same pattern as working cloudmock dashboard) ===
  useEffect(() => {
    // SSE connection — use same pattern as cloudmock dashboard:
    // In dev (port 1420), connect to admin port 4599 directly
    // In production (port 4500), use relative URL (same-origin)
    const port = window.location.port;
    const adminBase = port === '1420'
      ? `${window.location.protocol}//${window.location.hostname}:4599`
      : '';
    const es = new EventSource(`${adminBase}/api/stream`);

    es.onopen = () => {
      setConnected(true);
      setDataSource('sse');
    };

    es.onerror = () => {
      setConnected(false);
    };

    es.onmessage = (e) => {
      if (pausedRef.current) return;
      try {
        const event = JSON.parse(e.data);
        if (event.type === 'request' && event.data) {
          const d = event.data;
          const req: RequestEvent = {
            id: d.id || `sse-${Date.now()}`,
            service: d.service || '',
            action: d.action || '',
            method: d.method || '',
            path: d.path || '',
            status_code: d.status_code || 200,
            latency_ms: d.latency_ms || (d.latency_ns ? d.latency_ns / 1_000_000 : 0),
            timestamp: d.timestamp || new Date().toISOString(),
            request_headers: d.request_headers,
            response_body: d.response_body,
          };
          setRequests((prev) => {
            if (pinnedRef.current) return prev;
            if (prev.some((r) => r.id === req.id)) return prev;
            return [req, ...prev].slice(0, 2000);
          });
        }
      } catch (e) { console.warn('[Activity] SSE parse error:', e); }
    };

    return () => es.close();
  }, []);

  // === DATA SOURCE 2: Poll /api/requests?level=all every 3s ===
  // No deps — runs once on mount, uses refs for mutable state
  useEffect(() => {
    let mounted = true;

    async function poll() {
      if (pausedRef.current || pinnedRef.current) return;
      try {
        const adminBase = getAdminBase();
        const res = await fetch(`${adminBase}/api/requests?level=all&limit=200`, {
          headers: { 'Content-Type': 'application/json' },
        });
        if (!res.ok || !mounted) return;

        const reqs: any[] = await res.json();
        if (!Array.isArray(reqs) || reqs.length === 0 || !mounted) return;

        const events: RequestEvent[] = reqs.map((r) => ({
          id: r.ID || r.id || r.TraceID || `${r.Timestamp}-${r.Service}-${r.Path}`,
          trace_id: r.TraceID || r.trace_id,
          service: r.Service || r.service || '',
          action: r.Action || r.action || '',
          method: r.Method || r.method || '',
          path: r.Path || r.path || '',
          status_code: r.StatusCode || r.status_code || 200,
          latency_ms: r.LatencyMs || r.latency_ms || 0,
          timestamp: r.Timestamp || r.timestamp || new Date().toISOString(),
          request_headers: r.RequestHeaders || r.request_headers,
          response_body: r.ResponseBody || r.response_body,
          source: r.Level || r.level || 'infra',
        }));

        setDataSource('polling');
        setRequests((prev) => {
          const existingIds = new Set(prev.map((r) => r.id));
          const unique = events.filter((r) => !existingIds.has(r.id));
          if (unique.length === 0) return prev;
          return [...unique, ...prev].slice(0, 2000);
        });
      } catch (e) { console.warn('[Activity] Poll error:', e); }
    }

    poll();
    const interval = setInterval(poll, 3000);
    return () => { mounted = false; clearInterval(interval); };
  }, []); // empty deps — uses refs for mutable state

  // Listen for navigate-activity events from topology node inspector
  useEffect(() => {
    const handler = (e: Event) => {
      const detail = (e as CustomEvent).detail;
      if (detail?.service) {
        setFilters((f) => ({ ...f, service: detail.service }));
      }
    };
    document.addEventListener('neureaux:navigate-activity', handler);
    return () => document.removeEventListener('neureaux:navigate-activity', handler);
  }, []);

  // Sync service filter to URL hash
  useEffect(() => {
    if (filters.service) {
      window.location.hash = `service=${encodeURIComponent(filters.service)}`;
    } else {
      if (window.location.hash.includes('service=')) {
        window.location.hash = '';
      }
    }
  }, [filters.service]);

  const handleTogglePause = useCallback(() => setPaused((p) => !p), []);
  const handleTogglePin = useCallback(() => setPinned((p) => !p), []);
  const handleSearchChange = useCallback((v: string) => setSearch(v), []);
  const handleFilterChange = useCallback(
    (key: keyof Filters, value: string) =>
      setFilters((f) => ({ ...f, [key]: value })),
    [],
  );

  // Source filter
  const [sourceFilter, setSourceFilter] = useState<Set<string>>(new Set());
  const uniqueSources = useMemo(() => {
    const counts = new Map<string, number>();
    for (const r of requests) {
      const src = r.service || 'unknown';
      counts.set(src, (counts.get(src) || 0) + 1);
    }
    return counts;
  }, [requests]);

  const handleToggleSource = useCallback((source: string) => {
    setSourceFilter((prev) => {
      const next = new Set(prev);
      if (next.has(source)) next.delete(source); else next.add(source);
      return next;
    });
  }, []);

  const handleResetSourceFilter = useCallback(() => setSourceFilter(new Set()), []);

  const handleClear = useCallback(() => {
    setRequests([]);
    setSelectedId(null);
    setSourceFilter(new Set());
  }, []);

  const handleExportHAR = useCallback(() => {
    const har = {
      log: {
        version: '1.2',
        creator: { name: 'neureaux-devtools', version: '0.1.0' },
        entries: requests.map((r) => ({
          startedDateTime: r.timestamp,
          time: r.latency_ms,
          request: { method: r.method, url: r.path, headers: Object.entries(r.request_headers || {}).map(([n, v]) => ({ name: n, value: v })) },
          response: { status: r.status_code, statusText: '', headers: [] },
          timings: { wait: r.latency_ms },
        })),
      },
    };
    const blob = new Blob([JSON.stringify(har, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'neureaux-devtools-export.har';
    a.click();
    URL.revokeObjectURL(url);
  }, [requests]);

  // Apply filters
  const filtered = requests.filter((r) => {
    if (sourceFilter.size > 0 && !sourceFilter.has(r.service || 'unknown')) return false;
    if (filters.service && r.service !== filters.service) return false;
    if (filters.status) {
      const low = parseInt(filters.status.charAt(0)) * 100;
      if (r.status_code < low || r.status_code >= low + 100) return false;
    }
    if (search) {
      const q = search.toLowerCase();
      if (!(r.action || '').toLowerCase().includes(q) &&
          !(r.path || '').toLowerCase().includes(q) &&
          !(r.service || '').toLowerCase().includes(q)) return false;
    }
    return true;
  });

  const selectedEvent = selectedId ? requests.find((r) => r.id === selectedId) ?? null : null;
  const serviceNames = [...new Set(requests.map((r) => r.service))].filter(Boolean).sort();

  return (
    <div class="activity-view">
      {/* Active service filter banner */}
      {filters.service && (
        <div class="activity-filter-banner">
          <span class="activity-filter-banner-text">
            Filtered to: <strong>{filters.service}</strong>
          </span>
          <button
            class="btn btn-ghost activity-filter-clear-btn"
            onClick={() => setFilters((f) => ({ ...f, service: '' }))}
          >
            Clear filter
          </button>
        </div>
      )}
      {/* Pin / buffer status bar */}
      <div class="activity-buffer-bar">
        <span class="activity-buffer-count">{requests.length} events buffered</span>
        <button
          class={`btn btn-ghost activity-pin-btn ${pinned ? 'active' : ''}`}
          onClick={handleTogglePin}
          title={pinned ? 'Unpin: allow new events to stream in' : 'Pin: freeze the current event list'}
        >
          {pinned ? '\uD83D\uDCCC Pinned' : '\uD83D\uDCCC Pin'}
        </button>
      </div>
      <SplitPanel
        initialSplit={60}
        direction="horizontal"
        minSize={200}
        left={
          <EventList
            events={filtered}
            selectedId={selectedId}
            onSelect={setSelectedId}
            paused={paused}
            onTogglePause={handleTogglePause}
            search={search}
            onSearchChange={handleSearchChange}
            filters={filters}
            onFilterChange={handleFilterChange}
            onClear={handleClear}
            connected={connected}
            serviceNames={serviceNames}
            dataSource={dataSource === 'sse' ? 'sse' : 'polling'}
            sourceFilter={sourceFilter}
            onToggleSource={handleToggleSource}
            onResetSourceFilter={handleResetSourceFilter}
            sourceCounts={uniqueSources}
            onExportHAR={handleExportHAR}
          />
        }
        right={<EventDetail event={selectedEvent} allEvents={requests} />}
      />
    </div>
  );
}
