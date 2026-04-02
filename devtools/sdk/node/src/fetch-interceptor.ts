import type { Connection } from './ws';

let requestCounter = 0;

/**
 * Intercepts globalThis.fetch (Node 18+) to capture outgoing HTTP traffic.
 * Injects X-CloudMock-Source header and sends http:response events.
 * Returns a teardown function that restores the original fetch.
 */
export function interceptFetch(conn: Connection, correlate: boolean): () => void {
  // Only works on Node 18+ where fetch is a global
  if (typeof globalThis.fetch !== 'function') {
    return () => {};
  }

  const originalFetch = globalThis.fetch;

  globalThis.fetch = async function patchedFetch(
    input: RequestInfo | URL,
    init?: RequestInit,
  ): Promise<Response> {
    const id = `fetch_${++requestCounter}_${Date.now()}`;
    const startTime = Date.now();

    // Build request info
    let url: string;
    let method: string;
    const headers = new Headers(init?.headers);

    if (input instanceof Request) {
      url = input.url;
      method = init?.method || input.method || 'GET';
      // Merge request headers into our headers
      input.headers.forEach((value, key) => {
        if (!headers.has(key)) {
          headers.set(key, value);
        }
      });
    } else {
      url = String(input);
      method = init?.method || 'GET';
    }

    // Inject correlation headers
    if (correlate) {
      headers.set('X-CloudMock-Source', conn.appName);
      headers.set('X-CloudMock-Request-Id', id);
    }

    // Build modified init
    const modifiedInit: RequestInit = {
      ...init,
      headers,
      method,
    };

    let response: Response;
    try {
      response = await originalFetch(input, modifiedInit);
    } catch (err: any) {
      conn.send({
        type: 'http:error',
        data: {
          id,
          method,
          url,
          error: err?.message || String(err),
          duration_ms: Date.now() - startTime,
        },
      });
      throw err;
    }

    const duration = Date.now() - startTime;

    // Clone response so we can read the body without consuming it
    const clonedResponse = response.clone();

    // Read body in background (don't block the caller)
    clonedResponse
      .text()
      .then((body) => {
        const requestHeaders: Record<string, string> = {};
        headers.forEach((value, key) => {
          requestHeaders[key] = value;
        });

        const responseHeaders: Record<string, string> = {};
        response.headers.forEach((value, key) => {
          responseHeaders[key] = value;
        });

        conn.send({
          type: 'http:response',
          data: {
            id,
            method,
            url,
            path: tryParsePath(url),
            status: response.status,
            duration_ms: duration,
            request_headers: requestHeaders,
            response_headers: responseHeaders,
            response_body: body.slice(0, 4096),
            content_length: responseHeaders['content-length'],
          },
        });
      })
      .catch(() => {
        // Body read failed; send event without body
        conn.send({
          type: 'http:response',
          data: {
            id,
            method,
            url,
            path: tryParsePath(url),
            status: response.status,
            duration_ms: duration,
          },
        });
      });

    return response;
  };

  return () => {
    globalThis.fetch = originalFetch;
  };
}

function tryParsePath(url: string): string {
  try {
    return new URL(url).pathname;
  } catch {
    return '/';
  }
}
