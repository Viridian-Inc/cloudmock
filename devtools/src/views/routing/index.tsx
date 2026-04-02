import { useState, useEffect, useCallback } from 'preact/hooks';
import { api, getAdminBase } from '../../lib/api';
import type { ServiceInfo } from '../../lib/types';
import './routing.css';

interface ServiceRoute {
  name: string;
  mode: 'local' | 'cloud';
  localEndpoint: string;
  cloudEndpoints: Record<string, string>; // env → url
  healthy: boolean;
  group: string;
}

interface ProxyRouteStatus {
  service: string;
  synced: boolean;
  error?: string;
}

interface EnvironmentInfo {
  name: string;
  source: 'env' | 'pulumi' | 'default';
  cloudmockEnv?: string;
}

interface TopologyNode {
  id: string;
  label: string;
  type: string;
  group: string;
  port?: number;
}

const STORAGE_KEY = 'neureaux-devtools:routing';

/** Derive a local endpoint from a topology node */
function deriveLocalEndpoint(node: TopologyNode): string {
  if (node.port) return `http://localhost:${node.port}`;
  if (node.group === 'AWS') return 'cloudmock';
  if (node.type === 'plugin') return 'cloudmock';
  if (node.type === 'lambda' || node.type === 'function') return 'cloudmock (lambda)';
  return 'cloudmock';
}

/** Map topology group names to routing display groups */
function mapGroup(group: string): string {
  if (group === 'AWS') return 'AWS Services';
  return group || 'Other';
}

/** Build routes from API data, merging with any saved user toggles */
function mergeWithSaved(
  apiRoutes: Omit<ServiceRoute, 'healthy'>[],
): ServiceRoute[] {
  let saved: ServiceRoute[] | null = null;
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) saved = JSON.parse(raw);
  } catch (e) { console.warn('[Routing] Failed to parse saved routes:', e); }

  if (!saved || saved.length === 0) {
    return apiRoutes.map((r) => ({ ...r, healthy: true }));
  }

  const savedByName = new Map(saved.map((r) => [r.name, r]));
  return apiRoutes.map((r) => {
    const prev = savedByName.get(r.name);
    return {
      ...r,
      mode: prev?.mode ?? r.mode,
      healthy: prev?.healthy ?? true,
    };
  });
}

function saveRoutes(routes: ServiceRoute[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(routes));
}

/** Derive a simple service key from the display name */
function serviceKey(name: string): string {
  return name.toLowerCase().replace(/\s+/g, '-');
}

/** Generate env var commands for a route */
function getEnvVarCommands(route: ServiceRoute, env: string): string[] {
  const cmds: string[] = [];
  const svcKey = serviceKey(route.name);

  if (route.group === 'AWS Services') {
    if (route.mode === 'local') {
      cmds.push(`export AWS_ENDPOINT_URL=http://localhost:4566`);
    } else {
      cmds.push(`unset AWS_ENDPOINT_URL`);
    }
  } else if (route.localEndpoint.startsWith('http://')) {
    const envVarName = `${svcKey.toUpperCase().replace(/-/g, '_')}_URL`;
    if (route.mode === 'local') {
      cmds.push(`export ${envVarName}=${route.localEndpoint}`);
    } else {
      const cloudUrl = route.cloudEndpoints[env] || route.cloudEndpoints.dev || '';
      cmds.push(`export ${envVarName}=${cloudUrl}`);
    }
  }

  return cmds;
}

/** Detect active environment from context */
function detectEnvironment(): EnvironmentInfo {
  // Check for CLOUDMOCK_ENV or NEUREAUX_ENV in localStorage (set by config)
  const storedEnv = localStorage.getItem('neureaux-devtools:active-env');
  if (storedEnv) {
    return { name: storedEnv, source: 'env', cloudmockEnv: storedEnv };
  }

  // Try to load from service-domains.json which may contain environment info
  return { name: 'local', source: 'default' };
}

/** Try to load cloud URLs from Pulumi stack outputs or service-domains.json */
async function loadCloudEndpoints(): Promise<Record<string, Record<string, string>> | null> {
  try {
    const res = await fetch('/service-domains.json');
    if (!res.ok) return null;
    const data = await res.json();
    if (data.routing && typeof data.routing === 'object') {
      return data.routing as Record<string, Record<string, string>>;
    }
  } catch (e) { console.warn('[Routing] Failed to load cloud endpoints:', e); }
  return null;
}

