import { Connection } from './ws';
import { interceptHttp } from './http-interceptor';
import { interceptFetch } from './fetch-interceptor';
import { interceptConsole } from './console-interceptor';
import { interceptErrors } from './error-interceptor';
import { createMiddleware } from './express-middleware';

export interface CloudMockOptions {
  /** Devtools host. Default: localhost */
  host?: string;
  /** Devtools TCP port. Default: 4580 */
  port?: number;
  /** App name shown in devtools source bar */
  appName?: string;
  /** Enable HTTP request interception (http/https modules). Default: true */
  http?: boolean;
  /** Enable globalThis.fetch interception (Node 18+). Default: true */
  fetch?: boolean;
  /** Enable console.log interception. Default: true */
  console?: boolean;
  /** Enable uncaught error capture. Default: true */
  errors?: boolean;
  /** Header injected into outgoing HTTP requests for correlation. Default: true */
  correlate?: boolean;
}

let _conn: Connection | null = null;
let _teardowns: (() => void)[] = [];

/**
 * Initialize the CloudMock devtools SDK.
 * Call once at app startup. No-ops gracefully if devtools isn't running.
 * Automatically tree-shaken in production when wrapped in NODE_ENV check.
 *
 * @example
 * ```ts
 * import { init } from '@cloudmock/node';
 * if (process.env.NODE_ENV !== 'production') {
 *   init({ appName: 'my-api' });
 * }
 * ```
 */
export function init(options: CloudMockOptions = {}): void {
  const {
    host = 'localhost',
    port = 4580,
    appName = process.env.npm_package_name || 'node-app',
    http: captureHttp = true,
    fetch: captureFetch = true,
    console: captureConsole = true,
    errors: captureErrors = true,
    correlate = true,
  } = options;

  // Don't double-init
  if (_conn) return;

  _conn = new Connection(host, port, appName);

  // Register source
  _conn.send({
    type: 'source:register',
    data: {
      runtime: 'node',
      appName,
      pid: process.pid,
      nodeVersion: process.version,
    },
  });

  if (captureHttp) {
    _teardowns.push(interceptHttp(_conn, correlate));
  }

  if (captureFetch) {
    _teardowns.push(interceptFetch(_conn, correlate));
  }

  if (captureConsole) {
    _teardowns.push(interceptConsole(_conn));
  }

  if (captureErrors) {
    _teardowns.push(interceptErrors(_conn));
  }
}

/**
 * Get Express/Connect middleware that captures inbound HTTP requests.
 * Add to your Express app: `app.use(getMiddleware())`
 *
 * @example
 * ```ts
 * import { init, getMiddleware } from '@cloudmock/node';
 * init({ appName: 'my-api' });
 * app.use(getMiddleware());
 * ```
 */
export function getMiddleware() {
  if (!_conn) {
    return (_req: any, _res: any, next: any) => next();
  }
  return createMiddleware(_conn);
}

/**
 * Disconnect from devtools and restore all intercepted functions.
 */
export function teardown(): void {
  for (const fn of _teardowns) fn();
  _teardowns = [];
  _conn?.close();
  _conn = null;
}

export type { Connection } from './ws';
