import { SearchIcon } from './Icons';

interface HeaderProps {
  connected: boolean;
  health: any;
  onOpenPalette: () => void;
}

export function Header({ connected, health, onOpenPalette }: HeaderProps) {
  return (
    <div class="page-header" style={{ background: 'var(--bg-primary)', borderBottom: '1px solid var(--border-subtle)' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
        <div class="header-badge" id="health-badge">
          <span
            class={`dot ${health && health.status === 'healthy' ? 'dot-green' : 'dot-yellow'}`}
            id="health-dot"
          />
          <span>{health ? (health.status === 'healthy' ? 'Healthy' : 'Degraded') : '...'}</span>
        </div>
        <div class="header-badge" id="sse-badge">
          <span class={`dot ${connected ? 'dot-green' : 'dot-red'}`} id="sse-dot" />
          <span>{connected ? 'Live' : 'Disconnected'}</span>
        </div>
      </div>
      <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
        <div class="search-box" onClick={onOpenPalette}>
          <SearchIcon />
          <span>Search...</span>
          <span class="search-kbd">&#8984;K</span>
        </div>
      </div>
    </div>
  );
}
