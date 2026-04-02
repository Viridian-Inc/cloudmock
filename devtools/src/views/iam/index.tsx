import { useState } from 'preact/hooks';
import { api } from '../../lib/api';
import './iam.css';

interface EvalResult {
  decision: string;
  reason?: string;
  matched_statement?: Record<string, unknown>;
}

interface HistoryEntry {
  principal: string;
  action: string;
  resource: string;
  decision: string;
  time: string;
}

export function IAMView() {
  const [principal, setPrincipal] = useState('');
  const [action, setAction] = useState('');
  const [resource, setResource] = useState('');
  const [result, setResult] = useState<EvalResult | null>(null);
  const [history, setHistory] = useState<HistoryEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  function evaluate() {
    if (!principal || !action || !resource) {
      setError('All fields are required');
      return;
    }

    setError('');
    setLoading(true);

    api<EvalResult>('/api/iam/evaluate', {
      method: 'POST',
      body: JSON.stringify({ principal, action, resource }),
    })
      .then((r) => {
        setResult(r);
        setHistory((prev) =>
          [
            {
              principal,
              action,
              resource,
              decision: r.decision,
              time: new Date().toISOString(),
            },
            ...prev,
          ].slice(0, 50),
        );
      })
      .catch(() => {
        setError('Evaluation failed');
      })
      .finally(() => {
        setLoading(false);
      });
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !loading) {
      evaluate();
    }
  }

  return (
    <div class="iam-view">
      <div class="iam-header">
        <h2 class="iam-title">IAM Debugger</h2>
        <p class="iam-desc">
          Evaluate IAM policies against principals and resources
        </p>
      </div>

      <div class="iam-body">
        {/* Evaluation form */}
        <div class="iam-form-card">
          <div class="iam-card-header">
            <h3 class="iam-card-title">Policy Evaluation</h3>
          </div>
          <div class="iam-card-body">
            <div class="iam-field">
              <label class="iam-label">Principal ARN</label>
              <input
                class="iam-input"
                placeholder="arn:aws:iam::123456789012:user/admin"
                value={principal}
                onInput={(e) =>
                  setPrincipal((e.target as HTMLInputElement).value)
                }
                onKeyDown={handleKeyDown}
              />
            </div>

            <div class="iam-field">
              <label class="iam-label">Action</label>
              <input
                class="iam-input"
                placeholder="s3:GetObject"
                value={action}
                onInput={(e) =>
                  setAction((e.target as HTMLInputElement).value)
                }
                onKeyDown={handleKeyDown}
              />
            </div>

            <div class="iam-field">
              <label class="iam-label">Resource ARN</label>
              <input
                class="iam-input"
                placeholder="arn:aws:s3:::my-bucket/*"
                value={resource}
                onInput={(e) =>
                  setResource((e.target as HTMLInputElement).value)
                }
                onKeyDown={handleKeyDown}
              />
            </div>

            {error && (
              <p style={{ color: 'var(--error)', fontSize: '12px', marginBottom: '10px' }}>
                {error}
              </p>
            )}

            <button
              class="iam-evaluate-btn"
              onClick={evaluate}
              disabled={loading}
            >
              {loading ? 'Evaluating...' : 'Evaluate'}
            </button>

            {result && (
              <>
                <div
                  class={`iam-result ${result.decision === 'ALLOW' ? 'iam-result-allow' : 'iam-result-deny'}`}
                >
                  {result.decision}
                </div>

                {result.reason && (
                  <p class="iam-reason">{result.reason}</p>
                )}

                {result.matched_statement && (
                  <>
                    <div class="iam-matched-title">Matched Statement</div>
                    <pre class="iam-json">
                      {JSON.stringify(result.matched_statement, null, 2)}
                    </pre>
                  </>
                )}
              </>
            )}
          </div>
        </div>

        {/* History sidebar */}
        <div class="iam-history">
          <div class="iam-card-header">
            <h3 class="iam-card-title">History</h3>
          </div>
          <div class="iam-history-body">
            {history.length === 0 ? (
              <div class="iam-history-empty">No evaluations yet</div>
            ) : (
              history.map((h, i) => (
                <div class="iam-history-item" key={i}>
                  <div class="iam-history-row">
                    <span
                      class={`iam-history-badge ${h.decision === 'ALLOW' ? 'iam-badge-allow' : 'iam-badge-deny'}`}
                    >
                      {h.decision}
                    </span>
                    <span class="iam-history-action">{h.action}</span>
                  </div>
                  <div class="iam-history-resource" title={h.resource}>
                    {h.resource}
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
