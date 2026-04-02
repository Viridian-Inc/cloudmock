import { describe, it, expect } from 'vitest';
import { computeCurrentBudgets, budgetColor, formatLatency, formatRate } from '../budget';

function makeWindow(overrides: Record<string, any> = {}) {
  return {
    Service: 'api',
    Action: '*',
    Total: 1000,
    Errors: 0,
    P50Ms: 10,
    P95Ms: 50,
    P99Ms: 100,
    ErrorRate: 0,
    Healthy: true,
    Violations: [],
    ...overrides,
  };
}

function makeRule(overrides: Record<string, any> = {}) {
  return {
    service: 'api',
    action: '*',
    p50_ms: 100,
    p95_ms: 500,
    p99_ms: 1000,
    error_rate: 0.01,
    ...overrides,
  };
}

describe('computeCurrentBudgets', () => {
  it('returns 100% budget when zero errors', () => {
    const budgets = computeCurrentBudgets([makeWindow()], [makeRule()]);
    expect(budgets.length).toBe(1);
    expect(budgets[0].budgetPct).toBe(100);
  });

  it('returns reduced budget when errors exist', () => {
    const budgets = computeCurrentBudgets(
      [makeWindow({ Errors: 5, Total: 1000 })],
      [makeRule({ error_rate: 0.01 })],
    );
    // actualRate = 5/1000 = 0.005, budget = (1 - 0.005/0.01) * 100 = 50%
    expect(budgets[0].budgetPct).toBe(50);
  });

  it('returns negative budget when errors exceed SLO', () => {
    const budgets = computeCurrentBudgets(
      [makeWindow({ Errors: 20, Total: 1000 })],
      [makeRule({ error_rate: 0.01 })],
    );
    // actualRate = 0.02, budget = (1 - 0.02/0.01) * 100 = -100, clamped to -50
    expect(budgets[0].budgetPct).toBe(-50);
  });

  it('returns 100% when no traffic', () => {
    const budgets = computeCurrentBudgets(
      [makeWindow({ Total: 0, Errors: 0 })],
      [makeRule()],
    );
    // actualRate = 0, budget = (1 - 0/0.01) * 100 = 100%
    expect(budgets[0].budgetPct).toBe(100);
  });

  it('skips windows without matching rules', () => {
    const budgets = computeCurrentBudgets(
      [makeWindow({ Service: 'orphan' })],
      [makeRule({ service: 'api' })],
    );
    expect(budgets.length).toBe(0);
  });

  it('skips rules with zero allowed error rate', () => {
    const budgets = computeCurrentBudgets(
      [makeWindow()],
      [makeRule({ error_rate: 0 })],
    );
    expect(budgets.length).toBe(0);
  });

  it('formats service name as "Service / Action"', () => {
    const budgets = computeCurrentBudgets(
      [makeWindow({ Service: 'billing', Action: 'CreateOrder' })],
      [makeRule({ service: 'billing', action: '*' })],
    );
    expect(budgets[0].service).toBe('billing / CreateOrder');
  });

  it('matches wildcard action in rules', () => {
    const budgets = computeCurrentBudgets(
      [makeWindow({ Action: 'GetUser' })],
      [makeRule({ action: '*' })],
    );
    expect(budgets.length).toBe(1);
  });
});

describe('budgetColor', () => {
  it('returns red for exhausted budget', () => {
    expect(budgetColor(0)).toBe('#ff4e5e');
    expect(budgetColor(-10)).toBe('#ff4e5e');
  });

  it('returns yellow for low budget (1-20%)', () => {
    expect(budgetColor(10)).toBe('#fad065');
    expect(budgetColor(20)).toBe('#fad065');
  });

  it('returns green for healthy budget (>20%)', () => {
    expect(budgetColor(50)).toBe('#36d982');
    expect(budgetColor(100)).toBe('#36d982');
  });
});

describe('formatLatency', () => {
  it('formats microseconds', () => {
    expect(formatLatency(0.5)).toBe('500us');
  });

  it('formats milliseconds', () => {
    expect(formatLatency(42.3)).toBe('42.3ms');
  });

  it('formats seconds', () => {
    expect(formatLatency(1500)).toBe('1.50s');
  });
});

describe('formatRate', () => {
  it('formats as percentage', () => {
    expect(formatRate(0.05)).toBe('5.00%');
  });

  it('formats zero rate', () => {
    expect(formatRate(0)).toBe('0.00%');
  });

  it('formats small rate', () => {
    expect(formatRate(0.001)).toBe('0.10%');
  });
});
