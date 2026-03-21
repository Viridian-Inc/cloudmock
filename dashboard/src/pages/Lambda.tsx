import { useState, useEffect, useMemo } from 'preact/hooks';
import { api } from '../api';
import { SSEState } from '../hooks/useSSE';
import { RefreshIcon } from '../components/Icons';
import { fmtTime } from '../utils';

interface LambdaPageProps {
  sse: SSEState;
}

export function LambdaPage({ sse }: LambdaPageProps) {
  const [logs, setLogs] = useState<any[]>([]);
  const [functions, setFunctions] = useState<string[]>([]);
  const [selected, setSelected] = useState('');
  const [search, setSearch] = useState('');

  function loadLogs(fn: string) {
    const params = new URLSearchParams();
    if (fn) params.set('function', fn);
    params.set('limit', '100');
    api(`/api/lambda/logs?${params.toString()}`).then((r: any) => {
      setLogs(r || []);
      const fns = [...new Set((r || []).map((l: any) => l.function_name).filter(Boolean))] as string[];
      setFunctions(fns.sort());
    }).catch(() => {});
  }

  useEffect(() => { loadLogs(''); }, []);

  useEffect(() => {
    if (sse.events.length === 0) return;
    const latest = sse.events[0];
    if (latest && latest.type === 'lambda_log' && latest.data) {
      setLogs(prev => [latest.data, ...prev].slice(0, 500));
    }
  }, [sse.events]);

  function selectFunction(fn: string) {
    setSelected(fn);
    loadLogs(fn);
  }

  const filtered = useMemo(() => {
    if (!search) return logs;
    const q = search.toLowerCase();
    return logs.filter((l: any) => (l.message || '').toLowerCase().includes(q) || (l.function_name || '').toLowerCase().includes(q));
  }, [logs, search]);

  return (
    <div>
      <div class="mb-6">
        <h1 class="page-title">Lambda Logs</h1>
        <p class="page-desc">Function execution logs and metrics</p>
      </div>

      <div class="flex gap-4" style="height:calc(100vh - var(--header-height) - 120px)">
        <div style="width:220px;flex-shrink:0">
          <div class="card" style="height:100%">
            <div class="card-header" style="padding:12px 16px">
              <span style="font-weight:600;font-size:14px">Functions</span>
            </div>
            <div class="card-body" style="padding:4px 8px;overflow-y:auto" id="lambda-filter">
              <div
                class={`nav-item ${!selected ? 'active' : ''}`}
                style="border-radius:var(--radius-md);border-left:none;padding:8px 12px"
                onClick={() => selectFunction('')}
              >
                All Functions
              </div>
              {functions.map(fn => (
                <div
                  class={`nav-item ${selected === fn ? 'active' : ''}`}
                  style="border-radius:var(--radius-md);border-left:none;padding:8px 12px"
                  onClick={() => selectFunction(fn)}
                >
                  <span class="truncate">{fn}</span>
                </div>
              ))}
            </div>
          </div>
        </div>

        <div style="flex:1;overflow:hidden;display:flex;flex-direction:column">
          <div class="filters-bar">
            <input
              class="input input-search"
              placeholder="Search logs..."
              style="flex:1"
              value={search}
              onInput={(e) => setSearch((e.target as HTMLInputElement).value)}
            />
            <button class="btn btn-ghost btn-sm" onClick={() => loadLogs(selected)}>
              <RefreshIcon /> Refresh
            </button>
          </div>
          <div class="card" style="flex:1;overflow:hidden;display:flex;flex-direction:column">
            <div class="table-wrap" style="flex:1;overflow-y:auto">
              <table id="lambda-table">
                <thead>
                  <tr>
                    <th style="width:100px">Time</th>
                    <th style="width:180px">Function</th>
                    <th style="width:120px">Request ID</th>
                    <th>Message</th>
                  </tr>
                </thead>
                <tbody id="lambda-tbody">
                  {filtered.length === 0 ? (
                    <tr><td colSpan={4} class="empty-state">No logs</td></tr>
                  ) : filtered.map((l: any) => (
                    <tr class={l.stream === 'stderr' ? 'stderr' : ''}>
                      <td class="font-mono text-sm">{fmtTime(l.timestamp || l.time)}</td>
                      <td class="truncate" style="max-width:180px">{l.function_name || ''}</td>
                      <td class="font-mono text-sm truncate" style="max-width:120px">{l.request_id || ''}</td>
                      <td class={`font-mono text-sm ${l.stream === 'stderr' ? 'stderr' : ''}`} style="white-space:pre-wrap">{l.message || ''}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
