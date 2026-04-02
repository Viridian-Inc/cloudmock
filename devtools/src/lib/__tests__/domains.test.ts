import { describe, it, expect } from 'vitest';
import { getServiceDomain, getUserFacingOverride, getRoutingConfig } from '../domains';
import type { DomainConfig } from '../domains';

const testConfig: DomainConfig = {
  domains: [
    { name: 'Payments', services: ['billing-service', 'payment-gateway'] },
    { name: 'Auth', services: ['cognito', 'iam'] },
    { name: 'Core', services: ['api-gateway'] },
  ],
  userFacingOverrides: {
    'billing-service': true,
    'iam': false,
  },
  routing: {
    'billing-service': { dev: 'http://localhost:3001', prod: 'https://billing.prod.example.com' },
    'api-gateway': { dev: 'http://localhost:4500' },
  },
};

describe('getServiceDomain', () => {
  it('returns domain name for a known service', () => {
    expect(getServiceDomain('billing-service', testConfig)).toBe('Payments');
  });

  it('returns correct domain for another group', () => {
    expect(getServiceDomain('cognito', testConfig)).toBe('Auth');
  });

  it('returns undefined for unknown service', () => {
    expect(getServiceDomain('unknown-service', testConfig)).toBeUndefined();
  });

  it('handles empty domains array', () => {
    expect(getServiceDomain('billing-service', { domains: [] })).toBeUndefined();
  });

  it('returns first matching domain when service listed once', () => {
    expect(getServiceDomain('api-gateway', testConfig)).toBe('Core');
  });
});

describe('getUserFacingOverride', () => {
  it('returns true when overridden to user-facing', () => {
    expect(getUserFacingOverride('billing-service', testConfig)).toBe(true);
  });

  it('returns false when overridden to non-user-facing', () => {
    expect(getUserFacingOverride('iam', testConfig)).toBe(false);
  });

  it('returns undefined when no override exists', () => {
    expect(getUserFacingOverride('cognito', testConfig)).toBeUndefined();
  });

  it('returns undefined when overrides map is absent', () => {
    expect(getUserFacingOverride('billing-service', { domains: [] })).toBeUndefined();
  });

  it('returns undefined for unknown service', () => {
    expect(getUserFacingOverride('nonexistent', testConfig)).toBeUndefined();
  });
});

describe('getRoutingConfig', () => {
  it('returns routing map for a known service', () => {
    const routing = getRoutingConfig('billing-service', testConfig);
    expect(routing).toEqual({
      dev: 'http://localhost:3001',
      prod: 'https://billing.prod.example.com',
    });
  });

  it('returns undefined for service without routing', () => {
    expect(getRoutingConfig('cognito', testConfig)).toBeUndefined();
  });

  it('returns undefined when routing map is absent', () => {
    expect(getRoutingConfig('billing-service', { domains: [] })).toBeUndefined();
  });

  it('returns partial routing config', () => {
    const routing = getRoutingConfig('api-gateway', testConfig);
    expect(routing).toEqual({ dev: 'http://localhost:4500' });
    expect(routing?.prod).toBeUndefined();
  });

  it('returns undefined for unknown service', () => {
    expect(getRoutingConfig('nonexistent', testConfig)).toBeUndefined();
  });
});
