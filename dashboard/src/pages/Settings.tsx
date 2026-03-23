import { useState, useEffect, useCallback } from 'preact/hooks';
import { fetchWebhooks, createWebhook, deleteWebhook, testWebhook, fetchUsers, updateUserRole, fetchAudit } from '../api';
import { fmtTime } from '../utils';

type SettingsTab = 'webhooks' | 'users' | 'audit';

interface SettingsPageProps {
  showToast: (msg: string) => void;
}

const inputStyle: any = {
  width: '100%', padding: '6px 8px', borderRadius: '4px', border: '1px solid var(--border)',
  background: 'var(--bg-primary)', color: 'var(--text-primary)', fontSize: '12px', boxSizing: 'border-box',
};

const thStyle: any = { padding: '8px', textAlign: 'left', fontWeight: 500, opacity: 0.8 };
const tdStyle: any = { padding: '8px' };

const WEBHOOK_EVENTS = ['incident.created', 'incident.resolved'] as const;

export function SettingsPage({ showToast }: SettingsPageProps) {
  const [tab, setTab] = useState<SettingsTab>('webhooks');

  const tabs: { key: SettingsTab; label: string }[] = [
    { key: 'webhooks', label: 'Webhooks' },
    { key: 'users', label: 'Users' },
    { key: 'audit', label: 'Audit' },
  ];

  return (
    <div style={{ padding: '24px' }}>
      <h2 style={{ margin: '0 0 16px', fontSize: '20px', fontWeight: 600 }}>Settings</h2>

      <div class="tabs" style={{ marginBottom: '20px' }}>
        {tabs.map(t => (
          <button
            key={t.key}
            class={`tab ${tab === t.key ? 'active' : ''}`}
            onClick={() => setTab(t.key)}
          >
            {t.label}
          </button>
        ))}
      </div>

      {tab === 'webhooks' && <WebhooksTab showToast={showToast} />}
      {tab === 'users' && <UsersTab showToast={showToast} />}
      {tab === 'audit' && <AuditTab />}
    </div>
  );
}

// --- Webhooks Tab ---

