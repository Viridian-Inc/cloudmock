export function statusClass(code: number): string {
  if (code >= 500) return 'status-5xx';
  if (code >= 400) return 'status-4xx';
  if (code >= 300) return 'status-3xx';
  return 'status-2xx';
}

interface StatusBadgeProps {
  code: number;
  style?: string;
}

export function StatusBadge({ code, style }: StatusBadgeProps) {
  return <span class={`status-pill ${statusClass(code)}`} style={style}>{code}</span>;
}
