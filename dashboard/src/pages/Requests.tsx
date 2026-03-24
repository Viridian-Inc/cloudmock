import { useState, useEffect, useMemo } from 'preact/hooks';
import { api } from '../api';
import { SSEState } from '../hooks/useSSE';
import { StatusBadge, statusClass } from '../components/StatusBadge';
import { JsonView } from '../components/JsonView';
import { Drawer } from '../components/Drawer';
import { ExpandIcon, RefreshIcon, PlayIcon, CopyIcon } from '../components/Icons';
import { fmtTime, fmtDuration, copyToClipboard } from '../utils';

interface RequestsPageProps {
  sse: SSEState;
  showToast: (msg: string) => void;
}

type DetailTab = 'overview' | 'request' | 'response' | 'timing';

export function RequestsPage({ sse, showToast }: RequestsPageProps) {
  const [requests, setRequests] = useState<any[]>([]);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [drawer, setDrawer] = useState<any>(null);
  const [svcFilter, setSvcFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [textFilter, setTextFilter] = useState('');
  const [services, setServices] = useState<string[]>([]);
  const [detailTab, setDetailTab] = useState<DetailTab>('overview');

  useEffect(() => {
    api('/api/requests?limit=200').then(setRequests).catch(() => {});
    api('/api/services').then((s: any[]) => setServices(s.map((x: any) => x.name).sort())).catch(() => {});
  }, []);

  useEffect(() => {
    if (sse.events.length === 0) return;
    const latest = sse.events[0];
    if (latest && latest.type === 'request' && latest.data) {
      setRequests(prev => [latest.data, ...prev].slice(0, 500));
    }
  }, [sse.events]);

  const filtered = useMemo(() => {
    return requests.filter((r: any) => {
      if (svcFilter && r.service !== svcFilter) return false;
      if (statusFilter) {
        const s = String(r.status);
        if (statusFilter === '2xx' && !s.startsWith('2')) return false;
        if (statusFilter === '4xx' && !s.startsWith('4')) return false;
        if (statusFilter === '5xx' && !s.startsWith('5')) return false;
      }
      if (textFilter) {
        const q = textFilter.toLowerCase();
        const haystack = `${r.service} ${r.action} ${r.method} ${r.id || ''}`.toLowerCase();
        if (!haystack.includes(q)) return false;
      }
      return true;
    });
  }, [requests, svcFilter, statusFilter, textFilter]);

  function toggleExpand(id: string) {
    setExpanded(prev => prev === id ? null : id);
  }

  function openDrawer(e: Event, req: any) {
    e.stopPropagation();
    setDrawer(req);
  }

  function replayRequest(id: string) {
    api(`/api/requests/${id}/replay`, { method: 'POST' })
      .then(() => showToast('Request replayed'))
      .catch(() => showToast('Replay failed'));
  }

  function renderDetail(req: any, tab: DetailTab) {
    if (!req) return null;
    switch (tab) {
      case 'request':
        return (
          <div>
            <div class="flex items-center justify-between mb-4">
              <span class="section-title" style="margin:0">Request Body</span>
              <button class="copy-btn" onClick={() => { copyToClipboard(JSON.stringify(req.request_body || req.body || '', null, 2)); showToast('Copied'); }}>
                <CopyIcon /> Copy
              </button>
            </div>
            <JsonView data={req.request_body || req.body || '(empty)'} />
          </div>
        );
      case 'response':
        return (
          <div>
            <div class="flex items-center justify-between mb-4">
              <span class="section-title" style="margin:0">Response Body</span>
              <button class="copy-btn" onClick={() => { copyToClipboard(JSON.stringify(req.response_body || '', null, 2)); showToast('Copied'); }}>
                <CopyIcon /> Copy
              </button>
            </div>
            <JsonView data={req.response_body || '(empty)'} />
          </div>
        );
      case 'timing':
        return (
          <div>
            <table>
              <tbody>
                <tr><td style="font-weight:600;width:150px">Total Latency</td><td>{fmtDuration(req.latency_ms || req.duration_ms)}</td></tr>
                <tr><td style="font-weight:600">Timestamp</td><td class="font-mono">{req.timestamp || req.time || ''}</td></tr>
              </tbody>
            </table>
          </div>
        );
      default:
        return (
          <div>
            <table>
              <tbody>
                <tr><td style="font-weight:600;width:150px">Method</td><td>{req.method || 'POST'}</td></tr>
                <tr><td style="font-weight:600">Service</td><td>{req.service}</td></tr>
                <tr><td style="font-weight:600">Action</td><td class="font-mono">{req.action}</td></tr>
                <tr><td style="font-weight:600">Status</td><td><StatusBadge code={req.status} /></td></tr>
                <tr><td style="font-weight:600">Latency</td><td>{fmtDuration(req.latency_ms || req.duration_ms)}</td></tr>
                <tr><td style="font-weight:600">Request ID</td><td class="font-mono text-sm">{req.id || ''}</td></tr>
                {req.trace_id && <tr><td style="font-weight:600">Trace</td><td><a href={`#/traces`} class="trace-link" style="color:var(--brand-blue);text-decoration:underline;cursor:pointer">View Trace</a></td></tr>}
                <tr><td style="font-weight:600">Time</td><td>{req.timestamp || req.time || ''}</td></tr>
              </tbody>
            </table>
          </div>
        );
    }
  }

  const tabs: DetailTab[] = ['overview', 'request', 'response', 'timing'];

  return (
    <div>
      <div class="flex items-center justify-between mb-6">
        <div>
          <h1 class="page-title">Request Log</h1>
          <p class="page-desc">All API requests to cloudmock services</p>
        </div>
        <button class="btn btn-ghost btn-sm" onClick={() => api('/api/requests?limit=200').then(setRequests)}>
          <RefreshIcon /> Refresh
        </button>
      </div>

      <div class="filters-bar">
        <select class="select" id="service-filter" value={svcFilter} onChange={(e) => setSvcFilter((e.target as HTMLSelectElement).value)}>
          <option value="">All Services</option>
          {services.map(s => <option value={s}>{s}</option>)}
        </select>
        <select class="select" value={statusFilter} onChange={(e) => setStatusFilter((e.target as HTMLSelectElement).value)}>
          <option value="">All Status</option>
          <option value="2xx">2xx Success</option>
          <option value="4xx">4xx Client Error</option>
          <option value="5xx">5xx Server Error</option>
        </select>
        <input class="input input-search" placeholder="Search requests..." value={textFilter} onInput={(e) => setTextFilter((e.target as HTMLInputElement).value)} />
        <span class="text-sm text-muted ml-auto">{filtered.length} requests</span>
      </div>

      <div class="card">
        <div class="table-wrap">
          <table id="requests-table">
            <thead>
              <tr>
                <th style="width:100px">Time</th>
                <th>Service</th>
                <th>Action</th>
                <th style="width:80px">Status</th>
                <th style="width:80px">Latency</th>
                <th style="width:40px"></th>
              </tr>
            </thead>
            <tbody id="requests-tbody">
              {filtered.length === 0 ? (
                <tr><td colSpan={6} class="empty-state">No requests recorded yet</td></tr>
              ) : filtered.map((req: any) => (
                <>
                  <tr
                    class={`clickable ${expanded === req.id ? 'expanded' : ''}`}
                    onClick={() => toggleExpand(req.id)}
                    key={req.id || Math.random()}
                  >
                    <td class="font-mono text-sm">{fmtTime(req.timestamp || req.time)}</td>
                    <td><span style="font-weight:600">{req.service}</span></td>
                    <td class="font-mono text-sm">{req.action}</td>
                    <td><span class={`status-pill ${statusClass(req.status)}`}>{req.status}</span></td>
                    <td class="font-mono text-sm">{fmtDuration(req.latency_ms || req.duration_ms)}</td>
                    <td>
                      <button class="btn-icon btn-sm btn-ghost" title="Open in drawer" onClick={(e) => openDrawer(e, req)}>
                        <ExpandIcon />
                      </button>
                    </td>
                  </tr>
                  {expanded === req.id && (
                    <tr>
                      <td colSpan={6} style="padding:0">
                        <div class="req-expand">
                          <div class="req-expand-inner">
                            <div class="tabs" style="padding:0 16px">
                              {tabs.map(t => (
                                <button class={`tab ${detailTab === t ? 'active' : ''}`} onClick={() => setDetailTab(t)}>
                                  {t.charAt(0).toUpperCase() + t.slice(1)}
                                </button>
                              ))}
                            </div>
                            <div class="req-expand-body">
                              {renderDetail(req, detailTab)}
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

      {drawer && (
        <Drawer
          title="Request Detail"
          onClose={() => setDrawer(null)}
          actions={
            <>
              <button class="btn btn-sm btn-ghost" onClick={() => replayRequest(drawer.id)}>
                <PlayIcon /> Replay
              </button>
              <a class="btn btn-sm btn-secondary" href={`#/requests/${drawer.id}`} style="text-decoration:none">
                Full Page
              </a>
            </>
          }
        >
          <div class="tabs">
            {tabs.map(t => (
              <button class={`tab ${detailTab === t ? 'active' : ''}`} onClick={() => setDetailTab(t)}>
                {t.charAt(0).toUpperCase() + t.slice(1)}
              </button>
            ))}
          </div>
          {renderDetail(drawer, detailTab)}
        </Drawer>
      )}
    </div>
  );
}
