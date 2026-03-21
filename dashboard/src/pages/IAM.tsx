import { useState } from 'preact/hooks';
import { ADMIN_BASE } from '../api';
import { JsonView } from '../components/JsonView';
import { statusClass } from '../components/StatusBadge';

interface IAMPageProps {
  showToast: (msg: string) => void;
}

export function IAMPage({ showToast }: IAMPageProps) {
  const [principal, setPrincipal] = useState('');
  const [action, setAction] = useState('');
  const [resource, setResource] = useState('');
  const [result, setResult] = useState<any>(null);
  const [history, setHistory] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);

  function evaluate() {
    if (!principal || !action || !resource) {
      showToast('All fields are required');
      return;
    }
    setLoading(true);
    fetch(`${ADMIN_BASE}/api/iam/evaluate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ principal, action, resource }),
    }).then(r => r.json()).then((r: any) => {
      setResult(r);
      setHistory(prev => [{ principal, action, resource, decision: r.decision, time: new Date().toISOString() }, ...prev].slice(0, 50));
      setLoading(false);
    }).catch(() => { showToast('Evaluation failed'); setLoading(false); });
  }

  return (
    <div>
      <div class="mb-6">
        <h1 class="page-title">IAM Debugger</h1>
        <p class="page-desc">Evaluate IAM policies against principals and resources</p>
      </div>

      <div class="flex gap-4">
        <div style="flex:1">
          <div class="card">
            <div class="card-header">
              <h3 style="font-weight:700">Policy Evaluation</h3>
            </div>
            <div class="card-body">
              <div class="mb-4">
                <div class="label">Principal ARN</div>
                <input class="input w-full" placeholder="arn:aws:iam::123456789012:user/admin" value={principal} onInput={(e) => setPrincipal((e.target as HTMLInputElement).value)} />
              </div>
              <div class="mb-4">
                <div class="label">Action</div>
                <input class="input w-full" placeholder="s3:GetObject" value={action} onInput={(e) => setAction((e.target as HTMLInputElement).value)} />
              </div>
              <div class="mb-4">
                <div class="label">Resource ARN</div>
                <input class="input w-full" placeholder="arn:aws:s3:::my-bucket/*" value={resource} onInput={(e) => setResource((e.target as HTMLInputElement).value)} />
              </div>
              <button class="btn btn-primary" onClick={evaluate} disabled={loading}>
                {loading ? 'Evaluating...' : 'Evaluate'}
              </button>

              {result && (
                <>
                  <div class={`iam-result ${result.decision === 'ALLOW' ? 'iam-allow' : 'iam-deny'}`}>
                    {result.decision}
                  </div>
                  {result.reason && <p class="text-sm text-muted mb-4">{result.reason}</p>}
                  {result.matched_statement && (
                    <div>
                      <div class="section-title">Matched Statement</div>
                      <JsonView data={result.matched_statement} />
                    </div>
                  )}
                </>
              )}
            </div>
          </div>
        </div>

        <div style="width:360px">
          <div class="card">
            <div class="card-header">
              <h3 style="font-weight:700">History</h3>
            </div>
            <div class="card-body" style="max-height:500px;overflow-y:auto">
              {history.length === 0 ? (
                <div class="text-sm text-muted" style="text-align:center;padding:24px">No evaluations yet</div>
              ) : history.map((h: any) => (
                <div style="padding:8px 0;border-bottom:1px solid var(--n100);font-size:13px">
                  <div class="flex items-center gap-2">
                    <span class={`status-pill ${h.decision === 'ALLOW' ? 'status-2xx' : 'status-5xx'}`} style="font-size:11px">{h.decision}</span>
                    <span class="font-mono">{h.action}</span>
                  </div>
                  <div class="text-muted truncate" style="margin-top:2px">{h.resource}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
