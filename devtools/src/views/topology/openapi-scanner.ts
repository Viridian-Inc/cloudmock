/**
 * OpenAPI Spec Scanner
 *
 * Scans for OpenAPI spec files served by the Vite dev server,
 * parses YAML/JSON specs, and extracts route/schema information
 * that can be mapped to topology service nodes.
 */

// --- localStorage key for configurable scan paths ---
const SCAN_PATHS_KEY = 'neureaux:openapi-scan-paths';

const DEFAULT_SCAN_PATHS = [
  'openapi.yaml',
  'openapi.json',
  'api-spec.yaml',
];

// --- Minimal YAML parser (handles the subset needed for OpenAPI specs) ---

/** Parse a simple YAML string into a JS object. Handles maps, lists, and scalars. */
function parseSimpleYaml(yaml: string): unknown {
  const lines = yaml.split('\n');
  return parseYamlLines(lines, 0, 0).value;
}

interface YamlParseResult {
  value: unknown;
  nextLine: number;
}

function getIndent(line: string): number {
  const match = line.match(/^(\s*)/);
  return match ? match[1].length : 0;
}

function parseYamlScalar(raw: string): unknown {
  const trimmed = raw.trim();
  if (trimmed === '' || trimmed === 'null' || trimmed === '~') return null;
  if (trimmed === 'true') return true;
  if (trimmed === 'false') return false;
  if (/^-?\d+$/.test(trimmed)) return parseInt(trimmed, 10);
  if (/^-?\d+\.\d+$/.test(trimmed)) return parseFloat(trimmed);
  // Strip quotes
  if ((trimmed.startsWith('"') && trimmed.endsWith('"')) ||
      (trimmed.startsWith("'") && trimmed.endsWith("'"))) {
    return trimmed.slice(1, -1);
  }
  return trimmed;
}

function parseYamlLines(
  lines: string[],
  startLine: number,
  expectedIndent: number,
): YamlParseResult {
  if (startLine >= lines.length) {
    return { value: null, nextLine: startLine };
  }

  // Skip blank lines and comments
  let i = startLine;
  while (i < lines.length && (lines[i].trim() === '' || lines[i].trim().startsWith('#'))) {
    i++;
  }
  if (i >= lines.length) return { value: null, nextLine: i };

  const firstLine = lines[i];
  const indent = getIndent(firstLine);
  const content = firstLine.trim();

  // Check if this is a list item
  if (content.startsWith('- ')) {
    const arr: unknown[] = [];
    while (i < lines.length) {
      const line = lines[i];
      if (line.trim() === '' || line.trim().startsWith('#')) { i++; continue; }
      const li = getIndent(line);
      if (li < indent) break;
      if (li > indent) break;
      const lc = line.trim();
      if (!lc.startsWith('- ')) break;
      const afterDash = lc.slice(2);
      if (afterDash.includes(':') && !afterDash.startsWith('{')) {
        // List item is a map
        const subResult = parseYamlMap(lines, i, indent + 2, afterDash);
        arr.push(subResult.value);
        i = subResult.nextLine;
      } else {
        arr.push(parseYamlScalar(afterDash));
        i++;
      }
    }
    return { value: arr, nextLine: i };
  }

  // Check if this is a map
  if (content.includes(':')) {
    return parseYamlMap(lines, i, indent, null);
  }

  // Scalar
  return { value: parseYamlScalar(content), nextLine: i + 1 };
}

function parseYamlMap(
  lines: string[],
  startLine: number,
  baseIndent: number,
  firstKeyLine: string | null,
): YamlParseResult {
  const obj: Record<string, unknown> = {};
  let i = startLine;

  // Handle inline first key (from list item)
  if (firstKeyLine !== null) {
    const colonIdx = firstKeyLine.indexOf(':');
    const key = firstKeyLine.slice(0, colonIdx).trim();
    const valueStr = firstKeyLine.slice(colonIdx + 1).trim();
    if (valueStr) {
      obj[key] = parseYamlScalar(valueStr);
    }
    i++;
  }

  while (i < lines.length) {
    const line = lines[i];
    if (line.trim() === '' || line.trim().startsWith('#')) { i++; continue; }
    const indent = getIndent(line);
    if (indent < baseIndent) break;
    if (indent > baseIndent) { i++; continue; }

    const content = line.trim();
    const colonIdx = content.indexOf(':');
    if (colonIdx === -1) { i++; continue; }

    const key = content.slice(0, colonIdx).trim();
    const valueStr = content.slice(colonIdx + 1).trim();

    if (valueStr) {
      // Inline value
      if (valueStr.startsWith('{') || valueStr.startsWith('[')) {
        // Attempt JSON inline parse
        try {
          obj[key] = JSON.parse(valueStr);
        } catch {
          obj[key] = parseYamlScalar(valueStr);
        }
      } else {
        obj[key] = parseYamlScalar(valueStr);
      }
      i++;
    } else {
      // Nested value on next lines
      i++;
      // Skip blanks
      while (i < lines.length && (lines[i].trim() === '' || lines[i].trim().startsWith('#'))) {
        i++;
      }
      if (i < lines.length) {
        const nextIndent = getIndent(lines[i]);
        if (nextIndent > baseIndent) {
          const sub = parseYamlLines(lines, i, nextIndent);
          obj[key] = sub.value;
          i = sub.nextLine;
        } else {
          obj[key] = null;
        }
      } else {
        obj[key] = null;
      }
    }
  }

  return { value: obj, nextLine: i };
}


