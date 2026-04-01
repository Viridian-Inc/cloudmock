import { getSessionId, shouldSample } from './session';
import { initTransport, enqueue, flush, destroyTransport } from './beacon';
import { observeWebVitals, collectPageLoad, disconnectObservers } from './vitals';
import { captureErrors } from './errors';

export interface CloudMockRUMOptions {
  /** Base URL of the CloudMock admin API (e.g. "http://localhost:4599"). */
  endpoint: string;

  /** Sample rate: 0.0 to 1.0. Default 1.0 (all sessions). */
  sampleRate?: number;

  /** Flush interval in milliseconds. Default 5000 (5 seconds). */
  flushIntervalMs?: number;

  /** Max events per batch before auto-flush. Default 20. */
  maxBatchSize?: number;

  /** Disable automatic web vitals collection. Default false. */
  disableVitals?: boolean;

  /** Disable automatic JS error capture. Default false. */
  disableErrors?: boolean;

  /** Disable automatic page load timing. Default false. */
  disablePageLoad?: boolean;

  /** Disable XHR/fetch resource timing collection. Default false. */
  disableResourceTiming?: boolean;

  /** Disable click tracking (including rage click detection). Default false. */
  disableClicks?: boolean;
}

let initialized = false;
let resourceObserver: PerformanceObserver | null = null;
let clickHandler: ((e: MouseEvent) => void) | null = null;

/**
 * Initialize CloudMock Real User Monitoring.
 * Call this once, as early as possible in your application.
 */
export function init(options: CloudMockRUMOptions): void {
  if (initialized) return;
  if (typeof window === 'undefined') return; // skip SSR

  const sampleRate = options.sampleRate ?? 1.0;

  // Decide sampling upfront — if not sampled, install nothing.
  if (!shouldSample(sampleRate)) return;

  initialized = true;

  const endpoint = options.endpoint.replace(/\/$/, '') + '/api/rum/events';

  initTransport({
    endpoint,
    flushIntervalMs: options.flushIntervalMs ?? 5000,
    maxBatchSize: options.maxBatchSize ?? 20,
  });

  if (!options.disableVitals) {
    observeWebVitals();
  }

  if (!options.disableErrors) {
    captureErrors();
  }

  if (!options.disablePageLoad) {
    collectPageLoad();
  }

  if (!options.disableResourceTiming) {
    observeResourceTiming();
  }

  if (!options.disableClicks) {
    observeClicks();
  }
}

/**
 * Manually track a custom event.
 */
export function track(
  type: string,
  data: Record<string, unknown>
): void {
  if (!initialized) return;
  enqueue({
    type,
    session_id: getSessionId(),
    url: typeof location !== 'undefined' ? location.href : '',
    user_agent: typeof navigator !== 'undefined' ? navigator.userAgent : '',
    timestamp: new Date().toISOString(),
    ...data,
  });
}

/**
 * Force-flush all queued events.
 */
export { flush };

/**
 * Get the current session ID (useful for correlating with backend logs).
 */
export { getSessionId };

/**
 * Tear down the RUM SDK — stops observers, flushes remaining events,
 * and clears timers. Useful for SPA cleanup.
 */
export function destroy(): void {
  if (!initialized) return;
  initialized = false;
  disconnectObservers();
  resourceObserver?.disconnect();
  resourceObserver = null;
  if (clickHandler) {
    document.removeEventListener('click', clickHandler, true);
    clickHandler = null;
  }
  destroyTransport();
}

// --- Click tracking with rage click detection ---

/** Map from CSS selector → array of click timestamps. */
const clickTimestamps = new Map<string, number[]>();

/** Rage click threshold: 3+ clicks on the same selector within 1 second. */
const RAGE_CLICK_THRESHOLD = 3;
const RAGE_CLICK_WINDOW_MS = 1000;

function getSelector(el: Element): string {
  if (el.id) return `#${el.id}`;
  const tag = el.tagName.toLowerCase();
  const classes = Array.from(el.classList).join('.');
  const parent = el.parentElement;
  if (classes) {
    const sel = `${tag}.${classes}`;
    // Check uniqueness within parent
    if (parent && parent.querySelectorAll(sel).length === 1) return sel;
  }
  if (parent) {
    const siblings = Array.from(parent.children);
    const idx = siblings.indexOf(el) + 1;
    return `${getSelector(parent)} > ${tag}:nth-child(${idx})`;
  }
  return tag;
}

function truncateText(text: string, maxLen = 50): string {
  const trimmed = text.trim().replace(/\s+/g, ' ');
  return trimmed.length > maxLen ? trimmed.slice(0, maxLen) + '...' : trimmed;
}

function observeClicks(): void {
  if (typeof document === 'undefined') return;

  clickHandler = (e: MouseEvent) => {
    const target = e.target as Element | null;
    if (!target) return;

    const selector = getSelector(target);
    const now = Date.now();

    // Track timestamps per selector for rage detection.
    let timestamps = clickTimestamps.get(selector);
    if (!timestamps) {
      timestamps = [];
      clickTimestamps.set(selector, timestamps);
    }
    timestamps.push(now);

    // Prune old timestamps outside the window.
    const cutoff = now - RAGE_CLICK_WINDOW_MS;
    while (timestamps.length > 0 && timestamps[0] < cutoff) {
      timestamps.shift();
    }

    const isRage = timestamps.length >= RAGE_CLICK_THRESHOLD;

    enqueue({
      type: 'click',
      session_id: getSessionId(),
      url: typeof location !== 'undefined' ? location.href : '',
      user_agent: typeof navigator !== 'undefined' ? navigator.userAgent : '',
      timestamp: new Date().toISOString(),
      click: {
        selector,
        text: truncateText((target as HTMLElement).innerText || ''),
        x: e.clientX,
        y: e.clientY,
        is_rage: isRage,
        url: typeof location !== 'undefined' ? location.href : '',
      },
    });
  };

  document.addEventListener('click', clickHandler, true);
}

// --- Resource timing (XHR / fetch) ---

function observeResourceTiming(): void {
  if (typeof PerformanceObserver === 'undefined') return;

  try {
    resourceObserver = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        const res = entry as PerformanceResourceTiming;

        // Only track fetch/XHR initiated requests (skip images, CSS, etc.
        // unless the developer opts in later).
        const initiator = res.initiatorType;
        if (initiator !== 'fetch' && initiator !== 'xmlhttprequest') {
          continue;
        }

        enqueue({
          type: 'resource_timing',
          session_id: getSessionId(),
          url: typeof location !== 'undefined' ? location.href : '',
          user_agent: typeof navigator !== 'undefined' ? navigator.userAgent : '',
          timestamp: new Date().toISOString(),
          resource_timing: {
            name: res.name,
            initiator_type: initiator,
            duration_ms: res.duration,
            transfer_size_kb: res.transferSize / 1024,
            status_code: 0, // not available via PerformanceResourceTiming
          },
        });
      }
    });
    resourceObserver.observe({ type: 'resource', buffered: false });
  } catch {
    // not supported
  }
}
