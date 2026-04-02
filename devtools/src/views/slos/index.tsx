import { useState, useEffect } from 'preact/hooks';
import { api, cachedApi } from '../../lib/api';
import './slos.css';

interface SLORule {
  service: string;
  action: string;
  p50_ms: number;
  p95_ms: number;
  p99_ms: number;
  error_rate: number;
}

interface SLOWindow {
  Service: string;
  Action: string;
  Total: number;
  Errors: number;
  P50Ms: number;
  P95Ms: number;
  P99Ms: number;
  ErrorRate: number;
  Healthy: boolean;
  Violations: string[];
}

interface SLOAlert {
  message: string;
  severity: string;
  timestamp: string;
}

interface SLOResponse {
  alerts: SLOAlert[];
  healthy: boolean;
  rules: SLORule[];
  windows: SLOWindow[];
}

/** Compute current error budget for each window as a single value */
function computeCurrentBudgets(
  windows: SLOWindow[],
  rules: SLORule[],
): { service: string; budgetPct: number }[] {
  const results: { service: string; budgetPct: number }[] = [];

  for (const w of windows) {
    const rule = rules.find(
      (r) => r.service === w.Service && (r.action === '*' || r.action === w.Action),
    );
    if (!rule) continue;

    const allowedErrorRate = rule.error_rate;
    if (allowedErrorRate <= 0) continue;

    const actualErrorRate = w.Total > 0 ? w.Errors / w.Total : 0;
    const currentBudget = Math.max(-50, (1 - actualErrorRate / allowedErrorRate) * 100);

    results.push({
      service: `${w.Service} / ${w.Action || '*'}`,
      budgetPct: currentBudget,
    });
  }

  return results;
}

/** Color for budget gauge based on remaining budget */
function budgetColor(currentPct: number): string {
  if (currentPct <= 0) return '#ff4e5e';
  if (currentPct <= 20) return '#fad065';
  return '#36d982';
}

