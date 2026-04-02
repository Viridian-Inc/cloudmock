import { useConnection } from '../../lib/connection';

export function Connections() {
  const { state, connect, disconnect } = useConnection();

  return (
    <div class="settings-section">
      <h3 class="settings-section-title">Connection Profile</h3>

      <div class="settings-field">
        <label class="settings-label">Admin URL</label>
        <div class="settings-field-row">
          <code class="settings-code">{state.adminUrl}</code>
          <span class={`settings-status-dot ${state.connected ? 'connected' : 'disconnected'}`} />
          <span class="settings-status-label">
            {state.connected ? 'Connected' : 'Disconnected'}
          </span>
        </div>
      </div>

      <div class="settings-field">
        <label class="settings-label">Gateway URL</label>
        <div class="settings-field-row">
          <code class="settings-code">{state.gatewayUrl}</code>
        </div>
      </div>

      {state.connected && (
        <div class="settings-field">
          <label class="settings-label">Details</label>
          <div class="settings-field-row" style="flex-direction: column; align-items: flex-start; gap: 4px;">
            <span>Region: {state.region || '—'}</span>
            <span>Profile: {state.profile || '—'}</span>
            <span>IAM Mode: {state.iamMode || '—'}</span>
            <span>Services: {state.serviceCount}</span>
            {state.pid && <span>PID: {state.pid}</span>}
          </div>
        </div>
      )}

      <div class="settings-actions">
        {state.connected ? (
          <button class="btn" onClick={disconnect}>Disconnect</button>
        ) : (
          <button
            class="btn btn-primary"
            onClick={() => connect('http://localhost:4599', 'http://localhost:4566')}
          >
            Connect to Local
          </button>
        )}
      </div>
    </div>
  );
}
