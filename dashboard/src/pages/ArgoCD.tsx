import { useState, useEffect } from 'preact/hooks';
import { argoGet, argoPost, argoDelete } from '../api';

interface ArgoApp {
  metadata: { name: string; uid?: string; creationTimestamp?: string };
  spec: { source: { repoURL: string; path?: string }; destination: { namespace?: string }; project?: string };
  status: { sync: { status: string }; health: { status: string; message?: string }; resources?: any[]; operationState?: any };
}

export function ArgoCDPage({ showToast }: { showToast: (msg: string) => void }) {
  const [tab, setTab] = useState<'apps' | 'repos' | 'clusters' | 'projects'>('apps');
  const [apps, setApps] = useState<ArgoApp[]>([]);
  const [repos, setRepos] = useState<any[]>([]);
  const [clusters, setClusters] = useState<any[]>([]);
  const [projects, setProjects] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [syncing, setSyncing] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    switch (tab) {
      case 'apps':
        argoGet('/api/v1/applications').then((d: any) => setApps(d.items || [])).catch(() => setApps([])).finally(() => setLoading(false));
        break;
      case 'repos':
        argoGet('/api/v1/repositories').then((d: any) => setRepos(d.items || [])).catch(() => setRepos([])).finally(() => setLoading(false));
        break;
      case 'clusters':
        argoGet('/api/v1/clusters').then((d: any) => setClusters(d.items || [])).catch(() => setClusters([])).finally(() => setLoading(false));
        break;
      case 'projects':
        argoGet('/api/v1/projects').then((d: any) => setProjects(d.items || [])).catch(() => setProjects([])).finally(() => setLoading(false));
        break;
    }
  }, [tab]);

  async function handleSync(name: string) {
    setSyncing(name);
    try {
      const updated: ArgoApp = await argoPost(`/api/v1/applications/${name}/sync`);
      setApps(prev => prev.map(a => a.metadata.name === name ? updated : a));
      showToast(`Synced "${name}" successfully`);
    } catch (e: any) {
      showToast(`Sync failed: ${e.message}`);
    } finally {
      setSyncing(null);
    }
  }

  async function handleDeleteApp(name: string) {
    try {
      await argoDelete(`/api/v1/applications/${name}`);
      setApps(prev => prev.filter(a => a.metadata.name !== name));
      showToast(`Deleted application "${name}"`);
    } catch (e: any) {
      showToast(`Error: ${e.message}`);
    }
  }

  const tabs = [
    { key: 'apps' as const, label: 'Applications', count: apps.length },
    { key: 'repos' as const, label: 'Repositories', count: repos.length },
    { key: 'clusters' as const, label: 'Clusters', count: clusters.length },
    { key: 'projects' as const, label: 'Projects', count: projects.length },
  ];

  return (
    <div style={{ padding: '24px' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '16px', marginBottom: '20px' }}>
        <h2 style={{ margin: 0 }}>ArgoCD</h2>
      </div>

      <div style={{ display: 'flex', gap: '8px', marginBottom: '16px' }}>
        {tabs.map(t => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            style={{
              padding: '6px 12px', borderRadius: '6px', border: '1px solid var(--border)',
              background: tab === t.key ? 'var(--accent)' : 'var(--bg-secondary)',
              color: tab === t.key ? '#fff' : 'var(--text-primary)',
              cursor: 'pointer', fontSize: '13px',
            }}
          >{t.label} ({t.count})</button>
        ))}
      </div>

      {loading ? (
        <div style={{ color: 'var(--text-tertiary)', padding: '32px', textAlign: 'center' }}>Loading...</div>
      ) : tab === 'apps' ? (
        apps.length === 0 ? (
          <div style={{ color: 'var(--text-tertiary)', padding: '32px', textAlign: 'center' }}>No applications. Use the ArgoCD CLI to create one.</div>
        ) : (
          <div style={{ display: 'grid', gap: '12px' }}>
            {apps.map(app => (
              <div key={app.metadata.name} style={{ border: '1px solid var(--border)', borderRadius: '8px', padding: '16px', background: 'var(--bg-secondary)' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                    <span style={{ fontWeight: 600, fontSize: '15px' }}>{app.metadata.name}</span>
                    <SyncBadge status={app.status?.sync?.status} />
                    <HealthBadge status={app.status?.health?.status} />
                  </div>
                  <div style={{ display: 'flex', gap: '8px' }}>
                    <button
                      onClick={() => handleSync(app.metadata.name)}
                      disabled={syncing === app.metadata.name}
                      style={{ padding: '4px 12px', borderRadius: '4px', border: '1px solid var(--accent)', background: 'transparent', color: 'var(--accent)', cursor: 'pointer', fontSize: '12px' }}
                    >{syncing === app.metadata.name ? 'Syncing...' : 'Sync'}</button>
                    <button
                      onClick={() => handleDeleteApp(app.metadata.name)}
                      style={{ padding: '4px 12px', borderRadius: '4px', border: '1px solid var(--border)', background: 'transparent', color: 'var(--text-secondary)', cursor: 'pointer', fontSize: '12px' }}
                    >Delete</button>
                  </div>
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '8px', fontSize: '12px', color: 'var(--text-secondary)' }}>
                  <div><span style={{ color: 'var(--text-tertiary)' }}>Repo:</span> {app.spec.source.repoURL}</div>
                  <div><span style={{ color: 'var(--text-tertiary)' }}>Path:</span> {app.spec.source.path || '/'}</div>
                  <div><span style={{ color: 'var(--text-tertiary)' }}>Namespace:</span> {app.spec.destination.namespace || 'default'}</div>
                  <div><span style={{ color: 'var(--text-tertiary)' }}>Project:</span> {app.spec.project || 'default'}</div>
                </div>
                {app.status?.resources && app.status.resources.length > 0 && (
                  <div style={{ marginTop: '8px', fontSize: '11px' }}>
                    <span style={{ color: 'var(--text-tertiary)' }}>Resources: </span>
                    {app.status.resources.map((r: any, i: number) => (
                      <span key={i} style={{ padding: '1px 6px', borderRadius: '3px', background: 'var(--bg-primary)', marginRight: '4px' }}>
                        {r.kind}/{r.name}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        )
      ) : tab === 'repos' ? (
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
          <thead>
            <tr style={{ borderBottom: '1px solid var(--border)', color: 'var(--text-secondary)' }}>
              <th style={{ textAlign: 'left', padding: '8px' }}>URL</th>
              <th style={{ textAlign: 'left', padding: '8px' }}>Type</th>
              <th style={{ textAlign: 'left', padding: '8px' }}>Status</th>
            </tr>
          </thead>
          <tbody>
            {repos.map(r => (
              <tr key={r.repo} style={{ borderBottom: '1px solid var(--border)' }}>
                <td style={{ padding: '8px', fontFamily: 'monospace' }}>{r.repo}</td>
                <td style={{ padding: '8px' }}>{r.type || 'git'}</td>
                <td style={{ padding: '8px' }}>
                  <span style={{ padding: '2px 6px', borderRadius: '4px', fontSize: '11px', background: r.connectionState?.status === 'Successful' ? '#22c55e' : '#ef4444', color: '#fff' }}>
                    {r.connectionState?.status || 'Unknown'}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : tab === 'clusters' ? (
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
          <thead>
            <tr style={{ borderBottom: '1px solid var(--border)', color: 'var(--text-secondary)' }}>
              <th style={{ textAlign: 'left', padding: '8px' }}>Name</th>
              <th style={{ textAlign: 'left', padding: '8px' }}>Server</th>
              <th style={{ textAlign: 'left', padding: '8px' }}>Status</th>
            </tr>
          </thead>
          <tbody>
            {clusters.map(c => (
              <tr key={c.server} style={{ borderBottom: '1px solid var(--border)' }}>
                <td style={{ padding: '8px' }}>{c.name}</td>
                <td style={{ padding: '8px', fontFamily: 'monospace' }}>{c.server}</td>
                <td style={{ padding: '8px' }}>
                  <span style={{ padding: '2px 6px', borderRadius: '4px', fontSize: '11px', background: '#22c55e', color: '#fff' }}>
                    {c.connectionState?.status || 'Connected'}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : (
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
          <thead>
            <tr style={{ borderBottom: '1px solid var(--border)', color: 'var(--text-secondary)' }}>
              <th style={{ textAlign: 'left', padding: '8px' }}>Name</th>
              <th style={{ textAlign: 'left', padding: '8px' }}>Description</th>
              <th style={{ textAlign: 'left', padding: '8px' }}>Source Repos</th>
            </tr>
          </thead>
          <tbody>
            {projects.map(p => (
              <tr key={p.metadata.name} style={{ borderBottom: '1px solid var(--border)' }}>
                <td style={{ padding: '8px' }}>{p.metadata.name}</td>
                <td style={{ padding: '8px', color: 'var(--text-secondary)' }}>{p.spec?.description || '-'}</td>
                <td style={{ padding: '8px', fontFamily: 'monospace', fontSize: '11px' }}>{(p.spec?.sourceRepos || []).join(', ')}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

function SyncBadge({ status }: { status?: string }) {
  const color = status === 'Synced' ? '#22c55e' : status === 'OutOfSync' ? '#f59e0b' : '#6b7280';
  return (
    <span style={{ padding: '2px 6px', borderRadius: '4px', fontSize: '11px', background: color, color: '#fff' }}>
      {status || 'Unknown'}
    </span>
  );
}

function HealthBadge({ status }: { status?: string }) {
  const color = status === 'Healthy' ? '#22c55e' : status === 'Degraded' ? '#f59e0b' : status === 'Progressing' ? '#3b82f6' : '#6b7280';
  return (
    <span style={{ padding: '2px 6px', borderRadius: '4px', fontSize: '11px', background: color, color: '#fff' }}>
      {status || 'Unknown'}
    </span>
  );
}
