import { useState, useEffect, useCallback } from 'preact/hooks';
import { api } from '../../lib/api';

const WEBHOOK_EVENTS = [
  'incident.created',
  'incident.resolved',
  'monitor.alert',
  'monitor.recovered',
  'chaos.started',
  'chaos.stopped',
  'deployment.started',
  'deployment.completed',
] as const;

type WebhookType = 'generic' | 'slack' | 'pagerduty';

interface Webhook {
  id: string;
  url: string;
  type: WebhookType;
  events: string[];
  headers?: Record<string, string>;
  active: boolean;
}

export function Webhooks() {
  const [webhooks, setWebhooks] = useState<Webhook[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [testing, setTesting] = useState<string | null>(null);

  // Form state
  const [formUrl, setFormUrl] = useState('');
  const [formType, setFormType] = useState<WebhookType>('generic');
  const [formEvents, setFormEvents] = useState<string[]>([]);
  const [formHeaders, setFormHeaders] = useState('{}');

  const load = useCallback(() => {
    setLoading(true);
    api<Webhook[]>('/api/webhooks')
      .then((data) => {
        setWebhooks(data);
        setError(null);
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : String(err));
        setWebhooks([]);
      })
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  function clearMessages() {
    setError(null);
    setSuccess(null);
  }

  function toggleEvent(event: string) {
    setFormEvents((prev) =>
      prev.includes(event) ? prev.filter((e) => e !== event) : [...prev, event],
    );
  }

  async function handleCreate() {
    clearMessages();
    let headers: Record<string, string> = {};
    try {
      headers = JSON.parse(formHeaders);
    } catch {
      setError('Invalid JSON in headers field');
      return;
    }

    try {
      await api<Webhook>('/api/webhooks', {
        method: 'POST',
        body: JSON.stringify({
          url: formUrl,
          type: formType,
          events: formEvents,
          headers,
        }),
      });
      setSuccess('Webhook created');
      setShowForm(false);
      setFormUrl('');
      setFormType('generic');
      setFormEvents([]);
      setFormHeaders('{}');
      load();
    } catch (err: any) {
      setError(err.message || 'Failed to create webhook');
    }
  }

  async function handleDelete(id: string) {
    clearMessages();
    try {
      await api<void>(`/api/webhooks/${encodeURIComponent(id)}`, {
        method: 'DELETE',
      });
      setSuccess('Webhook deleted');
      load();
    } catch (err: any) {
      setError(err.message || 'Failed to delete webhook');
    }
  }

  async function handleTest(id: string) {
    clearMessages();
    setTesting(id);
    try {
      await api<void>(`/api/webhooks/${encodeURIComponent(id)}/test`, {
        method: 'POST',
      });
      setSuccess('Test payload sent');
    } catch (err: any) {
      setError(err.message || 'Test failed');
    } finally {
      setTesting(null);
    }
  }

  async function handleToggleActive(wh: Webhook) {
    clearMessages();
    try {
      // Re-create with toggled active state
      await api<Webhook>(`/api/webhooks/${encodeURIComponent(wh.id)}`, {
        method: 'POST',
        body: JSON.stringify({ ...wh, active: !wh.active }),
      });
      setSuccess(wh.active ? 'Webhook deactivated' : 'Webhook activated');
      load();
    } catch (err: any) {
      setError(err.message || 'Failed to toggle webhook');
    }
  }

  return (
    <div class="settings-section" style="max-width: 700px;">
      <div class="webhooks-header">
        <div>
          <h3 class="settings-section-title">
            Webhooks ({webhooks.length})
          </h3>
          <p class="settings-section-desc">
            Deliver event notifications to external services via HTTP webhooks.
          </p>
        </div>
        <button
          class={`btn ${showForm ? '' : 'btn-primary'}`}
          onClick={() => {
            setShowForm(!showForm);
            clearMessages();
          }}
        >
          {showForm ? 'Cancel' : 'Add Webhook'}
        </button>
      </div>

      {error && (
        <div class="settings-error">{error}</div>
      )}

      {success && (
        <div class="webhooks-success">{success}</div>
      )}

      {showForm && (
        <div class="webhooks-form">
          <div class="webhooks-form-grid">
            <div class="webhooks-form-field">
              <label class="settings-label">URL</label>
              <input
                class="input"
                type="text"
                value={formUrl}
                onInput={(e) => setFormUrl((e.target as HTMLInputElement).value)}
                placeholder="https://hooks.example.com/..."
              />
            </div>

            <div class="webhooks-form-field">
              <label class="settings-label">Type</label>
              <select
                class="input"
                value={formType}
                onChange={(e) =>
                  setFormType((e.target as HTMLSelectElement).value as WebhookType)
                }
              >
                <option value="generic">Generic</option>
                <option value="slack">Slack</option>
                <option value="pagerduty">PagerDuty</option>
              </select>
            </div>

            <div class="webhooks-form-field">
              <label class="settings-label">Events</label>
              <div class="webhooks-events-grid">
                {WEBHOOK_EVENTS.map((ev) => (
                  <label key={ev} class="webhooks-event-checkbox">
                    <input
                      type="checkbox"
                      checked={formEvents.includes(ev)}
                      onChange={() => toggleEvent(ev)}
                    />
                    <span>{ev}</span>
                  </label>
                ))}
              </div>
            </div>

            <div class="webhooks-form-field webhooks-form-field-full">
              <label class="settings-label">Headers (JSON)</label>
              <textarea
                class="input webhooks-headers-input"
                value={formHeaders}
                onInput={(e) =>
                  setFormHeaders((e.target as HTMLTextAreaElement).value)
                }
                placeholder='{"Authorization": "Bearer ..."}'
                rows={3}
              />
            </div>
          </div>

          <button
            class="btn btn-primary"
            onClick={handleCreate}
            disabled={!formUrl.trim()}
          >
            Create Webhook
          </button>
        </div>
      )}

      {loading ? (
        <div class="settings-placeholder">Loading webhooks...</div>
      ) : (
        <div class="webhooks-table-wrapper">
          <table class="webhooks-table">
            <thead>
              <tr>
                <th>URL</th>
                <th>Type</th>
                <th>Events</th>
                <th>Active</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {webhooks.map((wh) => (
                <tr key={wh.id}>
                  <td>
                    <code class="webhooks-url">{wh.url}</code>
                  </td>
                  <td>
                    <span class="webhooks-type-badge">{wh.type || 'generic'}</span>
                  </td>
                  <td>
                    <span class="webhooks-events-list">
                      {(wh.events || []).join(', ') || '-'}
                    </span>
                  </td>
                  <td>
                    <button
                      class={`webhooks-toggle ${wh.active !== false ? 'active' : ''}`}
                      onClick={() => handleToggleActive(wh)}
                      title={wh.active !== false ? 'Click to deactivate' : 'Click to activate'}
                    >
                      <span class="webhooks-toggle-track">
                        <span class="webhooks-toggle-thumb" />
                      </span>
                    </button>
                  </td>
                  <td>
                    <div class="webhooks-actions">
                      <button
                        class="btn btn-ghost webhooks-action-btn"
                        onClick={() => handleTest(wh.id)}
                        disabled={testing === wh.id}
                      >
                        {testing === wh.id ? 'Sending...' : 'Test'}
                      </button>
                      <button
                        class="btn btn-ghost webhooks-delete-btn"
                        onClick={() => handleDelete(wh.id)}
                        title="Delete webhook"
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
              {webhooks.length === 0 && (
                <tr>
                  <td colSpan={5} class="webhooks-empty">
                    No webhooks configured. Click "Add Webhook" to create one.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
