import { useState, useEffect, useCallback } from 'preact/hooks';
import { api } from '../api';

interface ChaosRule {
  id: string;
  service: string;
  action: string;
  enabled: boolean;
  type: string;
  errorCode: number;
  errorMsg: string;
  latencyMs: number;
  percentage: number;
}

interface ChaosState {
  rules: ChaosRule[];
  active: boolean;
}

const PRESETS: { label: string; rule: Partial<ChaosRule> }[] = [
  {
    label: 'Simulate DynamoDB Throttling',
    rule: { service: 'dynamodb', action: '*', type: 'error', errorCode: 400, errorMsg: 'ProvisionedThroughputExceededException', percentage: 10, enabled: true },
  },
  {
    label: 'Add Network Latency',
    rule: { service: '*', action: '*', type: 'latency', latencyMs: 500, percentage: 100, enabled: true },
  },
  {
    label: 'S3 Intermittent Failures',
    rule: { service: 's3', action: '*', type: 'error', errorCode: 500, errorMsg: 'InternalError', percentage: 20, enabled: true },
  },
  {
    label: 'Lambda Cold Start',
    rule: { service: 'lambda', action: 'Invoke', type: 'latency', latencyMs: 3000, percentage: 30, enabled: true },
  },
];

const FAULT_TYPES = ['error', 'latency', 'timeout', 'blackhole'] as const;

