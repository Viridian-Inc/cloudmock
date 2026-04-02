import { useState, useEffect, useMemo, useRef, useCallback } from 'preact/hooks';
import { api, getAdminBase } from '../../lib/api';
import './lambda.css';

interface LogEntry {
  timestamp?: string;
  time?: string;
  function_name?: string;
  request_id?: string;
  message?: string;
  stream?: string;
}

function formatTime(ts: string | undefined): string {
  if (!ts) return '--:--:--';
  const d = new Date(ts);
  if (isNaN(d.getTime())) return '--:--:--';
  return d.toTimeString().slice(0, 8);
}

export function LambdaView() {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [functions, setFunctions] = useState<string[]>([]);
  const [selected, setSelected] = useState('');
  const [search, setSearch] = useState('');
  const [sseConnected, setSseConnected] = useState(false);
  const tableRef = useRef<HTMLDivElement>(null);
  const sseRef = useRef<EventSource | null>(null);

  const loadLogs = useCallback(
    (fn: string) => {
      const params = new URLSearchParams();
      if (fn) params.set('function', fn);
      params.set('limit', '100');
      api<LogEntry[]>(`/api/lambda/logs?${params.toString()}`)
        .then((r) => {
          const entries = r || [];
          setLogs(entries);
          const fns = [
            ...new Set(
              entries.map((l) => l.function_name).filter(Boolean) as string[],
            ),
          ].sort();
          setFunctions(fns);
        })
        .catch(() => {});
    },
    [],
  );

  // Initial load
  useEffect(() => {
    loadLogs('');
  }, [loadLogs]);

  // SSE streaming
  useEffect(() => {
    const base = getAdminBase();
    const url = `${base}/api/lambda/logs/stream`;
    const es = new EventSource(url);
    sseRef.current = es;

    es.onopen = () => setSseConnected(true);

    es.onmessage = (ev) => {
      try {
        const data = JSON.parse(ev.data) as LogEntry;
        setLogs((prev) => [data, ...prev].slice(0, 500));
        if (data.function_name) {
          setFunctions((prev) => {
            if (prev.includes(data.function_name!)) return prev;
            return [...prev, data.function_name!].sort();
          });
        }
      } catch {
        // ignore parse errors
      }
    };

    es.onerror = () => setSseConnected(false);

    return () => {
      es.close();
      sseRef.current = null;
    };
  }, []);

  // Auto-scroll to top (newest) when new logs arrive
  useEffect(() => {
    if (tableRef.current) {
      tableRef.current.scrollTop = 0;
    }
  }, [logs.length]);

  function selectFunction(fn: string) {
    setSelected(fn);
    loadLogs(fn);
  }

  const filtered = useMemo(() => {
    let result = logs;

    // Filter by selected function
    if (selected) {
      result = result.filter((l) => l.function_name === selected);
    }

    // Filter by search text
    if (search) {
      const q = search.toLowerCase();
      result = result.filter(
        (l) =>
          (l.message || '').toLowerCase().includes(q) ||
          (l.function_name || '').toLowerCase().includes(q) ||
          (l.request_id || '').toLowerCase().includes(q),
      );
    }

    return result;
  }, [logs, search, selected]);

  return (
    <div class="lambda-view">
      <div class="lambda-header">
        <h2 class="lambda-title">Lambda Logs</h2>
        <p class="lambda-desc">
          <span
            class={`lambda-sse-dot ${sseConnected ? 'lambda-sse-connected' : 'lambda-sse-disconnected'}`}
          />
          Function execution logs and real-time streaming
        </p>
      </div>

      <div class="lambda-body">
        {/* Function sidebar */}
        <div class="lambda-sidebar">
          <div class="lambda-sidebar-header">Functions</div>
          <div class="lambda-sidebar-list">
            <div
              class={`lambda-fn-item ${!selected ? 'lambda-fn-item-active' : ''}`}
              onClick={() => selectFunction('')}
            >
              All Functions
            </div>
            {functions.map((fn) => (
              <div
                key={fn}
                class={`lambda-fn-item ${selected === fn ? 'lambda-fn-item-active' : ''}`}
                onClick={() => selectFunction(fn)}
                title={fn}
              >
                {fn}
              </div>
            ))}
          </div>
        </div>

        {/* Main log area */}
        <div class="lambda-main">
          <div class="lambda-toolbar">
            <input
              class="lambda-search"
              placeholder="Search logs..."
              value={search}
              onInput={(e) =>
                setSearch((e.target as HTMLInputElement).value)
              }
            />
            <button
              class="lambda-refresh-btn"
              onClick={() => loadLogs(selected)}
            >
              Refresh
            </button>
          </div>

          <div class="lambda-table-wrap" ref={tableRef}>
            <table class="lambda-table">
              <thead>
                <tr>
                  <th style={{ width: '90px' }}>Time</th>
                  <th style={{ width: '170px' }}>Function</th>
                  <th style={{ width: '110px' }}>Request ID</th>
                  <th>Message</th>
                </tr>
              </thead>
              <tbody>
                {filtered.length === 0 ? (
                  <tr>
                    <td colSpan={4}>
                      <div class="lambda-empty">No logs</div>
                    </td>
                  </tr>
                ) : (
                  filtered.map((l, i) => (
                    <tr
                      key={`${l.request_id}-${i}`}
                      class={l.stream === 'stderr' ? 'lambda-stderr' : ''}
                    >
                      <td class="lambda-mono">
                        {formatTime(l.timestamp || l.time)}
                      </td>
                      <td
                        class="lambda-truncate"
                        style={{ maxWidth: '170px' }}
                        title={l.function_name || ''}
                      >
                        {l.function_name || ''}
                      </td>
                      <td
                        class="lambda-mono lambda-truncate"
                        style={{ maxWidth: '110px' }}
                        title={l.request_id || ''}
                      >
                        {l.request_id || ''}
                      </td>
                      <td class="lambda-msg">{l.message || ''}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}
