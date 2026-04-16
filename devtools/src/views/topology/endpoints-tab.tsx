import { useState, useEffect, useMemo, useCallback } from 'preact/hooks';
import { api } from '../../lib/api';
import {
  scanOpenAPISpecsDetailed,
  getSpecEndpointsForService,
  getScanPaths,
  setScanPaths,
  type OpenAPISpec,
  type OpenAPIEndpoint,
  type OpenAPIParameter,
  type OpenAPIResponse,
  type ScanAttempt,
} from './openapi-scanner';

/* ---------- Observed-from-traffic endpoint derivation ---------- */

interface RawRequest {
  Service?: string; service?: string;
  Method?: string; method?: string;
  Path?: string; path?: string;
  Action?: string; action?: string;
  StatusCode?: number; status_code?: number;
}

interface ObservedEndpoint {
  method: string;
  path: string;
  action?: string;
  count: number;
  lastStatus?: number;
}

/** Fuzzy service-name match: strips -handler/-sync/-service suffixes and
 *  checks both directions case-insensitively. */
function serviceMatches(svcKey: string, recordedService: string): boolean {
  const normalize = (s: string) =>
    s.toLowerCase().replace(/[-_](handler|sync|service|lambda|fn)$/i, '');
  const a = normalize(svcKey);
  const b = normalize(recordedService);
  if (a === b) return true;
  if (a.includes(b) || b.includes(a)) return true;
  return false;
}

function deriveObservedEndpoints(requests: RawRequest[], svcKey: string): ObservedEndpoint[] {
  const bucket = new Map<string, ObservedEndpoint>();
  for (const r of requests) {
    const svc = r.Service ?? r.service ?? '';
    if (!serviceMatches(svcKey, svc)) continue;
    const method = (r.Method ?? r.method ?? 'UNKNOWN').toUpperCase();
    let path = r.Path ?? r.path ?? '/';
    // Strip query string for aggregation — keeps GET /x?a=1 and GET /x?a=2 together.
    const q = path.indexOf('?');
    if (q >= 0) path = path.slice(0, q);
    const action = r.Action ?? r.action ?? undefined;
    const key = `${method} ${path} ${action ?? ''}`;
    const existing = bucket.get(key);
    if (existing) {
      existing.count += 1;
      existing.lastStatus = r.StatusCode ?? r.status_code ?? existing.lastStatus;
    } else {
      bucket.set(key, {
        method,
        path,
        action,
        count: 1,
        lastStatus: r.StatusCode ?? r.status_code,
      });
    }
  }
  // Sort by count descending
  return [...bucket.values()].sort((a, b) => b.count - a.count);
}

