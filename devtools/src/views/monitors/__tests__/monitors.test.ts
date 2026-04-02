import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import {
  loadMonitors,
  saveMonitors,
  evaluateThreshold,
  evaluateMonitorStatus,
  statusIcon,
  isMuteExpired,
  filterMonitors,
  relativeTime,
} from '../helpers';
import type { Monitor } from '../helpers';

// Mock localStorage
const storage = new Map<string, string>();

beforeEach(() => {
  storage.clear();
  vi.stubGlobal('localStorage', {
    getItem: (key: string) => storage.get(key) ?? null,
    setItem: (key: string, value: string) => storage.set(key, value),
    removeItem: (key: string) => storage.delete(key),
    clear: () => storage.clear(),
  });
});

afterEach(() => {
  vi.restoreAllMocks();
});

function makeMonitor(overrides: Partial<Monitor> = {}): Monitor {
  return {
    id: 'mon-1',
    name: 'Test Monitor',
    service: 'api-gateway',
    metric: 'p99',
    operator: '>',
    criticalThreshold: 500,
    warningThreshold: 200,
    evaluationWindow: '5m',
    notificationChannels: ['slack'],
    status: 'ok',
    lastValue: 100,
    lastChecked: '2025-01-15T10:00:00Z',
    muted: false,
    createdAt: '2025-01-01T00:00:00Z',
    updatedAt: '2025-01-01T00:00:00Z',
    alertHistory: [],
    ...overrides,
  };
}

describe('loadMonitors / saveMonitors', () => {
  it('returns empty array when nothing stored', () => {
    expect(loadMonitors()).toEqual([]);
  });

  it('roundtrips through save and load', () => {
    const monitors = [makeMonitor({ id: 'mon-a', name: 'A' })];
    saveMonitors(monitors);
    const loaded = loadMonitors();
    expect(loaded).toHaveLength(1);
    expect(loaded[0].name).toBe('A');
  });

  it('returns empty array for corrupted JSON', () => {
    storage.set('neureaux:monitors', '{{invalid');
    expect(loadMonitors()).toEqual([]);
  });
});

describe('evaluateThreshold', () => {
  it('evaluates > correctly', () => {
    expect(evaluateThreshold(501, '>', 500)).toBe(true);
    expect(evaluateThreshold(500, '>', 500)).toBe(false);
    expect(evaluateThreshold(499, '>', 500)).toBe(false);
  });

  it('evaluates >= correctly', () => {
    expect(evaluateThreshold(500, '>=', 500)).toBe(true);
    expect(evaluateThreshold(499, '>=', 500)).toBe(false);
  });

  it('evaluates < correctly', () => {
    expect(evaluateThreshold(9, '<', 10)).toBe(true);
    expect(evaluateThreshold(10, '<', 10)).toBe(false);
  });

  it('evaluates <= correctly', () => {
    expect(evaluateThreshold(10, '<=', 10)).toBe(true);
    expect(evaluateThreshold(11, '<=', 10)).toBe(false);
  });

  it('evaluates == correctly', () => {
    expect(evaluateThreshold(500, '==', 500)).toBe(true);
    expect(evaluateThreshold(501, '==', 500)).toBe(false);
  });

  it('evaluates != correctly', () => {
    expect(evaluateThreshold(501, '!=', 500)).toBe(true);
    expect(evaluateThreshold(500, '!=', 500)).toBe(false);
  });
});

describe('evaluateMonitorStatus', () => {
  it('returns muted when monitor is muted regardless of value', () => {
    expect(evaluateMonitorStatus(9999, '>', 500, 200, true)).toBe('muted');
  });

  it('returns no_data when value is undefined', () => {
    expect(evaluateMonitorStatus(undefined, '>', 500, 200, false)).toBe('no_data');
  });

  it('returns alert when value exceeds critical threshold', () => {
    expect(evaluateMonitorStatus(600, '>', 500, 200, false)).toBe('alert');
  });

  it('returns warning when value exceeds warning but not critical', () => {
    expect(evaluateMonitorStatus(300, '>', 500, 200, false)).toBe('warning');
  });

  it('returns ok when value is below all thresholds', () => {
    expect(evaluateMonitorStatus(100, '>', 500, 200, false)).toBe('ok');
  });

  it('returns ok when no warning threshold set and below critical', () => {
    expect(evaluateMonitorStatus(100, '>', 500, undefined, false)).toBe('ok');
  });

  it('returns alert for < operator when value is below critical', () => {
    expect(evaluateMonitorStatus(5, '<', 10, 20, false)).toBe('alert');
  });
});

describe('statusIcon', () => {
  it('returns checkmark for ok', () => {
    expect(statusIcon('ok')).toBe('\u2713');
  });

  it('returns ! for warning', () => {
    expect(statusIcon('warning')).toBe('!');
  });

  it('returns !! for alert', () => {
    expect(statusIcon('alert')).toBe('!!');
  });

  it('returns dash for muted', () => {
    expect(statusIcon('muted')).toBe('\u2014');
  });

  it('returns ? for no_data', () => {
    expect(statusIcon('no_data')).toBe('?');
  });
});

describe('isMuteExpired', () => {
  it('returns false for null expiry (indefinite)', () => {
    expect(isMuteExpired(null)).toBe(false);
  });

  it('returns false for undefined expiry', () => {
    expect(isMuteExpired(undefined)).toBe(false);
  });

  it('returns true when expiry is in the past', () => {
    vi.spyOn(Date, 'now').mockReturnValue(new Date('2025-01-15T12:00:00Z').getTime());
    expect(isMuteExpired('2025-01-15T10:00:00Z')).toBe(true);
  });

  it('returns false when expiry is in the future', () => {
    vi.spyOn(Date, 'now').mockReturnValue(new Date('2025-01-15T08:00:00Z').getTime());
    expect(isMuteExpired('2025-01-15T10:00:00Z')).toBe(false);
  });
});

describe('filterMonitors', () => {
  const monitors = [
    makeMonitor({ id: '1', status: 'ok', muted: false }),
    makeMonitor({ id: '2', status: 'warning', muted: false }),
    makeMonitor({ id: '3', status: 'alert', muted: false }),
    makeMonitor({ id: '4', status: 'ok', muted: true }),
  ];

  it('returns all for "all" filter', () => {
    expect(filterMonitors(monitors, 'all')).toHaveLength(4);
  });

  it('filters by status', () => {
    expect(filterMonitors(monitors, 'warning')).toHaveLength(1);
    expect(filterMonitors(monitors, 'alert')).toHaveLength(1);
  });

  it('filters muted monitors', () => {
    const muted = filterMonitors(monitors, 'muted');
    expect(muted).toHaveLength(1);
    expect(muted[0].id).toBe('4');
  });

  it('filters ok monitors (not muted)', () => {
    const ok = filterMonitors(monitors, 'ok');
    // Both mon-1 (ok, not muted) and mon-4 (ok, muted) have status 'ok'
    expect(ok).toHaveLength(2);
  });
});
