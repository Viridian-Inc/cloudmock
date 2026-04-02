export interface Incident {
  id: string;
  title: string;
  service: string;
  severity: 'critical' | 'warning' | 'info';
  status: 'open' | 'acknowledged' | 'resolved';
  message: string;
  timestamp: string;
  acknowledged_at?: string;
  resolved_at?: string;
  details?: Record<string, unknown>;
  first_seen?: string;
  last_seen?: string;
  affected_services?: string[];
}

export const SEVERITY_COLORS: Record<string, string> = {
  critical: '#ff4e5e',
  high: '#F7711E',
  warning: '#fad065',
  medium: '#fad065',
  low: '#538eff',
  info: '#538eff',
};

/** Group incidents close in time (within groupWindowMs) */
export interface IncidentGroup {
  incidents: Incident[];
  time: number; // average timestamp of group
}

export function groupIncidents(incidents: Incident[], groupWindowMs: number): IncidentGroup[] {
  if (incidents.length === 0) return [];

  const sorted = [...incidents].sort(
    (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime(),
  );
  const groups: IncidentGroup[] = [];
  let current: Incident[] = [sorted[0]];
  let lastTs = new Date(sorted[0].timestamp).getTime();

  for (let i = 1; i < sorted.length; i++) {
    const ts = new Date(sorted[i].timestamp).getTime();
    if (ts - lastTs <= groupWindowMs) {
      current.push(sorted[i]);
    } else {
      const avgTime = current.reduce((sum, inc) => sum + new Date(inc.timestamp).getTime(), 0) / current.length;
      groups.push({ incidents: current, time: avgTime });
      current = [sorted[i]];
    }
    lastTs = ts;
  }
  const avgTime = current.reduce((sum, inc) => sum + new Date(inc.timestamp).getTime(), 0) / current.length;
  groups.push({ incidents: current, time: avgTime });

  return groups;
}

/** Filter incidents by status */
export function filterByStatus(incidents: Incident[], filter: string): Incident[] {
  if (filter === 'all') return incidents;
  return incidents.filter((inc) => inc.status === filter);
}

/** Sort incidents by severity (critical first) */
export function sortBySeverity(incidents: Incident[]): Incident[] {
  const order: Record<string, number> = { critical: 0, warning: 1, info: 2 };
  return [...incidents].sort((a, b) => (order[a.severity] ?? 99) - (order[b.severity] ?? 99));
}

export function severityIcon(severity: string): string {
  switch (severity) {
    case 'critical': return '!!';
    case 'warning': return '!';
    default: return 'i';
  }
}

export function formatTime(ts: string): string {
  const d = new Date(ts);
  if (isNaN(d.getTime())) return '--:--:--';
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
