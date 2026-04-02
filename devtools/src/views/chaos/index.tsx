import { useState, useEffect, useRef, useCallback } from 'preact/hooks';
import { api } from '../../lib/api';
import './chaos.css';

const DURATION_OPTIONS: { label: string; seconds: number }[] = [
  { label: 'Indefinite', seconds: 0 },
  { label: '1 min', seconds: 60 },
  { label: '5 min', seconds: 300 },
  { label: '15 min', seconds: 900 },
  { label: '30 min', seconds: 1800 },
  { label: '1 hour', seconds: 3600 },
];

/** Format milliseconds remaining into a human-readable countdown string. */
function formatCountdown(ms: number): string {
  const totalSeconds = Math.ceil(ms / 1000);
  if (totalSeconds <= 0) return '0s';
  const m = Math.floor(totalSeconds / 60);
  const s = totalSeconds % 60;
  if (m === 0) return `${s}s`;
  return `${m}m ${s.toString().padStart(2, '0')}s`;
}

interface ChaosRule {
  service: string;
  action?: string;
  type: 'latency' | 'error' | 'throttle';
  value: number;
}

interface ChaosState {
  active: boolean;
  rules: ChaosRule[];
}

const TYPE_OPTIONS: { value: ChaosRule['type']; label: string }[] = [
  { value: 'latency', label: 'Latency (ms)' },
  { value: 'error', label: 'Error (status)' },
  { value: 'throttle', label: 'Throttle (%)' },
];

function typeLabel(type: ChaosRule['type']): string {
  switch (type) {
    case 'latency': return 'Latency';
    case 'error': return 'Error';
    case 'throttle': return 'Throttle';
  }
}

function valueLabel(rule: ChaosRule): string {
  switch (rule.type) {
    case 'latency': return `${rule.value}ms`;
    case 'error': return `HTTP ${rule.value}`;
    case 'throttle': return `${rule.value}%`;
  }
}

function typeBadgeClass(type: ChaosRule['type']): string {
  switch (type) {
    case 'latency': return 'chaos-badge-latency';
    case 'error': return 'chaos-badge-error';
    case 'throttle': return 'chaos-badge-throttle';
  }
}

interface ChaosPreset {
  label: string;
  icon: string;
  description: string;
  rules: ChaosRule[];
}

const CHAOS_PRESETS: ChaosPreset[] = [
  {
    label: 'Slow Database',
    icon: '\uD83D\uDC0C',
    description: 'DynamoDB +2s latency',
    rules: [{ service: 'dynamodb', type: 'latency', value: 2000 }],
  },
  {
    label: 'Auth Failure',
    icon: '\uD83D\uDD12',
    description: 'Cognito returns 403',
    rules: [{ service: 'cognito-idp', type: 'error', value: 403 }],
  },
  {
    label: 'Queue Backlog',
    icon: '\uD83D\uDCEC',
    description: 'SQS +5s latency',
    rules: [{ service: 'sqs', type: 'latency', value: 5000 }],
  },
  {
    label: 'Network Partition',
    icon: '\uD83C\uDF10',
    description: 'All services return 503',
    rules: [{ service: '*', type: 'error', value: 503 }],
  },
  {
    label: 'Lambda Timeout',
    icon: '\u26A1',
    description: 'Lambda +30s latency',
    rules: [{ service: 'lambda', type: 'latency', value: 30000 }],
  },
];

