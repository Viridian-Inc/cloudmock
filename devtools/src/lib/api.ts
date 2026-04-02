import type {
  ServiceInfo,
  HealthStatus,
  RequestEvent,
  RequestFilters,
} from './types';
import { cacheSet, cacheGet } from './cache';

// Port-based admin API detection:
// Vite dev server (1420) or old dashboard (4501) → proxy to admin API on :4599
// Production: UI + API on same origin (:4500) → no base needed
function detectAdminBase(): string {
  if (typeof window === 'undefined') return '';
  const port = window.location.port;
  if (port === '1420' || port === '4501') {
    return `${window.location.protocol}//${window.location.hostname}:4599`;
  }
  // Production: UI + API on same origin (:4500)
  return '';
}

let _adminBase = detectAdminBase();

export function getAdminBase(): string {
  return _adminBase;
}

export function setAdminBase(url: string): void {
  _adminBase = url;
}

export async function api<T>(path: string, options?: RequestInit): Promise<T> {
  const url = `${_adminBase}${path}`;
  const res = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (!res.ok) {
    const body = await res.text().catch(() => '');
    throw new Error(`API ${res.status}: ${res.statusText} — ${body}`);
  }

  return res.json() as Promise<T>;
}

export async function cachedApi<T>(path: string, cacheKey: string, ttlMs?: number): Promise<T> {
  try {
    const data = await api<T>(path);
    cacheSet(cacheKey, data, ttlMs);
    return data;
  } catch (e) {
    // Offline fallback: return cached data if available
    const cached = cacheGet<T>(cacheKey);
    if (cached) {
      console.warn(`[API] Using cached data for ${path} (${cached.stale ? 'stale' : 'fresh'})`);
      return cached.data;
    }
    throw e;
  }
}

export function getHealth(): Promise<HealthStatus> {
  return api<HealthStatus>('/api/health');
}

export function getServices(): Promise<ServiceInfo[]> {
  return api<ServiceInfo[]>('/api/services');
}

export function getRequests(filters?: RequestFilters): Promise<RequestEvent[]> {
  const params = new URLSearchParams();
  params.set('level', 'all'); // cloudmock defaults to "app" which hides infra traffic
  if (filters?.service) params.set('service', filters.service);
  if (filters?.limit != null) params.set('limit', String(filters.limit));
  const qs = params.toString();
  return api<RequestEvent[]>(`/api/requests${qs ? `?${qs}` : ''}`);
}

export function getResources(service: string): Promise<{ service: string; resources: any }> {
  return api<{ service: string; resources: any }>(`/api/resources/${encodeURIComponent(service)}`);
}

export function getConfig(): Promise<any> {
  return api<any>('/api/config');
}

export function getTraces(): Promise<any[]> {
  return api<any[]>('/api/traces');
}

export function resetService(name?: string): Promise<void> {
  const path = name ? `/api/reset?service=${encodeURIComponent(name)}` : '/api/reset';
  return api<void>(path, { method: 'POST' });
}
