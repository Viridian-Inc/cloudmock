export interface ServiceRoute {
  name: string;
  mode: 'local' | 'cloud';
  localEndpoint: string;
  cloudEndpoints: Record<string, string>; // env -> url
  healthy: boolean;
  group: string;
}

export const STORAGE_KEY = 'neureaux-devtools:routing';

/** Derive a simple service key from the display name */
export function serviceKey(name: string): string {
  return name.toLowerCase().replace(/\s+/g, '-');
}

/** Generate env var commands for a route */
export function getEnvVarCommands(route: ServiceRoute, env: string): string[] {
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

interface TopologyNode {
  id: string;
  label: string;
  type: string;
  group: string;
  port?: number;
}

/** Derive a local endpoint from a topology node */
export function deriveLocalEndpoint(node: TopologyNode): string {
  if (node.port) return `http://localhost:${node.port}`;
  if (node.group === 'AWS') return 'cloudmock';
  if (node.type === 'plugin') return 'cloudmock';
  if (node.type === 'lambda' || node.type === 'function') return 'cloudmock (lambda)';
  return 'cloudmock';
}

/** Map topology group names to routing display groups */
export function mapGroup(group: string): string {
  if (group === 'AWS') return 'AWS Services';
  return group || 'Other';
}

/** Build routes from API data, merging with any saved user toggles */
export function mergeWithSaved(
  apiRoutes: Omit<ServiceRoute, 'healthy'>[],
  savedRoutes: ServiceRoute[] | null,
): ServiceRoute[] {
  if (!savedRoutes || savedRoutes.length === 0) {
    return apiRoutes.map((r) => ({ ...r, healthy: true }));
  }

  const savedByName = new Map(savedRoutes.map((r) => [r.name, r]));
  return apiRoutes.map((r) => {
    const prev = savedByName.get(r.name);
    return {
      ...r,
      mode: prev?.mode ?? r.mode,
      healthy: prev?.healthy ?? true,
    };
  });
}
