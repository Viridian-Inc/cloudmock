import { useState, useEffect, useCallback } from 'preact/hooks';
import {
  getLocalDataInfo,
  deleteLocalData,
  type LocalDataInfo,
  type LocalDataDeleteResult,
} from '../../lib/api';

const UNNAMED = '(unnamed — default ~/.cloudmock)';

export function LocalData() {
  const [info, setInfo] = useState<LocalDataInfo | null>(null);
  const [loadError, setLoadError] = useState('');
  const [showConfirm, setShowConfirm] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [actionError, setActionError] = useState('');
  const [result, setResult] = useState<LocalDataDeleteResult | null>(null);

  const refresh = useCallback(() => {
    getLocalDataInfo()
      .then((d) => {
        setInfo(d);
        setLoadError('');
      })
      .catch((e) => setLoadError(String(e?.message ?? e)));
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const handleConfirmDelete = useCallback(() => {
    setDeleting(true);
    setActionError('');
    deleteLocalData()
      .then((r) => {
        setResult(r);
        setShowConfirm(false);
        refresh();
      })
      .catch((e) => setActionError(String(e?.message ?? e)))
      .finally(() => setDeleting(false));
  }, [refresh]);

  const projectLabel = info?.project ? info.project : UNNAMED;

  return (
    <div class="settings-section">
      <h3 class="settings-section-title">Local Data</h3>
      <p class="settings-section-desc">
        On-disk persistent storage for this cloudmock project. Deleting wipes all
        locally stored data for this project and resets in-memory state — installed
        plugins and other projects are not affected.
      </p>

      {loadError && <p class="settings-error">{loadError}</p>}

      <div class="settings-field">
        <label class="settings-label">Project / Database</label>
        <span class="settings-status-label" style="font-size: 12px;">
          {projectLabel}
        </span>
      </div>

      <div class="settings-field">
        <label class="settings-label">Data Directory</label>
        <code class="settings-code">{info?.dir || '—'}</code>
      </div>

      <div class="settings-field">
        <label class="settings-label">Persistence</label>
        <div class="settings-field-row">
          <span
            class={`settings-status-dot ${info?.persistent ? 'connected' : 'disconnected'}`}
          />
          <span class="settings-status-label" style="font-size: 12px;">
            {info?.persistent
              ? `Enabled → ${info.stateFile}`
              : 'In-memory only (no disk persistence)'}
          </span>
        </div>
      </div>

      {result && (
        <p
          class="settings-status-label"
          style="font-size: 12px; color: var(--text-secondary); margin-top: 8px;"
        >
          Deleted {result.removed?.length ?? 0} path(s) and reset{' '}
          {result.reset_services?.length ?? 0} service(s).
          {result.failures && result.failures.length > 0
            ? ` ${result.failures.length} path(s) failed to delete.`
            : ''}
        </p>
      )}

      <div class="settings-actions">
        <button
          class="btn settings-btn-danger"
          disabled={!info?.onDisk}
          title={info?.onDisk ? '' : 'Nothing is stored on disk for this project'}
          onClick={() => {
            setResult(null);
            setActionError('');
            setShowConfirm(true);
          }}
        >
          Delete all local data
        </button>
      </div>

      {showConfirm && (
        <div
          class="settings-modal-overlay"
          onClick={() => {
            if (!deleting) setShowConfirm(false);
          }}
        >
          <div class="settings-modal" onClick={(e) => e.stopPropagation()}>
            <div class="settings-modal-header">
              <span class="settings-modal-title">Delete all local data?</span>
            </div>
            <div class="settings-modal-body">
              <p style="font-size: 13px; margin: 0 0 12px; color: var(--text-primary);">
                This permanently deletes all locally stored on-disk data for project{' '}
                <strong>{projectLabel}</strong> and resets in-memory service state.
                This cannot be undone.
              </p>
              <code class="settings-code">{info?.dir}</code>
              {actionError && (
                <p class="settings-error" style="margin-top: 12px; margin-bottom: 0;">
                  {actionError}
                </p>
              )}
            </div>
            <div class="settings-modal-footer">
              <button
                class="btn"
                disabled={deleting}
                onClick={() => setShowConfirm(false)}
              >
                Cancel
              </button>
              <button
                class="btn settings-btn-danger"
                disabled={deleting}
                onClick={handleConfirmDelete}
              >
                {deleting ? 'Deleting…' : 'Delete everything'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
