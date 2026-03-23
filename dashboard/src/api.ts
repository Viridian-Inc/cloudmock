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

export { ADMIN_BASE, GW_BASE };
