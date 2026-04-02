import type { Connection } from './ws';

/**
 * Intercepts console.log/warn/error/info/debug to forward to devtools.
 * Original console methods continue to work normally.
 */
export function interceptConsole(conn: Connection): () => void {
  const originals = {
    log: console.log,
    warn: console.warn,
    error: console.error,
    info: console.info,
    debug: console.debug,
  };

  function wrap(level: keyof typeof originals) {
    const original = originals[level];
    console[level] = function (...args: any[]) {
      // Call original
      original.apply(console, args);

      // Forward to devtools
      const message = args
        .map((a) => {
          if (typeof a === 'string') return a;
          try { return JSON.stringify(a); } catch { return String(a); }
        })
        .join(' ');

      // Get caller location
      const stack = new Error().stack;
      const callerLine = stack?.split('\n')[2]?.trim() || '';
      const match = callerLine.match(/\((.+):(\d+):(\d+)\)$/) ||
                    callerLine.match(/at (.+):(\d+):(\d+)$/);

      conn.send({
        type: 'console',
        data: {
          level,
          message,
          file: match?.[1] || undefined,
          line: match?.[2] ? parseInt(match[2]) : undefined,
          column: match?.[3] ? parseInt(match[3]) : undefined,
        },
      });
    };
  }

  wrap('log');
  wrap('warn');
  wrap('error');
  wrap('info');
  wrap('debug');

  return () => {
    console.log = originals.log;
    console.warn = originals.warn;
    console.error = originals.error;
    console.info = originals.info;
    console.debug = originals.debug;
  };
}
