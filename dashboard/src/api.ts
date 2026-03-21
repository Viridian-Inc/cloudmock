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

export { ADMIN_BASE, GW_BASE };