function ErrorBudgetSection({
  windows,
  rules,
}: {
  windows: SLOWindow[];
  rules: SLORule[];
}) {
  const budgets = computeCurrentBudgets(windows, rules);

  if (budgets.length === 0) {
    return null;
  }

  return (
    <div class="slos-section">
      <h3 class="slos-section-title">Current Error Budget</h3>
      <div class="slos-budget-charts">
        {budgets.map((b) => {
          const color = budgetColor(b.budgetPct);
          const barWidth = Math.max(0, Math.min(100, b.budgetPct));
          return (
            <div key={b.service} class="slos-budget-chart-item">
              <div class="slos-budget-gauge">
                <div class="slos-budget-gauge-label">{b.service}</div>
                <div class="slos-budget-gauge-bar-track">
                  <div
                    class="slos-budget-gauge-bar-fill"
                    style={{ width: `${barWidth}%`, background: color }}
                  />
                </div>
                <div class="slos-budget-gauge-value" style={{ color }}>
                  {b.budgetPct.toFixed(1)}%
                </div>
              </div>
              {b.budgetPct <= 0 && (
                <div class="slos-budget-exhausted">Budget Exhausted</div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

function formatLatency(ms: number): string {
  if (ms < 1) return `${(ms * 1000).toFixed(0)}us`;
  if (ms < 1000) return `${ms.toFixed(1)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

function formatRate(rate: number): string {
  return `${(rate * 100).toFixed(2)}%`;
}

interface ServiceOption {
  name: string;
}

export function SLOsView() {
  const [data, setData] = useState<SLOResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Add-rule form state
  const [services, setServices] = useState<string[]>([]);
  const [newService, setNewService] = useState('');
  const [newP50, setNewP50] = useState('100');
  const [newP95, setNewP95] = useState('500');
  const [newP99, setNewP99] = useState('1000');
  const [newErrorRate, setNewErrorRate] = useState('0.01');
  const [formError, setFormError] = useState<string | null>(null);
  const [formSuccess, setFormSuccess] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    cachedApi<SLOResponse>('/api/slo', 'slos:data')
      .then(setData)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
    // Load service names for dropdown
    cachedApi<ServiceOption[]>('/api/services', 'slos:services')
      .then((svcs) => {
        if (Array.isArray(svcs)) {
          setServices(svcs.map((s) => s.name).sort());
        }
      })
      .catch((e) => { console.warn('[SLOs] Failed to load service list:', e); });
  }, []);

  async function handleAddRule() {
    if (!newService) {
      setFormError('Service is required');
      return;
    }
    const rule: SLORule = {
      service: newService,
      action: '*',
      p50_ms: Number(newP50) || 100,
      p95_ms: Number(newP95) || 500,
      p99_ms: Number(newP99) || 1000,
      error_rate: Number(newErrorRate) || 0.01,
    };
    const existingRules = data?.rules || [];
    const updatedRules = [...existingRules, rule];

    setSubmitting(true);
    setFormError(null);
    setFormSuccess(null);
    try {
      const result = await api<SLOResponse>('/api/slo', {
        method: 'POST',
        body: JSON.stringify({ rules: updatedRules }),
      });
      setData(result);
      setFormSuccess(`Added SLO rule for ${newService}`);
      setNewService('');
      setNewP50('100');
      setNewP95('500');
      setNewP99('1000');
      setNewErrorRate('0.01');
    } catch (err: any) {
      setFormError(err.message || 'Failed to add rule');
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDeleteRule(index: number) {
    const existingRules = data?.rules || [];
    const updatedRules = existingRules.filter((_, i) => i !== index);

    setFormError(null);
    setFormSuccess(null);
    try {
      const result = await api<SLOResponse>('/api/slo', {
        method: 'POST',
        body: JSON.stringify({ rules: updatedRules }),
      });
      setData(result);
      setFormSuccess('Rule deleted');
    } catch (err: any) {
      setFormError(err.message || 'Failed to delete rule');
    }
  }

  if (loading) {
    return (
      <div class="slos-view slos-view-empty">
        <div class="slos-placeholder">Loading SLO data...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div class="slos-view slos-view-empty">
        <div class="slos-placeholder slos-placeholder-error">Failed to load SLOs: {error}</div>
      </div>
    );
  }

  if (!data || (!data.rules?.length && !data.windows?.length)) {
    return (
      <div class="slos-view slos-view-empty">
        <div class="slos-placeholder">No SLO rules configured</div>
      </div>
    );
  }

  return (
    <div class="slos-view">
      <div class="slos-header">
        <h2 class="slos-title">Service Level Objectives</h2>
        <div class={`slos-health-badge ${data.healthy ? 'slos-healthy' : 'slos-unhealthy'}`}>
          <span class="slos-health-dot" />
          {data.healthy ? 'All Healthy' : 'Violations Detected'}
        </div>
      </div>

      {data.alerts && data.alerts.length > 0 && (
        <div class="slos-alerts">
          {data.alerts.map((alert, i) => (
            <div key={i} class={`slos-alert slos-alert-${alert.severity || 'warning'}`}>
              <span class="slos-alert-message">{alert.message}</span>
              {alert.timestamp && (
                <span class="slos-alert-time">
                  {new Date(alert.timestamp).toLocaleTimeString()}
                </span>
              )}
            </div>
          ))}
        </div>
      )}

      {data.rules && data.rules.length > 0 && (
        <div class="slos-section">
          <h3 class="slos-section-title">SLO Rules</h3>
          <div class="slos-table-wrap">
            <table class="slos-table">
              <thead>
                <tr>
                  <th>Service</th>
                  <th>Action</th>
                  <th>P50</th>
                  <th>P95</th>
                  <th>P99</th>
                  <th>Error Rate</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {data.rules.map((rule, i) => (
                  <tr key={i}>
                    <td class="slos-service-name">{rule.service}</td>
                    <td class="slos-action">{rule.action || '*'}</td>
                    <td class="slos-mono">{formatLatency(rule.p50_ms)}</td>
                    <td class="slos-mono">{formatLatency(rule.p95_ms)}</td>
                    <td class="slos-mono">{formatLatency(rule.p99_ms)}</td>
                    <td class="slos-mono">{formatRate(rule.error_rate)}</td>
                    <td>
                      <button
                        class="slos-delete-btn"
                        onClick={() => handleDeleteRule(i)}
                        title="Delete rule"
                      >
                        &times;
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {data.windows && data.windows.length > 0 && (
        <div class="slos-section">
          <h3 class="slos-section-title">SLO Compliance Windows</h3>
          <div class="slos-table-wrap">
            <table class="slos-table">
              <thead>
                <tr>
                  <th>Status</th>
                  <th>Service</th>
                  <th>Action</th>
                  <th>Total</th>
                  <th>Errors</th>
                  <th>P50</th>
                  <th>P95</th>
                  <th>P99</th>
                  <th>Error Rate</th>
                  <th>Violations</th>
                </tr>
              </thead>
              <tbody>
                {data.windows.map((w, i) => (
                  <tr key={i} class={w.Healthy ? '' : 'slos-row-violation'}>
                    <td>
                      <span class={`slos-status-dot ${w.Healthy ? 'slos-status-healthy' : 'slos-status-violation'}`} />
                    </td>
                    <td class="slos-service-name">{w.Service}</td>
                    <td class="slos-action">{w.Action || '*'}</td>
                    <td class="slos-mono">{w.Total}</td>
                    <td class={`slos-mono ${w.Errors > 0 ? 'slos-text-error' : ''}`}>{w.Errors}</td>
                    <td class="slos-mono">{formatLatency(w.P50Ms)}</td>
                    <td class="slos-mono">{formatLatency(w.P95Ms)}</td>
                    <td class="slos-mono">{formatLatency(w.P99Ms)}</td>
                    <td class={`slos-mono ${w.ErrorRate > 0.05 ? 'slos-text-error' : ''}`}>
                      {formatRate(w.ErrorRate)}
                    </td>
                    <td>
                      {w.Violations && w.Violations.length > 0 ? (
                        <div class="slos-violations">
                          {w.Violations.map((v, vi) => (
                            <span key={vi} class="slos-violation-tag">{v}</span>
                          ))}
                        </div>
                      ) : (
                        <span class="slos-text-muted">--</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {data.windows && data.windows.length > 0 && data.rules && data.rules.length > 0 && (
        <ErrorBudgetSection windows={data.windows} rules={data.rules} />
      )}

      <div class="slos-section">
        <h3 class="slos-section-title">Add Rule</h3>

        {formError && (
          <div class="slos-form-feedback slos-form-error">
            <span class="slos-form-feedback-icon">!</span>
            {formError}
          </div>
        )}
        {formSuccess && (
          <div class="slos-form-feedback slos-form-success">
            <span class="slos-form-feedback-icon">&#x2713;</span>
            {formSuccess}
          </div>
        )}

        <div class="slos-add-form">
          <div class="slos-form-field">
            <label class="slos-form-label">Service</label>
            <select
              class="input slos-form-select"
              value={newService}
              onChange={(e) => setNewService((e.target as HTMLSelectElement).value)}
            >
              <option value="">Select service...</option>
              {services.map((s) => (
                <option key={s} value={s}>{s}</option>
              ))}
            </select>
          </div>
          <div class="slos-form-field">
            <label class="slos-form-label">P50 (ms)</label>
            <input
              class="input slos-form-input"
              type="number"
              value={newP50}
              onInput={(e) => setNewP50((e.target as HTMLInputElement).value)}
            />
          </div>
          <div class="slos-form-field">
            <label class="slos-form-label">P95 (ms)</label>
            <input
              class="input slos-form-input"
              type="number"
              value={newP95}
              onInput={(e) => setNewP95((e.target as HTMLInputElement).value)}
            />
          </div>
          <div class="slos-form-field">
            <label class="slos-form-label">P99 (ms)</label>
            <input
              class="input slos-form-input"
              type="number"
              value={newP99}
              onInput={(e) => setNewP99((e.target as HTMLInputElement).value)}
            />
          </div>
          <div class="slos-form-field">
            <label class="slos-form-label">Error Rate</label>
            <input
              class="input slos-form-input"
              type="number"
              step="0.001"
              value={newErrorRate}
              onInput={(e) => setNewErrorRate((e.target as HTMLInputElement).value)}
            />
          </div>
          <div class="slos-form-field slos-form-field-action">
            <button
              class="btn btn-primary slos-add-btn"
              onClick={handleAddRule}
              disabled={submitting || !newService}
            >
              {submitting ? 'Adding...' : 'Add Rule'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
