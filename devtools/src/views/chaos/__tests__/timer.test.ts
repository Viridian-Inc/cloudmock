import { describe, it, expect, vi, afterEach } from 'vitest';
import {
  formatCountdown,
  typeLabel,
  valueLabel,
  isExpired,
  remainingSeconds,
  DURATION_OPTIONS,
} from '../timer';
import type { ChaosRule } from '../timer';

describe('formatCountdown', () => {
  it('formats zero seconds', () => {
    expect(formatCountdown(0)).toBe('0s');
  });

  it('formats negative seconds as 0s', () => {
    expect(formatCountdown(-5)).toBe('0s');
  });

  it('formats seconds only', () => {
    expect(formatCountdown(45)).toBe('45s');
  });

  it('formats minutes and seconds', () => {
    expect(formatCountdown(90)).toBe('1m 30s');
  });

  it('pads single-digit seconds', () => {
    expect(formatCountdown(65)).toBe('1m 05s');
  });

  it('formats exact minutes', () => {
    expect(formatCountdown(300)).toBe('5m 00s');
  });

  it('formats large durations', () => {
    expect(formatCountdown(3661)).toBe('61m 01s');
  });
});

describe('typeLabel', () => {
  it('returns Latency for latency type', () => {
    expect(typeLabel('latency')).toBe('Latency');
  });

  it('returns Error for error type', () => {
    expect(typeLabel('error')).toBe('Error');
  });

  it('returns Throttle for throttle type', () => {
    expect(typeLabel('throttle')).toBe('Throttle');
  });
});

describe('valueLabel', () => {
  it('formats latency rule', () => {
    const rule: ChaosRule = { service: 'dynamodb', type: 'latency', value: 2000 };
    expect(valueLabel(rule)).toBe('2000ms');
  });

  it('formats error rule', () => {
    const rule: ChaosRule = { service: 'cognito', type: 'error', value: 403 };
    expect(valueLabel(rule)).toBe('HTTP 403');
  });

  it('formats throttle rule', () => {
    const rule: ChaosRule = { service: 's3', type: 'throttle', value: 50 };
    expect(valueLabel(rule)).toBe('50%');
  });
});

describe('isExpired', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('returns false for null expiresAt', () => {
    expect(isExpired(null)).toBe(false);
  });

  it('returns true when current time is past expiresAt', () => {
    vi.spyOn(Date, 'now').mockReturnValue(2000);
    expect(isExpired(1000)).toBe(true);
  });

  it('returns false when current time is before expiresAt', () => {
    vi.spyOn(Date, 'now').mockReturnValue(500);
    expect(isExpired(1000)).toBe(false);
  });

  it('returns true when current time equals expiresAt', () => {
    vi.spyOn(Date, 'now').mockReturnValue(1000);
    expect(isExpired(1000)).toBe(true);
  });
});

describe('remainingSeconds', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('returns 0 for null expiresAt', () => {
    expect(remainingSeconds(null)).toBe(0);
  });

  it('returns remaining seconds rounded up', () => {
    vi.spyOn(Date, 'now').mockReturnValue(5000);
    // 10000 - 5000 = 5000ms = 5s
    expect(remainingSeconds(10000)).toBe(5);
  });

  it('returns 0 when expired', () => {
    vi.spyOn(Date, 'now').mockReturnValue(15000);
    expect(remainingSeconds(10000)).toBe(0);
  });

  it('rounds up fractional seconds', () => {
    vi.spyOn(Date, 'now').mockReturnValue(5500);
    // 10000 - 5500 = 4500ms -> ceil(4.5) = 5
    expect(remainingSeconds(10000)).toBe(5);
  });
});

describe('DURATION_OPTIONS', () => {
  it('has 6 options', () => {
    expect(DURATION_OPTIONS.length).toBe(6);
  });

  it('starts with Indefinite (0 seconds)', () => {
    expect(DURATION_OPTIONS[0].label).toBe('Indefinite');
    expect(DURATION_OPTIONS[0].seconds).toBe(0);
  });

  it('ends with 1 hour', () => {
    const last = DURATION_OPTIONS[DURATION_OPTIONS.length - 1];
    expect(last.label).toBe('1 hour');
    expect(last.seconds).toBe(3600);
  });
});