export function RoutingView() {
  const [routes, setRoutes] = useState<ServiceRoute[]>([]);
  const [loadingRoutes, setLoadingRoutes] = useState(true);
  const [env, setEnv] = useState<'dev' | 'staging' | 'prod'>('dev');
  const [search, setSearch] = useState('');
  const [proxyStatuses, setProxyStatuses] = useState<Map<string, ProxyRouteStatus>>(new Map());
  const [showHowToApply, setShowHowToApply] = useState(false);
  const [envInfo, setEnvInfo] = useState<EnvironmentInfo>(() => detectEnvironment());
  const [proxyAvailable, setProxyAvailable] = useState<boolean | null>(null);

  // Load routes from API on mount
  useEffect(() => {
    const info = detectEnvironment();
    setEnvInfo(info);

    (async () => {
      try {
        // Fetch active services and topology config in parallel
        const [servicesResult, topologyResult, cloudRouting] = await Promise.all([
          api<ServiceInfo[]>('/api/services').catch(() => [] as ServiceInfo[]),
          api<{ nodes: TopologyNode[] | null; edges: any[] | null }>('/api/topology/config').catch(() => ({ nodes: null, edges: null })),
          loadCloudEndpoints(),
        ]);

        const nodes = topologyResult.nodes || [];
        const serviceNames = new Set((servicesResult || []).map((s) => s.name));

        // Build routes from topology nodes
        const apiRoutes: Omit<ServiceRoute, 'healthy'>[] = nodes.map((node) => {
          const cloudEndpoints: Record<string, string> = {};
          // Merge cloud endpoints from service-domains.json if available
          const key = serviceKey(node.label);
          if (cloudRouting && cloudRouting[key]) {
            Object.assign(cloudEndpoints, cloudRouting[key]);
          }

          return {
            name: node.label,
            mode: 'local' as const,
            localEndpoint: deriveLocalEndpoint(node),
            cloudEndpoints,
            group: mapGroup(node.group),
          };
        });

        // Add any active services not already represented in topology
        for (const svc of servicesResult || []) {
          if (!apiRoutes.some((r) => serviceKey(r.name) === serviceKey(svc.name))) {
            const key = serviceKey(svc.name);
            const cloudEndpoints: Record<string, string> = {};
            if (cloudRouting && cloudRouting[key]) {
              Object.assign(cloudEndpoints, cloudRouting[key]);
            }
            apiRoutes.push({
              name: svc.name,
              mode: 'local',
              localEndpoint: 'cloudmock',
              cloudEndpoints,
              group: 'Other',
            });
          }
        }

        setRoutes(mergeWithSaved(apiRoutes));
      } catch (e) {
        console.warn('[Routing] API fetch failed:', e);
        // If all fetches fail, start with empty — no hardcoded fallback
        setRoutes([]);
      } finally {
        setLoadingRoutes(false);
      }
    })();

    // Check if cloudmock proxy API is available
    fetch(`${getAdminBase()}/api/proxy/routes`, { method: 'GET', signal: AbortSignal.timeout(2000) })
      .then((res) => setProxyAvailable(res.ok || res.status === 404))
      .catch(() => setProxyAvailable(false));
  }, []);

  // Health check local services
  useEffect(() => {
    async function checkHealth() {
      const updated = await Promise.all(
        routes.map(async (r) => {
          if (r.mode === 'local' && r.localEndpoint.startsWith('http://')) {
            try {
              const res = await fetch(`${r.localEndpoint}/health`, { signal: AbortSignal.timeout(2000) });
              return { ...r, healthy: res.ok };
            } catch (e) {
              console.debug('[Routing] Health check failed for', r.name, e);
              return { ...r, healthy: false };
            }
          }
          return { ...r, healthy: true };
        }),
      );
      setRoutes(updated);
    }
    checkHealth();
    const interval = setInterval(checkHealth, 30000);
    return () => clearInterval(interval);
  }, []);

  /** POST route change to cloudmock proxy, with fallback to env var instructions */
  const syncRouteToProxy = useCallback(async (route: ServiceRoute) => {
    const svc = serviceKey(route.name);
    const endpoint = route.mode === 'local'
      ? route.localEndpoint
      : (route.cloudEndpoints[env] || route.cloudEndpoints.dev || '');

    try {
      const url = `${getAdminBase()}/api/proxy/routes`;
      const res = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          service: svc,
          mode: route.mode,
          endpoint,
        }),
      });

      if (res.ok) {
        setProxyStatuses((prev) => {
          const next = new Map(prev);
          next.set(svc, { service: svc, synced: true });
          return next;
        });
        return;
      }

      // If endpoint doesn't exist, flag as not synced
      const body = await res.text().catch(() => '');
      setProxyStatuses((prev) => {
        const next = new Map(prev);
        next.set(svc, { service: svc, synced: false, error: `${res.status}: ${body || res.statusText}` });
        return next;
      });
    } catch (err: any) {
      setProxyStatuses((prev) => {
        const next = new Map(prev);
        next.set(svc, { service: svc, synced: false, error: err.message || 'Network error' });
        return next;
      });
    }
  }, [env]);

  const toggleMode = async (name: string) => {
    setRoutes((prev) => {
      const updated = prev.map((r) =>
        r.name === name ? { ...r, mode: r.mode === 'local' ? 'cloud' as const : 'local' as const } : r,
      );
      saveRoutes(updated);
      const changedRoute = updated.find((r) => r.name === name);
      if (changedRoute) syncRouteToProxy(changedRoute);
      return updated;
    });
  };

  const setGroupMode = async (group: string, mode: 'local' | 'cloud') => {
    setRoutes((prev) => {
      const updated = prev.map((r) => r.group === group ? { ...r, mode } : r);
      saveRoutes(updated);
      for (const r of updated.filter((r) => r.group === group)) syncRouteToProxy(r);
      return updated;
    });
  };

  const setAllMode = async (mode: 'local' | 'cloud') => {
    setRoutes((prev) => {
      const updated = prev.map((r) => ({ ...r, mode }));
      saveRoutes(updated);
      for (const r of updated) syncRouteToProxy(r);
      return updated;
    });
  };

  // Group routes
  const groups = new Map<string, ServiceRoute[]>();
  for (const r of routes) {
    if (search && !r.name.toLowerCase().includes(search.toLowerCase())) continue;
    if (!groups.has(r.group)) groups.set(r.group, []);
    groups.get(r.group)!.push(r);
  }

  const localCount = routes.filter((r) => r.mode === 'local').length;
  const cloudCount = routes.filter((r) => r.mode === 'cloud').length;

  // Collect env var commands for "How to apply"
  const envVarCommands = routes
    .filter((r) => r.mode === 'cloud' || r.group === 'AWS Services')
    .flatMap((r) => getEnvVarCommands(r, env));
  const uniqueCommands = [...new Set(envVarCommands)];

  if (loadingRoutes) {
    return (
      <div class="routing-view">
        <div class="routing-header">
          <div class="routing-title">
            <h2>Service Routing</h2>
            <p>Loading services...</p>
          </div>
        </div>
      </div>
    );
  }

  if (routes.length === 0) {
    return (
      <div class="routing-view">
        <div class="routing-header">
          <div class="routing-title">
            <h2>Service Routing</h2>
            <p>No services found. Start cloudmock and seed a topology config to populate the routing table.</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div class="routing-view">
      {/* Header */}
      <div class="routing-header">
        <div class="routing-title">
          <h2>
            Service Routing
            {envInfo.source !== 'default' && (
              <span class="routing-env-badge">{envInfo.name}</span>
            )}
          </h2>
          <p>
            Choose which services route locally (cloudmock) or to the cloud environment.
            {proxyAvailable === true && (
              <span class="routing-proxy-status synced"> Proxy connected</span>
            )}
            {proxyAvailable === false && (
              <span class="routing-proxy-status not-synced"> Proxy unavailable - use env vars below</span>
            )}
          </p>
        </div>
      </div>

      {/* Controls bar */}
      <div class="routing-bar">
        <div class="routing-bar-left">
          <input
            class="input routing-search"
            placeholder="Filter services..."
            value={search}
            onInput={(e) => setSearch((e.target as HTMLInputElement).value)}
          />
          <div class="routing-summary">
            <span class="routing-count local">{localCount} local</span>
            <span class="routing-count cloud">{cloudCount} cloud</span>
          </div>
        </div>
        <div class="routing-bar-right">
          <span class="routing-env-label">Environment:</span>
          <div class="routing-env-pills">
            {(['dev', 'staging', 'prod'] as const).map((e) => (
              <button
                key={e}
                class={`routing-env-pill ${env === e ? 'active' : ''} ${e === 'prod' ? 'prod' : ''}`}
                onClick={() => setEnv(e)}
              >
                {e}
              </button>
            ))}
          </div>
          <div class="routing-bulk">
            <button class="btn btn-ghost" onClick={() => setAllMode('local')}>All Local</button>
            <button class="btn btn-ghost" onClick={() => setAllMode('cloud')}>All Cloud</button>
          </div>
        </div>
      </div>

      {/* Service groups */}
      <div class="routing-groups">
        {[...groups].map(([group, services]) => (
          <div key={group} class="routing-group">
            <div class="routing-group-header">
              <span class="routing-group-name">{group}</span>
              <span class="routing-group-count">{services.length} services</span>
              <div class="routing-group-actions">
                <button class="btn btn-ghost routing-group-btn" onClick={() => setGroupMode(group, 'local')}>All Local</button>
                <button class="btn btn-ghost routing-group-btn" onClick={() => setGroupMode(group, 'cloud')}>All Cloud</button>
              </div>
            </div>
            <div class="routing-group-body">
              {services.map((r) => {
                const svc = serviceKey(r.name);
                const status = proxyStatuses.get(svc);
                return (
                  <div key={r.name} class={`routing-row ${r.mode}`}>
                    <div class="routing-row-name">
                      <span class={`routing-health ${r.mode === 'local' ? (r.healthy ? 'up' : 'down') : 'cloud'}`} />
                      <span class="routing-name">{r.name}</span>
                      {status && (
                        <span class={`routing-sync-dot ${status.synced ? 'synced' : 'not-synced'}`}
                          title={status.synced ? 'Proxy synced' : `Not synced: ${status.error || 'unknown'}`}
                        />
                      )}
                    </div>
                    <div class="routing-row-toggle">
                      <button
                        class={`routing-toggle ${r.mode}`}
                        onClick={() => toggleMode(r.name)}
                      >
                        <span class={`routing-toggle-track`}>
                          <span class="routing-toggle-thumb" />
                        </span>
                        <span class="routing-toggle-label">
                          {r.mode === 'local' ? 'Local' : 'Cloud'}
                        </span>
                      </button>
                    </div>
                    <div class="routing-row-endpoint">
                      {r.mode === 'local' ? r.localEndpoint : (r.cloudEndpoints[env] || r.cloudEndpoints.dev || '\u2014')}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        ))}

        {/* How to Apply section */}
        <div class="routing-how-to-apply">
          <button
            class="routing-how-to-apply-toggle"
            onClick={() => setShowHowToApply(!showHowToApply)}
          >
            <span class="routing-how-to-apply-icon">{showHowToApply ? '\u25BC' : '\u25B6'}</span>
            How to apply
          </button>
          {showHowToApply && (
            <div class="routing-how-to-apply-body">
              <p class="routing-how-to-apply-desc">
                {proxyAvailable
                  ? 'Route changes are automatically sent to the cloudmock proxy. You can also set these env vars for services that connect directly:'
                  : 'Set these environment variables before starting your services to route traffic as configured:'}
              </p>
              {uniqueCommands.length > 0 ? (
                <pre class="routing-env-commands">{uniqueCommands.join('\n')}</pre>
              ) : (
                <p class="routing-how-to-apply-hint">All services are routing to local. No env var changes needed.</p>
              )}
              <div class="routing-how-to-apply-note">
                For AWS services, set <code>AWS_ENDPOINT_URL=http://localhost:4566</code> to route to cloudmock, or unset it to use real AWS.
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
