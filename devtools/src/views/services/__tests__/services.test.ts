import { describe, it, expect } from 'vitest';
import {
  buildAppServices,
  groupByDomain,
  healthDotClass,
  parseAwsDep,
  splitAwsServices,
  nodeIcon,
} from '../helpers';
import type { ServiceInfo } from '../../../lib/types';

describe('buildAppServices', () => {
  it('maps topology nodes to AppService entries', () => {
    const nodes = [
      { id: 'external:bff', label: 'BFF', type: 'server', group: 'API' },
    ];
    const edges = [
      { source: 'external:bff', target: 'dynamodb:users' },
      { source: 'external:bff', target: 'sqs:events' },
    ];
    const result = buildAppServices(nodes, edges);
    expect(result).toHaveLength(1);
    expect(result[0].id).toBe('external:bff');
    expect(result[0].name).toBe('BFF');
    expect(result[0].awsDeps).toEqual(['dynamodb:users', 'sqs:events']);
  });

  it('excludes external: and plugin: targets from AWS deps', () => {
    const nodes = [
      { id: 'lambda:handler', label: 'Handler', type: 'server', group: 'Compute' },
    ];
    const edges = [
      { source: 'lambda:handler', target: 'external:other-api' },
      { source: 'lambda:handler', target: 'plugin:auth' },
      { source: 'lambda:handler', target: 'dynamodb:table1' },
    ];
    const result = buildAppServices(nodes, edges);
    expect(result[0].awsDeps).toEqual(['dynamodb:table1']);
  });

  it('returns empty array for empty input', () => {
    expect(buildAppServices([], [])).toEqual([]);
  });

  it('handles nodes with no outgoing edges', () => {
    const nodes = [
      { id: 'client:app', label: 'App', type: 'client', group: 'Client' },
    ];
    const result = buildAppServices(nodes, []);
    expect(result[0].awsDeps).toEqual([]);
  });
});

describe('groupByDomain', () => {
  const services = [
    { id: '1', name: 'BFF', icon: '', type: 'server', group: 'API', awsDeps: [] },
    { id: '2', name: 'Auth', icon: '', type: 'plugin', group: 'Plugins', awsDeps: [] },
    { id: '3', name: 'Gateway', icon: '', type: 'server', group: 'API', awsDeps: [] },
    { id: '4', name: 'Worker', icon: '', type: 'lambda', group: 'Compute', awsDeps: [] },
  ];

  it('groups services by their group field', () => {
    const groups = groupByDomain(services, '');
    expect(groups.get('API')?.length).toBe(2);
    expect(groups.get('Plugins')?.length).toBe(1);
    expect(groups.get('Compute')?.length).toBe(1);
  });

  it('filters by search query (case insensitive)', () => {
    const groups = groupByDomain(services, 'gate');
    expect(groups.get('API')?.length).toBe(1);
    expect(groups.get('API')![0].name).toBe('Gateway');
    expect(groups.has('Plugins')).toBe(false);
  });

  it('returns empty map when no matches', () => {
    const groups = groupByDomain(services, 'zzz-no-match');
    expect(groups.size).toBe(0);
  });
});

describe('healthDotClass', () => {
  it('returns healthy for healthy service', () => {
    const svc: ServiceInfo = { name: 's3', healthy: true, action_count: 5 };
    expect(healthDotClass(svc)).toBe('healthy');
  });

  it('returns unhealthy for unhealthy service', () => {
    const svc: ServiceInfo = { name: 's3', healthy: false, action_count: 5 };
    expect(healthDotClass(svc)).toBe('unhealthy');
  });

  it('returns healthy when service is undefined', () => {
    expect(healthDotClass(undefined)).toBe('healthy');
  });
});

describe('parseAwsDep', () => {
  const awsServices: ServiceInfo[] = [
    { name: 'dynamodb', healthy: true, action_count: 10 },
    { name: 'sqs', healthy: false, action_count: 3 },
  ];

  it('parses colon-separated dep string', () => {
    const result = parseAwsDep('dynamodb:users', awsServices);
    expect(result.svcName).toBe('dynamodb');
    expect(result.resourceName).toBe('users');
    expect(result.healthy).toBe(true);
  });

  it('returns svcName as resourceName when no colon', () => {
    const result = parseAwsDep('cognito', awsServices);
    expect(result.svcName).toBe('cognito');
    expect(result.resourceName).toBe('cognito');
  });

  it('reports unhealthy for unhealthy service', () => {
    const result = parseAwsDep('sqs:events', awsServices);
    expect(result.healthy).toBe(false);
  });

  it('returns healthy when service not found in list', () => {
    const result = parseAwsDep('lambda:handler', awsServices);
    expect(result.healthy).toBe(true);
  });
});

describe('splitAwsServices', () => {
  const services: ServiceInfo[] = [
    { name: 'dynamodb', healthy: true, action_count: 10 },
    { name: 'sqs', healthy: true, action_count: 3 },
    { name: 'cognito', healthy: true, action_count: 0 },
    { name: 'iam', healthy: true, action_count: 0 },
    { name: 's3', healthy: false, action_count: 7 },
  ];

  it('separates active and stub services', () => {
    const { active, stubs } = splitAwsServices(services, '');
    expect(active.length).toBe(3);
    expect(stubs.length).toBe(2);
  });

  it('sorts active by action_count descending', () => {
    const { active } = splitAwsServices(services, '');
    expect(active[0].name).toBe('dynamodb');
    expect(active[1].name).toBe('s3');
    expect(active[2].name).toBe('sqs');
  });

  it('sorts stubs alphabetically', () => {
    const { stubs } = splitAwsServices(services, '');
    expect(stubs[0].name).toBe('cognito');
    expect(stubs[1].name).toBe('iam');
  });

  it('filters by search query', () => {
    const { active, stubs } = splitAwsServices(services, 'dyn');
    expect(active.length).toBe(1);
    expect(active[0].name).toBe('dynamodb');
    expect(stubs.length).toBe(0);
  });
});

describe('nodeIcon', () => {
  it('returns type icon when type matches', () => {
    expect(nodeIcon('client', 'API')).toBe('\uD83D\uDCF1');
  });

  it('falls back to group icon when type not found', () => {
    expect(nodeIcon('unknown', 'Compute')).toBe('\u2699\uFE0F');
  });

  it('falls back to default gear icon', () => {
    expect(nodeIcon('unknown', 'Unknown')).toBe('\u2699\uFE0F');
  });
});
