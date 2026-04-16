import { useState, useEffect, useCallback } from 'preact/hooks';
import { api } from '../../lib/api';
import { EnvPicker } from '../env-picker/env-picker';
import { TipBanner } from '../tip-banner/tip-banner';
import './source-bar.css';

interface ConnectedSource {
  runtime: string;
  app_name: string;
  pid?: number;
  connected_at: string;
}

interface SourceStatus {
  tcp_sources: ConnectedSource[];
  http_sources: string[];
  total_events: number;
}

const RUNTIME_COLORS: Record<string, string> = {
  node: 'rgba(34,197,94,0.15)',
  go: 'rgba(0,173,216,0.15)',
  python: 'rgba(255,212,59,0.15)',
  swift: 'rgba(59,130,246,0.15)',
  kotlin: 'rgba(245,158,11,0.15)',
  dart: 'rgba(56,189,248,0.15)',
  java: 'rgba(236,112,99,0.15)',
  rust: 'rgba(222,165,132,0.15)',
};

const RUNTIME_BORDERS: Record<string, string> = {
  node: 'rgba(34,197,94,0.3)',
  go: 'rgba(0,173,216,0.3)',
  python: 'rgba(255,212,59,0.3)',
  swift: 'rgba(59,130,246,0.3)',
  kotlin: 'rgba(245,158,11,0.3)',
  dart: 'rgba(56,189,248,0.3)',
  java: 'rgba(236,112,99,0.3)',
  rust: 'rgba(222,165,132,0.3)',
};

export function SourceBar() {
  const [sources, setSources] = useState<ConnectedSource[]>([]);
  const [httpSources, setHttpSources] = useState<string[]>([]);
  const [eventCount, setEventCount] = useState(0);
  const [env, setEnv] = useState('local');
  const [showAddForm, setShowAddForm] = useState(false);
  const [addUrl, setAddUrl] = useState('');
  const [addName, setAddName] = useState('');

  // Poll /api/source/status every 5s to discover connected SDKs
  useEffect(() => {
    let cancelled = false;

    async function poll() {
      while (!cancelled) {
        try {
          const status = await api<SourceStatus>('/api/source/status');
          if (!cancelled) {
            setSources(status.tcp_sources || []);
            setHttpSources(status.http_sources || []);
            setEventCount(status.total_events || 0);
          }
        } catch {
          // Admin API unreachable — keep showing last known sources
        }
        await new Promise((r) => setTimeout(r, 5000));
      }
    }

    poll();
    return () => { cancelled = true; };
  }, []);

  const handleAddSource = useCallback(async () => {
    if (!addUrl.trim()) return;
    const name = addName.trim() || new URL(addUrl).hostname;
    try {
      // Register as an HTTP source by sending a registration event
      await api('/api/source/events', {
        method: 'POST',
        body: JSON.stringify({
          type: 'source:register',
          source: name,
          runtime: 'custom',
          timestamp: Date.now(),
          data: { url: addUrl.trim() },
        }),
      });
      setShowAddForm(false);
      setAddUrl('');
      setAddName('');
    } catch (e) {
      console.error('[SourceBar] Failed to add source:', e);
    }
  }, [addUrl, addName]);

  const allSources = [
    ...sources,
    ...httpSources.map((name) => ({ app_name: name, runtime: 'http', connected_at: '' })),
  ];

  return (
    <div class="source-bar">
      <span class="source-bar-label">Sources</span>
      <div class="source-chip">
        <span class="source-chip-dot connected" />
        <span class="source-chip-icon">☁️</span>
        <span class="source-chip-name">cloudmock</span>
        {eventCount > 0 && (
          <span class="source-chip-count">{eventCount.toLocaleString()}</span>
        )}
      </div>
      {allSources.map((s) => (
        <div
          key={s.app_name}
          class="source-chip"
          style={{
            background: RUNTIME_COLORS[s.runtime] || 'rgba(255,255,255,0.08)',
            borderColor: RUNTIME_BORDERS[s.runtime] || 'rgba(255,255,255,0.15)',
          }}
        >
          <span class="source-chip-dot connected" />
          <span class="source-chip-name">
            {s.runtime !== 'http' ? `${s.runtime} · ` : ''}{s.app_name}
          </span>
        </div>
      ))}
      <button
        class="source-add-btn"
        onClick={() => setShowAddForm(!showAddForm)}
        title="Add a source endpoint"
      >
        +
      </button>
      {showAddForm && (
        <div class="source-add-form">
          <input
            class="input source-add-input"
            type="text"
            placeholder="App name"
            value={addName}
            onInput={(e) => setAddName((e.target as HTMLInputElement).value)}
          />
          <input
            class="input source-add-input"
            type="text"
            placeholder="http://localhost:3000"
            value={addUrl}
            onInput={(e) => setAddUrl((e.target as HTMLInputElement).value)}
            onKeyDown={(e) => { if (e.key === 'Enter') handleAddSource(); }}
          />
          <button class="btn btn-primary source-add-submit" onClick={handleAddSource}>
            Add
          </button>
        </div>
      )}
      <div class="source-bar-spacer" />
      <TipBanner />
      <EnvPicker value={env} onChange={setEnv} />
    </div>
  );
}
