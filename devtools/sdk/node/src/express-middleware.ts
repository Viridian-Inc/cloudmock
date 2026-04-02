import type { Connection } from './ws';
import type { IncomingMessage, ServerResponse } from 'node:http';

/**
 * Express/Connect middleware that captures inbound HTTP requests.
 * Logs every request hitting the server to the devtools.
 */
export function createMiddleware(conn: Connection) {
  return function cloudmockMiddleware(
    req: IncomingMessage,
    res: ServerResponse,
    next: (...args: any[]) => void,
  ) {
    const startTime = Date.now();
    const id = `inbound_${startTime}_${Math.random().toString(36).slice(2, 8)}`;

    // Capture response
    const originalEnd = res.end;
    let responseBody = '';

    res.end = function (this: ServerResponse, ...args: any[]) {
      const duration = Date.now() - startTime;

      // Try to capture first chunk of response body
      if (args[0] && typeof args[0] === 'string') {
        responseBody = args[0].slice(0, 4096);
      } else if (args[0] instanceof Buffer) {
        responseBody = args[0].toString('utf8').slice(0, 4096);
      }

      conn.send({
        type: 'http:inbound',
        data: {
          id,
          direction: 'inbound',
          method: req.method || 'GET',
          url: req.url || '/',
          path: req.url || '/',
          status: res.statusCode,
          duration_ms: duration,
          request_headers: req.headers as Record<string, string>,
          response_body: responseBody,
          content_length: res.getHeader('content-length'),
          user_agent: req.headers['user-agent'],
          remote_addr: req.socket?.remoteAddress,
        },
      });

      return originalEnd.apply(this, args as any);
    } as any;

    next();
  };
}
