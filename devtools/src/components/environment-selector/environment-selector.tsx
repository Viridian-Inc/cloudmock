import { useState, useEffect, useRef, useCallback } from 'preact/hooks';
import {
  type Environment,
  loadEnvironments,
  saveEnvironments,
  loadActiveEnvironmentId,
  saveActiveEnvironmentId,
  getActiveEnvironment,
} from '../../lib/environments';
import './environment-selector.css';

interface EnvironmentSelectorProps {
  onEnvironmentChange: (env: Environment) => void;
}

export function EnvironmentSelector({ onEnvironmentChange }: EnvironmentSelectorProps) {
  const [environments, setEnvironments] = useState<Environment[]>(loadEnvironments);
  const [activeId, setActiveId] = useState<string>(loadActiveEnvironmentId);
  const [open, setOpen] = useState(false);
  const [confirmingProd, setConfirmingProd] = useState<Environment | null>(null);
  const [editingEnv, setEditingEnv] = useState<Environment | null>(null);
  const [editEndpoint, setEditEndpoint] = useState('');
  const dropdownRef = useRef<HTMLDivElement>(null);

  const active = getActiveEnvironment(environments, activeId);

  // Update window title with environment name
  useEffect(() => {
    if (active) {
      const base = 'Neureaux DevTools';
      document.title = active.id === 'local' ? base : `${base} — ${active.name}`;
    }
  }, [active]);

  // Close dropdown on outside click
  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open]);

  const switchTo = useCallback(
    (env: Environment) => {
      if (env.isProduction) {
        setConfirmingProd(env);
        setOpen(false);
        return;
      }
      doSwitch(env);
    },
    [environments],
  );

  const doSwitch = useCallback(
    (env: Environment) => {
      setActiveId(env.id);
      saveActiveEnvironmentId(env.id);
      setOpen(false);
      setConfirmingProd(null);
      onEnvironmentChange(env);
    },
    [onEnvironmentChange],
  );

  const startEdit = (env: Environment, e: MouseEvent) => {
    e.stopPropagation();
    setEditingEnv(env);
    setEditEndpoint(env.endpoint);
    setOpen(false);
  };

  const saveEdit = () => {
    if (!editingEnv) return;
    const updated = environments.map((env) =>
      env.id === editingEnv.id ? { ...env, endpoint: editEndpoint.replace(/\/+$/, '') } : env,
    );
    setEnvironments(updated);
    saveEnvironments(updated);
    setEditingEnv(null);

    // If editing the active environment, notify parent
    if (editingEnv.id === activeId) {
      const newEnv = updated.find((e) => e.id === editingEnv.id);
      if (newEnv) onEnvironmentChange(newEnv);
    }
  };

  if (!active) return null;

  return (
    <>
      <div class="env-selector" ref={dropdownRef}>
        <button
          class="env-selector-trigger"
          onClick={() => setOpen(!open)}
          title={`Environment: ${active.name}`}
        >
          <span class="env-dot" style={{ background: active.color }} />
          <span class="env-name">{active.name}</span>
          <svg class="env-chevron" width="10" height="10" viewBox="0 0 10 10">
            <path d="M2 4l3 3 3-3" stroke="currentColor" fill="none" stroke-width="1.5" />
          </svg>
        </button>

        {open && (
          <div class="env-dropdown">
            {environments.map((env) => (
              <div
                key={env.id}
                class={`env-option ${env.id === activeId ? 'active' : ''} ${!env.endpoint && env.id !== 'local' ? 'unconfigured' : ''}`}
                onClick={() => env.endpoint ? switchTo(env) : startEdit(env, event as MouseEvent)}
              >
                <span class="env-dot" style={{ background: env.color }} />
                <div class="env-option-content">
                  <span class="env-option-name">{env.name}</span>
                  <span class="env-option-endpoint">
                    {env.endpoint || 'Click to configure'}
                  </span>
                </div>
                {env.id === activeId && (
                  <span class="env-active-badge">active</span>
                )}
                <button
                  class="env-edit-btn"
                  onClick={(e) => startEdit(env, e)}
                  title="Edit endpoint"
                >
                  <svg width="12" height="12" viewBox="0 0 12 12">
                    <path
                      d="M8.5 1.5l2 2-7 7H1.5V8.5l7-7z"
                      stroke="currentColor"
                      fill="none"
                      stroke-width="1"
                    />
                  </svg>
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Production warning modal */}
      {confirmingProd && (
        <div class="env-modal-overlay" onClick={() => setConfirmingProd(null)}>
          <div class="env-modal" onClick={(e) => e.stopPropagation()}>
            <div class="env-modal-icon">!</div>
            <div class="env-modal-title">Switch to Production?</div>
            <div class="env-modal-body">
              You are about to connect to <strong>{confirmingProd.name}</strong> ({confirmingProd.endpoint}).
              Changes made in production may affect real users and data.
            </div>
            <div class="env-modal-actions">
              <button
                class="env-modal-btn cancel"
                onClick={() => setConfirmingProd(null)}
              >
                Cancel
              </button>
              <button
                class="env-modal-btn confirm"
                onClick={() => doSwitch(confirmingProd)}
              >
                Connect to Production
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Edit endpoint modal */}
      {editingEnv && (
        <div class="env-modal-overlay" onClick={() => setEditingEnv(null)}>
          <div class="env-modal" onClick={(e) => e.stopPropagation()}>
            <div class="env-modal-title">
              Configure {editingEnv.name} Endpoint
            </div>
            <div class="env-modal-body">
              <label class="env-edit-label">
                Admin API URL
                <input
                  class="env-edit-input"
                  type="text"
                  placeholder="https://dev.cloudmock.example.com:4599"
                  value={editEndpoint}
                  onInput={(e) => setEditEndpoint((e.target as HTMLInputElement).value)}
                  onKeyDown={(e) => { if (e.key === 'Enter') saveEdit(); }}
                  autoFocus
                />
              </label>
            </div>
            <div class="env-modal-actions">
              <button
                class="env-modal-btn cancel"
                onClick={() => setEditingEnv(null)}
              >
                Cancel
              </button>
              <button
                class="env-modal-btn confirm"
                onClick={saveEdit}
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
