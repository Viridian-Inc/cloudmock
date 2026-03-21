import { useState, useEffect } from 'preact/hooks';
import { CloudIcon, SearchIcon, SunIcon, MoonIcon } from './Icons';

interface HeaderProps {
  connected: boolean;
  health: any;
  onOpenPalette: () => void;
}

export function Header({ connected, health, onOpenPalette }: HeaderProps) {
  const [dark, setDark] = useState(() => {
    if (typeof window !== 'undefined') {
      return document.documentElement.dataset.theme === 'dark' ||
        localStorage.getItem('theme') === 'dark';
    }
    return false;
  });

  useEffect(() => {
    document.documentElement.dataset.theme = dark ? 'dark' : 'light';
    localStorage.setItem('theme', dark ? 'dark' : 'light');
  }, [dark]);

  return (
    <header class="header">
      <div class="header-logo">
        <CloudIcon />
        <span>cloudmock</span>
      </div>
      <div class="header-spacer" />
      <div class="header-badge" id="health-badge">
        <span
          class={`dot ${health && health.status === 'healthy' ? 'dot-green' : 'dot-yellow'}`}
          id="health-dot"
        />
        <span>{health ? (health.status === 'healthy' ? 'Healthy' : 'Degraded') : '...'}</span>
      </div>
      <div class="header-badge" id="sse-badge">
        <span class={`dot ${connected ? 'dot-green' : 'dot-red'}`} id="sse-dot" />
        <span>{connected ? 'Connected' : 'Disconnected'}</span>
      </div>
      <button
        class="header-theme-toggle"
        onClick={() => setDark(d => !d)}
        title={dark ? 'Switch to light mode' : 'Switch to dark mode'}
      >
        {dark ? <SunIcon /> : <MoonIcon />}
      </button>
      <button class="cmd-k-btn" onClick={onOpenPalette}>
        <SearchIcon /> Search <kbd>Cmd+K</kbd>
      </button>
    </header>
  );
}