export function ChaosView() {
  const [active, setActive] = useState(false);
  const [rules, setRules] = useState<ChaosRule[]>([]);
  const [dirty, setDirty] = useState(false);
  const [applying, setApplying] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [loadingState, setLoadingState] = useState(true);

  // Timer state
  const [duration, setDuration] = useState(0); // 0 = indefinite, otherwise seconds
  const [expiresAt, setExpiresAt] = useState<number | null>(null);
  const [countdown, setCountdown] = useState<number>(0);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Countdown tick + auto-disable
  // Uses expiresAt timestamp and computes remaining ms from Date.now() on
  // each tick so the display stays accurate regardless of setInterval drift.
  useEffect(() => {
    if (expiresAt == null) {
      setCountdown(0);
      if (timerRef.current) {
        clearInterval(timerRef.current);
        timerRef.current = null;
      }
      return;
    }

    const deadline = expiresAt; // capture once so the closure never sees null

    function tick() {
      const remaining = Math.max(0, deadline - Date.now());
      setCountdown(remaining);
      if (remaining <= 0) {
        // Auto-disable chaos
        setExpiresAt(null);
        api<ChaosState>('/api/chaos', {
          method: 'POST',
          body: JSON.stringify({ active: false, rules: [] }),
        })
          .then((result) => {
            setActive(result.active);
            setRules(result.rules || []);
            setDirty(false);
            setSuccess('Chaos timer expired — rules disabled');
          })
          .catch(() => {
            setError('Failed to auto-disable chaos');
          });
      }
    }

    tick();
    timerRef.current = setInterval(tick, 1000);
    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current);
        timerRef.current = null;
      }
    };
  }, [expiresAt]);

  const handleStopEarly = useCallback(() => {
    setExpiresAt(null);
    api<ChaosState>('/api/chaos', {
      method: 'POST',
      body: JSON.stringify({ active: false, rules: [] }),
    })
      .then((result) => {
        setActive(result.active);
        setRules(result.rules || []);
        setDirty(false);
        setSuccess('Chaos stopped early');
      })
      .catch(() => {
        setError('Failed to stop chaos');
      });
  }, []);

  // Form state
  const [service, setService] = useState('');
  const [action, setAction] = useState('');
  const [type, setType] = useState<ChaosRule['type']>('latency');
  const [value, setValue] = useState('');

  useEffect(() => {
    setLoadingState(true);
    api<ChaosState>('/api/chaos')
      .then((data) => {
        setActive(data.active);
        setRules(data.rules || []);
      })
      .catch((err) => {
        setError(`Failed to load chaos state: ${err.message}`);
      })
      .finally(() => setLoadingState(false));
  }, []);

  function handleAddRule() {
    if (!service.trim() || !value.trim()) return;

    const numValue = Number(value);
    if (isNaN(numValue)) return;

    const rule: ChaosRule = {
      service: service.trim(),
      type,
      value: numValue,
    };
    if (action.trim()) {
      rule.action = action.trim();
    }

    setRules([...rules, rule]);
    setDirty(true);
    setSuccess(null);
    setService('');
    setAction('');
    setValue('');
  }

  function handleDeleteRule(index: number) {
    setRules(rules.filter((_, i) => i !== index));
    setDirty(true);
    setSuccess(null);
  }

  function handleToggleActive() {
    setActive(!active);
    setDirty(true);
    setSuccess(null);
  }

  async function handleApply() {
    setApplying(true);
    setError(null);
    setSuccess(null);
    try {
      const result = await api<ChaosState>('/api/chaos', {
        method: 'POST',
        body: JSON.stringify({ active, rules }),
      });
      // Sync with server response in case it normalized anything
      setActive(result.active);
      setRules(result.rules || []);
      setDirty(false);

      // Start timer if activating with a duration
      if (active && duration > 0) {
        setExpiresAt(Date.now() + duration * 1000);
        setSuccess(
          `Chaos rules applied (${rules.length} rule${rules.length !== 1 ? 's' : ''}) — auto-disabling in ${DURATION_OPTIONS.find((o) => o.seconds === duration)?.label ?? `${duration}s`}`,
        );
      } else {
        // Clear timer if deactivating
        if (!active) setExpiresAt(null);
        setSuccess(
          active
            ? `Chaos rules applied (${rules.length} rule${rules.length !== 1 ? 's' : ''} active)`
            : 'Chaos injection disabled',
        );
      }
    } catch (err: any) {
      setError(err.message || 'Failed to apply chaos rules');
    } finally {
      setApplying(false);
    }
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleAddRule();
    }
  }

  function handleApplyPreset(preset: ChaosPreset) {
    const newRules = [...rules, ...preset.rules];
    setRules(newRules);
    setDirty(true);
    setSuccess(`Added preset: ${preset.label}`);
    setError(null);
  }

  return (
    <div class="chaos-view">
      <div class="chaos-header">
        <div class="chaos-header-left">
          <h2 class="chaos-title">Chaos Engineering</h2>
          <span class="chaos-subtitle">Inject failures into service calls</span>
        </div>
        <div class="chaos-header-right">
          <button
            class={`chaos-toggle ${active ? 'chaos-toggle-active' : ''}`}
            onClick={handleToggleActive}
            disabled={loadingState}
          >
            <span class="chaos-toggle-track">
              <span class="chaos-toggle-thumb" />
            </span>
            <span class="chaos-toggle-label">{active ? 'Active' : 'Inactive'}</span>
          </button>
          <select
            class="input chaos-duration-select"
            value={duration}
            onChange={(e) => setDuration(Number((e.target as HTMLSelectElement).value))}
            title="Auto-disable after duration"
          >
            {DURATION_OPTIONS.map((opt) => (
              <option key={opt.seconds} value={opt.seconds}>
                {opt.label}
              </option>
            ))}
          </select>
          <button
            class={`btn btn-primary chaos-apply-btn ${dirty ? '' : 'chaos-apply-clean'}`}
            onClick={handleApply}
            disabled={applying || !dirty}
          >
            {applying ? 'Applying...' : 'Apply Rules'}
          </button>
        </div>
      </div>

      {expiresAt != null && countdown > 500 && (
        <div class="chaos-timer-banner">
          <span class="chaos-timer-text">
            {'\u23F1'} Chaos active: {formatCountdown(countdown)} remaining
          </span>
          <button class="btn btn-ghost chaos-stop-early-btn" onClick={handleStopEarly}>
            Stop early
          </button>
        </div>
      )}

      {error && (
        <div class="chaos-error">
          <span class="chaos-error-icon">!</span>
          {error}
        </div>
      )}

      {success && (
        <div class="chaos-success">
          <span class="chaos-success-icon">&#x2713;</span>
          {success}
        </div>
      )}

      <div class="chaos-body">
        <div class="chaos-presets-section">
          <div class="chaos-presets-title">Presets</div>
          <div class="chaos-presets-grid">
            {CHAOS_PRESETS.map((preset) => (
              <button
                key={preset.label}
                class="chaos-preset-btn"
                onClick={() => handleApplyPreset(preset)}
                title={preset.description}
              >
                <span class="chaos-preset-icon">{preset.icon}</span>
                <span class="chaos-preset-label">{preset.label}</span>
                <span class="chaos-preset-desc">{preset.description}</span>
              </button>
            ))}
          </div>
        </div>

        <div class="chaos-form-section">
          <div class="chaos-form-title">Add Rule</div>
          <div class="chaos-form">
            <div class="chaos-form-field">
              <label class="chaos-label">Service</label>
              <input
                class="input chaos-input"
                type="text"
                placeholder="e.g. s3, dynamodb"
                value={service}
                onInput={(e) => setService((e.target as HTMLInputElement).value)}
                onKeyDown={handleKeyDown}
              />
            </div>
            <div class="chaos-form-field">
              <label class="chaos-label">Action <span class="chaos-optional">(optional)</span></label>
              <input
                class="input chaos-input"
                type="text"
                placeholder="e.g. GetObject"
                value={action}
                onInput={(e) => setAction((e.target as HTMLInputElement).value)}
                onKeyDown={handleKeyDown}
              />
            </div>
            <div class="chaos-form-field">
              <label class="chaos-label">Fault Type</label>
              <select
                class="input chaos-select"
                value={type}
                onChange={(e) => setType((e.target as HTMLSelectElement).value as ChaosRule['type'])}
              >
                {TYPE_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
            </div>
            <div class="chaos-form-field">
              <label class="chaos-label">Value</label>
              <input
                class="input chaos-input chaos-input-value"
                type="number"
                placeholder={type === 'latency' ? '500' : type === 'error' ? '503' : '50'}
                value={value}
                onInput={(e) => setValue((e.target as HTMLInputElement).value)}
                onKeyDown={handleKeyDown}
              />
            </div>
            <div class="chaos-form-field chaos-form-field-action">
              <button
                class="btn btn-primary chaos-add-btn"
                onClick={handleAddRule}
                disabled={!service.trim() || !value.trim()}
              >
                + Add
              </button>
            </div>
          </div>
        </div>

        <div class="chaos-rules-section">
          <div class="chaos-rules-header">
            <span class="chaos-rules-title">Rules</span>
            <span class="chaos-rules-count">{rules.length}</span>
          </div>

          {rules.length === 0 ? (
            <div class="chaos-rules-empty">
              <div class="chaos-rules-empty-icon">&#x26A1;</div>
              <div class="chaos-rules-empty-text">No chaos rules configured</div>
              <div class="chaos-rules-empty-hint">
                Add a rule above to inject latency, errors, or throttling
              </div>
            </div>
          ) : (
            <div class="chaos-rules-table-wrapper">
              <table class="chaos-rules-table">
                <thead>
                  <tr>
                    <th>Service</th>
                    <th>Action</th>
                    <th>Type</th>
                    <th>Value</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {rules.map((rule, i) => (
                    <tr key={i} class="chaos-rule-row">
                      <td>
                        <span class="chaos-service-name">{rule.service}</span>
                      </td>
                      <td>
                        <span class={rule.action ? 'chaos-action-name' : 'chaos-action-any'}>
                          {rule.action || 'All'}
                        </span>
                      </td>
                      <td>
                        <span class={`chaos-type-badge ${typeBadgeClass(rule.type)}`}>
                          {typeLabel(rule.type)}
                        </span>
                      </td>
                      <td>
                        <span class="chaos-value-mono">{valueLabel(rule)}</span>
                      </td>
                      <td>
                        <button
                          class="chaos-delete-btn"
                          onClick={() => handleDeleteRule(i)}
                          title="Remove rule"
                        >
                          &times;
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
