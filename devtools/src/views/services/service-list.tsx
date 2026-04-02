import { useState, useRef, useCallback } from 'preact/hooks';
import type { ServiceInfo } from '../../lib/types';

interface AppService {
  id: string;
  name: string;
  icon: string;
  type: string;
  group: string;
  awsDeps: string[];
}

interface ServiceListProps {
  services: ServiceInfo[];
  appServices: AppService[];
  selectedService: string | null;
  selectedType: 'app' | 'aws' | null;
  onSelectApp: (id: string) => void;
  onSelectAws: (name: string) => void;
}

export function ServiceList({
  services,
  appServices,
  selectedService,
  selectedType,
  onSelectApp,
  onSelectAws,
}: ServiceListProps) {
  const [search, setSearch] = useState('');
  const [showStubs, setShowStubs] = useState(false);
  const listRef = useRef<HTMLDivElement>(null);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      const container = listRef.current;
      if (!container) return;
      const items = container.querySelectorAll<HTMLElement>('.service-list-item');
      if (items.length === 0) return;

      const focused = container.querySelector<HTMLElement>('.service-list-item:focus');
      const idx = focused ? Array.from(items).indexOf(focused) : -1;

      if (e.key === 'ArrowDown') {
        e.preventDefault();
        const next = idx < items.length - 1 ? idx + 1 : 0;
        items[next].focus();
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        const prev = idx > 0 ? idx - 1 : items.length - 1;
        items[prev].focus();
      } else if (e.key === 'Enter' && focused) {
        e.preventDefault();
        focused.click();
      }
    },
    [],
  );

  const q = search.toLowerCase();

  // Group app services by their group
  const appGroups = new Map<string, AppService[]>();
  for (const app of appServices) {
    if (!app.name.toLowerCase().includes(q)) continue;
    if (!appGroups.has(app.group)) appGroups.set(app.group, []);
    appGroups.get(app.group)!.push(app);
  }

  const filteredAws = services.filter((s) => s.name.toLowerCase().includes(q));
  const active = filteredAws.filter((s) => s.action_count > 0).sort((a, b) => b.action_count - a.action_count);
  const stubs = filteredAws.filter((s) => s.action_count === 0).sort((a, b) => a.name.localeCompare(b.name));

  return (
    <div class="service-list">
      <div class="service-list-header">
        <input
          class="input service-list-search"
          type="text"
          placeholder="Filter services..."
          value={search}
          onInput={(e) => setSearch((e.target as HTMLInputElement).value)}
        />
      </div>
      <div class="service-list-body" ref={listRef} tabIndex={0} onKeyDown={handleKeyDown}>

        {/* App services grouped by category */}
        {[...appGroups].map(([group, apps]) => (
          <div key={group}>
            <div class="service-group-header app-group">
              {group} ({apps.length})
            </div>
            {apps.map((app) => (
              <div
                key={app.id}
                class={`service-list-item app-item ${
                  selectedService === app.id && selectedType === 'app' ? 'service-list-item-selected' : ''
                }`}
                tabIndex={-1}
                onClick={() => onSelectApp(app.id)}
              >
                <span class="service-app-icon">{app.icon}</span>
                <span class="service-name">{app.name}</span>
                {app.awsDeps.length > 0 && (
                  <span class="service-action-count">{app.awsDeps.length}</span>
                )}
                <span class="service-runtime-badge">{app.type}</span>
              </div>
            ))}
          </div>
        ))}

        {/* Active AWS services */}
        {active.length > 0 && (
          <>
            <div class="service-group-header">
              Active AWS Services ({active.length})
            </div>
            {active.map((svc) => (
              <div
                key={`aws:${svc.name}`}
                class={`service-list-item ${
                  svc.name === selectedService && selectedType === 'aws' ? 'service-list-item-selected' : ''
                }`}
                tabIndex={-1}
                onClick={() => onSelectAws(svc.name)}
              >
                <span class={`service-health-dot ${svc.healthy ? 'healthy' : 'unhealthy'}`} />
                <span class="service-name">{svc.name}</span>
                {svc.action_count > 0 && (
                  <span class="service-action-count">{svc.action_count}</span>
                )}
              </div>
            ))}
          </>
        )}

        {/* Stub services */}
        {stubs.length > 0 && (
          <>
            <div
              class="service-group-header service-group-toggle"
              onClick={() => setShowStubs(!showStubs)}
            >
              {showStubs ? '▾' : '▸'} Stub Services ({stubs.length})
            </div>
            {showStubs &&
              stubs.map((svc) => (
                <div
                  key={`aws:${svc.name}`}
                  class={`service-list-item ${
                    svc.name === selectedService && selectedType === 'aws' ? 'service-list-item-selected' : ''
                  }`}
                  tabIndex={-1}
                  onClick={() => onSelectAws(svc.name)}
                >
                  <span class={`service-health-dot ${svc.healthy ? 'healthy' : 'unhealthy'}`} />
                  <span class="service-name">{svc.name}</span>
                </div>
              ))}
          </>
        )}

        {appGroups.size === 0 && filteredAws.length === 0 && (
          <div class="service-list-empty">No services found</div>
        )}
      </div>
    </div>
  );
}
