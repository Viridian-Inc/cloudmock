import { useState, useEffect, useCallback } from 'preact/hooks';
import { SplitPanel } from '../../components/panels/split-panel';
import { api } from '../../lib/api';
import './monitors.css';

type MonitorStatus = 'ok' | 'warning' | 'alert' | 'no_data' | 'muted';
type MetricName = 'p50' | 'p95' | 'p99' | 'error_rate' | 'request_count' | 'avg_latency';
type Operator = '>' | '>=' | '<' | '<=' | '==' | '!=';

interface AlertEvent {
  id: string;
  monitorId: string;
  status: MonitorStatus;
  value: number;
  threshold: number;
  timestamp: string;
  message: string;
}

interface Monitor {
  id: string;
  name: string;
  service: string;
  metric: MetricName;
  operator: Operator;
  criticalThreshold: number;
  warningThreshold?: number;
  evaluationWindow: string;  // e.g. "5m", "15m", "1h"
  notificationChannels: string[];
  status: MonitorStatus;
  lastValue?: number;
  lastChecked?: string;
  muted: boolean;
  createdAt: string;
  updatedAt: string;
  alertHistory: AlertEvent[];
}

type FilterTab = 'all' | MonitorStatus;

const FILTER_TABS: { key: FilterTab; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'ok', label: 'OK' },
  { key: 'warning', label: 'Warning' },
  { key: 'alert', label: 'Alert' },
  { key: 'no_data', label: 'No Data' },
  { key: 'muted', label: 'Muted' },
];

const METRICS: { value: MetricName; label: string }[] = [
  { value: 'p50', label: 'P50 Latency' },
  { value: 'p95', label: 'P95 Latency' },
  { value: 'p99', label: 'P99 Latency' },
  { value: 'error_rate', label: 'Error Rate' },
  { value: 'request_count', label: 'Request Count' },
  { value: 'avg_latency', label: 'Avg Latency' },
];

const OPERATORS: { value: Operator; label: string }[] = [
  { value: '>', label: '>' },
  { value: '>=', label: '>=' },
  { value: '<', label: '<' },
  { value: '<=', label: '<=' },
  { value: '==', label: '==' },
  { value: '!=', label: '!=' },
];

const EVAL_WINDOWS = ['1m', '5m', '15m', '30m', '1h'];

const NOTIFICATION_CHANNELS = ['email', 'slack', 'pagerduty', 'webhook'];

function statusIcon(status: MonitorStatus): string {
  switch (status) {
    case 'ok': return '\u2713';
    case 'warning': return '!';
    case 'alert': return '!!';
    case 'muted': return '\u2014';
    default: return '?';
  }
}

function formatTime(ts: string): string {
  const d = new Date(ts);
  if (isNaN(d.getTime())) return '--';
  return d.toLocaleString();
}

