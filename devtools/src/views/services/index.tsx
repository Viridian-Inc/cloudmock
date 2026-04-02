import { useState, useEffect } from 'preact/hooks';
import { SplitPanel } from '../../components/panels/split-panel';
import { api, cachedApi, getServices, getResources } from '../../lib/api';
import type { ServiceInfo } from '../../lib/types';
import { ServiceList } from './service-list';
import { ResourceBrowser } from './resource-browser';
import './services.css';

/** App service loaded from IaC topology config */
interface AppService {
  id: string;
  name: string;
  icon: string;
  type: string;
  group: string;
  awsDeps: string[]; // AWS service node IDs this app connects to
}

const GROUP_ICONS: Record<string, string> = {
  Client: '📱',
  API: '🔀',
  Compute: '⚙️',
  Plugins: '🔌',
};

const TYPE_ICONS: Record<string, string> = {
  client: '📱',
  server: '🖥️',
  plugin: '🔌',
};

/**
 * Load app services from IaC topology config (Pulumi/Terraform).
 * Falls back to empty if not seeded.
 */
async function loadAppServicesFromIaC(): Promise<AppService[]> {
  try {
    const config = await api<{ nodes: any[] | null; edges: any[] | null }>('/api/topology/config');
    const nodes = config.nodes || [];
    const edges = config.edges || [];

    return nodes.map((n) => {
      // Find AWS deps: edges where this node is the source
      const deps = edges
        .filter((e: any) => e.source === n.id)
        .map((e: any) => e.target)
        .filter((t: string) => !t.startsWith('external:') && !t.startsWith('plugin:'));

      return {
        id: n.id,
        name: n.label,
        icon: TYPE_ICONS[n.type] || GROUP_ICONS[n.group] || '⚙️',
        type: n.type,
        group: n.group,
        awsDeps: deps,
      };
    });
  } catch {
    return [];
  }
}

export function ServicesView() {
  const [services, setServices] = useState<ServiceInfo[]>([]);
  const [appServices, setAppServices] = useState<AppService[]>([]);
  const [selectedService, setSelectedService] = useState<string | null>(null);
  const [selectedType, setSelectedType] = useState<'app' | 'aws' | null>(null);
  const [resources, setResources] = useState<any>(null);
  const [loadingResources, setLoadingResources] = useState(false);

  useEffect(() => {
    cachedApi<ServiceInfo[]>('/api/services', 'services:list').then(setServices).catch(() => setServices([]));
    loadAppServicesFromIaC().then(setAppServices);
  }, []);

  useEffect(() => {
    if (!selectedService || selectedType !== 'aws') {
      if (selectedType === 'app') return;
      setResources(null);
      return;
    }

    setLoadingResources(true);
    getResources(selectedService)
      .then((data) => setResources(data.resources))
      .catch(() => setResources(null))
      .finally(() => setLoadingResources(false));
  }, [selectedService, selectedType]);

  const handleSelectApp = (id: string) => {
    setSelectedService(id);
    setSelectedType('app');
    setResources(null);
  };

  const handleSelectAws = (name: string) => {
    setSelectedService(name);
    setSelectedType('aws');
  };

  const selectedApp = selectedType === 'app'
    ? appServices.find((a) => a.id === selectedService) ?? null
    : null;

  return (
    <div class="services-view">
      <SplitPanel
        initialSplit={30}
        direction="horizontal"
        minSize={160}
        left={
          <ServiceList
            services={services}
            appServices={appServices}
            selectedService={selectedService}
            selectedType={selectedType}
            onSelectApp={handleSelectApp}
            onSelectAws={handleSelectAws}
          />
        }
        right={
          selectedApp ? (
            <AppServiceDetail app={selectedApp} awsServices={services} />
          ) : (
            <ResourceBrowser
              service={selectedType === 'aws' ? selectedService : null}
              resources={resources}
              loading={loadingResources}
            />
          )
        }
      />
    </div>
  );
}

function AppServiceDetail({ app, awsServices }: { app: AppService; awsServices: ServiceInfo[] }) {
  return (
    <div class="app-service-detail">
      <div class="app-service-detail-header">
        <span class="app-service-detail-icon">{app.icon}</span>
        <div>
          <div class="app-service-detail-name">{app.name}</div>
          <div class="app-service-detail-runtime">{app.group} · {app.type}</div>
        </div>
      </div>
      <div class="app-service-detail-body">
        {app.awsDeps.length > 0 && (
          <div class="app-service-section">
            <div class="app-service-section-title">AWS Dependencies ({app.awsDeps.length})</div>
            <div class="app-service-deps">
              {app.awsDeps.map((dep) => {
                const parts = dep.split(':');
                const svcName = parts[0];
                const resourceName = parts.slice(1).join(':');
                const info = awsServices.find((s) => s.name === svcName);
                return (
                  <div key={dep} class="app-service-dep">
                    <span class={`service-health-dot ${info?.healthy !== false ? 'healthy' : 'unhealthy'}`} />
                    <span class="app-service-dep-name">{resourceName || svcName}</span>
                    <span class="app-service-dep-type">{svcName}</span>
                  </div>
                );
              })}
            </div>
          </div>
        )}
        {app.awsDeps.length === 0 && (
          <div class="app-service-section">
            <div class="app-service-section-title">No AWS Dependencies</div>
            <p style="color: var(--text-tertiary); font-size: 12px;">
              This service connects to other services but not directly to AWS resources.
            </p>
          </div>
        )}
        <div class="app-service-section">
          <div class="app-service-section-title">ID</div>
          <code style="font-size: 11px; color: var(--text-accent);">{app.id}</code>
        </div>
      </div>
    </div>
  );
}
