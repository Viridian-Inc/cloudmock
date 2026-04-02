/**
 * Naming convention:
 * - Interface names: PascalCase (TypeScript standard).
 * - Field names that come from the cloudmock API: snake_case, matching the Go
 *   JSON serialization (e.g. `status_code`, `trace_id`, `latency_ms`).
 * - SLOWindow is a special case: cloudmock's Go SLO endpoint serializes with
 *   PascalCase field names (Service, Action, P50Ms, ...). We keep them as-is
 *   to avoid a mapping layer and match the API 1:1.
 * - Internal-only fields use camelCase where possible.
 */

/** Service health summary from /api/services. */
export interface ServiceInfo {
  /** snake_case — matches cloudmock API */
  name: string;
  healthy: boolean;
  action_count: number;
}

/** Matches cloudmock SSE event `data.data` shape. All fields are snake_case per the Go JSON tags. */
export interface RequestEvent {
  id: string;
  trace_id?: string;
  span_id?: string;
  service: string;
  action: string;
  method: string;
  path: string;
  /** HTTP status code — snake_case from API */
  status_code: number;
  /** Duration in milliseconds — snake_case from API */
  latency_ms: number;
  latency_ns?: number;
  timestamp: string;
  caller_id?: string;
  level?: string;
  mem_alloc_kb?: number;
  goroutines?: number;
  request_headers?: Record<string, string>;
  response_headers?: Record<string, string>;
  request_body?: string;
  response_body?: string;
  source?: string;
}

/** Internal wrapper for SSE connection events. */
export interface SSEEvent {
  type: string;
  data: any;
  timestamp: number;
}

/** Response from /api/health. */
export interface HealthStatus {
  status: string;
  services: Record<string, boolean>;
  dataplane?: string;
}

/** Query parameters for /api/traces. */
export interface RequestFilters {
  service?: string;
  limit?: number;
}

/** Incident from /api/incidents — snake_case from API. */
export interface IncidentInfo {
  id: string;
  title: string;
  severity: string;
  status: string;
  first_seen: string;
  last_seen: string;
  alert_count: number;
  affected_services: string[];
  affected_tenants: string[];
}

/** Aggregate response from /api/slo. */
export interface SLOData {
  healthy: boolean;
  alerts: any[];
  rules: SLORule[];
  windows: SLOWindow[];
}

/** SLO threshold rules — snake_case from API. */
export interface SLORule {
  service: string;
  action: string;
  p50_ms: number;
  p95_ms: number;
  p99_ms: number;
  error_rate: number;
}

/**
 * SLO sliding-window data from /api/slo.
 *
 * NOTE: PascalCase field names are intentional — cloudmock's Go SLO handler
 * serializes these with uppercase first letters (Go's default JSON encoding
 * for exported struct fields). Renaming would require a mapping layer across
 * every consumer (slos/, metrics/, traces/).
 */
export interface SLOWindow {
  /** PascalCase — matches cloudmock Go JSON (exported struct field) */
  Service: string;
  /** PascalCase — matches cloudmock Go JSON */
  Action: string;
  Total: number;
  Errors: number;
  P50Ms: number;
  P95Ms: number;
  P99Ms: number;
  ErrorRate: number;
  Healthy: boolean;
  Violations: string[];
}

/** Chaos engineering state from /api/chaos. */
export interface ChaosState {
  active: boolean;
  rules: ChaosRule[];
}

/** Individual chaos fault injection rule. */
export interface ChaosRule {
  service: string;
  action?: string;
  type: 'latency' | 'error' | 'throttle';
  value: number;
}