function WebhooksTab({ showToast }: { showToast: (msg: string) => void }) {
  const [webhooks, setWebhooks] = useState<any[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({
    url: '',
    type: 'generic',
    events: [] as string[],
    headers: '{}',
  });

  const load = useCallback(() => {
    fetchWebhooks().then(setWebhooks).catch(() => {});
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleCreate = async () => {
    try {
      let headers: Record<string, string> = {};
      try { headers = JSON.parse(form.headers); } catch { /* ignore */ }
      await createWebhook({
        url: form.url,
        type: form.type,
        events: form.events,
        headers,
      });
      showToast('Webhook created');
      setShowForm(false);
      setForm({ url: '', type: 'generic', events: [], headers: '{}' });
      load();
    } catch {
      showToast('Failed to create webhook');
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteWebhook(id);
      showToast('Webhook deleted');
      load();
    } catch {
      showToast('Failed to delete webhook');
    }
  };

  const handleTest = async (id: string) => {
    try {
      await testWebhook(id);
      showToast('Test sent');
    } catch {
      showToast('Test failed');
    }
  };

  const toggleActive = async (wh: any) => {
    // Re-create with toggled active state via the API
    // Since there's no dedicated toggle endpoint, we just show state
    showToast(wh.active ? 'Webhook is active' : 'Webhook is inactive');
  };

  const toggleEvent = (event: string) => {
    setForm(prev => ({
      ...prev,
      events: prev.events.includes(event)
        ? prev.events.filter(e => e !== event)
        : [...prev.events, event],
    }));
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
        <h3 style={{ margin: 0, fontSize: '14px', fontWeight: 600 }}>Webhooks ({webhooks.length})</h3>
        <button
          onClick={() => setShowForm(!showForm)}
          style={{
            padding: '6px 14px', borderRadius: '6px', border: 'none',
            background: '#3b82f6', color: 'white', cursor: 'pointer', fontWeight: 600, fontSize: '13px',
          }}
        >
          {showForm ? 'Cancel' : 'Add Webhook'}
        </button>
      </div>

      {showForm && (
        <div class="card" style={{ padding: '16px', marginBottom: '20px' }}>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))', gap: '12px', marginBottom: '12px' }}>
            <label style={{ fontSize: '12px' }}>
              <div style={{ marginBottom: '4px', opacity: 0.7 }}>URL</div>
              <input
                type="text"
                value={form.url}
                onInput={(e: any) => setForm({ ...form, url: e.target.value })}
                placeholder="https://hooks.example.com/..."
                style={inputStyle}
              />
            </label>
            <label style={{ fontSize: '12px' }}>
              <div style={{ marginBottom: '4px', opacity: 0.7 }}>Type</div>
              <select
                value={form.type}
                onChange={(e: any) => setForm({ ...form, type: e.target.value })}
                style={inputStyle}
              >
                <option value="generic">Generic</option>
                <option value="slack">Slack</option>
                <option value="pagerduty">PagerDuty</option>
              </select>
            </label>
            <div style={{ fontSize: '12px' }}>
              <div style={{ marginBottom: '4px', opacity: 0.7 }}>Events</div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                {WEBHOOK_EVENTS.map(ev => (
                  <label key={ev} style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer' }}>
                    <input
                      type="checkbox"
                      checked={form.events.includes(ev)}
                      onChange={() => toggleEvent(ev)}
                    />
                    {ev}
                  </label>
                ))}
              </div>
            </div>
            <label style={{ fontSize: '12px', gridColumn: '1 / -1' }}>
              <div style={{ marginBottom: '4px', opacity: 0.7 }}>Headers (JSON)</div>
              <textarea
                value={form.headers}
                onInput={(e: any) => setForm({ ...form, headers: e.target.value })}
                placeholder='{"Authorization": "Bearer ..."}'
                style={{ ...inputStyle, minHeight: '60px', fontFamily: 'monospace', resize: 'vertical' }}
              />
            </label>
          </div>
          <button
            onClick={handleCreate}
            style={{
              padding: '8px 16px', borderRadius: '6px', border: 'none',
              background: '#3b82f6', color: 'white', cursor: 'pointer', fontWeight: 600, fontSize: '13px',
            }}
          >
            Create Webhook
          </button>
        </div>
      )}

      <div class="card" style={{ padding: '16px' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
          <thead>
            <tr style={{ borderBottom: '1px solid var(--border)' }}>
              <th style={thStyle}>URL</th>
              <th style={thStyle}>Type</th>
              <th style={thStyle}>Events</th>
              <th style={{ ...thStyle, textAlign: 'center' }}>Active</th>
              <th style={{ ...thStyle, textAlign: 'center' }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {webhooks.map((wh: any) => (
              <tr key={wh.id} style={{ borderBottom: '1px solid var(--border)' }}>
                <td style={{ ...tdStyle, maxWidth: '280px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  <code style={{ fontSize: '11px' }}>{wh.url}</code>
                </td>
                <td style={tdStyle}>
                  <span style={{
                    display: 'inline-block', padding: '2px 6px', borderRadius: '4px', fontSize: '11px', fontWeight: 600,
                    background: 'var(--bg-secondary)', color: 'var(--text-primary)',
                  }}>
                    {wh.type || 'generic'}
                  </span>
                </td>
                <td style={tdStyle}>{(wh.events || []).join(', ') || '-'}</td>
                <td style={{ ...tdStyle, textAlign: 'center' }}>
                  <button
                    onClick={() => toggleActive(wh)}
                    style={{
                      width: '40px', height: '22px', borderRadius: '11px', border: 'none', cursor: 'pointer',
                      background: wh.active !== false ? '#22c55e' : 'var(--border)',
                      position: 'relative', transition: 'background 0.2s',
                    }}
                  >
                    <span style={{
                      position: 'absolute', top: '2px', width: '18px', height: '18px', borderRadius: '50%',
                      background: 'white', transition: 'left 0.2s',
                      left: wh.active !== false ? '20px' : '2px',
                    }} />
                  </button>
                </td>
                <td style={{ ...tdStyle, textAlign: 'center', whiteSpace: 'nowrap' }}>
                  <button
                    onClick={() => handleTest(wh.id)}
                    style={{
                      padding: '3px 8px', borderRadius: '4px', border: '1px solid var(--border)',
                      background: 'transparent', color: 'var(--text-primary)', cursor: 'pointer', fontSize: '11px',
                      marginRight: '4px',
                    }}
                  >
                    Test
                  </button>
                  <button
                    onClick={() => handleDelete(wh.id)}
                    style={{ background: 'none', border: 'none', color: '#ef4444', cursor: 'pointer', fontSize: '16px', padding: '2px 6px' }}
                    title="Delete webhook"
                  >
                    x
                  </button>
                </td>
              </tr>
            ))}
            {webhooks.length === 0 && (
              <tr><td colSpan={5} style={{ padding: '24px', textAlign: 'center', opacity: 0.5 }}>No webhooks configured. Click "Add Webhook" to create one.</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// --- Users Tab ---

function UsersTab({ showToast }: { showToast: (msg: string) => void }) {
  const [users, setUsers] = useState<any[] | null>(null);

  useEffect(() => {
    fetchUsers().then(setUsers).catch(() => setUsers([]));
  }, []);

  const handleRoleChange = async (userId: string, role: string) => {
    try {
      await updateUserRole(userId, role);
      showToast('Role updated');
      setUsers(prev => prev ? prev.map(u => u.id === userId ? { ...u, role } : u) : prev);
    } catch {
      showToast('Failed to update role');
    }
  };

  if (users === null) {
    return <div style={{ padding: '24px', textAlign: 'center', opacity: 0.5 }}>Loading users...</div>;
  }

  if (users.length === 0) {
    return (
      <div class="card" style={{ padding: '24px', textAlign: 'center' }}>
        <p style={{ opacity: 0.6, margin: 0 }}>Auth not enabled -- no users found.</p>
      </div>
    );
  }

  return (
    <div class="card" style={{ padding: '16px' }}>
      <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
        <thead>
          <tr style={{ borderBottom: '1px solid var(--border)' }}>
            <th style={thStyle}>Email</th>
            <th style={thStyle}>Name</th>
            <th style={thStyle}>Role</th>
            <th style={thStyle}>Tenant ID</th>
            <th style={thStyle}>Created At</th>
          </tr>
        </thead>
        <tbody>
          {users.map((u: any) => (
            <tr key={u.id} style={{ borderBottom: '1px solid var(--border)' }}>
              <td style={tdStyle}>{u.email}</td>
              <td style={tdStyle}>{u.name || '-'}</td>
              <td style={tdStyle}>
                <select
                  value={u.role || 'viewer'}
                  onChange={(e: any) => handleRoleChange(u.id, e.target.value)}
                  style={{ ...inputStyle, width: 'auto' }}
                >
                  <option value="admin">admin</option>
                  <option value="editor">editor</option>
                  <option value="viewer">viewer</option>
                </select>
              </td>
              <td style={tdStyle}><code style={{ fontSize: '11px' }}>{u.tenant_id || '-'}</code></td>
              <td style={tdStyle} class="font-mono text-sm">{u.created_at ? fmtTime(u.created_at) : '-'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// --- Audit Tab ---

function AuditTab() {
  const [entries, setEntries] = useState<any[]>([]);
  const [actorFilter, setActorFilter] = useState('');
  const [actionFilter, setActionFilter] = useState('');

  const load = useCallback(() => {
    fetchAudit({
      actor: actorFilter || undefined,
      action: actionFilter || undefined,
      limit: 100,
    }).then(setEntries).catch(() => setEntries([]));
  }, [actorFilter, actionFilter]);

  useEffect(() => { load(); }, [load]);

  return (
    <div>
      <div class="filters-bar" style={{ marginBottom: '16px' }}>
        <input
          class="input input-search"
          placeholder="Filter by actor..."
          value={actorFilter}
          onInput={(e) => setActorFilter((e.target as HTMLInputElement).value)}
          style={{ maxWidth: '220px' }}
        />
        <input
          class="input input-search"
          placeholder="Filter by action..."
          value={actionFilter}
          onInput={(e) => setActionFilter((e.target as HTMLInputElement).value)}
          style={{ maxWidth: '220px' }}
        />
        <span class="text-sm text-muted ml-auto">{entries.length} entries</span>
      </div>

      <div class="card" style={{ padding: '16px' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
          <thead>
            <tr style={{ borderBottom: '1px solid var(--border)' }}>
              <th style={thStyle}>Timestamp</th>
              <th style={thStyle}>Actor</th>
              <th style={thStyle}>Action</th>
              <th style={thStyle}>Resource</th>
            </tr>
          </thead>
          <tbody>
            {entries.map((entry: any, i: number) => (
              <tr key={entry.id || i} style={{ borderBottom: '1px solid var(--border)' }}>
                <td style={tdStyle} class="font-mono text-sm">{fmtTime(entry.timestamp)}</td>
                <td style={tdStyle}>{entry.actor || '-'}</td>
                <td style={tdStyle}>
                  <code style={{ fontSize: '11px' }}>{entry.action}</code>
                </td>
                <td style={tdStyle}>{entry.resource || '-'}</td>
              </tr>
            ))}
            {entries.length === 0 && (
              <tr><td colSpan={4} style={{ padding: '24px', textAlign: 'center', opacity: 0.5 }}>No audit entries found.</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
