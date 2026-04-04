import { useState, useEffect } from 'preact/hooks';
import { api } from '../../lib/api';
import './platform-apps.css';

type InfraType = 'shared' | 'dedicated';

interface App {
  id: string;
  name: string;
  slug: string;
  endpoint: string;
  infra_type: InfraType;
  status: string;
  request_count: number;
}

function statusClass(status: string): string {
  if (status === 'running') return 'badge-green';
  if (status === 'shared') return 'badge-yellow';
  if (status === 'dedicated') return 'badge-blue';
  return 'badge-gray';
}

function infraClass(infra: string): string {
  if (infra === 'dedicated') return 'badge-blue';
  return 'badge-yellow';
}

interface CreateAppDialogProps {
  onClose: () => void;
  onCreate: (name: string, infra: InfraType) => void;
}

function CreateAppDialog({ onClose, onCreate }: CreateAppDialogProps) {
  const [name, setName] = useState('');
  const [infra, setInfra] = useState<InfraType>('shared');
  const [error, setError] = useState('');

  function handleSubmit() {
    if (!name.trim()) {
      setError('App name is required');
      return;
    }
    onCreate(name.trim(), infra);
  }

  return (
    <div class="platform-modal-overlay" onClick={onClose}>
      <div class="platform-modal" onClick={(e) => e.stopPropagation()}>
        <div class="platform-modal-header">
          <span class="platform-modal-title">New App</span>
          <button class="platform-modal-close" onClick={onClose}>×</button>
        </div>
        <div class="platform-modal-body">
          <div class="platform-field">
            <label class="platform-label">App Name</label>
            <input
              class="input platform-input"
              placeholder="e.g. staging"
              value={name}
              onInput={(e) => setName((e.target as HTMLInputElement).value)}
            />
          </div>
          <div class="platform-field">
            <label class="platform-label">Infrastructure Type</label>
            <div class="platform-radio-group">
              <label class="platform-radio-option">
                <input
                  type="radio"
                  name="infra"
                  value="shared"
                  checked={infra === 'shared'}
                  onChange={() => setInfra('shared')}
                />
                <span class="platform-radio-label">
                  <strong>Shared</strong>
                  <span class="platform-radio-desc">Multi-tenant, lower cost</span>
                </span>
              </label>
              <label class="platform-radio-option">
                <input
                  type="radio"
                  name="infra"
                  value="dedicated"
                  checked={infra === 'dedicated'}
                  onChange={() => setInfra('dedicated')}
                />
                <span class="platform-radio-label">
                  <strong>Dedicated</strong>
                  <span class="platform-radio-desc">Isolated instance, higher performance</span>
                </span>
              </label>
            </div>
          </div>
          {error && <p class="platform-error">{error}</p>}
        </div>
        <div class="platform-modal-footer">
          <button class="btn" onClick={onClose}>Cancel</button>
          <button class="btn btn-primary" onClick={handleSubmit}>Create App</button>
        </div>
      </div>
    </div>
  );
}

export function PlatformAppsView() {
  const [apps, setApps] = useState<App[]>([]);
  const [showCreate, setShowCreate] = useState(false);

  useEffect(() => {
    api<App[]>('/api/platform/apps').then(setApps).catch(console.error);
  }, []);

  function handleCreate(name: string, infra: InfraType) {
    api<App>('/api/platform/apps', {
      method: 'POST',
      body: JSON.stringify({ name, infra_type: infra }),
    })
      .then((newApp) => {
        setApps((prev) => [...prev, newApp]);
        setShowCreate(false);
      })
      .catch(console.error);
  }

  function handleDelete(id: string) {
    api(`/api/platform/apps/${id}`, { method: 'DELETE' })
      .then(() => setApps((prev) => prev.filter((a) => a.id !== id)))
      .catch(console.error);
  }

  return (
    <div class="platform-view">
      {showCreate && (
        <CreateAppDialog
          onClose={() => setShowCreate(false)}
          onCreate={handleCreate}
        />
      )}
      <div class="platform-header">
        <div class="platform-header-left">
          <h2 class="platform-title">Apps</h2>
          <p class="platform-subtitle">Manage your CloudMock app instances</p>
        </div>
        <button class="btn btn-primary" onClick={() => setShowCreate(true)}>
          + New App
        </button>
      </div>

      <div class="apps-grid">
        {apps.map((app) => (
          <div class="app-card" key={app.id}>
            <div class="app-card-header">
              <span class="app-name">{app.name}</span>
              <span class={`badge ${statusClass(app.status)}`}>{app.status}</span>
            </div>
            <div class="app-card-body">
              <div class="app-field">
                <span class="app-field-label">Endpoint</span>
                <span class="app-endpoint">{app.endpoint}</span>
              </div>
              <div class="app-field">
                <span class="app-field-label">Infrastructure</span>
                <span class={`badge ${infraClass(app.infra_type)}`}>{app.infra_type}</span>
              </div>
              <div class="app-field">
                <span class="app-field-label">Requests</span>
                <span class="app-requests">{app.request_count.toLocaleString()}</span>
              </div>
            </div>
            <div class="app-card-footer">
              <button class="btn btn-danger-ghost" onClick={() => handleDelete(app.id)}>Delete</button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
