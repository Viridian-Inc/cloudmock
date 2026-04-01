export interface TransportOptions {
  endpoint: string;
  flushIntervalMs: number;
  maxBatchSize: number;
}

type RUMPayload = Record<string, unknown>;

let queue: RUMPayload[] = [];
let timer: ReturnType<typeof setInterval> | null = null;
let opts: TransportOptions | null = null;

export function initTransport(options: TransportOptions): void {
  opts = options;
  timer = setInterval(flush, opts.flushIntervalMs);

  // Flush on page hide (beforeunload is unreliable in modern browsers).
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'hidden') {
        flush();
      }
    });
  }
  if (typeof window !== 'undefined') {
    window.addEventListener('pagehide', flush);
  }
}

export function enqueue(payload: RUMPayload): void {
  queue.push(payload);
  if (opts && queue.length >= opts.maxBatchSize) {
    flush();
  }
}

export function flush(): void {
  if (!opts || queue.length === 0) return;

  const batch = queue.splice(0);
  const body = JSON.stringify(batch);

  // Prefer sendBeacon (non-blocking, survives page unload).
  if (typeof navigator !== 'undefined' && navigator.sendBeacon) {
    const blob = new Blob([body], { type: 'application/json' });
    const sent = navigator.sendBeacon(opts.endpoint, blob);
    if (sent) return;
  }

  // Fallback to fetch with keepalive.
  if (typeof fetch !== 'undefined') {
    fetch(opts.endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body,
      keepalive: true,
    }).catch(() => {
      // silently drop — this is best-effort telemetry
    });
  }
}

export function destroyTransport(): void {
  flush();
  if (timer !== null) {
    clearInterval(timer);
    timer = null;
  }
  opts = null;
}
