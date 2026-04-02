import { describe, it, expect } from 'vitest';
import {
  serviceKey,
  getEnvVarCommands,
  deriveLocalEndpoint,
  mapGroup,
  mergeWithSaved,
} from '../routing-utils';
import type { ServiceRoute } from '../routing-utils';

function makeRoute(overrides: Partial<ServiceRoute> = {}): ServiceRoute {
  return {
    name: 'Billing Service',
    mode: 'local',
    localEndpoint: 'http://localhost:3001',
    cloudEndpoints: { dev: 'https://billing.dev.example.com', prod: 'https://billing.prod.example.com' },
    healthy: true,
    group: 'API',
    ...overrides,
  };
}

describe('serviceKey', () => {
  it('lowercases and replaces spaces with hyphens', () => {
    expect(serviceKey('Billing Service')).toBe('billing-service');
  });

  it('handles multiple spaces (collapsed to single hyphen)', () => {
    expect(serviceKey('My  Cool  Service')).toBe('my-cool-service');
  });

  it('handles already lowercase with no spaces', () => {
    expect(serviceKey('dynamodb')).toBe('dynamodb');
  });

  it('handles empty string', () => {
    expect(serviceKey('')).toBe('');
  });

  it('handles single word', () => {
    expect(serviceKey('Auth')).toBe('auth');
  });
});

describe('getEnvVarCommands', () => {
  it('generates export for local AWS service', () => {
    const route = makeRoute({ group: 'AWS Services', mode: 'local' });
    const cmds = getEnvVarCommands(route, 'dev');
    expect(cmds).toEqual(['export AWS_ENDPOINT_URL=http://localhost:4566']);
  });

  it('generates unset for cloud AWS service', () => {
    const route = makeRoute({ group: 'AWS Services', mode: 'cloud' });
    const cmds = getEnvVarCommands(route, 'dev');
    expect(cmds).toEqual(['unset AWS_ENDPOINT_URL']);
  });

  it('generates export with service URL for local non-AWS service', () => {
    const route = makeRoute({ mode: 'local' });
    const cmds = getEnvVarCommands(route, 'dev');
    expect(cmds).toEqual(['export BILLING_SERVICE_URL=http://localhost:3001']);
  });

  it('generates export with cloud URL for cloud non-AWS service', () => {
    const route = makeRoute({ mode: 'cloud' });
    const cmds = getEnvVarCommands(route, 'dev');
    expect(cmds).toEqual(['export BILLING_SERVICE_URL=https://billing.dev.example.com']);
  });

  it('uses prod URL when env is prod', () => {
    const route = makeRoute({ mode: 'cloud' });
    const cmds = getEnvVarCommands(route, 'prod');
    expect(cmds).toEqual(['export BILLING_SERVICE_URL=https://billing.prod.example.com']);
  });

  it('falls back to dev URL when requested env is missing', () => {
    const route = makeRoute({
      mode: 'cloud',
      cloudEndpoints: { dev: 'https://fallback.dev.example.com' },
    });
    const cmds = getEnvVarCommands(route, 'staging');
    expect(cmds).toEqual(['export BILLING_SERVICE_URL=https://fallback.dev.example.com']);
  });

  it('returns empty array for cloudmock-only routes', () => {
    const route = makeRoute({ localEndpoint: 'cloudmock', group: 'Other' });
    const cmds = getEnvVarCommands(route, 'dev');
    expect(cmds).toEqual([]);
  });
});

