import type { MetricQuery, AggregationFn } from './types';

const VALID_AGGREGATIONS: AggregationFn[] = [
  'avg', 'sum', 'min', 'max', 'count', 'p50', 'p95', 'p99',
];

/**
 * Parse a DSL query string into a MetricQuery.
 *
 * Format: `aggregation(metric){ key=value, ... } by group`
 *
 * Examples:
 *   avg(http.request.duration){ service=api-gateway }
 *   p99(http.request.duration){ service=api-gateway, method=GET } by service
 *   count(http.request.count)
 */
export function parseQuery(input: string): MetricQuery {
  const trimmed = input.trim();
  if (!trimmed) {
    throw new Error('Query cannot be empty');
  }

  // Match: aggregation(metric){ filters } by groupBy
  const re = /^(\w+)\(([^)]+)\)\s*(?:\{([^}]*)\})?\s*(?:by\s+(\w+))?$/i;
  const match = trimmed.match(re);

  if (!match) {
    throw new Error(
      `Invalid query syntax. Expected: aggregation(metric){ key=value } by group`,
    );
  }

  const [, rawAgg, metric, rawFilters, groupBy] = match;
  const aggregation = rawAgg.toLowerCase() as AggregationFn;

  if (!VALID_AGGREGATIONS.includes(aggregation)) {
    throw new Error(
      `Unknown aggregation "${rawAgg}". Valid: ${VALID_AGGREGATIONS.join(', ')}`,
    );
  }

  const filters: Record<string, string> = {};
  if (rawFilters) {
    const parts = rawFilters.split(',');
    for (const part of parts) {
      const eqIdx = part.indexOf('=');
      if (eqIdx === -1) continue;
      const key = part.slice(0, eqIdx).trim();
      const val = part.slice(eqIdx + 1).trim();
      if (key) filters[key] = val;
    }
  }

  return {
    metric: metric.trim(),
    aggregation,
    filters,
    groupBy: groupBy?.trim(),
  };
}

/**
 * Format a MetricQuery back into the DSL string.
 */
export function formatQuery(query: MetricQuery): string {
  let out = `${query.aggregation}(${query.metric})`;

  const filterEntries = Object.entries(query.filters);
  if (filterEntries.length > 0) {
    const pairs = filterEntries.map(([k, v]) => `${k}=${v}`).join(', ');
    out += `{ ${pairs} }`;
  }

  if (query.groupBy) {
    out += ` by ${query.groupBy}`;
  }

  return out;
}

/**
 * Validate a MetricQuery and return an array of error strings.
 * Returns an empty array if the query is valid.
 */
export function validateQuery(query: MetricQuery): string[] {
  const errors: string[] = [];

  if (!query.metric || !query.metric.trim()) {
    errors.push('Metric name is required');
  }

  if (!VALID_AGGREGATIONS.includes(query.aggregation)) {
    errors.push(
      `Unknown aggregation "${query.aggregation}". Valid: ${VALID_AGGREGATIONS.join(', ')}`,
    );
  }

  // Metric names should be dot-separated identifiers
  if (query.metric && !/^[\w][\w.]*[\w]$/.test(query.metric) && query.metric.length > 1) {
    errors.push('Metric name must be dot-separated identifiers (e.g. http.request.duration)');
  }

  // Filter keys must be non-empty identifiers
  for (const key of Object.keys(query.filters)) {
    if (!/^\w+$/.test(key)) {
      errors.push(`Invalid filter key "${key}"`);
    }
  }

  // groupBy must be a simple identifier
  if (query.groupBy && !/^\w+$/.test(query.groupBy)) {
    errors.push(`Invalid groupBy "${query.groupBy}"`);
  }

  return errors;
}
