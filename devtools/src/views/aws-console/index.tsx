import { useState, useEffect } from 'preact/hooks';
import { api } from '../../lib/api';
import {
  categorizeServices,
  filterServices,
  displayName,
  type AWSServiceInfo,
} from './service-catalog';
import './aws-console.css';

interface ServiceFromAPI {
  name: string;
  actions: string[];
  healthy: boolean;
}

export function AWSConsoleView() {
  const [services, setServices] = useState<AWSServiceInfo[]>([]);
  const [search, setSearch] = useState('');
  const [selectedService, setSelectedService] = useState<string | null>(null);
  const [resources, setResources] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api<ServiceFromAPI[]>('/api/services')
      .then((data) => {
        const mapped: AWSServiceInfo[] = data.map((s) => ({
          name: s.name,
          actions: s.actions?.length || 0,
          healthy: s.healthy,
        }));
        setServices(mapped);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (!selectedService) {
      setResources(null);
      return;
    }
    api<any>(`/api/resources/${selectedService}`)
      .then(setResources)
      .catch(() => setResources(null));
  }, [selectedService]);

  const filtered = filterServices(services, search);
  const groups = categorizeServices(filtered);

  // Sort categories: put ones with services first
  const categoryOrder = [
    'Compute', 'Storage', 'Database', 'Networking', 'Security',
    'Integration', 'Management', 'Developer Tools', 'AI & ML',
    'Analytics', 'Containers & Serverless', 'IoT', 'Media & Migration', 'Other',
  ];
  const sortedCategories = categoryOrder.filter((c) => groups[c]?.length > 0);

  if (loading) {
    return (
      <div class="aws-console">
        <div class="aws-console-header">
          <h2 class="aws-console-title">AWS Console</h2>
        </div>
        <div class="aws-console-loading">Loading services...</div>
      </div>
    );
  }

  return (
    <div class="aws-console">
      <div class="aws-console-header">
        <div class="aws-console-header-left">
          <h2 class="aws-console-title">AWS Console</h2>
          <span class="aws-console-count">{services.length} services</span>
        </div>
        <input
          class="input aws-console-search"
          type="text"
          placeholder="Search services..."
          value={search}
          onInput={(e) => {
            setSearch((e.target as HTMLInputElement).value);
            setSelectedService(null);
          }}
        />
      </div>

      <div class="aws-console-body">
        {/* Service grid */}
        <div class={`aws-console-grid ${selectedService ? 'with-detail' : ''}`}>
          {sortedCategories.map((category) => (
            <div class="aws-category" key={category}>
              <h3 class="aws-category-title">{category}</h3>
              <div class="aws-category-services">
                {groups[category].map((svc) => (
                  <button
                    key={svc.name}
                    class={`aws-service-card ${selectedService === svc.name ? 'selected' : ''} ${svc.healthy ? '' : 'degraded'}`}
                    onClick={() => setSelectedService(selectedService === svc.name ? null : svc.name)}
                  >
                    <span class="aws-service-dot" />
                    <span class="aws-service-name">{displayName(svc.name)}</span>
                    <span class="aws-service-ops">{svc.actions} ops</span>
                  </button>
                ))}
              </div>
            </div>
          ))}

          {sortedCategories.length === 0 && (
            <div class="aws-console-empty">
              No services match "{search}"
            </div>
          )}
        </div>

        {/* Detail panel */}
        {selectedService && (
          <div class="aws-detail-panel">
            <div class="aws-detail-header">
              <h3 class="aws-detail-title">{displayName(selectedService)}</h3>
              <button class="aws-detail-close" onClick={() => setSelectedService(null)}>×</button>
            </div>
            <div class="aws-detail-meta">
              <span class="aws-detail-label">Service Name</span>
              <code class="aws-detail-value">{selectedService}</code>
            </div>
            <div class="aws-detail-meta">
              <span class="aws-detail-label">Status</span>
              <span class={`badge ${services.find((s) => s.name === selectedService)?.healthy ? 'badge-green' : 'badge-red'}`}>
                {services.find((s) => s.name === selectedService)?.healthy ? 'Healthy' : 'Degraded'}
              </span>
            </div>
            <div class="aws-detail-meta">
              <span class="aws-detail-label">Operations</span>
              <span class="aws-detail-value">{services.find((s) => s.name === selectedService)?.actions || 0}</span>
            </div>

            <h4 class="aws-detail-section-title">Resources</h4>
            {resources ? (
              <div class="aws-detail-resources">
                {resources.resources && typeof resources.resources === 'object' ? (
                  Object.entries(resources.resources).map(([type, items]: [string, any]) => (
                    <div key={type} class="aws-resource-group">
                      <div class="aws-resource-type">{type}</div>
                      {Array.isArray(items) ? (
                        <div class="aws-resource-count">{items.length} items</div>
                      ) : (
                        <div class="aws-resource-count">—</div>
                      )}
                    </div>
                  ))
                ) : (
                  <div class="aws-resource-empty">No resources created yet</div>
                )}
              </div>
            ) : (
              <div class="aws-resource-empty">Loading resources...</div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
