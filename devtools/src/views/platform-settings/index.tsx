import { useState, useEffect } from 'preact/hooks';
import { api } from '../../lib/api';
import './platform-settings.css';

interface RetentionConfig {
  audit_log: number;
  request_log: number;
  state_snapshot: number;
}

interface OrgInfo {
  name: string;
  slug: string;
  plan: string;
  owner_email?: string;
  request_count?: number;
  request_limit?: number;
  retention: RetentionConfig;
}

export function PlatformSettingsView() {
  const [org, setOrg] = useState<OrgInfo | null>(null);
  const [retention, setRetention] = useState<RetentionConfig | null>(null);
  const [saved, setSaved] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleteInput, setDeleteInput] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    api<OrgInfo>('/api/platform/settings')
      .then((data) => {
        setOrg(data);
        setRetention(data.retention);
      })
      .catch((e) => setError(e.message));
  }, []);

  function handleRetentionChange(key: keyof RetentionConfig, raw: string) {
    const val = parseInt(raw, 10);
    if (!isNaN(val) && val > 0 && retention) {
      setRetention((prev) => prev ? { ...prev, [key]: val } : prev);
    }
  }

  function handleSave() {
    api('/api/platform/settings', {
      method: 'PUT',
      body: JSON.stringify({ retention }),
    })
      .then(() => {
        setSaved(true);
        setTimeout(() => setSaved(false), 2000);
      })
      .catch((e) => setError(e.message));
  }

  function handleDeleteOrg() {
    // In production this calls the real delete endpoint
    setShowDeleteConfirm(false);
    setDeleteInput('');
  }

  if (error) {
    return (
      <div class="platform-view">
        <div class="platform-header">
          <h2 class="platform-title">Platform Settings</h2>
        </div>
        <div class="platform-error">{error}</div>
      </div>
    );
  }

  if (!org || !retention) {
    return (
      <div class="platform-view">
        <div class="platform-header">
          <h2 class="platform-title">Platform Settings</h2>
        </div>
        <div class="platform-loading">Loading...</div>
      </div>
    );
  }

  return (
    <div class="platform-view">
      <div class="platform-header">
        <div class="platform-header-left">
          <h2 class="platform-title">Platform Settings</h2>
          <p class="platform-subtitle">Manage your organization and data preferences</p>
        </div>
      </div>

      <section class="settings-section">
        <h3 class="settings-section-title">Organization</h3>
        <div class="settings-card">
          <div class="settings-info-grid">
            <div class="settings-info-row">
              <span class="settings-info-label">Name</span>
              <span class="settings-info-value">{org.name}</span>
            </div>
            <div class="settings-info-row">
              <span class="settings-info-label">Slug</span>
              <code class="settings-info-mono">{org.slug}</code>
            </div>
            <div class="settings-info-row">
              <span class="settings-info-label">Plan</span>
              <span class="badge badge-green">{org.plan}</span>
            </div>
            {org.owner_email && (
              <div class="settings-info-row">
                <span class="settings-info-label">Owner</span>
                <span class="settings-info-value">{org.owner_email}</span>
              </div>
            )}
            {org.request_count != null && (
              <div class="settings-info-row">
                <span class="settings-info-label">Requests</span>
                <span class="settings-info-value">
                  {org.request_count.toLocaleString()}
                  {org.request_limit ? ` / ${org.request_limit.toLocaleString()}` : ' (unlimited)'}
                </span>
              </div>
            )}
          </div>
        </div>
      </section>

      <section class="settings-section">
        <h3 class="settings-section-title">Data Retention</h3>
        <p class="settings-section-desc">
          Configure how long CloudMock retains logs and snapshots.
        </p>
        <div class="settings-card">
          <div class="retention-rows">
            {(['audit_log', 'request_log', 'state_snapshot'] as const).map((key) => (
              <div class="retention-row" key={key}>
                <div class="retention-row-info">
                  <span class="retention-label">
                    {key === 'audit_log' ? 'Audit Log' : key === 'request_log' ? 'Request Log' : 'State Snapshot'}
                  </span>
                </div>
                <div class="retention-input-wrap">
                  <input
                    type="number"
                    class="input retention-input"
                    min="1"
                    max="3650"
                    value={retention[key]}
                    onInput={(e) => handleRetentionChange(key, (e.target as HTMLInputElement).value)}
                  />
                  <span class="retention-unit">days</span>
                </div>
              </div>
            ))}
          </div>
          <div class="settings-card-footer">
            <button class="btn btn-primary" onClick={handleSave}>
              {saved ? 'Saved!' : 'Save Changes'}
            </button>
          </div>
        </div>
      </section>

      <section class="settings-section settings-danger-section">
        <h3 class="settings-section-title settings-danger-title">Danger Zone</h3>
        <div class="settings-card settings-danger-card">
          <div class="danger-row">
            <div>
              <div class="danger-label">Delete Organization</div>
              <div class="danger-desc">
                Permanently delete this organization and all associated data.
              </div>
            </div>
            <button class="btn btn-danger" onClick={() => setShowDeleteConfirm(true)}>
              Delete Org
            </button>
          </div>
        </div>
      </section>

      {showDeleteConfirm && (
        <div class="platform-modal-overlay" onClick={() => setShowDeleteConfirm(false)}>
          <div class="platform-modal" onClick={(e) => e.stopPropagation()}>
            <div class="platform-modal-header">
              <span class="platform-modal-title">Delete Organization</span>
              <button class="platform-modal-close" onClick={() => setShowDeleteConfirm(false)}>×</button>
            </div>
            <div class="platform-modal-body">
              <p class="danger-confirm-text">
                This will permanently delete <strong>{org.name}</strong>.
                Type <strong>{org.slug}</strong> to confirm.
              </p>
              <input
                class="input platform-input"
                placeholder={org.slug}
                value={deleteInput}
                onInput={(e) => setDeleteInput((e.target as HTMLInputElement).value)}
              />
            </div>
            <div class="platform-modal-footer">
              <button class="btn" onClick={() => setShowDeleteConfirm(false)}>Cancel</button>
              <button
                class="btn btn-danger"
                disabled={deleteInput !== org.slug}
                onClick={handleDeleteOrg}
              >
                Delete Forever
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
