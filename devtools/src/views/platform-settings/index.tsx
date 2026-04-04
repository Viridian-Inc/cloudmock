import { useState } from 'preact/hooks';
import './platform-settings.css';

interface RetentionConfig {
  audit_log: number;
  request_log: number;
  state_snapshot: number;
}

// TODO: Replace with API call to GET /api/platform/settings
const DEFAULT_RETENTION: RetentionConfig = {
  audit_log: 365,
  request_log: 90,
  state_snapshot: 30,
};

interface OrgInfo {
  name: string;
  slug: string;
  plan: string;
  owner_email: string;
}

// TODO: Replace with API call to GET /api/platform/org
const MOCK_ORG: OrgInfo = {
  name: 'My Organization',
  slug: 'my-org',
  plan: 'Free',
  owner_email: 'admin@example.com',
};

export function PlatformSettingsView() {
  const [retention, setRetention] = useState<RetentionConfig>(DEFAULT_RETENTION);
  const [saved, setSaved] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleteInput, setDeleteInput] = useState('');

  function handleRetentionChange(key: keyof RetentionConfig, raw: string) {
    const val = parseInt(raw, 10);
    if (!isNaN(val) && val > 0) {
      setRetention((prev) => ({ ...prev, [key]: val }));
    }
  }

  function handleSave() {
    // TODO: PUT /api/platform/settings with retention config
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  }

  function handleDeleteOrg() {
    // TODO: DELETE /api/platform/org
    alert('Organization deleted (mock). In production this would be irreversible.');
    setShowDeleteConfirm(false);
    setDeleteInput('');
  }

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
              <span class="settings-info-value">{MOCK_ORG.name}</span>
            </div>
            <div class="settings-info-row">
              <span class="settings-info-label">Slug</span>
              <code class="settings-info-mono">{MOCK_ORG.slug}</code>
            </div>
            <div class="settings-info-row">
              <span class="settings-info-label">Plan</span>
              <span class="badge badge-green">{MOCK_ORG.plan}</span>
            </div>
            <div class="settings-info-row">
              <span class="settings-info-label">Owner</span>
              <span class="settings-info-value">{MOCK_ORG.owner_email}</span>
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
                This will permanently delete <strong>{MOCK_ORG.name}</strong> and all its data.
                Type <strong>{MOCK_ORG.slug}</strong> to confirm.
              </p>
              <input
                class="input platform-input"
                placeholder={MOCK_ORG.slug}
                value={deleteInput}
                onInput={(e) => setDeleteInput((e.target as HTMLInputElement).value)}
              />
            </div>
            <div class="platform-modal-footer">
              <button class="btn" onClick={() => setShowDeleteConfirm(false)}>Cancel</button>
              <button
                class="btn btn-danger"
                disabled={deleteInput !== MOCK_ORG.slug}
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
