import { describe, it, expect } from 'vitest';
import {
  categorizeServices,
  filterServices,
  AWS_SERVICE_CATEGORIES,
  type AWSServiceInfo,
} from '../service-catalog';

describe('AWS Console: Service Catalog', () => {
  const mockServices: AWSServiceInfo[] = [
    { name: 's3', actions: 41, healthy: true },
    { name: 'dynamodb', actions: 47, healthy: true },
    { name: 'lambda', actions: 36, healthy: true },
    { name: 'iam', actions: 61, healthy: true },
    { name: 'kms', actions: 21, healthy: true },
    { name: 'sqs', actions: 12, healthy: true },
    { name: 'sns', actions: 10, healthy: true },
    { name: 'ecs', actions: 45, healthy: true },
    { name: 'ecr', actions: 15, healthy: true },
    { name: 'rds', actions: 42, healthy: false },
    { name: 'guardduty', actions: 87, healthy: true },
    { name: 'secretsmanager', actions: 10, healthy: true },
    { name: 'cloudwatch', actions: 20, healthy: true },
    { name: 'elasticfilesystem', actions: 31, healthy: true },
    { name: 'unknown-service', actions: 5, healthy: true },
  ];

  describe('categorizeServices', () => {
    it('groups services by AWS category', () => {
      const groups = categorizeServices(mockServices);

      // Should have multiple categories
      expect(Object.keys(groups).length).toBeGreaterThan(3);

      // S3 should be in Storage
      const storage = groups['Storage'];
      expect(storage).toBeDefined();
      expect(storage.some((s) => s.name === 's3')).toBe(true);

      // Lambda should be in Compute
      const compute = groups['Compute'];
      expect(compute).toBeDefined();
      expect(compute.some((s) => s.name === 'lambda')).toBe(true);

      // DynamoDB should be in Database
      const database = groups['Database'];
      expect(database).toBeDefined();
      expect(database.some((s) => s.name === 'dynamodb')).toBe(true);

      // IAM and KMS should be in Security
      const security = groups['Security'];
      expect(security).toBeDefined();
      expect(security.some((s) => s.name === 'iam')).toBe(true);
      expect(security.some((s) => s.name === 'kms')).toBe(true);

      // GuardDuty should also be in Security
      expect(security.some((s) => s.name === 'guardduty')).toBe(true);
    });

    it('puts unknown services in Other', () => {
      const groups = categorizeServices(mockServices);
      const other = groups['Other'];
      expect(other).toBeDefined();
      expect(other.some((s) => s.name === 'unknown-service')).toBe(true);
    });

    it('handles empty input', () => {
      const groups = categorizeServices([]);
      expect(Object.keys(groups).length).toBe(0);
    });
  });

  describe('filterServices', () => {
    it('filters by search query (case-insensitive)', () => {
      const result = filterServices(mockServices, 'dynamo');
      expect(result.length).toBe(1);
      expect(result[0].name).toBe('dynamodb');
    });

    it('returns all services for empty query', () => {
      const result = filterServices(mockServices, '');
      expect(result.length).toBe(mockServices.length);
    });

    it('matches partial names', () => {
      const result = filterServices(mockServices, 'ec');
      // Should match ecs, ecr, secretsmanager
      expect(result.length).toBeGreaterThanOrEqual(2);
      expect(result.some((s) => s.name === 'ecs')).toBe(true);
      expect(result.some((s) => s.name === 'ecr')).toBe(true);
    });

    it('returns empty for no matches', () => {
      const result = filterServices(mockServices, 'zzzznotaservice');
      expect(result.length).toBe(0);
    });
  });

  describe('AWS_SERVICE_CATEGORIES', () => {
    it('has all major AWS categories', () => {
      const categories = Object.keys(AWS_SERVICE_CATEGORIES);
      expect(categories).toContain('Compute');
      expect(categories).toContain('Storage');
      expect(categories).toContain('Database');
      expect(categories).toContain('Security');
      expect(categories).toContain('Networking');
      expect(categories).toContain('Integration');
      expect(categories).toContain('Management');
    });

    it('maps s3 to Storage', () => {
      expect(AWS_SERVICE_CATEGORIES['Storage']).toContain('s3');
    });

    it('maps lambda to Compute', () => {
      expect(AWS_SERVICE_CATEGORIES['Compute']).toContain('lambda');
    });

    it('maps guardduty to Security', () => {
      expect(AWS_SERVICE_CATEGORIES['Security']).toContain('guardduty');
    });
  });
});
