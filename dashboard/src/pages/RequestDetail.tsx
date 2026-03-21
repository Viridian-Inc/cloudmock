import { useState, useEffect } from 'preact/hooks';
import { api } from '../api';
import { StatusBadge } from '../components/StatusBadge';
import { JsonView } from '../components/JsonView';
import { CopyIcon } from '../components/Icons';
import { fmtDuration, copyToClipboard } from '../utils';
import { statusClass } from '../components/StatusBadge';

interface RequestDetailPageProps {
  id: string;
  showToast: (msg: string) => void;
}

type Tab = 'overview' | 'request' | 'response' | 'timing';

export function RequestDetailPage({ id, showToast }: RequestDetailPageProps) {
  const [req, setReq] = useState<any>(null);
  const [tab, setTab] = useState<Tab>('overview');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    api(`/api/requests/${id}`).then(r => { setReq(r); setLoading(false); }).catch(() => setLoading(false));
  }, [id]);

  if (loading) return <div class="empty-state">Loading...</div>;
  if (!req) return <div class="empty-state">Request not found</div>;

  function renderBody(body: any, label: string) {
    return (
      <div>
        <div class="flex items-center justify-between mb-4">
          <span class="section-title" style="margin:0">{label}</span>
          <button class="copy-btn" onClick={() => { copyToClipboard(JSON.stringify(body || '', null, 2)); showToast('Copied'); }}>
            <CopyIcon /> Copy
          </button>
        </div>
        <JsonView data={body || '(empty)'} />
      </div>
    );
  }

  const tabs: Tab[] = ['overview', 'request', 'response', 'timing'];

  return (
    <div>
      <div class="flex items-center gap-3 mb-6">
        <a href="#/requests" class="btn btn-ghost btn-sm">Back</a>
        <div>
          <h1 class="page-title">{req.service} / {req.action}</h1>
          <p class="page-desc font-mono">{id}</p>
        </div>
        <div class="ml-auto">
          <span class={`status-pill ${statusClass(req.status)}`} style="font-size:16px;padding:4px 14px">{req.status}</span>
        </div>
      </div>

      <div class="card">
        <div class="tabs" style="padding:0 20px">
          {tabs.map(t => (
            <button class={`tab ${tab === t ? 'active' : ''}`} onClick={() => setTab(t)}>
              {t.charAt(0).toUpperCase() + t.slice(1)}
            </button>
          ))}
        </div>
        <div class="card-body">
          {tab === 'overview' && (
            <table>
              <tbody>
                <tr><td style="font-weight:600;width:160px">Method</td><td>{req.method || 'POST'}</td></tr>
                <tr><td style="font-weight:600">Service</td><td>{req.service}</td></tr>
                <tr><td style="font-weight:600">Action</td><td class="font-mono">{req.action}</td></tr>
                <tr><td style="font-weight:600">Status</td><td><StatusBadge code={req.status} /></td></tr>
                <tr><td style="font-weight:600">Latency</td><td>{fmtDuration(req.latency_ms || req.duration_ms)}</td></tr>
                <tr><td style="font-weight:600">Timestamp</td><td class="font-mono">{req.timestamp || req.time || ''}</td></tr>
              </tbody>
            </table>
          )}
          {tab === 'request' && renderBody(req.request_body || req.body, 'Request Body')}
          {tab === 'response' && renderBody(req.response_body, 'Response Body')}
          {tab === 'timing' && (
            <table>
              <tbody>
                <tr><td style="font-weight:600;width:160px">Total Latency</td><td>{fmtDuration(req.latency_ms || req.duration_ms)}</td></tr>
                <tr><td style="font-weight:600">Timestamp</td><td class="font-mono">{req.timestamp || req.time || ''}</td></tr>
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  );
}
