import { useState, useEffect } from 'preact/hooks';
import {
  categorizeServices,
  filterServices,
  displayName,
  type AWSServiceInfo,
} from './service-catalog';
import './styles.css';

interface ServiceFromAPI {
  name: string;
  actions: string[];
  healthy: boolean;
}

const ADMIN_BASE = detectAdminBase();

function detectAdminBase(): string {
  if (typeof window === 'undefined') return '';
  const port = window.location.port;
  // Dev server on 4501 proxies to admin API on 4599
  if (port === '4501') return '';
  // Production: served by gateway on 4500, admin API same origin
  return '';
}

async function api<T>(path: string): Promise<T> {
  const res = await fetch(`${ADMIN_BASE}${path}`);
  if (!res.ok) throw new Error(`API ${res.status}`);
  return res.json();
}

const CATEGORY_ORDER = [
  'Compute', 'Storage', 'Database', 'Networking', 'Security',
  'Integration', 'Management', 'Developer Tools', 'AI & ML',
  'Analytics', 'Containers & Serverless', 'IoT', 'Media & Migration', 'Other',
];

export function App() {
  const [services, setServices] = useState<AWSServiceInfo[]>([]);
  const [search, setSearch] = useState('');
  const [selectedService, setSelectedService] = useState<string | null>(null);
  const [resources, setResources] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    api<ServiceFromAPI[]>('/api/services')
      .then((data) => {
        setServices(data.map((s) => ({
          name: s.name,
          actions: s.actions?.length || 0,
          healthy: s.healthy,
        })));
        setLoading(false);
      })
      .catch((e) => {
        setError('Cannot reach CloudMock gateway. Is it running on port 4599?');
        setLoading(false);
      });
  }, []);

  useEffect(() => {
    if (!selectedService) { setResources(null); return; }
    api<any>(`/api/resources/${selectedService}`)
      .then(setResources)
      .catch(() => setResources({ resources: {} }));
  }, [selectedService]);

  const filtered = filterServices(services, search);
  const groups = categorizeServices(filtered);
  const sortedCategories = CATEGORY_ORDER.filter((c) => groups[c]?.length > 0);

  return (
    <div class="console">
      {/* Header */}
      <header class="header">
        <div class="header-brand">
          <span class="header-logo">☁️</span>
          <h1 class="header-title">CloudMock</h1>
          <span class="header-subtitle">AWS Console</span>
        </div>
        <div class="header-search">
          <input
            class="search-input"
            type="text"
            placeholder={`Search ${services.length} services...`}
            value={search}
            onInput={(e) => {
              setSearch((e.target as HTMLInputElement).value);
              setSelectedService(null);
            }}
          />
        </div>
        <div class="header-status">
          <span class={`status-dot ${services.length > 0 ? 'connected' : ''}`} />
          <span class="status-text">
            {services.length > 0 ? `${services.length} services` : 'Disconnected'}
          </span>
        </div>
      </header>

      {/* Error */}
      {error && <div class="error-banner">{error}</div>}

      {/* Loading */}
      {loading && <div class="loading">Loading AWS services...</div>}

      {/* Main content */}
      {!loading && !error && (
        <div class="main">
          {/* Service grid */}
          <div class={`grid ${selectedService ? 'with-detail' : ''}`}>
            {sortedCategories.map((category) => (
              <section class="category" key={category}>
                <h2 class="category-title">{category}</h2>
                <div class="category-grid">
                  {groups[category].map((svc) => (
                    <button
                      key={svc.name}
                      class={`service-card ${selectedService === svc.name ? 'selected' : ''} ${svc.healthy ? '' : 'degraded'}`}
                      onClick={() => setSelectedService(selectedService === svc.name ? null : svc.name)}
                    >
                      <div class="service-card-top">
                        <span class={`service-dot ${svc.healthy ? 'healthy' : 'unhealthy'}`} />
                        <span class="service-name">{displayName(svc.name)}</span>
                      </div>
                      <div class="service-card-bottom">
                        <code class="service-id">{svc.name}</code>
                        <span class="service-ops">{svc.actions} ops</span>
                      </div>
                    </button>
                  ))}
                </div>
              </section>
            ))}

            {sortedCategories.length === 0 && search && (
              <div class="empty">No services match "{search}"</div>
            )}
          </div>

          {/* Detail panel */}
          {selectedService && (
            <aside class="detail">
              <div class="detail-header">
                <h2 class="detail-title">{displayName(selectedService)}</h2>
                <button class="detail-close" onClick={() => setSelectedService(null)}>×</button>
              </div>

              <div class="detail-section">
                <div class="detail-row">
                  <span class="detail-label">Service</span>
                  <code class="detail-value">{selectedService}</code>
                </div>
                <div class="detail-row">
                  <span class="detail-label">Status</span>
                  <span class={`badge ${services.find((s) => s.name === selectedService)?.healthy ? 'badge-green' : 'badge-red'}`}>
                    {services.find((s) => s.name === selectedService)?.healthy ? 'Active' : 'Degraded'}
                  </span>
                </div>
                <div class="detail-row">
                  <span class="detail-label">Operations</span>
                  <span class="detail-value">{services.find((s) => s.name === selectedService)?.actions || 0}</span>
                </div>
                <div class="detail-row">
                  <span class="detail-label">Endpoint</span>
                  <code class="detail-value">http://localhost:4566</code>
                </div>
              </div>

              <h3 class="detail-subtitle">Resources</h3>
              {resources?.resources && typeof resources.resources === 'object' ? (
                <div class="resource-list">
                  {Object.entries(resources.resources).length > 0 ? (
                    Object.entries(resources.resources).map(([type, items]: [string, any]) => (
                      <div key={type} class="resource-item">
                        <span class="resource-type">{type}</span>
                        <span class="resource-count">
                          {Array.isArray(items) ? items.length : '—'}
                        </span>
                      </div>
                    ))
                  ) : (
                    <div class="resource-empty">No resources created yet. Use the AWS SDK to create resources.</div>
                  )}
                </div>
              ) : (
                <div class="resource-empty">Loading...</div>
              )}

              <div class="detail-actions">
                <a
                  class="detail-link"
                  href={`http://localhost:4500`}
                  target="_blank"
                  rel="noopener"
                >
                  Open in DevTools →
                </a>
              </div>
            </aside>
          )}
        </div>
      )}
    </div>
  );
}
