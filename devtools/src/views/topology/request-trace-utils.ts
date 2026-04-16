import type { TopoNode, TopoEdge } from './index';
import type { RequestEvent } from '../../lib/types';

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

/** A grouped request flow: one inbound request with its outbound calls. */
export interface RequestFlow {
  id: string;
  /** Original cloudmock request ID (for replay). May differ from id when id falls back to traceId. */
  requestId: string;
  traceId: string | undefined;
  method: string;
  path: string;
  statusCode: number;
  durationMs: number;
  timestamp: string;
  /** The source service that sent the inbound request. */
  inboundSource: string | undefined;
  inboundHeaders: Record<string, string> | undefined;
  /** Outbound calls made during this request. */
  outbound: OutboundCall[];
  /** Response body summary (item count, etc.). */
  responseSummary: string;
}

export interface OutboundCall {
  service: string;
  action: string;
  statusCode: number;
  durationMs: number;
  detail: string;
}

/* ------------------------------------------------------------------ */
/*  Normalization: PascalCase -> snake_case                            */
/* ------------------------------------------------------------------ */

/**
 * Normalize a raw PascalCase API response into a snake_case RequestEvent.
 */
export function normalizeRequestEvent(r: Record<string, any>): RequestEvent {
  return {
    id: r.ID || r.id || '',
    trace_id: r.TraceID || r.trace_id || '',
    service: r.Service || r.service || '',
    action: r.Action || r.action || '',
    method: r.Method || r.method || '',
    path: r.Path || r.path || '',
    status_code: r.StatusCode || r.status_code || 200,
    latency_ms: r.LatencyMs || r.latency_ms || 0,
    timestamp: r.Timestamp || r.timestamp || '',
    caller_id: r.CallerID || r.caller_id,
    request_headers: r.RequestHeaders || r.request_headers,
    response_body: r.ResponseBody || r.response_body,
    source: r.Level || r.level || 'infra',
  };
}

/* ------------------------------------------------------------------ */
/*  Service Key Derivation                                             */
/* ------------------------------------------------------------------ */

/**
 * Derive the logical service key from a topology node.
 * For IaC nodes (external/plugin), uses the ID suffix after the colon.
 * Otherwise, uses the service property or strips common ID prefixes.
 */
export function getServiceKey(node: TopoNode): string {
  // For IaC nodes (external/plugin), use ID suffix -- "external" matches nothing
  if (node.service === 'external' || node.service === 'plugin') {
    const colonIdx = node.id.indexOf(':');
    if (colonIdx >= 0) return node.id.substring(colonIdx + 1);
    return node.label;
  }
  return node.service || node.id.replace(/^svc:|^ms:/, '');
}

/* ------------------------------------------------------------------ */
/*  Response Summary                                                   */
/* ------------------------------------------------------------------ */

export function buildResponseSummary(body: string | undefined): string {
  if (!body) return '';
  try {
    const parsed = JSON.parse(body);
    if (Array.isArray(parsed)) return `${parsed.length} item${parsed.length !== 1 ? 's' : ''}`;
    if (parsed && typeof parsed === 'object') {
      const keys = Object.keys(parsed);
      // Look for common list patterns
      for (const k of keys) {
        if (Array.isArray(parsed[k])) {
          return `${parsed[k].length} ${k}`;
        }
      }
      return `${keys.length} field${keys.length !== 1 ? 's' : ''}`;
    }
  } catch {
    // not JSON
  }
  return '';
}

/* ------------------------------------------------------------------ */
/*  Request Filtering                                                  */
/* ------------------------------------------------------------------ */

/**
 * Filter raw request events by service connections: requests whose service
 * matches the node's service key, stripped variants, or any connected edge
 * service (inbound/outbound).
 */
