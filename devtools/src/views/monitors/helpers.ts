export type MonitorStatus = 'ok' | 'warning' | 'alert' | 'no_data' | 'muted';
export type Operator = '>' | '>=' | '<' | '<=' | '==' | '!=';

export interface Monitor {
  id: string;
  name: string;
  service: string;
  metric: string;
  operator: Operator;
  criticalThreshold: number;
  warningThreshold?: number;
  evaluationWindow: string;
  notificationChannels: string[];
  status: MonitorStatus;
  lastValue?: number;
  lastChecked?: string;
  muted: boolean;
  createdAt: string;
  updatedAt: string;
  alertHistory: AlertEvent[];
}

export interface AlertEvent {
  id: string;
  monitorId: string;
  status: MonitorStatus;
  value: number;
  threshold: number;
  timestamp: string;
  message: string;
}

const MONITORS_KEY = 'neureaux:monitors';

export function loadMonitors(): Monitor[] {
  try {
    const raw = localStorage.getItem(MONITORS_KEY);
    return raw ? JSON.parse(raw) : [];
  } catch {
    return [];
  }
}

export function saveMonitors(monitors: Monitor[]): void {
  localStorage.setItem(MONITORS_KEY, JSON.stringify(monitors));
}

/**
 * Evaluate a threshold comparison.
 * Returns true when the value meets the condition.
 */
export function evaluateThreshold(value: number, operator: Operator, threshold: number): boolean {
  switch (operator) {
    case '>': return value > threshold;
    case '>=': return value >= threshold;
    case '<': return value < threshold;
    case '<=': return value <= threshold;
    case '==': return value === threshold;
    case '!=': return value !== threshold;
  }
}

/**
 * Determine the effective monitor status based on current value and thresholds.
 * Used for evaluation logic.
 */
export function evaluateMonitorStatus(
  value: number | undefined,
  operator: Operator,
  criticalThreshold: number,
  warningThreshold: number | undefined,
  muted: boolean,
): MonitorStatus {
  if (muted) return 'muted';
  if (value === undefined) return 'no_data';
  if (evaluateThreshold(value, operator, criticalThreshold)) return 'alert';
  if (warningThreshold !== undefined && evaluateThreshold(value, operator, warningThreshold)) return 'warning';
  return 'ok';
}

export function statusIcon(status: MonitorStatus): string {
  switch (status) {
    case 'ok': return '\u2713';
    case 'warning': return '!';
    case 'alert': return '!!';
    case 'muted': return '\u2014';
    default: return '?';
  }
}

export function formatTime(ts: string): string {
  const d = new Date(ts);
  if (isNaN(d.getTime())) return '--';
  return d.toLocaleString();
}

export function relativeTime(ts: string): string {
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

/**
 * Check if a mute expiry timestamp has passed.
 * null/undefined means indefinite (no expiry).
 */
export function isMuteExpired(expiresAt: string | null | undefined): boolean {
  if (!expiresAt) return false;
  return new Date(expiresAt).getTime() <= Date.now();
}

/** Filter monitors by tab filter. */
export function filterMonitors(monitors: Monitor[], filter: string): Monitor[] {
  if (filter === 'all') return monitors;
  if (filter === 'muted') return monitors.filter((m) => m.muted);
  return monitors.filter((m) => m.status === filter);
}
