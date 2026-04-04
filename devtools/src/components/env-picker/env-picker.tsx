import { useState, useEffect } from 'preact/hooks';
import { api } from '../../lib/api';
import './env-picker.css';

interface Environment {
  name: string;
  source: string; // "local" or "cloud"
}

const LOCAL_ENV: Environment = { name: 'Local', source: 'local' };
const ALL_ENV: Environment = { name: 'All', source: 'all' };

export function EnvPicker({
  value,
  onChange
}: {
  value: string;
  onChange: (env: string) => void;
}) {
  const [environments, setEnvironments] = useState<Environment[]>([LOCAL_ENV]);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    // Try to fetch cloud environments from the API
    api<{ environments: string[] }>('/api/platform/environments')
      .then((data) => {
        const cloudEnvs = (data.environments || []).map((name) => ({
          name,
          source: 'cloud' as const,
        }));
        setEnvironments([LOCAL_ENV, ...cloudEnvs, ALL_ENV]);
      })
      .catch(() => {
        // No cloud configured -- just show Local
        setEnvironments([LOCAL_ENV]);
      });
  }, []);

  const current = environments.find((e) => e.name.toLowerCase() === value) || LOCAL_ENV;

  function dotColor(env: Environment): string {
    if (env.source === 'local') return 'var(--brand-green)';
    if (env.source === 'all') return 'var(--brand-cyan)';
    if (env.name.toLowerCase().includes('prod')) return 'var(--brand-orange)';
    return 'var(--brand-blue)';
  }

  return (
    <div class="env-picker">
      <button class="env-picker-trigger" onClick={() => setOpen(!open)}>
        <span class="env-dot" style={{ background: dotColor(current) }} />
        <span class="env-name">{current.name}</span>
        <span class="env-chevron">▾</span>
      </button>
      {open && (
        <div class="env-picker-dropdown">
          {environments.map((env) => (
            <button
              key={env.name}
              class={`env-option ${env.name === current.name ? 'active' : ''}`}
              onClick={() => {
                onChange(env.name.toLowerCase());
                setOpen(false);
              }}
            >
              <span class="env-dot" style={{ background: dotColor(env) }} />
              <span>{env.name}</span>
              {env.source === 'cloud' && <span class="env-badge">cloud</span>}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