// --- OpenAPI types ---

export interface OpenAPIParameter {
  name: string;
  in: string;
  required?: boolean;
  description?: string;
  schema?: Record<string, unknown>;
}

export interface OpenAPIResponse {
  statusCode: string;
  description?: string;
  schema?: Record<string, unknown>;
}

export interface OpenAPIEndpoint {
  path: string;
  method: string;
  summary?: string;
  description?: string;
  operationId?: string;
  parameters: OpenAPIParameter[];
  responses: OpenAPIResponse[];
  tags: string[];
}

export interface OpenAPISpec {
  title: string;
  version: string;
  basePath?: string;
  endpoints: OpenAPIEndpoint[];
  raw: Record<string, unknown>;
}

// --- Scanner ---

export function getScanPaths(): string[] {
  try {
    const stored = localStorage.getItem(SCAN_PATHS_KEY);
    if (stored) {
      const parsed = JSON.parse(stored);
      if (Array.isArray(parsed) && parsed.length > 0) return parsed;
    }
  } catch { /* ignore */ }
  return DEFAULT_SCAN_PATHS;
}

export function setScanPaths(paths: string[]): void {
  localStorage.setItem(SCAN_PATHS_KEY, JSON.stringify(paths));
}

function isYamlPath(path: string): boolean {
  return path.endsWith('.yaml') || path.endsWith('.yml');
}

function parseSpec(content: string, path: string): Record<string, unknown> | null {
  try {
    if (isYamlPath(path)) {
      const parsed = parseSimpleYaml(content);
      if (parsed && typeof parsed === 'object') return parsed as Record<string, unknown>;
    }
    return JSON.parse(content);
  } catch {
    return null;
  }
}

function extractEndpoints(spec: Record<string, unknown>): OpenAPIEndpoint[] {
  const endpoints: OpenAPIEndpoint[] = [];
  const paths = spec.paths as Record<string, unknown> | undefined;
  if (!paths || typeof paths !== 'object') return endpoints;

  for (const [path, methods] of Object.entries(paths)) {
    if (!methods || typeof methods !== 'object') continue;

    for (const [method, operation] of Object.entries(methods as Record<string, unknown>)) {
      if (['get', 'post', 'put', 'patch', 'delete', 'head', 'options'].indexOf(method) === -1) continue;
      if (!operation || typeof operation !== 'object') continue;

      const op = operation as Record<string, unknown>;

      // Extract parameters
      const params: OpenAPIParameter[] = [];
      const rawParams = op.parameters;
      if (Array.isArray(rawParams)) {
        for (const p of rawParams) {
          if (p && typeof p === 'object') {
            const param = p as Record<string, unknown>;
            params.push({
              name: String(param.name ?? ''),
              in: String(param.in ?? ''),
              required: Boolean(param.required),
              description: param.description ? String(param.description) : undefined,
              schema: param.schema as Record<string, unknown> | undefined,
            });
          }
        }
      }

      // Extract responses
      const responses: OpenAPIResponse[] = [];
      const rawResponses = op.responses;
      if (rawResponses && typeof rawResponses === 'object') {
        for (const [statusCode, resp] of Object.entries(rawResponses as Record<string, unknown>)) {
          if (resp && typeof resp === 'object') {
            const r = resp as Record<string, unknown>;
            const schema = (r.content as Record<string, unknown>)?.['application/json'] as Record<string, unknown> | undefined;
            responses.push({
              statusCode,
              description: r.description ? String(r.description) : undefined,
              schema: schema?.schema as Record<string, unknown> | undefined,
            });
          }
        }
      }

      // Extract tags
      const tags: string[] = [];
      if (Array.isArray(op.tags)) {
        for (const t of op.tags) {
          if (typeof t === 'string') tags.push(t);
        }
      }

      endpoints.push({
        path,
        method: method.toUpperCase(),
        summary: op.summary ? String(op.summary) : undefined,
        description: op.description ? String(op.description) : undefined,
        operationId: op.operationId ? String(op.operationId) : undefined,
        parameters: params,
        responses,
        tags,
      });
    }
  }

  return endpoints;
}

/**
 * Scan for OpenAPI spec files from the configured paths.
 * Tries to fetch each path from the Vite dev server.
 */
export interface ScanAttempt {
  path: string;
  ok: boolean;
  error?: string;
  endpointCount?: number;
  title?: string;
}

/** Holds both successful specs and per-path diagnostics for every attempted URL. */
export interface ScanResult {
  specs: OpenAPISpec[];
  attempts: ScanAttempt[];
}

