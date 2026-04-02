/**
 * Pure functions extracted from RUMView for testing.
 * Vital color classification and score coloring.
 */

/**
 * Map a web vital rating string to a display color.
 * 'good' -> green, 'needs-improvement' -> yellow, 'poor' -> red
 */
export function vitalColor(rating: string): string {
  if (rating === 'good') return '#22c55e';
  if (rating === 'needs-improvement') return '#fbbf24';
  return '#ef4444';
}

/**
 * Map a numeric UX score to a display color.
 * >= 75 -> green, >= 50 -> yellow, < 50 -> red
 */
export function scoreColor(score: number): string {
  if (score >= 75) return '#22c55e';
  if (score >= 50) return '#fbbf24';
  return '#ef4444';
}

/**
 * Time window options available in the RUM view.
 * Each entry has minutes and a display label.
 */
export const TIME_WINDOW_OPTIONS: { minutes: number; label: string }[] = [
  { minutes: 15, label: '15m' },
  { minutes: 30, label: '30m' },
  { minutes: 60, label: '1h' },
  { minutes: 120, label: '2h' },
];

/**
 * Format a time window option for display.
 * minutes < 60 shows as "Xm", otherwise "Xh".
 */
export function formatTimeWindow(minutes: number): string {
  if (minutes < 60) return `${minutes}m`;
  return `${minutes / 60}h`;
}

/**
 * Available RUM tabs.
 */
export const RUM_TABS = ['vitals', 'pages', 'errors', 'sessions'] as const;
export type RumTab = typeof RUM_TABS[number];

/**
 * Get the display label for a RUM tab.
 */
export function tabLabel(tab: RumTab): string {
  switch (tab) {
    case 'vitals': return 'Web Vitals';
    case 'pages': return 'Pages';
    case 'errors': return 'Errors';
    case 'sessions': return 'Sessions';
  }
}

/**
 * Classify a web vital rating from a numeric p75 value using Web Vitals thresholds.
 * Returns 'good', 'needs-improvement', or 'poor'.
 */
export function classifyVital(
  metric: 'lcp' | 'fid' | 'cls' | 'ttfb' | 'fcp',
  p75: number,
): string {
  const thresholds: Record<string, { good: number; poor: number }> = {
    lcp: { good: 2500, poor: 4000 },
    fid: { good: 100, poor: 300 },
    cls: { good: 0.1, poor: 0.25 },
    ttfb: { good: 800, poor: 1800 },
    fcp: { good: 1800, poor: 3000 },
  };

  const t = thresholds[metric];
  if (!t) return 'poor';
  if (p75 <= t.good) return 'good';
  if (p75 <= t.poor) return 'needs-improvement';
  return 'poor';
}
