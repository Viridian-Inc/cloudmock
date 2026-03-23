import { useState, useCallback } from 'preact/hooks';

export interface RequestFilters {
  service?: string;
  path?: string;
  method?: string;
  caller_id?: string;
  action?: string;
  error?: boolean;
  trace_id?: string;
  level?: string;
  limit?: number;
  tenant_id?: string;
  org_id?: string;
  user_id?: string;
  min_latency_ms?: number;
  max_latency_ms?: number;
  from?: string;
  to?: string;
}

export function useFilters(initial: Partial<RequestFilters> = {}) {
  const [filters, setFilters] = useState<RequestFilters>({ level: 'app', limit: 100, ...initial });

  const setFilter = useCallback(<K extends keyof RequestFilters>(key: K, value: RequestFilters[K]) => {
    setFilters(prev => ({ ...prev, [key]: value }));
  }, []);

  const clearFilters = useCallback(() => setFilters({ level: 'app', limit: 100 }), []);

  const hasActiveFilters = Object.entries(filters).some(([k, v]) =>
    v !== undefined && v !== '' && k !== 'level' && k !== 'limit'
  );

  return { filters, setFilter, clearFilters, hasActiveFilters };
}
