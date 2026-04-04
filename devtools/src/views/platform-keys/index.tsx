import { useState, useEffect, useCallback } from 'preact/hooks';
import { api } from '../../lib/api';
import './platform-keys.css';

type Role = 'admin' | 'developer' | 'viewer';

function CopyInline({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  const handleCopy = useCallback((e: Event) => {
    e.stopPropagation();
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }, [text]);
  return (
    <button class="copy-inline" onClick={handleCopy} title="Copy to clipboard">
      {copied ? 'Copied!' : 'Copy'}
    </button>
  );
}

interface ApiKey {
  id: string;
  prefix: string;
  name: string;
  role: Role;
  last_used_at: string | null;
  created_at: string;
}

function relativeTime(iso: string | null): string {
  if (!iso) return 'Never';
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'Just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  return `${days}d ago`;
}

function roleClass(role: Role): string {
  if (role === 'admin') return 'badge badge-red';
  if (role === 'developer') return 'badge badge-blue';
  return 'badge badge-gray';
}

interface CreateKeyDialogProps {
  onClose: () => void;
  onCreate: (name: string, role: Role) => void;
}

function CreateKeyDialog({ onClose, onCreate }: CreateKeyDialogProps) {
  const [name, setName] = useState('');
  const [role, setRole] = useState<Role>('developer');
  const [error, setError] = useState('');

  function handleSubmit() {
    if (!name.trim()) {
      setError('Key name is required');
      return;
    }
    onCreate(name.trim(), role);
  }

  return (
    <div class="platform-modal-overlay" onClick={onClose}>
      <div class="platform-modal" onClick={(e) => e.stopPropagation()}>
        <div class="platform-modal-header">
          <span class="platform-modal-title">Create API Key</span>
          <button class="platform-modal-close" onClick={onClose}>×</button>
        </div>
        <div class="platform-modal-body">
          <div class="platform-field">
            <label class="platform-label">Key Name</label>
            <input
              class="input platform-input"
              placeholder="e.g. CI Pipeline"
              value={name}
              onInput={(e) => setName((e.target as HTMLInputElement).value)}
            />
          </div>
          <div class="platform-field">
            <label class="platform-label">Role</label>
            <select
              class="input platform-input"
              value={role}
              onChange={(e) => setRole((e.target as HTMLSelectElement).value as Role)}
            >
              <option value="viewer">Viewer — read-only access</option>
              <option value="developer">Developer — read + write</option>
              <option value="admin">Admin — full access</option>
            </select>
          </div>
          {error && <p class="platform-error">{error}</p>}
        </div>
        <div class="platform-modal-footer">
          <button class="btn" onClick={onClose}>Cancel</button>
          <button class="btn btn-primary" onClick={handleSubmit}>Create Key</button>
        </div>
      </div>
    </div>
  );
}

interface NewKeyRevealProps {
  keyValue: string;
  onDone: () => void;
}

function NewKeyReveal({ keyValue, onDone }: NewKeyRevealProps) {
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    navigator.clipboard.writeText(keyValue).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  return (
    <div class="platform-modal-overlay">
      <div class="platform-modal">
        <div class="platform-modal-header">
          <span class="platform-modal-title">Your New API Key</span>
        </div>
        <div class="platform-modal-body">
          <div class="key-reveal-warning">
            Save this key — it will only be shown once.
          </div>
          <div class="key-reveal-box">
            <code class="key-reveal-value">{keyValue}</code>
            <button class="btn" onClick={handleCopy}>
              {copied ? 'Copied!' : 'Copy'}
            </button>
          </div>
        </div>
        <div class="platform-modal-footer">
          <button class="btn btn-primary" onClick={onDone}>Done</button>
        </div>
      </div>
    </div>
  );
}

export function PlatformKeysView() {
  const [keys, setKeys] = useState<ApiKey[]>([]);
  const [showCreate, setShowCreate] = useState(false);
  const [newKeyValue, setNewKeyValue] = useState<string | null>(null);

  useEffect(() => {
    api<ApiKey[]>('/api/platform/keys').then(setKeys).catch(console.error);
  }, []);

  function handleCreate(name: string, role: Role) {
    api<ApiKey & { key: string }>('/api/platform/keys', {
      method: 'POST',
      body: JSON.stringify({ name, role }),
    })
      .then((resp) => {
        const { key, ...keyMeta } = resp;
        setKeys((prev) => [keyMeta, ...prev]);
        setShowCreate(false);
        setNewKeyValue(key);
      })
      .catch(console.error);
  }

  function handleRevoke(id: string) {
    api(`/api/platform/keys/${id}`, { method: 'DELETE' })
      .then(() => setKeys((prev) => prev.filter((k) => k.id !== id)))
      .catch(console.error);
  }

  return (
    <div class="platform-view">
      {showCreate && (
        <CreateKeyDialog
          onClose={() => setShowCreate(false)}
          onCreate={handleCreate}
        />
      )}
      {newKeyValue && (
        <NewKeyReveal
          keyValue={newKeyValue}
          onDone={() => setNewKeyValue(null)}
        />
      )}

      <div class="platform-header">
        <div class="platform-header-left">
          <h2 class="platform-title">API Keys</h2>
          <p class="platform-subtitle">Manage authentication keys for your apps and CI pipelines</p>
        </div>
        <button class="btn btn-primary" onClick={() => setShowCreate(true)}>
          Create Key
        </button>
      </div>

      <div class="platform-table-wrap">
        <table class="platform-table">
          <thead>
            <tr>
              <th>Prefix</th>
              <th>Name</th>
              <th>Role</th>
              <th>Last Used</th>
              <th>Created</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {keys.length === 0 && (
              <tr>
                <td colspan={6} class="platform-table-empty">No API keys yet</td>
              </tr>
            )}
            {keys.map((key) => (
              <tr key={key.id}>
                <td>
                  <div class="key-prefix-row">
                    <code class="key-prefix">{key.prefix}…</code>
                    <CopyInline text={key.prefix} />
                  </div>
                </td>
                <td class="key-name">{key.name}</td>
                <td>
                  <span class={roleClass(key.role as Role)}>{key.role}</span>
                </td>
                <td class="key-time">{relativeTime(key.last_used_at)}</td>
                <td class="key-time">{relativeTime(key.created_at)}</td>
                <td>
                  <button
                    class="btn btn-danger-ghost"
                    onClick={() => handleRevoke(key.id)}
                  >
                    Revoke
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
