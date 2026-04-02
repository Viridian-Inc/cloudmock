import { useState, useEffect, useRef } from 'preact/hooks';
import { api, getAdminBase, cachedApi } from '../lib/api';
import { cacheSet, cacheGet } from '../lib/cache';
import type { ServiceMetrics, DeployEvent } from '../lib/health';
import type { IncidentInfo } from '../lib/types';

/**
 * Fetch JSON from the admin API with 429 rate-limit awareness.
 * Returns `null` if rate-limited (caller should treat as empty/skip).
 */
async function fetchWithRateLimit<T>(path: string): Promise<T | null> {
  const url = `${getAdminBase()}${path}`;
  const res = await fetch(url, {
    headers: { 'Content-Type': 'application/json' },
  });

  if (res.status === 429) {
    const retryAfter = parseInt(res.headers.get('Retry-After') || '30');
    console.warn(`[TopologyMetrics] Rate limited, backing off ${retryAfter}s`);
    await new Promise((r) => setTimeout(r, retryAfter * 1000));
    return null;
  }

  if (!res.ok) {
    const body = await res.text().catch(() => '');
    throw new Error(`API ${res.status}: ${res.statusText} — ${body}`);
  }

  return res.json() as Promise<T>;
}

interface TraceEntry {
  TraceID: string;
  RootService: string;
  Method: string;
  Path: string;
  DurationMs: number;
  StatusCode: number;
  SpanCount: number;
  HasError: boolean;
  StartTime: string;
}

/** Raw deploy shape from cloudmock (PascalCase fields). */
interface RawDeploy {
  ID: string;
  Service: string;
  CommitSHA: string;
  Author: string;
  Description: string;
  DeployedAt: string;
  Metadata?: Record<string, any>;
  // camelCase variants if the API ever normalizes
  id?: string;
  service?: string;
  commit?: string;
  author?: string;
  message?: string;
  timestamp?: string;
  branch?: string;
}

interface TopologyMetricsResult {
  metrics: ServiceMetrics[];
  deploys: DeployEvent[];
  incidents: IncidentInfo[];
  loading: boolean;
}

function percentile(sorted: number[], p: number): number {
  if (sorted.length === 0) return 0;
  const idx = Math.ceil((p / 100) * sorted.length) - 1;
  return sorted[Math.max(0, idx)];
}

/** Compute ServiceMetrics from trace data. */
function computeMetricsFromTraces(traces: TraceEntry[]): ServiceMetrics[] {
  const byService = new Map<string, TraceEntry[]>();
  for (const t of traces) {
    const svc = t.RootService || 'unknown';
    if (!byService.has(svc)) byService.set(svc, []);
    byService.get(svc)!.push(t);
  }

  const results: ServiceMetrics[] = [];
  for (const [svc, svcTraces] of byService) {
    const count = svcTraces.length;
    const errors = svcTraces.filter((t) => t.HasError || t.StatusCode >= 500).length;
    const durations = svcTraces.map((t) => t.DurationMs).sort((a, b) => a - b);
    const avg = durations.reduce((s, d) => s + d, 0) / count;

    results.push({
      service: svc,
      totalCalls: count,
      errorCalls: errors,
      errorRate: count > 0 ? errors / count : 0,
      avgMs: avg,
      p50ms: percentile(durations, 50),
      p95ms: percentile(durations, 95),
      p99ms: percentile(durations, 99),
    });
  }

  return results;
}

/** Normalize cloudmock PascalCase deploy to our DeployEvent type. */
function normalizeDeploy(raw: RawDeploy): DeployEvent {
  return {
    id: raw.ID || raw.id || '',
    service: raw.Service || raw.service || '',
    commit: raw.CommitSHA || raw.commit || '',
    author: raw.Author || raw.author || '',
    message: raw.Description || raw.message || '',
    timestamp: raw.DeployedAt || raw.timestamp || '',
    branch: raw.branch || '',
  };
}

export interface TimeWindow {
  start: number;
  end: number;
}

/** Filter traces to a time window. */
function filterTracesByTime(traces: TraceEntry[], window: TimeWindow): TraceEntry[] {
  return traces.filter((t) => {
    const ts = new Date(t.StartTime).getTime();
    return ts >= window.start && ts <= window.end;
  });
}

/** Filter deploys to a time window. */
function filterDeploysByTime(deploys: DeployEvent[], window: TimeWindow): DeployEvent[] {
  return deploys.filter((d) => {
    const ts = new Date(d.timestamp).getTime();
    return ts >= window.start && ts <= window.end;
  });
}

/** Filter incidents to a time window. */
function filterIncidentsByTime(incidents: IncidentInfo[], window: TimeWindow): IncidentInfo[] {
  return incidents.filter((inc) => {
    const firstTs = new Date(inc.first_seen).getTime();
    const lastTs = new Date(inc.last_seen).getTime();
    // Include if the incident overlaps with the window
    return lastTs >= window.start && firstTs <= window.end;
  });
}

/**
 * Polls cloudmock for live operational data:
 * - Metrics (via traces) every 10s
 * - Deploys + incidents every 30s
 *
 * When `paused` is true (historical mode), polling stops and data is frozen.
 * When `timeWindow` is provided, traces/deploys/incidents are filtered to that range.
 */
