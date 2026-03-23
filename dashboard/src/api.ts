const ADMIN_BASE = window.location.port === '4501'
  ? 'http://localhost:4599'
  : `${window.location.protocol}//${window.location.hostname}:4599`;

const GW_BASE = ADMIN_BASE.replace(/:\d+$/, ':4566');

export async function api<T = any>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(ADMIN_BASE + path, options);
  if (!res.ok) throw new Error(`API error: ${res.status}`);
  return res.json();
}

export async function ddbRequest(action: string, body: Record<string, any> = {}) {
  const res = await fetch(GW_BASE, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-amz-json-1.0',
      'X-Amz-Target': `DynamoDB_20120810.${action}`,
      'Authorization': 'AWS4-HMAC-SHA256 Credential=test/20260321/us-east-1/dynamodb/aws4_request, SignedHeaders=host, Signature=fake',
    },
    body: JSON.stringify(body),
  });
  return res.json();
}

export async function getHomeData() {
  const [services, stats, health] = await Promise.all([
    api('/api/services'),
    api('/api/stats'),
    api('/api/health'),
  ]);
  return { services, stats, health };
}

export function getNodeRequests(service: string, limit = 20) {
  return api<any[]>(`/api/requests?service=${encodeURIComponent(service)}&limit=${limit}`);
}

export function getNodeTraces(service: string, limit = 10) {
  return api<any[]>(`/api/traces?service=${encodeURIComponent(service)}&limit=${limit}`);
}

export function getNodeResources(service: string) {
  return api<any>(`/api/resources/${encodeURIComponent(service)}`);
}

export function getStats() {
  return api<Record<string, number>>('/api/stats');
}

export function getMetrics() {
  return api<any>('/api/metrics');
}

export function getViews() {
  return api<any[]>('/api/views');
}

export function createView(view: any) {
  return api<any>('/api/views', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(view),
  });
}

export function deleteView(id: string) {
  return api('/api/views?id=' + id, { method: 'DELETE' });
}

export function getSLOStatus() {
  return api<any>('/api/slo');
}

export function getBlastRadius(nodeId: string) {
  return api<any>('/api/blast-radius?node=' + encodeURIComponent(nodeId));
}

// --- Incidents ---

export async function fetchIncidents(params?: { status?: string; severity?: string; service?: string; limit?: number }) {
  const query = new URLSearchParams();
  if (params?.status) query.set('status', params.status);
  if (params?.severity) query.set('severity', params.severity);
  if (params?.service) query.set('service', params.service);
  if (params?.limit) query.set('limit', String(params.limit));
  return api<any[]>(`/api/incidents?${query}`);
}

export async function acknowledgeIncident(id: string, owner: string) {
  return api<any>(`/api/incidents/${id}/acknowledge`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ owner }),
  });
}

export async function resolveIncident(id: string) {
  return api(`/api/incidents/${id}/resolve`, { method: 'POST' });
}

export async function fetchIncidentReport(id: string, format = 'html') {
  const res = await fetch(ADMIN_BASE + `/api/incidents/${id}/report?format=${format}`);
  if (!res.ok) throw new Error(`API error: ${res.status}`);
  return format === 'json' ? res.json() : res.text();
}

// --- Regressions ---

export async function fetchRegressions(params?: { service?: string; severity?: string; status?: string; limit?: number }) {
  const query = new URLSearchParams();
  if (params?.service) query.set('service', params.service);
  if (params?.severity) query.set('severity', params.severity);
  if (params?.status) query.set('status', params.status);
  if (params?.limit) query.set('limit', String(params.limit));
  return api<any[]>(`/api/regressions?${query}`);
}

export async function dismissRegression(id: string) {
  return api(`/api/regressions/${id}/dismiss`, { method: 'POST' });
}

// --- Trace comparison ---

export async function compareTraces(a: string, b: string) {
  return api<any>(`/api/traces/compare?a=${a}&b=${b}`);
}

export async function compareBaseline(a: string) {
  return api<any>(`/api/traces/compare?a=${a}&baseline=true`);
}

// --- Cost ---

export async function fetchCostRoutes(limit = 20) {
  return api<any[]>(`/api/cost/routes?limit=${limit}`);
}

export async function fetchCostTenants(limit = 20) {
  return api<any[]>(`/api/cost/tenants?limit=${limit}`);
}

export async function fetchCostTrend(window = '24h', bucket = '1h') {
  return api<any[]>(`/api/cost/trend?window=${window}&bucket=${bucket}`);
}

// --- Profiling ---

export async function captureProfile(service: string, type: string, duration = '5s') {
  const method = type === 'cpu' ? 'POST' : 'GET';
  const res = await fetch(ADMIN_BASE + `/api/profile/${service}?type=${type}&duration=${duration}&format=flamegraph`, { method });
  if (!res.ok) throw new Error(`API error: ${res.status}`);
  return res.text();
}

export async function fetchProfiles(service?: string) {
  const query = service ? `?service=${service}` : '';
  return api<any[]>(`/api/profiles${query}`);
}

export async function fetchProfileFlamegraph(id: string) {
  const res = await fetch(ADMIN_BASE + `/api/profiles/${id}?format=flamegraph`);
  if (!res.ok) throw new Error(`API error: ${res.status}`);
  return res.text();
}

// --- Audit ---

export async function fetchAudit(params?: { actor?: string; action?: string; limit?: number }) {
  const query = new URLSearchParams();
  if (params?.actor) query.set('actor', params.actor);
  if (params?.action) query.set('action', params.action);
  if (params?.limit) query.set('limit', String(params.limit));
  return api<any[]>(`/api/audit?${query}`);
}

// --- Webhooks ---

export async function fetchWebhooks() {
  return api<any[]>('/api/webhooks');
}

export async function createWebhook(config: Record<string, unknown>) {
  return api<any>('/api/webhooks', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  });
}

export async function deleteWebhook(id: string) {
  return api(`/api/webhooks/${id}`, { method: 'DELETE' });
}

export async function testWebhook(id: string) {
  return api(`/api/webhooks/${id}/test`, { method: 'POST' });
}

// --- Users ---

export async function fetchUsers() {
  return api<any[]>('/api/users');
}

export async function updateUserRole(id: string, role: string) {
  return api(`/api/users/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ role }),
  });
}

export { ADMIN_BASE, GW_BASE };
