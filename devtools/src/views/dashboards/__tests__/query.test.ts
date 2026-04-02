import { describe, it, expect } from 'vitest';
import { parseQuery, formatQuery, validateQuery } from '../query';
import type { MetricQuery } from '../types';

describe('parseQuery', () => {
  it('parses a simple query without filters or groupBy', () => {
    const result = parseQuery('avg(http.request.duration)');
    expect(result).toEqual({
      metric: 'http.request.duration',
      aggregation: 'avg',
      filters: {},
      groupBy: undefined,
    });
  });

  it('parses a query with filters', () => {
    const result = parseQuery('p99(http.request.duration){ service=api-gateway }');
    expect(result).toEqual({
      metric: 'http.request.duration',
      aggregation: 'p99',
      filters: { service: 'api-gateway' },
      groupBy: undefined,
    });
  });

  it('parses a query with multiple filters', () => {
    const result = parseQuery('avg(http.request.duration){ service=api, method=GET }');
    expect(result).toEqual({
      metric: 'http.request.duration',
      aggregation: 'avg',
      filters: { service: 'api', method: 'GET' },
      groupBy: undefined,
    });
  });

  it('parses a query with groupBy', () => {
    const result = parseQuery('count(http.request.count) by service');
    expect(result).toEqual({
      metric: 'http.request.count',
      aggregation: 'count',
      filters: {},
      groupBy: 'service',
    });
  });

  it('parses a query with filters and groupBy', () => {
    const result = parseQuery('sum(http.request.count){ env=prod } by service');
    expect(result).toEqual({
      metric: 'http.request.count',
      aggregation: 'sum',
      filters: { env: 'prod' },
      groupBy: 'service',
    });
  });

  it('is case-insensitive for aggregation', () => {
    const result = parseQuery('AVG(http.request.duration)');
    expect(result.aggregation).toBe('avg');
  });

  it('trims whitespace', () => {
    const result = parseQuery('  avg( http.request.duration )  ');
    expect(result.metric).toBe('http.request.duration');
  });

  it('throws on empty input', () => {
    expect(() => parseQuery('')).toThrow('Query cannot be empty');
  });

  it('throws on invalid syntax', () => {
    expect(() => parseQuery('just some random text')).toThrow('Invalid query syntax');
  });

  it('throws on unknown aggregation', () => {
    expect(() => parseQuery('banana(http.request.duration)')).toThrow('Unknown aggregation');
  });

  it('parses all valid aggregations', () => {
    const aggs = ['avg', 'sum', 'min', 'max', 'count', 'p50', 'p95', 'p99'];
    for (const agg of aggs) {
      const result = parseQuery(`${agg}(test.metric)`);
      expect(result.aggregation).toBe(agg);
    }
  });
});

describe('formatQuery', () => {
  it('formats a simple query', () => {
    const query: MetricQuery = {
      metric: 'http.request.duration',
      aggregation: 'avg',
      filters: {},
    };
    expect(formatQuery(query)).toBe('avg(http.request.duration)');
  });

  it('formats a query with filters', () => {
    const query: MetricQuery = {
      metric: 'http.request.duration',
      aggregation: 'p99',
      filters: { service: 'api-gateway' },
    };
    expect(formatQuery(query)).toBe('p99(http.request.duration){ service=api-gateway }');
  });

  it('formats a query with groupBy', () => {
    const query: MetricQuery = {
      metric: 'http.request.count',
      aggregation: 'count',
      filters: {},
      groupBy: 'service',
    };
    expect(formatQuery(query)).toBe('count(http.request.count) by service');
  });

  it('formats a query with filters and groupBy', () => {
    const query: MetricQuery = {
      metric: 'http.request.count',
      aggregation: 'sum',
      filters: { env: 'prod', region: 'us-east-1' },
      groupBy: 'service',
    };
    expect(formatQuery(query)).toBe(
      'sum(http.request.count){ env=prod, region=us-east-1 } by service',
    );
  });
});

describe('parseQuery/formatQuery roundtrip', () => {
  const cases = [
    'avg(http.request.duration)',
    'p99(http.request.duration){ service=api }',
    'count(http.request.count) by service',
    'sum(http.request.count){ env=prod } by service',
  ];

  for (const input of cases) {
    it(`roundtrips: ${input}`, () => {
      const parsed = parseQuery(input);
      const formatted = formatQuery(parsed);
      const reParsed = parseQuery(formatted);
      expect(reParsed).toEqual(parsed);
    });
  }
});

describe('validateQuery', () => {
  it('returns empty array for a valid query', () => {
    const query: MetricQuery = {
      metric: 'http.request.duration',
      aggregation: 'avg',
      filters: { service: 'api' },
    };
    expect(validateQuery(query)).toEqual([]);
  });

  it('rejects empty metric name', () => {
    const query: MetricQuery = {
      metric: '',
      aggregation: 'avg',
      filters: {},
    };
    const errors = validateQuery(query);
    expect(errors).toContain('Metric name is required');
  });

  it('rejects unknown aggregation', () => {
    const query: MetricQuery = {
      metric: 'http.request.duration',
      aggregation: 'banana' as any,
      filters: {},
    };
    const errors = validateQuery(query);
    expect(errors.some((e) => e.includes('Unknown aggregation'))).toBe(true);
  });

  it('rejects invalid filter keys', () => {
    const query: MetricQuery = {
      metric: 'http.request.duration',
      aggregation: 'avg',
      filters: { 'bad key!': 'value' },
    };
    const errors = validateQuery(query);
    expect(errors.some((e) => e.includes('Invalid filter key'))).toBe(true);
  });

  it('rejects invalid groupBy', () => {
    const query: MetricQuery = {
      metric: 'http.request.duration',
      aggregation: 'avg',
      filters: {},
      groupBy: 'bad group!',
    };
    const errors = validateQuery(query);
    expect(errors.some((e) => e.includes('Invalid groupBy'))).toBe(true);
  });

  it('accepts single-segment metric names', () => {
    const query: MetricQuery = {
      metric: 'cpu',
      aggregation: 'avg',
      filters: {},
    };
    // Single word is acceptable
    expect(validateQuery(query)).toEqual([]);
  });
});