export function filterRequestsByEdgeServices(
  allRequests: RequestEvent[],
  serviceName: string,
  node: TopoNode,
  edges: TopoEdge[],
): RequestEvent[] {
  if (!serviceName) return allRequests.slice(0, 50);

  const outboundServices = new Set(
    edges.filter((e) => e.source === node.id).map((e) => {
      const t = e.target;
      if (t.includes(':')) return t.split(':').pop()!;
      return t.replace(/^svc:/, '').replace(/^ms:/, '');
    }),
  );
  const inboundServices = new Set(
    edges.filter((e) => e.target === node.id).map((e) => {
      const s = e.source;
      if (s.includes(':')) return s.split(':').pop()!;
      return s.replace(/^svc:/, '').replace(/^ms:/, '');
    }),
  );

  const svcLower = serviceName.toLowerCase();
  const stripped = svcLower.replace(/-handler$/, '').replace(/-sync$/, '').replace(/-service$/, '');

  return allRequests.filter((r) => {
    const rs = (r.service || '').toLowerCase();
    // Strip common suffixes for fuzzy matching
    const rsStripped = rs.replace(/-handler$/, '').replace(/-sync$/, '').replace(/-service$/, '');
    return rs === svcLower || rs === stripped ||
      rsStripped === stripped || rsStripped === svcLower.replace(/-service$/, '') ||
      svcLower.includes(rs) || rs.includes(stripped) ||
      outboundServices.has(rs) || outboundServices.has(rsStripped) ||
      inboundServices.has(rs) || inboundServices.has(rsStripped);
  });
}

/* ------------------------------------------------------------------ */
/*  Build Request Flows                                                */
/* ------------------------------------------------------------------ */

export function buildFlows(
  requests: RequestEvent[],
  serviceName: string,
  allNodes: TopoNode[],
  edges: TopoEdge[],
): RequestFlow[] {
  // Group by trace_id when available
  const byTrace = new Map<string, RequestEvent[]>();
  const noTrace: RequestEvent[] = [];

  for (const r of requests) {
    if (r.trace_id) {
      if (!byTrace.has(r.trace_id)) byTrace.set(r.trace_id, []);
      byTrace.get(r.trace_id)!.push(r);
    } else {
      noTrace.push(r);
    }
  }

  const flows: RequestFlow[] = [];

  // Build flows from trace groups
  for (const [traceId, group] of byTrace) {
    const sorted = group.sort(
      (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime(),
    );
    const primary = sorted[0];
    const outbound: OutboundCall[] = sorted.slice(1).map((r) => ({
      service: r.service,
      action: r.action || r.method,
      statusCode: r.status_code,
      durationMs: r.latency_ms,
      detail: r.path || r.action || '',
    }));

    // Determine inbound source from edges
    const nodeId = allNodes.find(
      (n) => getServiceKey(n) === serviceName || n.service === serviceName,
    )?.id;
    const inboundEdge = nodeId
      ? edges.find((e) => e.target === nodeId)
      : undefined;
    const inboundNode = inboundEdge
      ? allNodes.find((n) => n.id === inboundEdge.source)
      : undefined;

    flows.push({
      id: primary.id || traceId,
      requestId: primary.id,
      traceId,
      method: primary.method || 'GET',
      path: (primary.path && primary.path !== '/') ? primary.path : (primary.action || primary.path || '/'),
      statusCode: primary.status_code,
      durationMs: primary.latency_ms,
      timestamp: primary.timestamp,
      inboundSource: inboundNode?.label || primary.source || primary.caller_id,
      inboundHeaders: primary.request_headers,
      outbound,
      responseSummary: buildResponseSummary(primary.response_body),
    });
  }

  // Build flows from requests without trace IDs (each becomes its own flow)
  // Include requests from the service itself AND from connected edge services
  const nodeId = allNodes.find(
    (n) => getServiceKey(n) === serviceName || n.service === serviceName,
  )?.id;
  const inboundEdge = nodeId ? edges.find((e) => e.target === nodeId) : undefined;
  const inboundNode = inboundEdge
    ? allNodes.find((n) => n.id === inboundEdge.source)
    : undefined;

  // Compute connected services from edges (same logic as filterRequestsByEdgeServices)
  const connectedServices = new Set<string>();
  for (const e of edges) {
    if (e.source === nodeId) {
      const t = e.target;
      connectedServices.add(t.includes(':') ? t.split(':').pop()! : t);
    }
    if (e.target === nodeId) {
      const s = e.source;
      connectedServices.add(s.includes(':') ? s.split(':').pop()! : s);
    }
  }

  for (const r of noTrace) {
    const rSvc = (r.service || '').toLowerCase();
    const isSelf = rSvc === serviceName.toLowerCase();
    const isConnected = connectedServices.has(rSvc);

    if (isSelf || isConnected) {
      flows.push({
        id: r.id,
        requestId: r.id,
        traceId: undefined,
        method: r.method || 'GET',
        path: (r.path && r.path !== '/') ? r.path : (r.action || r.path || '/'),
        statusCode: r.status_code,
        durationMs: r.latency_ms,
        timestamp: r.timestamp,
        inboundSource: inboundNode?.label || r.source || r.caller_id,
        inboundHeaders: r.request_headers,
        outbound: [],
        responseSummary: buildResponseSummary(r.response_body),
      });
    }
  }

  // Sort by timestamp descending (most recent first)
  flows.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());

  return flows;
}

