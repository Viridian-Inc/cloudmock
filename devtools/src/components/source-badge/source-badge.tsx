import './source-badge.css';

export function SourceBadge({ source }: { source: string }) {
  const label = source === 'local' ? 'local' : source === 'agent-sdk' ? 'sdk' : source === 'agent-proxy' ? 'proxy' : source;
  const colorClass = source === 'local' ? 'source-local' : 'source-cloud';

  return <span class={`source-badge ${colorClass}`}>{label}</span>;
}