export async function scanOpenAPISpecsDetailed(): Promise<ScanResult> {
  const paths = getScanPaths();
  const specs: OpenAPISpec[] = [];
  const attempts: ScanAttempt[] = [];

  const results = await Promise.allSettled(
    paths.map(async (path) => {
      const url = path.startsWith('http') ? path : `/${path}`;
      let res: Response;
      try {
        res = await fetch(url);
      } catch (e) {
        // TypeError on fetch is usually a CORS or network failure. Record it
        // so the diagnostic can surface the real reason.
        throw new Error(`fetch failed for ${url}: ${(e as Error).message || 'network/CORS error'}`);
      }
      if (!res.ok) throw new Error(`${res.status} ${res.statusText} at ${url}`);
      const text = await res.text();
      // Vite's dev server serves `index.html` for any unmatched path under
      // SPA mode — so `/openapi.yaml` returns HTML, not a 404. Drop anything
      // that clearly isn't a spec before we try to parse.
      const head = text.trimStart().slice(0, 16).toLowerCase();
      if (head.startsWith('<!doctype') || head.startsWith('<html')) {
        throw new Error(`HTML response at ${url} (likely SPA fallback — wrong URL)`);
      }
      const parsed = parseSpec(text, path);
      if (!parsed) throw new Error(`couldn't parse ${url} as YAML or JSON`);
      // Sanity check: a real OpenAPI/Swagger doc advertises itself.
      if (!parsed.openapi && !parsed.swagger && !parsed.paths) {
        throw new Error(`${url} parsed but has no openapi/swagger/paths — not a spec`);
      }

      const info = parsed.info as Record<string, unknown> | undefined;
      const title = info?.title ? String(info.title) : path;
      const version = info?.version ? String(info.version) : 'unknown';

      // Determine basePath from servers or basePath field
      let basePath: string | undefined;
      if (parsed.basePath && typeof parsed.basePath === 'string') {
        basePath = parsed.basePath;
      } else if (Array.isArray(parsed.servers) && parsed.servers.length > 0) {
        const server = parsed.servers[0] as Record<string, unknown>;
        if (server.url && typeof server.url === 'string') {
          try {
            basePath = new URL(server.url).pathname;
          } catch {
            basePath = server.url;
          }
        }
      }

      const endpoints = extractEndpoints(parsed);

      return { title, version, basePath, endpoints, raw: parsed };
    }),
  );

  for (let i = 0; i < results.length; i++) {
    const result = results[i];
    const path = paths[i];
    if (result.status === 'fulfilled' && result.value) {
      specs.push(result.value);
      attempts.push({
        path,
        ok: true,
        endpointCount: result.value.endpoints.length,
        title: result.value.title,
      });
    } else {
      const err = result.status === 'rejected'
        ? (result.reason instanceof Error ? result.reason.message : String(result.reason))
        : 'unknown';
      attempts.push({ path, ok: false, error: err });
    }
  }

  return { specs, attempts };
}

/** Back-compat: return only successful specs. */
export async function scanOpenAPISpecs(): Promise<OpenAPISpec[]> {
  const r = await scanOpenAPISpecsDetailed();
  return r.specs;
}

/**
 * Match an OpenAPI spec to a service name.
 * Matches by: spec title containing the service name, or
 * basePath matching the service name pattern.
 */
export function matchSpecToService(
  spec: OpenAPISpec,
  serviceName: string,
): boolean {
  const lower = serviceName.toLowerCase().replace(/[-_]/g, '');
  const titleLower = spec.title.toLowerCase().replace(/[-_]/g, '');

  // Direct title match
  if (titleLower.includes(lower) || lower.includes(titleLower)) return true;

  // BasePath match
  if (spec.basePath) {
    const pathLower = spec.basePath.toLowerCase().replace(/[-_/]/g, '');
    if (pathLower.includes(lower)) return true;
  }

  return false;
}

/**
 * Get endpoints from OpenAPI specs for a specific service.
 *
 * Match priority:
 *   1. spec whose title / basePath mentions the service name
 *   2. spec whose endpoint tags include the service name
 *   3. fallback: if the user configured any specs at all, return everything
 *      from all of them — they added those paths deliberately, so showing
 *      their full API surface is more useful than "No route metadata."
 */
export function getSpecEndpointsForService(
  specs: OpenAPISpec[],
  serviceName: string,
): OpenAPIEndpoint[] {
  // 1. title / basePath match
  for (const spec of specs) {
    if (matchSpecToService(spec, serviceName)) {
      return spec.endpoints;
    }
  }

  // 2. tag match — OpenAPI ops often tag routes with their owning service/module.
  const svcLower = serviceName.toLowerCase().replace(/[-_]/g, '');
  const tagged: OpenAPIEndpoint[] = [];
  for (const spec of specs) {
    for (const ep of spec.endpoints) {
      const hasTag = ep.tags?.some((t) => {
        const tl = t.toLowerCase().replace(/[-_]/g, '');
        return tl.includes(svcLower) || svcLower.includes(tl);
      });
      if (hasTag) tagged.push(ep);
    }
  }
  if (tagged.length > 0) return tagged;

  // 3. fallback — flatten all specs so the user at least sees what they wired up.
  if (specs.length > 0) {
    return specs.flatMap((s) => s.endpoints);
  }
  return [];
}
