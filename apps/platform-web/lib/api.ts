const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

// --- Types ---

export interface App {
  id: string;
  name: string;
  description: string;
  owner_org_id: string;
  created_at: string;
  updated_at: string;
}

export interface APIKeyResponse {
  id: string;
  app_id: string;
  name: string;
  key: string; // only present on creation
  created_at: string;
}

export interface APIKeyListItem {
  id: string;
  app_id: string;
  name: string;
  prefix: string;
  created_at: string;
  last_used_at: string | null;
}

// --- Internal helper ---

async function apiFetch<T>(
  path: string,
  token: string,
  options: RequestInit = {}
): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
      ...(options.headers ?? {}),
    },
  });

  if (!res.ok) {
    const body = await res.text();
    throw new Error(`API error ${res.status}: ${body}`);
  }

  // 204 No Content
  if (res.status === 204) return undefined as unknown as T;

  return res.json() as Promise<T>;
}

// --- Apps ---

export function listApps(token: string): Promise<App[]> {
  return apiFetch<App[]>("/v1/apps", token);
}

export function getApp(token: string, appId: string): Promise<App> {
  return apiFetch<App>(`/v1/apps/${appId}`, token);
}

export function createApp(
  token: string,
  payload: { name: string; description?: string }
): Promise<App> {
  return apiFetch<App>("/v1/apps", token, {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function deleteApp(token: string, appId: string): Promise<void> {
  return apiFetch<void>(`/v1/apps/${appId}`, token, { method: "DELETE" });
}

// --- API Keys ---

export function listKeys(
  token: string,
  appId: string
): Promise<APIKeyListItem[]> {
  return apiFetch<APIKeyListItem[]>(`/v1/apps/${appId}/keys`, token);
}

export function createKey(
  token: string,
  appId: string,
  payload: { name: string }
): Promise<APIKeyResponse> {
  return apiFetch<APIKeyResponse>(`/v1/apps/${appId}/keys`, token, {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function revokeKey(
  token: string,
  appId: string,
  keyId: string
): Promise<void> {
  return apiFetch<void>(`/v1/apps/${appId}/keys/${keyId}`, token, {
    method: "DELETE",
  });
}
