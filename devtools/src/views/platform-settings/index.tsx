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
  owner_email: string;
  retention: RetentionConfig;
}

export function PlatformSettingsView() {
  const [org, setOrg] = useState<OrgInfo | null>(null);
  const [retention, setRetention] = useState<RetentionConfig>({
    audit_log: 365,
    request_log: 90,
    state_snapshot: 30,
  });
  const [saved, setSaved] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleteInput, setDeleteInput] = useState('');

  useEffect(() => {
    api<OrgInfo>('/api/platform/settings')
      .then((data) => {
        setOrg(data);
        setRetention(data.retention);
      })
      .catch(console.error);
  }, []);

  function handleRetentionChange(key: keyof RetentionConfig, raw: string) {
    const val = parseInt(raw, 10);
    if (!isNaN(val) && val > 0) {
      setRetention((prev) => ({ ...prev, [key]: val }));
    }
  }

  function handleSave() {
    api<OrgInfo>('/api/platform/settings', {
      method: 'PUT',
      body: JSON.stringify({ retention }),
    })
      .then((data) => {
        setOrg(data);
        setRetention(data.retention);
        setSaved(true);
        setTimeout(() => setSaved(false), 2000);
      })
      .catch(console.error);
  }

  function handleDeleteOrg() {
    alert('Organization deleted (local mode). In production this would be irreversible.');
    setShowDeleteConfirm(false);
    setDeleteInput('');
  }

  const orgName = org?.name ?? 'My Organization';
  const orgSlug = org?.slug ?? 'my-org';

  return (
    <div class="platform-view">
      <div class="platform-header">
        <div class="platform-header-left">
          <h2 class="platform-title">Platform Settings</h2>
          <p class="platform-subtitle">Manage your organization and data preferences</p>
        </div>
      </div>

      {/* Organization info */}
      <section class="settings-section">
        <h3 class="settings-section-title">Organization</h3>
        <div class="settings-card">
          <div class="settings-info-grid">
            <div class="settings-info-row">
              <span class="settings-info-label">Name</span>
              <span class="settings-info-value">{orgName}</span>
            </div>
            <div class="settings-info-row">
              <span class="settings-info-label">Slug</span>
              <code class="settings-info-mono">{orgSlug}</code>
            </div>
            <div class="settings-info-row">
              <span class="settings-info-label">Plan</span>
              <span class="badge badge-green">{org?.plan ?? 'Free'}</span>
            </div>
            <div class="settings-info-row">
              <span class="settings-info-label">Owner</span>
              <span class="settings-info-value">{org?.owner_email ?? ''}</span>
            </div>
          </div>
        </div>
      </section>

      {/* Data retention */}
      <section class="settings-section">
        <h3 class="settings-section-title">Data Retention</h3>
        <p class="settings-section-desc">
          Configure how long CloudMock retains logs and snapshots for your apps.
        </p>
        <div class="settings-card">
          <div class="retention-rows">
            <div class="retention-row">
              <div class="retention-row-info">
                <span class="retention-label">Audit Log</span>
                <span class="retention-desc">User and key actions across the organization</span>
              </div>
              <div class="retention-input-wrap">
                <input
                  type="number"
                  class="input retention-input"
                  min="1"
                  max="3650"
                  value={retention.audit_log}
                  onInput={(e) => handleRetentionChange('audit_log', (e.target as HTMLInputElement).value)}
                />
                <span class="retention-unit">days</span>
              </div>
            </div>

            <div class="retention-row">
              <div class="retention-row-info">
                <span class="retention-label">Request Log</span>
                <span class="retention-desc">AWS API request and response history</span>
              </div>
              <div class="retention-input-wrap">
                <input
                  type="number"
                  class="input retention-input"
                  min="1"
                  max="3650"
                  value={retention.request_log}
                  onInput={(e) => handleRetentionChange('request_log', (e.target as HTMLInputElement).value)}
                />
                <span class="retention-unit">days</span>
              </div>
            </div>

            <div class="retention-row">
              <div class="retention-row-info">
                <span class="retention-label">State Snapshot</span>
                <span class="retention-desc">Periodic snapshots of your mocked infrastructure state</span>
              </div>
              <div class="retention-input-wrap">
                <input
                  type="number"
                  class="input retention-input"
                  min="1"
                  max="365"
                  value={retention.state_snapshot}
                  onInput={(e) => handleRetentionChange('state_snapshot', (e.target as HTMLInputElement).value)}
                />
                <span class="retention-unit">days</span>
              </div>
            </div>
          </div>

          <div class="settings-card-footer">
            <button class="btn btn-primary" onClick={handleSave}>
              {saved ? 'Saved!' : 'Save Changes'}
            </button>
          </div>
        </div>
      </section>

      {/* Danger zone */}
      <section class="settings-section settings-danger-section">
        <h3 class="settings-section-title settings-danger-title">Danger Zone</h3>
        <div class="settings-card settings-danger-card">
          <div class="danger-row">
            <div>
              <div class="danger-label">Delete Organization</div>
              <div class="danger-desc">
                Permanently delete this organization and all associated apps, keys, and data.
                This action cannot be undone.
              </div>
            </div>
            <button
              class="btn btn-danger"
              onClick={() => setShowDeleteConfirm(true)}
            >
              Delete Org
            </button>
          </div>
        </div>
      </section>

      {/* Delete confirmation dialog */}
      {showDeleteConfirm && (
        <div class="platform-modal-overlay" onClick={() => setShowDeleteConfirm(false)}>
          <div class="platform-modal" onClick={(e) => e.stopPropagation()}>
            <div class="platform-modal-header">
              <span class="platform-modal-title">Delete Organization</span>
              <button class="platform-modal-close" onClick={() => setShowDeleteConfirm(false)}>×</button>
            </div>
            <div class="platform-modal-body">
              <p class="danger-confirm-text">
                This will permanently delete <strong>{orgName}</strong> and all its data.
                Type <strong>{orgSlug}</strong> to confirm.
              </p>
              <input
                class="input platform-input"
                placeholder={orgSlug}
                value={deleteInput}
                onInput={(e) => setDeleteInput((e.target as HTMLInputElement).value)}
              />
            </div>
            <div class="platform-modal-footer">
              <button class="btn" onClick={() => setShowDeleteConfirm(false)}>Cancel</button>
              <button
                class="btn btn-danger"
                disabled={deleteInput !== orgSlug}
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
