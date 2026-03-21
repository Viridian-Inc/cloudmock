export function fmtTime(ts: string | undefined | null): string {
  if (!ts) return '';
  const d = new Date(ts);
  return d.toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

export function fmtDuration(ms: number | undefined | null): string {
  if (ms === undefined || ms === null) return '';
  if (ms < 1) return '<1ms';
  if (ms < 1000) return ms + 'ms';
  return (ms / 1000).toFixed(1) + 's';
}

export function copyToClipboard(text: string): void {
  navigator.clipboard.writeText(text).catch(() => {});
}
