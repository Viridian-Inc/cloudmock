import { getSessionId } from './session';
import { enqueue } from './beacon';

interface VitalEntry {
  name: string;
  value: number;
  delta: number;
}

let lcpObserver: PerformanceObserver | null = null;
let fidObserver: PerformanceObserver | null = null;
let clsObserver: PerformanceObserver | null = null;
let fcpObserver: PerformanceObserver | null = null;

// CLS is cumulative — track running total.
let clsValue = 0;

function sendVital(vital: VitalEntry): void {
  enqueue({
    type: 'web_vital',
    session_id: getSessionId(),
    url: location.href,
    user_agent: navigator.userAgent,
    timestamp: new Date().toISOString(),
    web_vital: {
      name: vital.name,
      value: vital.value,
      delta: vital.delta,
      // Rating is computed server-side by the engine, but we include a
      // client-side hint as well for immediate display.
      rating: '',
    },
  });
}

export function observeWebVitals(): void {
  if (typeof PerformanceObserver === 'undefined') return;

  // --- LCP (Largest Contentful Paint) ---
  try {
    lcpObserver = new PerformanceObserver((list) => {
      const entries = list.getEntries();
      if (entries.length === 0) return;
      const last = entries[entries.length - 1] as PerformanceEntry;
      sendVital({ name: 'LCP', value: last.startTime, delta: last.startTime });
    });
    lcpObserver.observe({ type: 'largest-contentful-paint', buffered: true });
  } catch {
    // not supported
  }

  // --- FID (First Input Delay) ---
  try {
    fidObserver = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        const fid = entry as PerformanceEventTiming;
        const delay = fid.processingStart - fid.startTime;
        sendVital({ name: 'FID', value: delay, delta: delay });
      }
    });
    fidObserver.observe({ type: 'first-input', buffered: true });
  } catch {
    // not supported
  }

  // --- CLS (Cumulative Layout Shift) ---
  try {
    clsObserver = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        const ls = entry as PerformanceEntry & {
          hadRecentInput?: boolean;
          value?: number;
        };
        if (!ls.hadRecentInput && ls.value !== undefined) {
          const delta = ls.value;
          clsValue += delta;
          sendVital({ name: 'CLS', value: clsValue, delta });
        }
      }
    });
    clsObserver.observe({ type: 'layout-shift', buffered: true });
  } catch {
    // not supported
  }

  // --- FCP (First Contentful Paint) ---
  try {
    fcpObserver = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        if (entry.name === 'first-contentful-paint') {
          sendVital({
            name: 'FCP',
            value: entry.startTime,
            delta: entry.startTime,
          });
        }
      }
    });
    fcpObserver.observe({ type: 'paint', buffered: true });
  } catch {
    // not supported
  }

  // --- TTFB (Time to First Byte) ---
  collectTTFB();

  // --- INP (Interaction to Next Paint) ---
  collectINP();
}

function collectTTFB(): void {
  try {
    const nav = performance.getEntriesByType(
      'navigation'
    )[0] as PerformanceNavigationTiming | undefined;
    if (nav && nav.responseStart > 0) {
      const ttfb = nav.responseStart - nav.requestStart;
      sendVital({ name: 'TTFB', value: ttfb, delta: ttfb });
    }
  } catch {
    // not supported
  }
}

function collectINP(): void {
  try {
    let maxINP = 0;
    const observer = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        const evt = entry as PerformanceEventTiming;
        const duration = evt.duration;
        if (duration > maxINP) {
          maxINP = duration;
          sendVital({ name: 'INP', value: duration, delta: duration });
        }
      }
    });
    observer.observe({ type: 'event', buffered: true, durationThreshold: 40 });
  } catch {
    // not supported
  }
}

/** Collect page load timing from the Navigation Timing API. */
export function collectPageLoad(): void {
  if (typeof performance === 'undefined') return;

  // Wait for the load event to have all timing data available.
  const collect = () => {
    try {
      const nav = performance.getEntriesByType(
        'navigation'
      )[0] as PerformanceNavigationTiming | undefined;
      if (!nav) return;

      const route =
        typeof location !== 'undefined' ? location.pathname : '';

      enqueue({
        type: 'page_load',
        session_id: getSessionId(),
        url: location.href,
        user_agent: navigator.userAgent,
        timestamp: new Date().toISOString(),
        page_load: {
          route,
          duration_ms: nav.loadEventEnd - nav.startTime,
          ttfb_ms: nav.responseStart - nav.requestStart,
          dom_content_loaded_ms:
            nav.domContentLoadedEventEnd - nav.startTime,
          load_ms: nav.loadEventEnd - nav.startTime,
          transfer_size_kb: nav.transferSize / 1024,
        },
      });
    } catch {
      // timing data not available
    }
  };

  if (document.readyState === 'complete') {
    // Use setTimeout to ensure loadEventEnd is populated.
    setTimeout(collect, 0);
  } else {
    window.addEventListener('load', () => setTimeout(collect, 0));
  }
}

export function disconnectObservers(): void {
  lcpObserver?.disconnect();
  fidObserver?.disconnect();
  clsObserver?.disconnect();
  fcpObserver?.disconnect();
}
