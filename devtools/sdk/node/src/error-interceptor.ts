import type { Connection } from './ws';

/**
 * Captures uncaught exceptions and unhandled promise rejections.
 */
export function interceptErrors(conn: Connection): () => void {
  function onUncaughtException(err: Error) {
    conn.send({
      type: 'error:uncaught',
      data: {
        name: err.name,
        message: err.message,
        stack: err.stack,
      },
    });
  }

  function onUnhandledRejection(reason: any) {
    const err = reason instanceof Error ? reason : new Error(String(reason));
    conn.send({
      type: 'error:unhandled-rejection',
      data: {
        name: err.name,
        message: err.message,
        stack: err.stack,
      },
    });
  }

  process.on('uncaughtException', onUncaughtException);
  process.on('unhandledRejection', onUnhandledRejection);

  return () => {
    process.removeListener('uncaughtException', onUncaughtException);
    process.removeListener('unhandledRejection', onUnhandledRejection);
  };
}