function relativeTime(ts: string): string {
  const d = new Date(ts);
  if (isNaN(d.getTime())) return '';
  const diff = Date.now() - d.getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

function newId(): string {
  return `mon-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

// ===== Persistence =====

const MONITORS_KEY = 'neureaux:monitors';

function loadMonitors(): Monitor[] {
  try {
    const raw = localStorage.getItem(MONITORS_KEY);
    return raw ? JSON.parse(raw) : [];
  } catch {
    return [];
  }
}

function saveMonitors(monitors: Monitor[]): void {
  localStorage.setItem(MONITORS_KEY, JSON.stringify(monitors));
}

// ===== Monitor List =====

function MonitorList({
  monitors,
  selectedId,
  onSelect,
  onNew,
}: {
  monitors: Monitor[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  onNew: () => void;
}) {
  const [filter, setFilter] = useState<FilterTab>('all');

  const filtered = monitors.filter((m) => {
    if (filter === 'all') return true;
    if (filter === 'muted') return m.muted;
    return m.status === filter;
  });

  const counts: Record<string, number> = {
    all: monitors.length,
    ok: monitors.filter((m) => m.status === 'ok' && !m.muted).length,
    warning: monitors.filter((m) => m.status === 'warning').length,
    alert: monitors.filter((m) => m.status === 'alert').length,
    no_data: monitors.filter((m) => m.status === 'no_data').length,
    muted: monitors.filter((m) => m.muted).length,
  };

  return (
    <div class="monitor-list">
      <div class="monitor-list-header">
        <div class="monitor-list-title-row">
          <div class="monitor-list-title">
            Monitors
            {counts.alert > 0 && (
              <span class="monitor-count-badge">{counts.alert}</span>
            )}
          </div>
          <button class="btn btn-primary monitor-new-btn" onClick={onNew}>
            + New
          </button>
        </div>
        <div class="monitor-list-filters">
          {FILTER_TABS.map((tab) => (
            <button
              key={tab.key}
              class={`monitor-filter-btn ${filter === tab.key ? 'active' : ''}`}
              onClick={() => setFilter(tab.key)}
            >
              {tab.label}
              {counts[tab.key] > 0 && (
                <span class="monitor-filter-count">{counts[tab.key]}</span>
              )}
            </button>
          ))}
        </div>
      </div>
      <div class="monitor-list-body">
        {filtered.length === 0 && monitors.length === 0 && (
          <div class="monitor-list-empty">
            <p>No monitors yet.</p>
            <button class="btn btn-primary" onClick={onNew}>Create your first monitor</button>
          </div>
        )}
        {filtered.length === 0 && monitors.length > 0 && (
          <div class="monitor-list-empty">No monitors match this filter</div>
        )}
        {filtered.map((m) => (
          <div
            key={m.id}
            class={`monitor-row ${m.id === selectedId ? 'monitor-row-selected' : ''}`}
            onClick={() => onSelect(m.id)}
          >
            <span class={`monitor-status-dot status-${m.muted ? 'muted' : m.status}`}>
              {statusIcon(m.muted ? 'muted' : m.status)}
            </span>
            <div class="monitor-row-content">
              <div class="monitor-row-name">{m.name}</div>
              <div class="monitor-row-meta">
                <span class="monitor-row-service">{m.service}</span>
                <span class="monitor-row-metric">{m.metric}</span>
                {m.lastChecked && (
                  <span class="monitor-row-time">{relativeTime(m.lastChecked)}</span>
                )}
              </div>
            </div>
            {m.lastValue !== undefined && (
              <span class="monitor-row-value">{m.lastValue}</span>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

// ===== Monitor Form =====

function MonitorForm({
  monitor,
  services,
  onSave,
  onCancel,
}: {
  monitor?: Monitor;
  services: string[];
  onSave: (monitor: Monitor) => void;
  onCancel: () => void;
}) {
  const isNew = !monitor;

  const [name, setName] = useState(monitor?.name ?? '');
  const [service, setService] = useState(monitor?.service ?? services[0] ?? '');
  const [metric, setMetric] = useState<MetricName>(monitor?.metric ?? 'p99');
  const [operator, setOperator] = useState<Operator>(monitor?.operator ?? '>');
  const [criticalThreshold, setCriticalThreshold] = useState(
    monitor?.criticalThreshold?.toString() ?? '',
  );
  const [warningThreshold, setWarningThreshold] = useState(
    monitor?.warningThreshold?.toString() ?? '',
  );
  const [evaluationWindow, setEvaluationWindow] = useState(
    monitor?.evaluationWindow ?? '5m',
  );
  const [channels, setChannels] = useState<Set<string>>(
    new Set(monitor?.notificationChannels ?? []),
  );

  const toggleChannel = (ch: string) => {
    setChannels((prev) => {
      const next = new Set(prev);
      if (next.has(ch)) next.delete(ch);
      else next.add(ch);
      return next;
    });
  };

  const handleSave = () => {
    const now = new Date().toISOString();
    const saved: Monitor = {
      id: monitor?.id ?? newId(),
      name: name.trim() || `${service} ${metric} monitor`,
      service,
      metric,
      operator,
      criticalThreshold: parseFloat(criticalThreshold) || 0,
      warningThreshold: warningThreshold ? parseFloat(warningThreshold) : undefined,
      evaluationWindow,
      notificationChannels: [...channels],
      status: monitor?.status ?? 'no_data',
      lastValue: monitor?.lastValue,
      lastChecked: monitor?.lastChecked,
      muted: monitor?.muted ?? false,
      createdAt: monitor?.createdAt ?? now,
      updatedAt: now,
      alertHistory: monitor?.alertHistory ?? [],
    };
    onSave(saved);
  };

  return (
    <div class="monitor-form">
      <div class="monitor-form-header">
        <h3 class="monitor-form-title">
          {isNew ? 'Create Monitor' : 'Edit Monitor'}
        </h3>
      </div>
      <div class="monitor-form-body">
        <div class="monitor-form-field">
          <label class="monitor-form-label">Name</label>
          <input
            class="monitor-form-input"
            type="text"
            value={name}
            onInput={(e) => setName((e.target as HTMLInputElement).value)}
            placeholder="Monitor name"
          />
        </div>

        <div class="monitor-form-field">
          <label class="monitor-form-label">Service</label>
          <select
            class="monitor-form-select"
            value={service}
            onChange={(e) => setService((e.target as HTMLSelectElement).value)}
          >
            {services.map((s) => (
              <option key={s} value={s}>{s}</option>
            ))}
          </select>
        </div>

        <div class="monitor-form-field">
          <label class="monitor-form-label">Metric</label>
          <select
            class="monitor-form-select"
            value={metric}
            onChange={(e) => setMetric((e.target as HTMLSelectElement).value as MetricName)}
          >
            {METRICS.map((m) => (
              <option key={m.value} value={m.value}>{m.label}</option>
            ))}
          </select>
        </div>

        <div class="monitor-form-row">
          <div class="monitor-form-field monitor-form-field-narrow">
            <label class="monitor-form-label">Operator</label>
            <select
              class="monitor-form-select"
              value={operator}
              onChange={(e) => setOperator((e.target as HTMLSelectElement).value as Operator)}
            >
              {OPERATORS.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </select>
          </div>
          <div class="monitor-form-field">
            <label class="monitor-form-label">Critical Threshold</label>
            <input
              class="monitor-form-input"
              type="number"
              value={criticalThreshold}
              onInput={(e) => setCriticalThreshold((e.target as HTMLInputElement).value)}
              placeholder="e.g. 500"
            />
          </div>
        </div>

        <div class="monitor-form-field">
          <label class="monitor-form-label">Warning Threshold (optional)</label>
          <input
            class="monitor-form-input"
            type="number"
            value={warningThreshold}
            onInput={(e) => setWarningThreshold((e.target as HTMLInputElement).value)}
            placeholder="e.g. 200"
          />
        </div>

        <div class="monitor-form-field">
          <label class="monitor-form-label">Evaluation Window</label>
          <select
            class="monitor-form-select"
            value={evaluationWindow}
            onChange={(e) => setEvaluationWindow((e.target as HTMLSelectElement).value)}
          >
            {EVAL_WINDOWS.map((w) => (
              <option key={w} value={w}>{w}</option>
            ))}
          </select>
        </div>

        <div class="monitor-form-field">
          <label class="monitor-form-label">Notification Channels</label>
          <div class="monitor-form-channels">
            {NOTIFICATION_CHANNELS.map((ch) => (
              <label key={ch} class="monitor-form-channel-label">
                <input
                  type="checkbox"
                  class="monitor-form-checkbox"
                  checked={channels.has(ch)}
                  onChange={() => toggleChannel(ch)}
                />
                {ch}
              </label>
            ))}
          </div>
        </div>
      </div>

      <div class="monitor-form-footer">
        <button class="btn" onClick={onCancel}>Cancel</button>
        <button class="btn btn-primary" onClick={handleSave}>
          {isNew ? 'Create' : 'Save'}
        </button>
      </div>
    </div>
  );
}

// ===== Monitor Detail =====

function MonitorDetail({
  monitor,
  onEdit,
  onMute,
  onDelete,
}: {
  monitor: Monitor | null;
  onEdit: (id: string) => void;
  onMute: (id: string) => void;
  onDelete: (id: string) => void;
}) {
  if (!monitor) {
    return (
      <div class="monitor-detail monitor-detail-empty">
        <div class="monitor-detail-placeholder">
          Select a monitor to view details
        </div>
      </div>
    );
  }

  const metricLabel = METRICS.find((m) => m.value === monitor.metric)?.label ?? monitor.metric;

  return (
    <div class="monitor-detail">
      <div class="monitor-detail-header">
        <div class="monitor-detail-title-row">
          <span class={`monitor-status-dot status-${monitor.muted ? 'muted' : monitor.status}`}>
            {statusIcon(monitor.muted ? 'muted' : monitor.status)}
          </span>
          <h3 class="monitor-detail-title">{monitor.name}</h3>
        </div>
        <div class="monitor-detail-actions">
          <button class="btn" onClick={() => onMute(monitor.id)}>
            {monitor.muted ? 'Unmute' : 'Mute'}
          </button>
          <button class="btn" onClick={() => onEdit(monitor.id)}>
            Edit
          </button>
          <button class="btn monitor-delete-btn" onClick={() => onDelete(monitor.id)}>
            Delete
          </button>
        </div>
      </div>

      <div class="monitor-detail-body">
        <div class="monitor-detail-field">
          <span class="monitor-field-label">Status</span>
          <span class={`monitor-status-badge status-${monitor.muted ? 'muted' : monitor.status}`}>
            {monitor.muted ? 'muted' : monitor.status.replace('_', ' ')}
          </span>
        </div>
        <div class="monitor-detail-field">
          <span class="monitor-field-label">Service</span>
          <span class="monitor-field-value monitor-field-accent">{monitor.service}</span>
        </div>
        <div class="monitor-detail-field">
          <span class="monitor-field-label">Metric</span>
          <span class="monitor-field-value">{metricLabel}</span>
        </div>
        <div class="monitor-detail-field">
          <span class="monitor-field-label">Condition</span>
          <span class="monitor-field-value monitor-field-mono">
            {monitor.metric} {monitor.operator} {monitor.criticalThreshold}
            {monitor.warningThreshold !== undefined && (
              <span class="monitor-field-sub">
                {' '}(warn: {monitor.warningThreshold})
              </span>
            )}
          </span>
        </div>
        <div class="monitor-detail-field">
          <span class="monitor-field-label">Window</span>
          <span class="monitor-field-value">{monitor.evaluationWindow}</span>
        </div>
        <div class="monitor-detail-field">
          <span class="monitor-field-label">Channels</span>
          <span class="monitor-field-value">
            {monitor.notificationChannels.length > 0
              ? monitor.notificationChannels.join(', ')
              : 'None'}
          </span>
        </div>
        {monitor.lastValue !== undefined && (
          <div class="monitor-detail-field">
            <span class="monitor-field-label">Last Value</span>
            <span class="monitor-field-value monitor-field-mono">
              {monitor.lastValue}
            </span>
          </div>
        )}
        {monitor.lastChecked && (
          <div class="monitor-detail-field">
            <span class="monitor-field-label">Last Checked</span>
            <span class="monitor-field-value">{formatTime(monitor.lastChecked)}</span>
          </div>
        )}

        {/* Alert History */}
        <div class="monitor-detail-section">
          <h4 class="monitor-section-heading">
            Alert History ({monitor.alertHistory.length})
          </h4>
          {monitor.alertHistory.length === 0 ? (
            <div class="monitor-history-empty">No alerts recorded</div>
          ) : (
            <div class="monitor-history-table-wrapper">
              <table class="monitor-history-table">
                <thead>
                  <tr>
                    <th>Status</th>
                    <th>Value</th>
                    <th>Threshold</th>
                    <th>Time</th>
                    <th>Message</th>
                  </tr>
                </thead>
                <tbody>
                  {monitor.alertHistory.map((evt) => (
                    <tr key={evt.id} class="monitor-history-row">
                      <td>
                        <span class={`monitor-status-badge status-${evt.status}`}>
                          {evt.status}
                        </span>
                      </td>
                      <td class="monitor-history-mono">{evt.value}</td>
                      <td class="monitor-history-mono">{evt.threshold}</td>
                      <td class="monitor-history-time">{relativeTime(evt.timestamp)}</td>
                      <td class="monitor-history-msg">{evt.message}</td>
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

// ===== Main View =====

export function MonitorsView() {
  const [monitors, setMonitors] = useState<Monitor[]>(loadMonitors);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [editing, setEditing] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [services, setServices] = useState<string[]>([]);

  // Persist monitors to localStorage on change
  useEffect(() => {
    saveMonitors(monitors);
  }, [monitors]);

  // Fetch services from the IaC config API
  useEffect(() => {
    api<any[]>('/api/services')
      .then((svcList) => {
        if (Array.isArray(svcList) && svcList.length > 0) {
          setServices(svcList.map((s) => s.name || s.Name).filter(Boolean));
        }
      })
      .catch(() => {
        // Try topology config as fallback
        api<{ nodes: any[] }>('/api/topology/config')
          .then((config) => setServices((config.nodes || []).map((n: any) => n.label || n.id)))
          .catch(() => setServices([]));
      });
  }, []);

  const selected = selectedId
    ? monitors.find((m) => m.id === selectedId) ?? null
    : null;

  const handleSave = useCallback((monitor: Monitor) => {
    setMonitors((prev) => {
      const idx = prev.findIndex((m) => m.id === monitor.id);
      if (idx >= 0) {
        return prev.map((m) => (m.id === monitor.id ? monitor : m));
      }
      return [...prev, monitor];
    });
    setEditing(null);
    setCreating(false);
    setSelectedId(monitor.id);
  }, []);

  const handleMute = useCallback((id: string) => {
    setMonitors((prev) =>
      prev.map((m) =>
        m.id === id ? { ...m, muted: !m.muted, updatedAt: new Date().toISOString() } : m,
      ),
    );
  }, []);

  const handleDelete = useCallback((id: string) => {
    setMonitors((prev) => prev.filter((m) => m.id !== id));
    if (selectedId === id) setSelectedId(null);
  }, [selectedId]);

  const handleNew = useCallback(() => {
    setCreating(true);
    setEditing(null);
    setSelectedId(null);
  }, []);

  const handleEdit = useCallback((id: string) => {
    setEditing(id);
    setCreating(false);
  }, []);

  const handleCancelForm = useCallback(() => {
    setEditing(null);
    setCreating(false);
  }, []);

  // Right panel content
  let rightPanel: preact.JSX.Element;
  if (creating) {
    rightPanel = (
      <MonitorForm
        services={services}
        onSave={handleSave}
        onCancel={handleCancelForm}
      />
    );
  } else if (editing) {
    const editMon = monitors.find((m) => m.id === editing);
    rightPanel = (
      <MonitorForm
        monitor={editMon}
        services={services}
        onSave={handleSave}
        onCancel={handleCancelForm}
      />
    );
  } else {
    rightPanel = (
      <MonitorDetail
        monitor={selected}
        onEdit={handleEdit}
        onMute={handleMute}
        onDelete={handleDelete}
      />
    );
  }

  return (
    <div class="monitors-view">
      <SplitPanel
        initialSplit={0.38}
        direction="horizontal"
        minSize={260}
        id="monitors"
        left={
          <MonitorList
            monitors={monitors}
            selectedId={selectedId}
            onSelect={setSelectedId}
            onNew={handleNew}
          />
        }
        right={rightPanel}
      />
    </div>
  );
}
