import { useState, useEffect } from 'preact/hooks';
import { EnvPicker } from '../env-picker/env-picker';
import './source-bar.css';

interface ConnectedSource {
  runtime: string;
  app_name: string;
  pid?: number;
  connected_at: number;
}

const RUNTIME_COLORS: Record<string, string> = {
  node: 'rgba(34,197,94,0.15)',
  swift: 'rgba(59,130,246,0.15)',
  kotlin: 'rgba(245,158,11,0.15)',
  dart: 'rgba(56,189,248,0.15)',
};

const RUNTIME_BORDERS: Record<string, string> = {
  node: 'rgba(34,197,94,0.3)',
  swift: 'rgba(59,130,246,0.3)',
  kotlin: 'rgba(245,158,11,0.3)',
  dart: 'rgba(56,189,248,0.3)',
};

export function SourceBar() {
  const [sources, setSources] = useState<ConnectedSource[]>([]);
  // TODO: wire env to API calls when cloud backend is connected
  const [env, setEnv] = useState('local');

  // TODO: poll /api/sources when admin API exposes connected SDK clients
  useEffect(() => {
    // Sources will be populated via admin API polling in future
  }, []);

  return (
    <div class="source-bar">
      <span class="source-bar-label">Sources:</span>
      <div class="source-chip">
        <span class="source-chip-dot connected" />
        <span class="source-chip-icon">☁️</span>
        <span class="source-chip-name">cloudmock</span>
      </div>
      {sources.map((s) => (
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
            {s.runtime} · {s.app_name}
          </span>
        </div>
      ))}
      <button class="source-add-btn" title="Sources connect automatically via SDK">
        + Add
      </button>
      <div class="source-bar-spacer" />
      <EnvPicker value={env} onChange={setEnv} />
    </div>
  );
}
