import { getSessionId } from './session';
import { enqueue } from './beacon';

let installed = false;

export function captureErrors(): void {
  if (installed) return;
  if (typeof window === 'undefined') return;
  installed = true;

  // window.onerror — catches synchronous errors and classic throw statements.
  const originalOnError = window.onerror;
  window.onerror = (
    message: Event | string,
    source?: string,
    lineno?: number,
    colno?: number,
    error?: Error
  ) => {
    enqueue({
      type: 'js_error',
      session_id: getSessionId(),
      url: location.href,
      user_agent: navigator.userAgent,
      timestamp: new Date().toISOString(),
      js_error: {
        message: String(message),
        source: source || '',
        lineno: lineno || 0,
        colno: colno || 0,
        stack: error?.stack || '',
      },
    });

    // Call the original handler if it exists.
    if (typeof originalOnError === 'function') {
      return originalOnError.call(
        window,
        message,
        source,
        lineno,
        colno,
        error
      );
    }
    return false;
  };

  // unhandledrejection — catches unhandled promise rejections.
  window.addEventListener('unhandledrejection', (event: PromiseRejectionEvent) => {
    const reason = event.reason;
    const message =
      reason instanceof Error ? reason.message : String(reason);
    const stack = reason instanceof Error ? reason.stack || '' : '';

    enqueue({
      type: 'js_error',
      session_id: getSessionId(),
      url: location.href,
      user_agent: navigator.userAgent,
      timestamp: new Date().toISOString(),
      js_error: {
        message: `Unhandled Promise Rejection: ${message}`,
        source: '',
        lineno: 0,
        colno: 0,
        stack,
      },
    });
  });
}