/* ------------------------------------------------------------------ */
/*  Fallback Flow Creation                                             */
/* ------------------------------------------------------------------ */

/* ------------------------------------------------------------------ */
/*  Merge Flows (accumulate across fetches)                            */
/* ------------------------------------------------------------------ */

/**
 * Merge incoming flows into existing, deduplicating by id.
 * Incoming wins on collision (fresher data). Sorted by timestamp desc.
 * Capped at 200 to prevent unbounded growth.
 */
export function mergeFlows(existing: RequestFlow[], incoming: RequestFlow[]): RequestFlow[] {
  const map = new Map<string, RequestFlow>();

  // Add existing first
  for (const f of existing) {
    map.set(f.id, f);
  }

  // Incoming overwrites existing (fresher data)
  for (const f of incoming) {
    map.set(f.id, f);
  }

  // Sort by timestamp descending (most recent first)
  const merged = Array.from(map.values()).sort(
    (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime(),
  );

  // Cap at 200
  return merged.slice(0, 200);
}

/* ------------------------------------------------------------------ */
/*  Compute Sliding Time Range                                         */
/* ------------------------------------------------------------------ */

/**
 * Compute a sliding time range that always includes "now" as the end.
 * If flows exist and the oldest is within 2x the window, extends start
 * to include it.
 */
export function computeTimeRange(
  flows: RequestFlow[],
  windowMs: number,
): { start: number; end: number } {
  const now = Date.now();

  if (flows.length === 0) {
    return { start: now - windowMs, end: now };
  }

  const timestamps = flows.map((f) => new Date(f.timestamp).getTime());
  const oldest = Math.min(...timestamps);
  const newest = Math.max(...timestamps);

  // End at `now` so live traffic lands inside the range, but fall back to the
  // newest flow timestamp if traffic is stale (otherwise flows captured hours
  // ago would all sit to the left of the window and the list would be empty).
  const end = newest > now - windowMs ? now : newest;

  // Start at `end - window`, but if that would exclude the oldest visible
  // flow, extend back far enough to show everything we fetched. Previously
  // this was capped at 2x window, which silently hid flows older than 10 min.
  let start = end - windowMs;
  if (oldest < start) {
    start = oldest;
  }

  return { start, end };
}

/* ------------------------------------------------------------------ */
/*  Method Filtering                                                   */
/* ------------------------------------------------------------------ */

/**
 * Filter flows by excluding specific HTTP methods (e.g., OPTIONS).
 * Comparison is case-insensitive.
 */
export function filterFlowsByMethod(
  flows: RequestFlow[],
  excludedMethods: Set<string>,
): RequestFlow[] {
  if (excludedMethods.size === 0) return flows;
  const upper = new Set(Array.from(excludedMethods).map((m) => m.toUpperCase()));
  return flows.filter((f) => !upper.has((f.method || '').toUpperCase()));
}

/* ------------------------------------------------------------------ */
/*  Fallback Flow Creation                                             */
/* ------------------------------------------------------------------ */

/**
 * When buildFlows returns empty but raw requests exist, create simple
 * flows directly from the request events.
 */
export function buildFallbackFlows(requests: RequestEvent[]): RequestFlow[] {
  return requests.slice(0, 50).map((r) => ({
    id: r.id || r.trace_id || `${r.timestamp}-${r.service}`,
    requestId: r.id,
    traceId: r.trace_id,
    method: r.method || 'GET',
    path: r.path || '/',
    statusCode: r.status_code || 200,
    durationMs: r.latency_ms || 0,
    timestamp: r.timestamp || '',
    inboundSource: r.caller_id || r.source,
    inboundHeaders: r.request_headers,
    outbound: [],
    responseSummary: '',
  }));
}