export function ChaosPage({ showToast }: { showToast: (msg: string) => void }) {
  const [state, setState] = useState<ChaosState>({ rules: [], active: false });
  const [form, setForm] = useState<Partial<ChaosRule>>({
    service: '*', action: '*', type: 'error', errorCode: 500, errorMsg: 'InternalError',
    latencyMs: 1000, percentage: 50, enabled: true,
  });

  const fetchRules = useCallback(() => {
    api('/api/chaos').then((data: ChaosState) => setState(data)).catch(() => {});
  }, []);

  useEffect(() => {
    fetchRules();
    const iv = setInterval(fetchRules, 5000);
    return () => clearInterval(iv);
  }, [fetchRules]);

  const createRule = async (rule: Partial<ChaosRule>) => {
    try {
      await api('/api/chaos', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(rule),
      });
      showToast('Chaos rule created');
      fetchRules();
    } catch { showToast('Failed to create rule'); }
  };

  const toggleRule = async (rule: ChaosRule) => {
    try {
      await api(`/api/chaos/${rule.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...rule, enabled: !rule.enabled }),
      });
      fetchRules();
    } catch { showToast('Failed to update rule'); }
  };

  const deleteRule = async (id: string) => {
    try {
      await api(`/api/chaos/${id}`, { method: 'DELETE' });
      showToast('Rule deleted');
      fetchRules();
    } catch { showToast('Failed to delete rule'); }
  };

  const disableAll = async () => {
    try {
      await api('/api/chaos', { method: 'DELETE' });
      showToast('All rules disabled');
      fetchRules();
    } catch { showToast('Failed to disable rules'); }
  };

  const typeDetails = (r: ChaosRule) => {
    switch (r.type) {
      case 'error': return `HTTP ${r.errorCode}: ${r.errorMsg}`;
      case 'latency': return `+${r.latencyMs}ms`;
      case 'timeout': return '30s timeout -> 504';
      case 'blackhole': return 'Connection reset';
      default: return r.type;
    }
  };

  return (
    <div style={{ padding: '24px' }}>
      <h2 style={{ margin: '0 0 16px', fontSize: '20px', fontWeight: 600 }}>Chaos Engineering</h2>

      {/* Status banner */}
      <div style={{
        padding: '12px 16px', borderRadius: '8px', marginBottom: '20px', fontSize: '13px', fontWeight: 600,
        background: state.active ? 'rgba(239, 68, 68, 0.1)' : 'rgba(34, 197, 94, 0.1)',
        border: `1px solid ${state.active ? 'rgba(239, 68, 68, 0.3)' : 'rgba(34, 197, 94, 0.3)'}`,
        color: state.active ? '#ef4444' : '#22c55e',
      }}>
        {state.active ? 'Fault injection is ACTIVE -- requests may be affected' : 'No active rules -- all requests pass through normally'}
      </div>

      {/* Presets */}
      <div class="card" style={{ padding: '16px', marginBottom: '20px' }}>
        <h3 style={{ margin: '0 0 12px', fontSize: '14px', fontWeight: 600 }}>Quick Presets</h3>
        <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
          {PRESETS.map(p => (
            <button
              key={p.label}
              onClick={() => createRule(p.rule)}
              style={{
                padding: '6px 12px', borderRadius: '6px', border: '1px solid var(--border)',
                background: 'var(--bg-secondary)', color: 'var(--text-primary)', cursor: 'pointer',
                fontSize: '12px', whiteSpace: 'nowrap',
              }}
            >
              {p.label}
            </button>
          ))}
        </div>
      </div>

      {/* Create rule form */}
      <div class="card" style={{ padding: '16px', marginBottom: '20px' }}>
        <h3 style={{ margin: '0 0 12px', fontSize: '14px', fontWeight: 600 }}>Create Rule</h3>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))', gap: '12px', marginBottom: '12px' }}>
          <label style={{ fontSize: '12px' }}>
            <div style={{ marginBottom: '4px', opacity: 0.7 }}>Service</div>
            <input
              type="text"
              value={form.service}
              onInput={(e: any) => setForm({ ...form, service: e.target.value })}
              placeholder="* or service name"
              style={inputStyle}
            />
          </label>
          <label style={{ fontSize: '12px' }}>
            <div style={{ marginBottom: '4px', opacity: 0.7 }}>Action</div>
            <input
              type="text"
              value={form.action}
              onInput={(e: any) => setForm({ ...form, action: e.target.value })}
              placeholder="* or action name"
              style={inputStyle}
            />
          </label>
          <label style={{ fontSize: '12px' }}>
            <div style={{ marginBottom: '4px', opacity: 0.7 }}>Fault Type</div>
            <select
              value={form.type}
              onChange={(e: any) => setForm({ ...form, type: e.target.value })}
              style={inputStyle}
            >
              {FAULT_TYPES.map(t => <option key={t} value={t}>{t.charAt(0).toUpperCase() + t.slice(1)}</option>)}
            </select>
          </label>
          {form.type === 'error' && (
            <>
              <label style={{ fontSize: '12px' }}>
                <div style={{ marginBottom: '4px', opacity: 0.7 }}>Status Code</div>
                <input
                  type="number"
                  value={form.errorCode}
                  onInput={(e: any) => setForm({ ...form, errorCode: parseInt(e.target.value) || 500 })}
                  style={inputStyle}
                />
              </label>
              <label style={{ fontSize: '12px' }}>
                <div style={{ marginBottom: '4px', opacity: 0.7 }}>Error Message</div>
                <input
                  type="text"
                  value={form.errorMsg}
                  onInput={(e: any) => setForm({ ...form, errorMsg: e.target.value })}
                  style={inputStyle}
                />
              </label>
            </>
          )}
          {form.type === 'latency' && (
            <label style={{ fontSize: '12px' }}>
              <div style={{ marginBottom: '4px', opacity: 0.7 }}>Latency (ms)</div>
              <input
                type="number"
                value={form.latencyMs}
                onInput={(e: any) => setForm({ ...form, latencyMs: parseInt(e.target.value) || 0 })}
                style={inputStyle}
              />
            </label>
          )}
          <label style={{ fontSize: '12px' }}>
            <div style={{ marginBottom: '4px', opacity: 0.7 }}>Percentage ({form.percentage}%)</div>
            <input
              type="range"
              min="0"
              max="100"
              value={form.percentage}
              onInput={(e: any) => setForm({ ...form, percentage: parseInt(e.target.value) })}
              style={{ width: '100%' }}
            />
          </label>
        </div>
        <button
          onClick={() => createRule(form)}
          style={{
            padding: '8px 16px', borderRadius: '6px', border: 'none',
            background: '#3b82f6', color: 'white', cursor: 'pointer', fontWeight: 600, fontSize: '13px',
          }}
        >
          Create Rule
        </button>
      </div>

      {/* Active rules table */}
      <div class="card" style={{ padding: '16px' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
          <h3 style={{ margin: 0, fontSize: '14px', fontWeight: 600 }}>Rules ({state.rules.length})</h3>
          {state.rules.length > 0 && (
            <button
              onClick={disableAll}
              style={{
                padding: '4px 12px', borderRadius: '4px', border: '1px solid var(--border)',
                background: 'transparent', color: '#ef4444', cursor: 'pointer', fontSize: '12px',
              }}
            >
              Disable All
            </button>
          )}
        </div>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
          <thead>
            <tr style={{ borderBottom: '1px solid var(--border)' }}>
              <th style={thStyle}>ID</th>
              <th style={thStyle}>Service</th>
              <th style={thStyle}>Action</th>
              <th style={thStyle}>Type</th>
              <th style={thStyle}>Details</th>
              <th style={{ ...thStyle, textAlign: 'right' }}>%</th>
              <th style={{ ...thStyle, textAlign: 'center' }}>Enabled</th>
              <th style={{ ...thStyle, textAlign: 'center' }}></th>
            </tr>
          </thead>
          <tbody>
            {state.rules.map(r => (
              <tr key={r.id} style={{ borderBottom: '1px solid var(--border)' }}>
                <td style={tdStyle}><code style={{ fontSize: '11px' }}>{r.id}</code></td>
                <td style={tdStyle}>{r.service}</td>
                <td style={tdStyle}>{r.action}</td>
                <td style={tdStyle}>
                  <span style={{
                    display: 'inline-block', padding: '2px 6px', borderRadius: '4px', fontSize: '11px', fontWeight: 600,
                    background: typeColor(r.type).bg, color: typeColor(r.type).fg,
                  }}>
                    {r.type}
                  </span>
                </td>
                <td style={{ ...tdStyle, maxWidth: '200px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{typeDetails(r)}</td>
                <td style={{ ...tdStyle, textAlign: 'right' }}>{r.percentage}%</td>
                <td style={{ ...tdStyle, textAlign: 'center' }}>
                  <button
                    onClick={() => toggleRule(r)}
                    style={{
                      width: '40px', height: '22px', borderRadius: '11px', border: 'none', cursor: 'pointer',
                      background: r.enabled ? '#22c55e' : 'var(--border)',
                      position: 'relative', transition: 'background 0.2s',
                    }}
                  >
                    <span style={{
                      position: 'absolute', top: '2px', width: '18px', height: '18px', borderRadius: '50%',
                      background: 'white', transition: 'left 0.2s',
                      left: r.enabled ? '20px' : '2px',
                    }} />
                  </button>
                </td>
                <td style={{ ...tdStyle, textAlign: 'center' }}>
                  <button
                    onClick={() => deleteRule(r.id)}
                    style={{ background: 'none', border: 'none', color: '#ef4444', cursor: 'pointer', fontSize: '16px', padding: '2px 6px' }}
                    title="Delete rule"
                  >
                    x
                  </button>
                </td>
              </tr>
            ))}
            {state.rules.length === 0 && (
              <tr><td colSpan={8} style={{ padding: '24px', textAlign: 'center', opacity: 0.5 }}>No chaos rules configured. Create one above or use a preset.</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function typeColor(type: string) {
  switch (type) {
    case 'error': return { bg: 'rgba(239, 68, 68, 0.1)', fg: '#ef4444' };
    case 'latency': return { bg: 'rgba(245, 158, 11, 0.1)', fg: '#f59e0b' };
    case 'timeout': return { bg: 'rgba(168, 85, 247, 0.1)', fg: '#a855f7' };
    case 'blackhole': return { bg: 'rgba(107, 114, 128, 0.1)', fg: '#6b7280' };
    default: return { bg: 'var(--bg-secondary)', fg: 'var(--text-primary)' };
  }
}

const inputStyle: any = {
  width: '100%', padding: '6px 8px', borderRadius: '4px', border: '1px solid var(--border)',
  background: 'var(--bg-primary)', color: 'var(--text-primary)', fontSize: '12px', boxSizing: 'border-box',
};

const thStyle: any = { padding: '8px', textAlign: 'left', fontWeight: 500, opacity: 0.8 };
const tdStyle: any = { padding: '8px' };

// Export a hook that other components can use to check if chaos is active.
export function useChaosActive() {
  const [active, setActive] = useState(false);
  useEffect(() => {
    const check = () => api('/api/chaos').then((d: ChaosState) => setActive(d.active)).catch(() => {});
    check();
    const iv = setInterval(check, 5000);
    return () => clearInterval(iv);
  }, []);
  return active;
}