describe('deriveLocalEndpoint', () => {
  it('returns http://localhost:{port} when port is set', () => {
    expect(deriveLocalEndpoint({ id: 'n1', label: 'BFF', type: 'server', group: 'API', port: 4500 }))
      .toBe('http://localhost:4500');
  });

  it('returns "cloudmock" for AWS group', () => {
    expect(deriveLocalEndpoint({ id: 'n2', label: 'DynamoDB', type: 'table', group: 'AWS' }))
      .toBe('cloudmock');
  });

  it('returns "cloudmock" for plugin type', () => {
    expect(deriveLocalEndpoint({ id: 'n3', label: 'Auth', type: 'plugin', group: 'Plugin' }))
      .toBe('cloudmock');
  });

  it('returns "cloudmock (lambda)" for lambda type', () => {
    expect(deriveLocalEndpoint({ id: 'n4', label: 'Handler', type: 'lambda', group: 'Compute' }))
      .toBe('cloudmock (lambda)');
  });

  it('returns "cloudmock" as default', () => {
    expect(deriveLocalEndpoint({ id: 'n5', label: 'Unknown', type: 'other', group: 'Misc' }))
      .toBe('cloudmock');
  });
});

describe('mapGroup', () => {
  it('maps AWS to "AWS Services"', () => {
    expect(mapGroup('AWS')).toBe('AWS Services');
  });

  it('passes through other group names', () => {
    expect(mapGroup('Compute')).toBe('Compute');
  });

  it('returns "Other" for empty string', () => {
    expect(mapGroup('')).toBe('Other');
  });
});

describe('mergeWithSaved', () => {
  it('returns routes with healthy=true when no saved routes', () => {
    const apiRoutes = [
      { name: 'BFF', mode: 'local' as const, localEndpoint: 'http://localhost:4500', cloudEndpoints: {}, group: 'API' },
    ];
    const merged = mergeWithSaved(apiRoutes, null);
    expect(merged[0].healthy).toBe(true);
    expect(merged[0].mode).toBe('local');
  });

  it('preserves saved mode toggles', () => {
    const apiRoutes = [
      { name: 'BFF', mode: 'local' as const, localEndpoint: 'http://localhost:4500', cloudEndpoints: {}, group: 'API' },
    ];
    const saved: ServiceRoute[] = [
      { name: 'BFF', mode: 'cloud', localEndpoint: 'http://localhost:4500', cloudEndpoints: {}, healthy: true, group: 'API' },
    ];
    const merged = mergeWithSaved(apiRoutes, saved);
    expect(merged[0].mode).toBe('cloud');
  });

  it('uses api route mode when saved is empty array', () => {
    const apiRoutes = [
      { name: 'BFF', mode: 'local' as const, localEndpoint: 'http://localhost:4500', cloudEndpoints: {}, group: 'API' },
    ];
    const merged = mergeWithSaved(apiRoutes, []);
    expect(merged[0].mode).toBe('local');
  });

  it('handles new routes not in saved', () => {
    const apiRoutes = [
      { name: 'BFF', mode: 'local' as const, localEndpoint: 'http://localhost:4500', cloudEndpoints: {}, group: 'API' },
      { name: 'Worker', mode: 'local' as const, localEndpoint: 'cloudmock', cloudEndpoints: {}, group: 'Compute' },
    ];
    const saved: ServiceRoute[] = [
      { name: 'BFF', mode: 'cloud', localEndpoint: 'http://localhost:4500', cloudEndpoints: {}, healthy: true, group: 'API' },
    ];
    const merged = mergeWithSaved(apiRoutes, saved);
    expect(merged[0].mode).toBe('cloud'); // BFF preserved from saved
    expect(merged[1].mode).toBe('local'); // Worker uses api default
  });

  it('preserves healthy state from saved', () => {
    const apiRoutes = [
      { name: 'BFF', mode: 'local' as const, localEndpoint: 'http://localhost:4500', cloudEndpoints: {}, group: 'API' },
    ];
    const saved: ServiceRoute[] = [
      { name: 'BFF', mode: 'local', localEndpoint: 'http://localhost:4500', cloudEndpoints: {}, healthy: false, group: 'API' },
    ];
    const merged = mergeWithSaved(apiRoutes, saved);
    expect(merged[0].healthy).toBe(false);
  });
});
