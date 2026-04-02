export const DURATION_OPTIONS: { label: string; seconds: number }[] = [
  { label: 'Indefinite', seconds: 0 },
  { label: '1 min', seconds: 60 },
  { label: '5 min', seconds: 300 },
  { label: '15 min', seconds: 900 },
  { label: '30 min', seconds: 1800 },
  { label: '1 hour', seconds: 3600 },
];

export function formatCountdown(seconds: number): string {
  if (seconds <= 0) return '0s';
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m === 0) return `${s}s`;
  return `${m}m ${s.toString().padStart(2, '0')}s`;
}

export function typeLabel(type: 'latency' | 'error' | 'throttle'): string {
  switch (type) {
    case 'latency': return 'Latency';
    case 'error': return 'Error';
    case 'throttle': return 'Throttle';
  }
}

export interface ChaosRule {
  service: string;
  action?: string;
  type: 'latency' | 'error' | 'throttle';
  value: number;
}

export function valueLabel(rule: ChaosRule): string {
  switch (rule.type) {
    case 'latency': return `${rule.value}ms`;
    case 'error': return `HTTP ${rule.value}`;
    case 'throttle': return `${rule.value}%`;
  }
}

/** Check if a chaos session has expired */
export function isExpired(expiresAt: number | null): boolean {
  if (expiresAt == null) return false;
  return Date.now() >= expiresAt;
}

/** Compute remaining seconds until expiry */
export function remainingSeconds(expiresAt: number | null): number {
  if (expiresAt == null) return 0;
  return Math.max(0, Math.ceil((expiresAt - Date.now()) / 1000));
}
