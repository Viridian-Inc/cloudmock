import { describe, it, expect } from 'vitest';
import { collapseTopology } from '../collapse';

describe('collapseTopology', () => {
  it('keeps external/plugin nodes as-is', () => {
    const rawNodes = [
      { id: 'external:bff-service', label: 'BFF', service: 'external', type: 'server', group: 'API' },
      { id: 'plugin:auth', label: 'Auth Plugin', service: 'plugin', type: 'plugin', group: 'Plugin' },
    ];
    const { nodes } = collapseTopology(rawNodes, []);
    expect(nodes.find((n) => n.id === 'external:bff-service')).toBeTruthy();
    expect(nodes.find((n) => n.id === 'plugin:auth')).toBeTruthy();
  });

  it('collapses DynamoDB tables into a single node', () => {
    const rawNodes = [
      { id: 'dynamodb:users', label: 'users', service: 'dynamodb', type: 'table', group: 'Storage' },
      { id: 'dynamodb:orders', label: 'orders', service: 'dynamodb', type: 'table', group: 'Storage' },
      { id: 'dynamodb:sessions', label: 'sessions', service: 'dynamodb', type: 'table', group: 'Storage' },
    ];
    const { nodes } = collapseTopology(rawNodes, []);
    const dynamo = nodes.filter((n) => n.service === 'dynamodb');
    expect(dynamo.length).toBe(1);
    expect(dynamo[0].id).toBe('svc:dynamodb');
    expect(dynamo[0].resourceCount).toBe(3);
  });

  it('creates Lambda functions as individual microservice nodes', () => {
    const rawNodes = [
      { id: 'external:bff-service', label: 'BFF', service: 'external', type: 'server', group: 'API' },
    ];
    const rawEdges = [
      { source: 'external:bff-service', target: 'lambda:autotend-attendance-handler' },
      { source: 'external:bff-service', target: 'lambda:autotend-order-handler' },
    ];
    const { nodes } = collapseTopology(rawNodes, rawEdges);
    const attendance = nodes.find((n) => n.label === 'Attendance');
    const billing = nodes.find((n) => n.label === 'Billing');
    expect(attendance).toBeTruthy();
    expect(attendance?.type).toBe('microservice');
    expect(billing).toBeTruthy();
  });

  it('deduplicates edges after collapse', () => {
    const rawNodes = [
      { id: 'external:bff', label: 'BFF', service: 'external', type: 'server', group: 'API' },
      { id: 'dynamodb:users', label: 'users', service: 'dynamodb', type: 'table', group: 'Storage' },
      { id: 'dynamodb:orders', label: 'orders', service: 'dynamodb', type: 'table', group: 'Storage' },
    ];
    const rawEdges = [
      { source: 'external:bff', target: 'dynamodb:users' },
      { source: 'external:bff', target: 'dynamodb:orders' },
    ];
    const { edges } = collapseTopology(rawNodes, rawEdges);
    // Both edges should collapse to BFF -> svc:dynamodb (one deduplicated edge)
    const bffToDynamo = edges.filter(
      (e) => e.source === 'external:bff' && e.target === 'svc:dynamodb',
    );
    expect(bffToDynamo.length).toBe(1);
  });

  it('removes self-loops created by collapse', () => {
    const rawNodes = [
      { id: 'dynamodb:users', label: 'users', service: 'dynamodb', type: 'table', group: 'Storage' },
      { id: 'dynamodb:orders', label: 'orders', service: 'dynamodb', type: 'table', group: 'Storage' },
    ];
    const rawEdges = [
      { source: 'dynamodb:users', target: 'dynamodb:orders' },
    ];
    const { edges } = collapseTopology(rawNodes, rawEdges);
    // After collapse, both map to svc:dynamodb, creating a self-loop which should be removed
    expect(edges.length).toBe(0);
  });

  it('uses friendly names for known Lambda functions', () => {
    const rawNodes = [
      { id: 'external:bff', label: 'BFF', service: 'external', type: 'server', group: 'API' },
    ];
    const rawEdges = [
      { source: 'external:bff', target: 'lambda:autotend-notification-handler' },
    ];
    const { nodes } = collapseTopology(rawNodes, rawEdges);
    expect(nodes.find((n) => n.label === 'Notifications')).toBeTruthy();
  });

  it('uses raw name for unknown Lambda functions', () => {
    const rawNodes = [
      { id: 'external:bff', label: 'BFF', service: 'external', type: 'server', group: 'API' },
    ];
    const rawEdges = [
      { source: 'external:bff', target: 'lambda:custom-handler' },
    ];
    const { nodes } = collapseTopology(rawNodes, rawEdges);
    expect(nodes.find((n) => n.label === 'custom-handler')).toBeTruthy();
  });

  it('handles empty input', () => {
    const { nodes, edges } = collapseTopology([], []);
    expect(nodes).toEqual([]);
    expect(edges).toEqual([]);
  });

  it('collapses S3 buckets to a single service node', () => {
    const rawNodes = [
      { id: 's3:uploads', label: 'uploads', service: 's3', type: 'bucket', group: 'Storage' },
      { id: 's3:logs', label: 'logs', service: 's3', type: 'bucket', group: 'Storage' },
    ];
    const { nodes } = collapseTopology(rawNodes, []);
    const s3Nodes = nodes.filter((n) => n.service === 's3');
    expect(s3Nodes.length).toBe(1);
    expect(s3Nodes[0].id).toBe('svc:s3');
    expect(s3Nodes[0].resourceCount).toBe(2);
  });

  it('capitalizes the label of collapsed AWS service nodes', () => {
    const rawNodes = [
      { id: 'sqs:queue1', label: 'queue1', service: 'sqs', type: 'queue', group: 'Messaging' },
    ];
    const { nodes } = collapseTopology(rawNodes, []);
    expect(nodes.find((n) => n.id === 'svc:sqs')?.label).toBe('Sqs');
  });
});