export function useTopologyMetrics(
  paused: boolean = false,
  timeWindow?: TimeWindow,
): TopologyMetricsResult {
  const [metrics, setMetrics] = useState<ServiceMetrics[]>([]);
  const [deploys, setDeploys] = useState<DeployEvent[]>([]);
  const [incidents, setIncidents] = useState<IncidentInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const mountedRef = useRef(true);
  // Keep timeWindow in a ref so polling closures always see the latest values
  const timeWindowRef = useRef(timeWindow);
  timeWindowRef.current = timeWindow;
  // Store all raw data so we can re-filter when timeWindow changes
  const rawTracesRef = useRef<TraceEntry[]>([]);
  const rawDeploysRef = useRef<DeployEvent[]>([]);
  const rawIncidentsRef = useRef<IncidentInfo[]>([]);

  // Re-filter data when timeWindow changes (historical scrubbing)
  useEffect(() => {
    if (!timeWindow) return;
    const filteredTraces = filterTracesByTime(rawTracesRef.current, timeWindow);
    const filteredMetrics = computeMetricsFromTraces(filteredTraces);
    const filteredDeploys = filterDeploysByTime(rawDeploysRef.current, timeWindow);
    const filteredIncidents = filterIncidentsByTime(rawIncidentsRef.current, timeWindow);
    setMetrics(filteredMetrics);
    setDeploys(filteredDeploys);
    setIncidents(filteredIncidents);
  }, [timeWindow?.start, timeWindow?.end]);

  useEffect(() => {
    mountedRef.current = true;

    async function fetchAllTraces(): Promise<TraceEntry[]> {
      try {
        const traces = await fetchWithRateLimit<TraceEntry[]>('/api/traces');
        if (traces === null) return []; // rate-limited, skip this cycle
        if (Array.isArray(traces)) return traces;
      } catch (e) { console.warn('[TopologyMetrics] Failed to fetch traces:', e); }
      return [];
    }

    async function fetchMetrics() {
      const tw = timeWindowRef.current;
      // In historical mode, always compute from traces so we can filter
      if (tw) {
        const traces = await fetchAllTraces();
        rawTracesRef.current = traces;
        const filtered = filterTracesByTime(traces, tw);
        const liveMetrics = computeMetricsFromTraces(filtered);
        return liveMetrics;
      }

      // Try /api/metrics first; if empty, fall back to computing from traces
      try {
        const m = await fetchWithRateLimit<ServiceMetrics[]>('/api/metrics');
        if (m === null) return []; // rate-limited, skip this cycle
        if (Array.isArray(m) && m.length > 0) return m;
      } catch (e) { console.debug('[TopologyMetrics] /api/metrics unavailable, falling back to traces:', e); }

      // Compute from traces
      const traces = await fetchAllTraces();
      rawTracesRef.current = traces;
      if (traces.length > 0) {
        return computeMetricsFromTraces(traces);
      }

      return [];
    }

    async function fetchDeploys(): Promise<DeployEvent[]> {
      try {
        const raw = await fetchWithRateLimit<RawDeploy[]>('/api/deploys');
        if (raw === null) return []; // rate-limited, skip this cycle
        if (Array.isArray(raw)) {
          const normalized = raw.map(normalizeDeploy);
          rawDeploysRef.current = normalized;
          const twD = timeWindowRef.current;
          if (twD) return filterDeploysByTime(normalized, twD);
          return normalized;
        }
      } catch (e) { console.warn('[TopologyMetrics] Failed to fetch deploys:', e); }
      return [];
    }

    async function fetchIncidents(): Promise<IncidentInfo[]> {
      try {
        const i = await fetchWithRateLimit<IncidentInfo[]>('/api/incidents');
        if (i === null) return []; // rate-limited, skip this cycle
        if (Array.isArray(i)) {
          rawIncidentsRef.current = i;
          const twI = timeWindowRef.current;
          if (twI) return filterIncidentsByTime(i, twI);
          return i;
        }
      } catch (e) { console.warn('[TopologyMetrics] Failed to fetch incidents:', e); }
      return [];
    }

    // Initial fetch with offline cache fallback
    async function fetchAll() {
      try {
        const [m, d, i] = await Promise.all([
          fetchMetrics(),
          fetchDeploys(),
          fetchIncidents(),
        ]);
        if (mountedRef.current) {
          setMetrics(m);
          setDeploys(d);
          setIncidents(i);
          setLoading(false);
          // Cache results for offline fallback
          cacheSet('topology:metrics', m);
          cacheSet('topology:deploys', d);
          cacheSet('topology:incidents', i);
        }
      } catch (e) {
        console.warn('[TopologyMetrics] Initial fetch failed, trying cache:', e);
        if (mountedRef.current) {
          const cachedM = cacheGet<ServiceMetrics[]>('topology:metrics');
          const cachedD = cacheGet<DeployEvent[]>('topology:deploys');
          const cachedI = cacheGet<IncidentInfo[]>('topology:incidents');
          if (cachedM) setMetrics(cachedM.data);
          if (cachedD) setDeploys(cachedD.data);
          if (cachedI) setIncidents(cachedI.data);
          setLoading(false);
        }
      }
    }

    fetchAll();

    // Only poll if NOT paused (live mode)
    if (paused) {
      return () => { mountedRef.current = false; };
    }

    // Poll metrics every 10s
    const metricsInterval = setInterval(async () => {
      const m = await fetchMetrics();
      if (mountedRef.current && m.length > 0) setMetrics(m);
    }, 10_000);

    // Poll deploys + incidents every 30s
    const slowInterval = setInterval(async () => {
      try {
        const [d, i] = await Promise.all([
          fetchDeploys(),
          fetchIncidents(),
        ]);
        if (mountedRef.current) {
          setDeploys(d);
          setIncidents(i);
        }
      } catch (e) { console.warn('[TopologyMetrics] Slow poll failed:', e); }
    }, 30_000);

    return () => {
      mountedRef.current = false;
      clearInterval(metricsInterval);
      clearInterval(slowInterval);
    };
  }, [paused, timeWindow ? 'historical' : 'live']);

  return { metrics, deploys, incidents, loading };
}
