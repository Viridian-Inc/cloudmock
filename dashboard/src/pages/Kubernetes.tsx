import { useState, useEffect } from 'preact/hooks';
import { k8sGet, k8sDelete } from '../api';

interface KubeResource {
  metadata: { name: string; namespace?: string; creationTimestamp?: string; labels?: Record<string, string>; uid?: string };
  spec?: any;
  status?: any;
}

type ResourceTab = 'pods' | 'deployments' | 'services' | 'configmaps' | 'secrets' | 'namespaces' | 'nodes';

export function KubernetesPage({ showToast }: { showToast: (msg: string) => void }) {
  const [tab, setTab] = useState<ResourceTab>('pods');
  const [namespace, setNamespace] = useState('default');
  const [namespaces, setNamespaces] = useState<string[]>([]);
  const [resources, setResources] = useState<KubeResource[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    k8sGet('/api/v1/namespaces').then((d: any) => {
      setNamespaces(d.items.map((n: any) => n.metadata.name));
    }).catch(() => {});
  }, []);

  useEffect(() => {
    setLoading(true);
    const path = getResourcePath(tab, namespace);
    k8sGet(path).then((d: any) => {
      setResources(d.items || []);
    }).catch(() => setResources([])).finally(() => setLoading(false));
  }, [tab, namespace]);

  function getResourcePath(t: ResourceTab, ns: string): string {
    switch (t) {
      case 'pods': return `/api/v1/namespaces/${ns}/pods`;
      case 'services': return `/api/v1/namespaces/${ns}/services`;
      case 'configmaps': return `/api/v1/namespaces/${ns}/configmaps`;
      case 'secrets': return `/api/v1/namespaces/${ns}/secrets`;
      case 'deployments': return `/apis/apps/v1/namespaces/${ns}/deployments`;
      case 'namespaces': return '/api/v1/namespaces';
      case 'nodes': return '/api/v1/nodes';
    }
  }

  async function handleDelete(r: KubeResource) {
    const ns = r.metadata.namespace || namespace;
    const name = r.metadata.name;
    let path: string;
    switch (tab) {
      case 'pods': path = `/api/v1/namespaces/${ns}/pods/${name}`; break;
      case 'services': path = `/api/v1/namespaces/${ns}/services/${name}`; break;
      case 'configmaps': path = `/api/v1/namespaces/${ns}/configmaps/${name}`; break;
      case 'secrets': path = `/api/v1/namespaces/${ns}/secrets/${name}`; break;
      case 'deployments': path = `/apis/apps/v1/namespaces/${ns}/deployments/${name}`; break;
      case 'namespaces': path = `/api/v1/namespaces/${name}`; break;
      default: return;
    }
    try {
      await k8sDelete(path);
      showToast(`Deleted ${tab.slice(0, -1)} "${name}"`);
      setResources(prev => prev.filter(p => p.metadata.name !== name));
    } catch (e: any) {
      showToast(`Error: ${e.message}`);
    }
  }

  const tabs: { key: ResourceTab; label: string }[] = [
    { key: 'pods', label: 'Pods' },
    { key: 'deployments', label: 'Deployments' },
    { key: 'services', label: 'Services' },
    { key: 'configmaps', label: 'ConfigMaps' },
    { key: 'secrets', label: 'Secrets' },
    { key: 'namespaces', label: 'Namespaces' },
    { key: 'nodes', label: 'Nodes' },
  ];

  const isClusterScoped = tab === 'namespaces' || tab === 'nodes';

  return (
    <div style={{ padding: '24px' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '16px', marginBottom: '20px' }}>
        <h2 style={{ margin: 0 }}>Kubernetes Resources</h2>
        <span style={{ fontSize: '12px', color: 'var(--text-tertiary)', background: 'var(--bg-secondary)', padding: '2px 8px', borderRadius: '4px' }}>
          {resources.length} items
        </span>
      </div>

      <div style={{ display: 'flex', gap: '8px', marginBottom: '16px', flexWrap: 'wrap' }}>
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
          >{t.label}</button>
        ))}
      </div>

      {!isClusterScoped && (
        <div style={{ marginBottom: '16px' }}>
          <label style={{ fontSize: '13px', color: 'var(--text-secondary)', marginRight: '8px' }}>Namespace:</label>
          <select
            value={namespace}
            onChange={(e: any) => setNamespace(e.target.value)}
            style={{ padding: '4px 8px', borderRadius: '4px', border: '1px solid var(--border)', background: 'var(--bg-secondary)', color: 'var(--text-primary)' }}
          >
            {namespaces.map(ns => <option key={ns} value={ns}>{ns}</option>)}
          </select>
        </div>
      )}

      {loading ? (
        <div style={{ color: 'var(--text-tertiary)', padding: '32px', textAlign: 'center' }}>Loading...</div>
      ) : resources.length === 0 ? (
        <div style={{ color: 'var(--text-tertiary)', padding: '32px', textAlign: 'center' }}>No {tab} found</div>
      ) : (
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
          <thead>
            <tr style={{ borderBottom: '1px solid var(--border)', color: 'var(--text-secondary)' }}>
              <th style={{ textAlign: 'left', padding: '8px' }}>Name</th>
              {!isClusterScoped && <th style={{ textAlign: 'left', padding: '8px' }}>Namespace</th>}
              <th style={{ textAlign: 'left', padding: '8px' }}>Status</th>
              <th style={{ textAlign: 'left', padding: '8px' }}>Age</th>
              <th style={{ textAlign: 'right', padding: '8px' }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {resources.map(r => (
              <tr key={r.metadata.uid || r.metadata.name} style={{ borderBottom: '1px solid var(--border)' }}>
                <td style={{ padding: '8px', fontFamily: 'monospace' }}>{r.metadata.name}</td>
                {!isClusterScoped && <td style={{ padding: '8px' }}>{r.metadata.namespace || '-'}</td>}
                <td style={{ padding: '8px' }}>
                  <span style={{
                    padding: '2px 6px', borderRadius: '4px', fontSize: '11px',
                    background: getStatusColor(r, tab),
                    color: '#fff',
                  }}>
                    {getStatus(r, tab)}
                  </span>
                </td>
                <td style={{ padding: '8px', color: 'var(--text-secondary)' }}>{r.metadata.creationTimestamp || '-'}</td>
                <td style={{ padding: '8px', textAlign: 'right' }}>
                  {tab !== 'nodes' && (
                    <button onClick={() => handleDelete(r)} style={{ padding: '2px 8px', borderRadius: '4px', border: '1px solid var(--border)', background: 'transparent', color: 'var(--text-secondary)', cursor: 'pointer', fontSize: '11px' }}>Delete</button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

function getStatus(r: KubeResource, tab: ResourceTab): string {
  switch (tab) {
    case 'pods': return r.status?.phase || 'Unknown';
    case 'deployments': return `${r.status?.readyReplicas || 0}/${r.spec?.replicas || 0} ready`;
    case 'services': return r.spec?.type || 'ClusterIP';
    case 'namespaces': return r.status?.phase || 'Active';
    case 'nodes': return r.status?.conditions?.find((c: any) => c.type === 'Ready')?.status === 'True' ? 'Ready' : 'NotReady';
    default: return 'Active';
  }
}

function getStatusColor(r: KubeResource, tab: ResourceTab): string {
  const status = getStatus(r, tab);
  if (status === 'Running' || status === 'Active' || status === 'Ready' || status.includes('ready')) return '#22c55e';
  if (status === 'Pending' || status === 'Progressing') return '#f59e0b';
  if (status === 'Failed' || status === 'NotReady') return '#ef4444';
  return '#6b7280';
}
