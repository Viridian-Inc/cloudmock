import { useState, useEffect, useRef } from 'preact/hooks';
import { api } from '../../lib/api';
import type { MetricQuery, QueryResult } from './types';

interface UseMetricQueryOptions {
  query: MetricQuery;
  timeWindow: string;
  refreshInterval: number; // seconds, 0 = no auto-refresh
  enabled?: boolean;
}

interface UseMetricQueryResult {
  data: QueryResult | null;
  loading: boolean;
  error: string | null;
  refetch: () => void;
}

/**
 * Hook that POSTs a metric query to /api/metrics/query and optionally
 * polls at the given refresh interval.
 */
export function useMetricQuery({
  query,
  timeWindow,
  refreshInterval,
  enabled = true,
}: UseMetricQueryOptions): UseMetricQueryResult {
  const [data, setData] = useState<QueryResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const fetchData = async () => {
    // Abort any in-flight request
    if (abortRef.current) abortRef.current.abort();
    const controller = new AbortController();
    abortRef.current = controller;

    setLoading(true);
    setError(null);

    try {
      const result = await api<QueryResult>('/api/metrics/query', {
        method: 'POST',
        body: JSON.stringify({
          metric: query.metric,
          aggregation: query.aggregation,
          filters: query.filters,
          group_by: query.groupBy,
          time_window: timeWindow,
        }),
        signal: controller.signal,
      });
      setData(result);
    } catch (err: any) {
      if (err.name === 'AbortError') return;
      setError(err.message || 'Failed to fetch metrics');
      setData(null);
    } finally {
      setLoading(false);
    }
  };

  // Initial fetch + re-fetch when inputs change
  useEffect(() => {
    if (!enabled) return;
    fetchData();

    return () => {
      if (abortRef.current) abortRef.current.abort();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query.metric, query.aggregation, JSON.stringify(query.filters), query.groupBy, timeWindow, enabled]);

  // Polling interval
  useEffect(() => {
    if (timerRef.current) clearInterval(timerRef.current);

    if (refreshInterval > 0 && enabled) {
      timerRef.current = setInterval(fetchData, refreshInterval * 1000);
    }

    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refreshInterval, enabled]);

  return { data, loading, error, refetch: fetchData };
}
