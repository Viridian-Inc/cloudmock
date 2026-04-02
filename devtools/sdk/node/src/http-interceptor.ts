import http from 'node:http';
import https from 'node:https';
import type { Connection } from './ws';

let requestCounter = 0;

/**
 * Monkey-patches http.request/get and https.request/get to capture outgoing HTTP traffic.
 * Injects X-CloudMock-Source header for server-side correlation.
 */
export function interceptHttp(conn: Connection, correlate: boolean): () => void {
  const originalHttpRequest = http.request;
  const originalHttpsRequest = https.request;
  const originalHttpGet = http.get;
  const originalHttpsGet = https.get;

  function wrapRequest(
    original: typeof http.request,
    protocol: string,
  ): typeof http.request {
    return function patchedRequest(this: any, ...args: any[]): http.ClientRequest {
      const req: http.ClientRequest = original.apply(this, args as any);
      const id = `req_${++requestCounter}_${Date.now()}`;
      const startTime = Date.now();

      // Extract URL info
      const options = typeof args[0] === 'string' ? new URL(args[0]) : args[0];
      const method = (req as any).method || options?.method || 'GET';
      const host = options?.hostname || options?.host || 'unknown';
      const path = options?.path || options?.pathname || '/';
      const url = `${protocol}://${host}${path}`;

      // Inject correlation header
      if (correlate) {
        try {
          req.setHeader('X-CloudMock-Source', conn.appName);
          req.setHeader('X-CloudMock-Request-Id', id);
        } catch {
          // Headers already sent
        }
      }

      // Capture response
      req.on('response', (res: http.IncomingMessage) => {
        const chunks: Buffer[] = [];
        res.on('data', (chunk: Buffer) => chunks.push(chunk));
        res.on('end', () => {
          const duration = Date.now() - startTime;
          const bodyStr = Buffer.concat(chunks).toString('utf8').slice(0, 4096);

          conn.send({
            type: 'http:response',
            data: {
              id,
              method,
              url,
              path,
              status: res.statusCode,
              duration_ms: duration,
              request_headers: (req as any).getHeaders?.() || {},
              response_headers: res.headers,
              response_body: bodyStr,
              content_length: res.headers['content-length'],
            },
          });
        });
      });

      req.on('error', (err: Error) => {
        conn.send({
          type: 'http:error',
          data: {
            id,
            method,
            url,
            path,
            error: err.message,
            duration_ms: Date.now() - startTime,
          },
        });
      });

      return req;
    } as any;
  }

  http.request = wrapRequest(originalHttpRequest, 'http');
  https.request = wrapRequest(originalHttpsRequest, 'https');

  // Patch http.get and https.get using the original implementations
  // to avoid recursive calls through patched request
  http.get = function patchedHttpGet(this: any, ...args: any[]) {
    const req = (http.request as any).apply(this, args);
    req.end();
    return req;
  } as any;

  https.get = function patchedHttpsGet(this: any, ...args: any[]) {
    const req = (https.request as any).apply(this, args);
    req.end();
    return req;
  } as any;

  return () => {
    http.request = originalHttpRequest;
    https.request = originalHttpsRequest;
    http.get = originalHttpGet;
    https.get = originalHttpsGet;
  };
}
