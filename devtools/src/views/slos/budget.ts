interface SLORule {
  service: string;
  action: string;
  p50_ms: number;
  p95_ms: number;
  p99_ms: number;
  error_rate: number;
}

interface SLOWindow {
  Service: string;
  Action: string;
  Total: number;
  Errors: number;
  P50Ms: number;
  P95Ms: number;
  P99Ms: number;
  ErrorRate: number;
  Healthy: boolean;
  Violations: string[];
}

/** Compute current error budget for each window as a single value */
export function computeCurrentBudgets(
  windows: SLOWindow[],
  rules: SLORule[],
): { service: string; budgetPct: number }[] {
  const results: { service: string; budgetPct: number }[] = [];

  for (const w of windows) {
    const rule = rules.find(
      (r) => r.service === w.Service && (r.action === '*' || r.action === w.Action),
    );
    if (!rule) continue;

    const allowedErrorRate = rule.error_rate;
    if (allowedErrorRate <= 0) continue;

    const actualErrorRate = w.Total > 0 ? w.Errors / w.Total : 0;
    const currentBudget = Math.max(-50, (1 - actualErrorRate / allowedErrorRate) * 100);

    results.push({
      service: `${w.Service} / ${w.Action || '*'}`,
      budgetPct: currentBudget,
    });
  }

  return results;
}

/** Color for budget gauge based on remaining budget */
export function budgetColor(currentPct: number): string {
  if (currentPct <= 0) return '#ff4e5e';
  if (currentPct <= 20) return '#fad065';
  return '#36d982';
}

export function formatLatency(ms: number): string {
  if (ms < 1) return `${(ms * 1000).toFixed(0)}us`;
  if (ms < 1000) return `${ms.toFixed(1)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

export function formatRate(rate: number): string {
  return `${(rate * 100).toFixed(2)}%`;
}
