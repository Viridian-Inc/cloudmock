import { useMemo, useCallback } from 'preact/hooks';
import { useConnection } from '../../lib/connection';
import {
  loadDashboards,
  loadDashboardPreferences,
  saveDashboardPreferences,
} from '../dashboards/storage';
import { PRESET_DASHBOARDS } from '../dashboards/presets';

export function Account() {
  const { state } = useConnection();

  const dashboardSummary = useMemo(() => {
    const saved = loadDashboards();
    const prefs = loadDashboardPreferences();
    const totalCustom = saved.length;
    const totalPresets = PRESET_DASHBOARDS.length;
    return {
      total: totalCustom + totalPresets,
      custom: totalCustom,
      presets: totalPresets,
      favorites: prefs.favorites.length,
      hidden: prefs.hidden.length,
    };
  }, []);

  const handleResetPreferences = useCallback(() => {
    saveDashboardPreferences({ hidden: [], favorites: [] });
    // Force re-render by dispatching a storage event
    window.dispatchEvent(new Event('storage'));
    // Inform the user
    alert('Dashboard preferences have been reset.');
  }, []);

  const isLocal = !state.adminUrl || state.adminUrl.includes('localhost');

  return (
    <div class="settings-section">
      <h3 class="settings-section-title">Account</h3>
      <p class="settings-section-desc">
        Connection mode and dashboard preferences summary.
      </p>

      {/* Connection mode */}
      <div class="settings-field">
        <label class="settings-label">Connection Mode</label>
        <div class="settings-field-row">
          <span class={`settings-status-dot ${state.connected ? 'connected' : 'disconnected'}`} />
          <span class="settings-status-label">
            {isLocal ? 'Local Development' : 'cloudmock.io'}
          </span>
        </div>
      </div>

      {isLocal ? (
        <div class="settings-field">
          <label class="settings-label">Authentication</label>
          <span class="settings-status-label" style="font-size: 12px; color: var(--text-tertiary);">
            No authentication required in local mode.
          </span>
        </div>
      ) : (
        <>
          <div class="settings-field">
            <label class="settings-label">Organization</label>
            <span class="settings-status-label" style="font-size: 12px;">
              {state.profile || 'Unknown'}
            </span>
          </div>
          <div class="settings-field">
            <label class="settings-label">Region</label>
            <span class="settings-status-label" style="font-size: 12px;">
              {state.region || 'Unknown'}
            </span>
          </div>
          <div class="settings-field">
            <label class="settings-label">IAM Mode</label>
            <span class="settings-status-label" style="font-size: 12px;">
              {state.iamMode || 'Unknown'}
            </span>
          </div>
        </>
      )}

      {/* Dashboard preferences summary */}
      <div class="settings-field" style="margin-top: 24px;">
        <label class="settings-label">Dashboard Preferences</label>
        <div class="settings-field-row" style="flex-direction: column; align-items: flex-start; gap: 4px;">
          <span style="font-size: 12px; color: var(--text-secondary);">
            {dashboardSummary.total} dashboards ({dashboardSummary.presets} presets, {dashboardSummary.custom} custom)
          </span>
          <span style="font-size: 12px; color: var(--text-secondary);">
            {dashboardSummary.favorites} favorited
          </span>
          <span style="font-size: 12px; color: var(--text-secondary);">
            {dashboardSummary.hidden} hidden
          </span>
        </div>
      </div>

      <div class="settings-actions">
        <button class="btn" onClick={handleResetPreferences}>
          Reset Dashboard Preferences
        </button>
      </div>
    </div>
  );
}
