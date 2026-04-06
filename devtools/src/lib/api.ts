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
let _authToken: string | null = null;

export function getAdminBase(): string {
  return _adminBase;
}

export function setAdminBase(url: string): void {
  _adminBase = url;
}

/** Set the Bearer token for authenticated API calls (hosted/SaaS mode). */
export function setAuthToken(token: string | null): void {
  _authToken = token;
}

export async function api<T>(path: string, options?: RequestInit): Promise<T> {
  const url = `${_adminBase}${path}`;
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options?.headers as Record<string, string>),
  };

  // Inject Bearer token for authenticated requests (hosted mode).
  // Local mode has no token — requests pass through without auth.
  if (_authToken) {
    headers['Authorization'] = `Bearer ${_authToken}`;
  }

  const res = await fetch(url, { ...options, headers });

  if (res.status === 401 && _authToken) {
    // Token expired or invalid — clear it and notify
    _authToken = null;
    localStorage.removeItem('cloudmock:auth-token');
    document.dispatchEvent(new CustomEvent('cloudmock:auth-expired'));
    throw new Error('Session expired — please sign in again');
  }

  if (res.status === 429) {
    const retryAfter = parseInt(res.headers.get('Retry-After') || '60', 10);
    const body = await res.json().catch(() => ({})) as any;
    throw new QuotaExceededError(
      body.request_count || 0,
      body.request_limit || 0,
      body.tier || 'free',
      retryAfter,
    );
  }

  if (!res.ok) {
    const body = await res.text().catch(() => '');
    throw new Error(`API ${res.status}: ${res.statusText} — ${body}`);
  }

  return res.json() as Promise<T>;
}

/** Error thrown when a tenant's request quota is exceeded (HTTP 429). */
export class QuotaExceededError extends Error {
  constructor(
    public requestCount: number,
    public requestLimit: number,
    public tier: string,
    public retryAfter: number,
  ) {
    super(`Quota exceeded: ${requestCount}/${requestLimit} requests (${tier} tier). Retry in ${retryAfter}s.`);
    this.name = 'QuotaExceededError';
  }
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