function ObservedSection({ endpoints }: { endpoints: ObservedEndpoint[] }) {
  return (
    <div>
      <div style={{ fontSize: '10px', color: 'var(--text-tertiary)', marginBottom: '8px' }}>
        {endpoints.length} endpoint{endpoints.length !== 1 ? 's' : ''} · observed from traffic
      </div>
      <div class="endpoint-group">
        <div class="endpoint-group-header">Observed</div>
        {endpoints.map((ep, i) => (
          <div key={i} class="endpoint-row" style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '6px 8px' }}>
            <span
              class="endpoint-method"
              style={{
                background: METHOD_COLORS[ep.method] ?? 'var(--bg-tertiary)',
                color: 'white',
                fontWeight: 600,
                padding: '1px 6px',
                borderRadius: '3px',
                fontSize: '10px',
                minWidth: '52px',
                textAlign: 'center',
              }}
            >
              {ep.method}
            </span>
            <span class="endpoint-path" style={{ flex: 1, fontFamily: 'var(--font-mono)', fontSize: '11px' }}>
              {ep.path}
              {ep.action && (
                <span style={{ color: 'var(--text-tertiary)', marginLeft: '6px' }}>
                  [{ep.action}]
                </span>
              )}
            </span>
            <span style={{ fontSize: '10px', color: 'var(--text-tertiary)', fontFamily: 'var(--font-mono)' }}>
              {ep.count}×
            </span>
            {ep.lastStatus !== undefined && (
              <span
                class={`status-pill ${ep.lastStatus >= 500 ? 'status-5xx' : ep.lastStatus >= 400 ? 'status-4xx' : 'status-2xx'}`}
                style={{ fontSize: '9px' }}
              >
                {ep.lastStatus}
              </span>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

export interface ManifestRoute {
  method: string;
  path: string;
}

export interface ManifestService {
  name: string;
  tables: any[];
  sdkClients: any[];
  routes: ManifestRoute[];
  schemas: any[];
  dependencies?: any[];
}

interface EndpointsTabProps {
  svcKey: string;
  manifest: ManifestService[] | null;
}

interface RouteGroup {
  prefix: string;
  routes: ManifestRoute[];
}

const METHOD_COLORS: Record<string, string> = {
  GET: '#22c55e',
  POST: '#3b82f6',
  PUT: '#fbbf24',
  PATCH: '#f97316',
  DELETE: '#ef4444',
};

/* ------------------------------------------------------------------ */
/*  Infer Required Headers                                             */
/* ------------------------------------------------------------------ */

interface InferredHeader {
  name: string;
  value: string;
}

const AUTH_PATTERNS = ['/auth', '/login', '/signin', '/signup', '/token', '/oauth'];
const MUTATING_METHODS = ['POST', 'PUT', 'PATCH'];

function inferHeaders(method: string, path: string): InferredHeader[] {
  const headers: InferredHeader[] = [];
  const lowerPath = path.toLowerCase();

  const isAuthEndpoint = AUTH_PATTERNS.some((p) => lowerPath.includes(p));

  if (isAuthEndpoint) {
    headers.push({ name: 'Authorization', value: 'Bearer <token>' });
    headers.push({ name: 'Content-Type', value: 'application/json' });
  } else {
    // Protected endpoint: most routes need auth
    headers.push({ name: 'Authorization', value: 'Bearer <token>  OR  x-admin-secret: <secret>' });
    headers.push({ name: 'x-user-id', value: '<userId>' });
  }

  if (MUTATING_METHODS.includes(method.toUpperCase()) && !headers.some((h) => h.name === 'Content-Type')) {
    headers.push({ name: 'Content-Type', value: 'application/json' });
  }

  headers.push({ name: 'Accept', value: 'application/json' });

  return headers;
}

function copyToClipboard(text: string): void {
  if (navigator.clipboard) {
    navigator.clipboard.writeText(text).catch(() => {
      // Fallback: silent fail in non-secure contexts
    });
  }
}

/* ------------------------------------------------------------------ */
/*  Required Headers Section                                           */
/* ------------------------------------------------------------------ */

function RequiredHeadersSection({ method, path }: { method: string; path: string }) {
  const [open, setOpen] = useState(false);
  const [copiedIdx, setCopiedIdx] = useState<number | null>(null);
  const headers = useMemo(() => inferHeaders(method, path), [method, path]);

  const handleCopy = useCallback((header: InferredHeader, idx: number) => {
    copyToClipboard(`${header.name}: ${header.value}`);
    setCopiedIdx(idx);
    setTimeout(() => setCopiedIdx(null), 1200);
  }, []);

  return (
    <div class="endpoint-headers-section">
      <button
        class="endpoint-headers-toggle"
        onClick={(e) => { e.stopPropagation(); setOpen(!open); }}
      >
        <span class="endpoint-headers-caret">{open ? '\u25BC' : '\u25B6'}</span>
        <span class="endpoint-headers-label">
          Required Headers
        </span>
        <span class="endpoint-headers-count">{headers.length}</span>
      </button>
      {open && (
        <div class="endpoint-headers-list">
          {headers.map((h, i) => (
            <div
              key={i}
              class="endpoint-header-row"
              onClick={() => handleCopy(h, i)}
              title="Click to copy"
            >
              <span class="endpoint-header-name">{h.name}:</span>
              <span class="endpoint-header-value">{h.value}</span>
              {copiedIdx === i && (
                <span class="endpoint-header-copied">copied</span>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

/**
 * Extract a grouping prefix from a route path.
 * e.g. /v1/attendance/check-in -> "attendance"
 *      /v1/admin/analytics/health -> "admin/analytics"
 *      /access/health -> "access"
 */
function getRoutePrefix(path: string): string {
  // Strip leading slash and version prefix
  const stripped = path.replace(/^\/+/, '').replace(/^v\d+\//, '');
  const segments = stripped.split('/');

  // For paths like :param/resource, skip param segments for grouping
  const meaningful = segments.filter((s) => !s.startsWith(':'));
  if (meaningful.length <= 1) return meaningful[0] || 'root';

  // Use first segment as group, or first two if second is also a namespace
  // (e.g. "admin/analytics")
  if (meaningful[0] === 'admin' && meaningful.length > 1) {
    return `${meaningful[0]}/${meaningful[1]}`;
  }
  return meaningful[0];
}

function groupRoutes(routes: ManifestRoute[]): RouteGroup[] {
  const groups = new Map<string, ManifestRoute[]>();

  for (const route of routes) {
    const prefix = getRoutePrefix(route.path);
    if (!groups.has(prefix)) groups.set(prefix, []);
    groups.get(prefix)!.push(route);
  }

  // Sort groups alphabetically, sort routes within each group by method then path
  const methodOrder = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'];

  return [...groups.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([prefix, routes]) => ({
      prefix,
      routes: routes.sort((a, b) => {
        const mi = methodOrder.indexOf(a.method) - methodOrder.indexOf(b.method);
        return mi !== 0 ? mi : a.path.localeCompare(b.path);
      }),
    }));
}

function ParametersList({ params }: { params: OpenAPIParameter[] }) {
  if (params.length === 0) return null;
  return (
    <div style={{ marginTop: '4px', paddingLeft: '12px' }}>
      <div style={{ fontSize: '9px', color: 'var(--text-tertiary)', marginBottom: '2px', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
        Parameters
      </div>
      {params.map((p, i) => (
        <div key={i} style={{ fontSize: '10px', color: 'var(--text-secondary)', display: 'flex', gap: '6px', alignItems: 'center', padding: '1px 0' }}>
          <span style={{ color: 'var(--brand-teal)', fontFamily: 'var(--font-mono)' }}>{p.name}</span>
          <span style={{ color: 'var(--text-tertiary)', fontSize: '9px' }}>({p.in})</span>
          {p.required && <span style={{ color: '#ff4e5e', fontSize: '8px' }}>required</span>}
          {p.schema?.type && (
            <span style={{ color: 'var(--text-tertiary)', fontSize: '9px', fontFamily: 'var(--font-mono)' }}>
              {String(p.schema.type)}
            </span>
          )}
        </div>
      ))}
    </div>
  );
}

function ResponsesList({ responses }: { responses: OpenAPIResponse[] }) {
  if (responses.length === 0) return null;
  return (
    <div style={{ marginTop: '4px', paddingLeft: '12px' }}>
      <div style={{ fontSize: '9px', color: 'var(--text-tertiary)', marginBottom: '2px', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
        Responses
      </div>
      {responses.map((r, i) => {
        const code = parseInt(r.statusCode, 10);
        const color = code >= 500 ? '#ef4444' : code >= 400 ? '#fbbf24' : code >= 300 ? '#3b82f6' : '#22c55e';
        return (
          <div key={i} style={{ fontSize: '10px', color: 'var(--text-secondary)', display: 'flex', gap: '6px', alignItems: 'center', padding: '1px 0' }}>
            <span style={{ color, fontFamily: 'var(--font-mono)', fontWeight: 600 }}>{r.statusCode}</span>
            {r.description && <span style={{ color: 'var(--text-tertiary)' }}>{r.description}</span>}
          </div>
        );
      })}
    </div>
  );
}

function OpenAPIEndpointRow({ endpoint }: { endpoint: OpenAPIEndpoint }) {
  const [expanded, setExpanded] = useState(false);
  const hasDetails = endpoint.parameters.length > 0 || endpoint.responses.length > 0 || endpoint.description;

  return (
    <div class="endpoint-route" style={{ flexDirection: 'column', alignItems: 'stretch' }}>
      <div
        style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: hasDetails ? 'pointer' : 'default' }}
        onClick={() => hasDetails && setExpanded(!expanded)}
      >
        {hasDetails && (
          <span style={{ fontSize: '8px', color: 'var(--text-tertiary)', width: '8px' }}>
            {expanded ? '\u25BC' : '\u25B6'}
          </span>
        )}
        <span
          class="method-badge"
          style={{ background: `${METHOD_COLORS[endpoint.method] || '#6b7280'}20`, color: METHOD_COLORS[endpoint.method] || '#6b7280' }}
        >
          {endpoint.method}
        </span>
        <span class="endpoint-path">{endpoint.path}</span>
      </div>
      {expanded && (
        <div style={{ borderLeft: '1px solid var(--border-default)', marginLeft: '4px', marginTop: '4px', paddingLeft: '8px' }}>
          {endpoint.summary && (
            <div style={{ fontSize: '10px', color: 'var(--text-secondary)', marginBottom: '4px' }}>
              {endpoint.summary}
            </div>
          )}
          {endpoint.description && (
            <div style={{ fontSize: '10px', color: 'var(--text-tertiary)', marginBottom: '4px' }}>
              {endpoint.description}
            </div>
          )}
          <ParametersList params={endpoint.parameters} />
          <ResponsesList responses={endpoint.responses} />
          <EndpointPayloadSection method={endpoint.method} path={endpoint.path} schemas={[]} />
          <RequiredHeadersSection method={endpoint.method} path={endpoint.path} />
        </div>
      )}
    </div>
  );
}

function OpenAPISection({
  endpoints,
}: {
  endpoints: OpenAPIEndpoint[];
}) {
  const groups = useMemo(() => {
    const routes: ManifestRoute[] = endpoints.map((e) => ({
      method: e.method,
      path: e.path,
    }));
    return groupRoutes(routes);
  }, [endpoints]);

  // Build a lookup from path+method to OpenAPIEndpoint
  const endpointMap = useMemo(() => {
    const map = new Map<string, OpenAPIEndpoint>();
    for (const ep of endpoints) {
      map.set(`${ep.method}:${ep.path}`, ep);
    }
    return map;
  }, [endpoints]);

  return (
    <div>
      <div style={{ fontSize: '10px', color: 'var(--brand-teal)', marginBottom: '6px', display: 'flex', alignItems: 'center', gap: '4px' }}>
        <span style={{ fontSize: '10px' }}>OpenAPI</span>
        <span style={{ fontSize: '9px', color: 'var(--text-tertiary)' }}>
          {endpoints.length} endpoint{endpoints.length !== 1 ? 's' : ''}
        </span>
      </div>
      {groups.map((group) => (
        <div key={group.prefix} class="endpoint-group">
          <div class="endpoint-group-header">
            /{group.prefix}
            <span class="endpoint-group-count">{group.routes.length}</span>
          </div>
          {group.routes.map((route, i) => {
            const ep = endpointMap.get(`${route.method}:${route.path}`);
            if (ep) {
              return <OpenAPIEndpointRow key={i} endpoint={ep} />;
            }
            return (
              <div key={i} class="endpoint-route">
                <span
                  class="method-badge"
                  style={{ background: `${METHOD_COLORS[route.method] || '#6b7280'}20`, color: METHOD_COLORS[route.method] || '#6b7280' }}
                >
                  {route.method}
                </span>
                <span class="endpoint-path">{route.path}</span>
              </div>
            );
          })}
        </div>
      ))}
    </div>
  );
}

function NoEndpointsHint({
  svcKey,
  reason,
  specs,
  attempts,
  onRescan,
}: {
  svcKey: string;
  reason: 'no-manifest' | 'no-routes';
  specs: OpenAPISpec[];
  attempts: ScanAttempt[];
  onRescan: () => void;
}) {
  const [url, setUrl] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [scanPaths, setScanPathsState] = useState<string[]>(() => getScanPaths());

  const addAndRescan = useCallback(() => {
    const trimmed = url.trim();
    if (!trimmed) return;
    if (!trimmed.startsWith('http') && !trimmed.endsWith('.yaml') && !trimmed.endsWith('.yml') && !trimmed.endsWith('.json')) {
      setError('Enter a full URL (http://...) or a file path ending in .yaml/.json');
      return;
    }
    const existing = getScanPaths();
    if (!existing.includes(trimmed)) {
      const next = [...existing, trimmed];
      setScanPaths(next);
      setScanPathsState(next);
    }
    setUrl('');
    setError(null);
    onRescan();
  }, [url, onRescan]);

  const removePath = useCallback((p: string) => {
    const next = getScanPaths().filter((x) => x !== p);
    setScanPaths(next);
    setScanPathsState(next);
    onRescan();
  }, [onRescan]);

  return (
    <div class="inspector-placeholder" style={{ textAlign: 'left', padding: '14px 16px', fontSize: '12px', lineHeight: 1.5 }}>
      <div style={{ marginBottom: '10px', color: 'var(--text-secondary)' }}>
        {reason === 'no-manifest'
          ? <>No manifest entry for <code>{svcKey}</code>.</>
          : <>No route metadata for <code>{svcKey}</code>.</>}
      </div>

      {/* Diagnostic — shows exactly why nothing matched, so the user can fix it. */}
      <div style={{ fontSize: '10px', color: 'var(--text-tertiary)', marginBottom: '10px', padding: '8px 10px', background: 'var(--bg-elevated)', borderRadius: '4px', fontFamily: 'var(--font-mono)', lineHeight: 1.7 }}>
        <div>Looking up: <span style={{ color: 'var(--text-primary)' }}>{svcKey}</span></div>
        {attempts.length === 0 && scanPaths.length === 0 && (
          <div style={{ color: 'var(--warning)' }}>No scan paths configured. Add one below.</div>
        )}
        {attempts.length > 0 && (
          <div>
            Tried {attempts.length} path{attempts.length === 1 ? '' : 's'}:
            <ul style={{ paddingLeft: '14px', margin: '2px 0' }}>
              {attempts.map((a, i) => (
                <li key={i} style={{ display: 'flex', gap: '8px', alignItems: 'baseline' }}>
                  <span style={{ color: a.ok ? 'var(--success-bright)' : 'var(--error)', flexShrink: 0 }}>
                    {a.ok ? '\u2713' : '\u2717'}
                  </span>
                  <span style={{ flex: 1, wordBreak: 'break-all' }}>
                    <span style={{ color: 'var(--text-primary)' }}>{a.path}</span>
                    {a.ok ? (
                      <span style={{ color: 'var(--text-tertiary)' }}>
                        {' '}— {a.title} ({a.endpointCount} endpoints)
                      </span>
                    ) : (
                      <span style={{ color: 'var(--text-tertiary)' }}>
                        {' '}— {a.error}
                      </span>
                    )}
                  </span>
                  {scanPaths.includes(a.path) && (
                    <button
                      onClick={() => removePath(a.path)}
                      style={{ background: 'none', border: 'none', color: 'var(--text-tertiary)', cursor: 'pointer', padding: 0, fontSize: '10px' }}
                      title="Remove"
                    >
                      {'\u2715'}
                    </button>
                  )}
                </li>
              ))}
            </ul>
          </div>
        )}
        <div>
          Total specs loaded: <span style={{ color: specs.length > 0 ? 'var(--success-bright)' : 'var(--warning)' }}>{specs.length}</span>
        </div>
      </div>

      <div style={{ fontSize: '11px', color: 'var(--text-tertiary)', marginBottom: '6px' }}>
        Add a spec URL:
      </div>
      <div style={{ display: 'flex', gap: '6px', marginBottom: '6px' }}>
        <input
          class="input"
          type="text"
          placeholder="http://localhost:3000/openapi.json"
          value={url}
          onInput={(e) => setUrl((e.target as HTMLInputElement).value)}
          onKeyDown={(e) => { if (e.key === 'Enter') addAndRescan(); }}
          style={{ flex: 1, fontSize: '11px' }}
        />
        <button class="btn btn-primary" style={{ fontSize: '11px', padding: '4px 10px' }} onClick={addAndRescan}>
          Scan
        </button>
      </div>
      {error && (
        <div style={{ fontSize: '10px', color: 'var(--error)', marginBottom: '6px' }}>{error}</div>
      )}
      <div style={{ fontSize: '10px', color: 'var(--text-tertiary)' }}>
        Paths are saved and re-used for every service in the topology.
      </div>
    </div>
  );
}

export function EndpointsTab({ svcKey, manifest }: EndpointsTabProps) {
  const [openApiSpecs, setOpenApiSpecs] = useState<OpenAPISpec[]>([]);
  const [specScanned, setSpecScanned] = useState(false);
  const [attempts, setAttempts] = useState<ScanAttempt[]>([]);
  const [observed, setObserved] = useState<ObservedEndpoint[]>([]);

  // Pull recent traffic and aggregate endpoints matching this service. Works
  // for any service that has ever received a request — no manifest or spec
  // required. This is the universal fallback and runs regardless of whether
  // manifest/spec data exists, so it can also supplement them.
  useEffect(() => {
    let cancelled = false;
    api<RawRequest[]>('/api/requests?level=all&limit=500')
      .then((reqs) => {
        if (cancelled) return;
        const list = Array.isArray(reqs) ? reqs : [];
        setObserved(deriveObservedEndpoints(list, svcKey));
      })
      .catch(() => { if (!cancelled) setObserved([]); });
    return () => { cancelled = true; };
  }, [svcKey]);

  const service = useMemo(() => {
    if (!manifest) return null;

    // Try exact match first
    const exact = manifest.find((s) => s.name === svcKey);
    if (exact) return exact;

    // Case-insensitive
    const ci = manifest.find((s) => s.name.toLowerCase() === svcKey.toLowerCase());
    if (ci) return ci;

    // Lambda function name → manifest service name mapping
    // Strip common prefixes/suffixes to find matching manifest entry
    const stripped = svcKey
      .replace(/-handler$/, '')
      .replace(/-sync$/, '');
    const byStripped = manifest.find((s) => s.name.toLowerCase() === stripped.toLowerCase());
    if (byStripped) return byStripped;

    // Partial match: manifest service name contained in svcKey or vice versa
    const partial = manifest.find((s) =>
      svcKey.toLowerCase().includes(s.name.toLowerCase()) ||
      s.name.toLowerCase().includes(stripped.toLowerCase()),
    );
    if (partial) return partial;

    return null;
  }, [manifest, svcKey]);

  const groups = useMemo(() => {
    if (!service || service.routes.length === 0) return [];
    return groupRoutes(service.routes);
  }, [service]);

  const hasManifestRoutes = groups.length > 0;

  const rescan = useCallback(() => {
    setSpecScanned(false);
    scanOpenAPISpecsDetailed().then(({ specs, attempts }) => {
      setOpenApiSpecs(specs);
      setAttempts(attempts);
      setSpecScanned(true);
    }).catch(() => setSpecScanned(true));
  }, []);

  // Scan for OpenAPI specs when manifest has no routes for this service
  useEffect(() => {
    if (hasManifestRoutes || specScanned) return;
    let cancelled = false;
    scanOpenAPISpecsDetailed().then(({ specs, attempts }) => {
      if (!cancelled) {
        setOpenApiSpecs(specs);
        setAttempts(attempts);
        setSpecScanned(true);
      }
    }).catch(() => {
      if (!cancelled) setSpecScanned(true);
    });
    return () => { cancelled = true; };
  }, [hasManifestRoutes, specScanned]);

  const specEndpoints = useMemo(() => {
    if (hasManifestRoutes || openApiSpecs.length === 0) return [];
    return getSpecEndpointsForService(openApiSpecs, svcKey);
  }, [openApiSpecs, svcKey, hasManifestRoutes]);

  if (!manifest) {
    return (
      <div class="inspector-placeholder">
        Loading service manifest...
      </div>
    );
  }

  // Priority:
  //   1. manifest with routes → authoritative
  //   2. OpenAPI spec matching this service → authoritative
  //   3. observed traffic → show what's actually being called (works for any handler with traffic)
  //   4. empty state + scan input
  if (!service && specEndpoints.length === 0) {
    if (!specScanned) {
      return (
        <div class="inspector-placeholder">
          Scanning for OpenAPI specs...
        </div>
      );
    }
    if (observed.length > 0) return <ObservedSection endpoints={observed} />;
    return <NoEndpointsHint svcKey={svcKey} reason="no-manifest" specs={openApiSpecs} attempts={attempts} onRescan={rescan} />;
  }

  // Show OpenAPI spec endpoints if manifest has no routes
  if (!hasManifestRoutes && specEndpoints.length > 0) {
    return <OpenAPISection endpoints={specEndpoints} />;
  }

  if (!hasManifestRoutes) {
    if (observed.length > 0) return <ObservedSection endpoints={observed} />;
    return <NoEndpointsHint svcKey={svcKey} reason="no-routes" specs={openApiSpecs} attempts={attempts} onRescan={rescan} />;
  }

  return (
    <div>
      <div style={{ fontSize: '10px', color: 'var(--text-tertiary)', marginBottom: '8px' }}>
        {service!.routes.length} endpoint{service!.routes.length !== 1 ? 's' : ''}
      </div>
      {groups.map((group) => (
        <div key={group.prefix} class="endpoint-group">
          <div class="endpoint-group-header">
            /{group.prefix}
            <span class="endpoint-group-count">{group.routes.length}</span>
          </div>
          {group.routes.map((route, i) => (
            <ManifestRouteRow key={i} route={route} schemas={service!.schemas || []} />
          ))}
        </div>
      ))}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Payload / Response Inference                                       */
/* ------------------------------------------------------------------ */

const MUTATING_METHODS_PAYLOAD = ['POST', 'PUT', 'PATCH'];

interface InferredPayload {
  fields: string;
}

interface InferredResponse {
  shape: string;
}

/**
 * Infer expected payload shape from method + path.
 * Returns null for methods that have no body.
 */
function inferPayload(method: string, path: string, schemas: any[]): InferredPayload | null {
  const upper = method.toUpperCase();
  if (!MUTATING_METHODS_PAYLOAD.includes(upper)) return null;

  // Check if schemas array has a matching entry
  if (schemas && schemas.length > 0) {
    const lastSegment = path.split('/').filter((s) => !s.startsWith(':')).pop() || '';
    const match = schemas.find(
      (s: any) => s && (s.name?.toLowerCase() === lastSegment.toLowerCase() || s.route === path),
    );
    if (match?.fields) {
      const fieldStr = typeof match.fields === 'string'
        ? match.fields
        : JSON.stringify(match.fields, null, 0);
      return { fields: fieldStr };
    }
  }

  const lowerPath = path.toLowerCase();

  // Detect route intent from last meaningful path segment
  const isUpdate = upper === 'PUT' || upper === 'PATCH';
  const isDelete = upper === 'DELETE';

  if (isDelete) return null;

  // Common CRUD fields
  if (isUpdate) {
    return { fields: '{ name?: string, description?: string, settings?: object }' };
  }

  // POST create routes
  if (lowerPath.includes('/auth') || lowerPath.includes('/login') || lowerPath.includes('/signin')) {
    return { fields: '{ email: string, password: string }' };
  }
  if (lowerPath.includes('/signup') || lowerPath.includes('/register')) {
    return { fields: '{ email: string, password: string, name?: string }' };
  }

  return { fields: '{ name: string, description?: string, capacity?: number, settings?: object }' };
}

/**
 * Infer expected response shape from method + path.
 */
function inferResponse(method: string, path: string): InferredResponse {
  const upper = method.toUpperCase();
  const lowerPath = path.toLowerCase();

  // Delete routes
  if (upper === 'DELETE') {
    return { shape: '{ success: boolean }' };
  }

  // Detect list vs single
  const segments = path.split('/').filter(Boolean);
  const lastSeg = segments[segments.length - 1] || '';
  const isParamLast = lastSeg.startsWith(':');

  if (upper === 'GET' && !isParamLast) {
    // Plural resource = list endpoint
    return { shape: '{ items: T[], total: number }' };
  }

  if (upper === 'GET' && isParamLast) {
    return { shape: '{ data: T }' };
  }

  // POST/PUT/PATCH
  if (lowerPath.includes('/auth') || lowerPath.includes('/login') || lowerPath.includes('/signin')) {
    return { shape: '{ success: boolean, token: string, user: { ... } }' };
  }

  return { shape: '{ success: boolean, data: T }' };
}

/* ------------------------------------------------------------------ */
/*  Endpoint Payload Section (compact, collapsible)                    */
/* ------------------------------------------------------------------ */

function EndpointPayloadSection({
  method,
  path,
  schemas,
}: {
  method: string;
  path: string;
  schemas: any[];
}) {
  const [open, setOpen] = useState(false);
  const payload = useMemo(() => inferPayload(method, path, schemas), [method, path, schemas]);
  const response = useMemo(() => inferResponse(method, path), [method, path]);

  const hasContent = payload || response;
  if (!hasContent) return null;

  return (
    <div class="endpoint-headers-section">
      <button
        class="endpoint-headers-toggle"
        onClick={(e) => { e.stopPropagation(); setOpen(!open); }}
      >
        <span class="endpoint-headers-caret">{open ? '\u25BC' : '\u25B6'}</span>
        <span class="endpoint-headers-label">Payload / Response</span>
      </button>
      {open && (
        <div class="endpoint-headers-list" style={{ gap: '4px' }}>
          {payload && (
            <div style={{ padding: '2px 0' }}>
              <div style={{
                fontSize: '9px',
                color: 'var(--text-tertiary)',
                textTransform: 'uppercase',
                letterSpacing: '0.5px',
                marginBottom: '2px',
              }}>
                Payload
              </div>
              <div style={{
                fontSize: '10px',
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-secondary)',
                padding: '2px 4px',
                background: 'rgba(255,255,255,0.03)',
                borderRadius: '3px',
                whiteSpace: 'pre',
                overflowX: 'auto',
              }}>
                {payload.fields}
              </div>
            </div>
          )}
          <div style={{ padding: '2px 0' }}>
            <div style={{
              fontSize: '9px',
              color: 'var(--text-tertiary)',
              textTransform: 'uppercase',
              letterSpacing: '0.5px',
              marginBottom: '2px',
            }}>
              Response
            </div>
            <div style={{
              fontSize: '10px',
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-secondary)',
              padding: '2px 4px',
              background: 'rgba(255,255,255,0.03)',
              borderRadius: '3px',
              whiteSpace: 'pre',
              overflowX: 'auto',
            }}>
              {response.shape}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Manifest Route Row (clickable with headers)                        */
/* ------------------------------------------------------------------ */

function ManifestRouteRow({ route, schemas }: { route: ManifestRoute; schemas: any[] }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div class="endpoint-route" style={{ flexDirection: 'column', alignItems: 'stretch' }}>
      <div
        style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer' }}
        onClick={() => setExpanded(!expanded)}
      >
        <span style={{ fontSize: '8px', color: 'var(--text-tertiary)', width: '8px' }}>
          {expanded ? '\u25BC' : '\u25B6'}
        </span>
        <span
          class="method-badge"
          style={{ background: `${METHOD_COLORS[route.method] || '#6b7280'}20`, color: METHOD_COLORS[route.method] || '#6b7280' }}
        >
          {route.method}
        </span>
        <span class="endpoint-path">{route.path}</span>
      </div>
      {expanded && (
        <div style={{ borderLeft: '1px solid var(--border-default)', marginLeft: '4px', marginTop: '4px', paddingLeft: '8px' }}>
          <EndpointPayloadSection method={route.method} path={route.path} schemas={schemas} />
          <RequiredHeadersSection method={route.method} path={route.path} />
        </div>
      )}
    </div>
  );
}
