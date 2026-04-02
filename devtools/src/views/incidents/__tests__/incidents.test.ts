import { describe, it, expect, vi, afterEach } from 'vitest';
import {
  groupIncidents,
  filterByStatus,
  sortBySeverity,
  severityIcon,
  formatTime,
  relativeTime,
  SEVERITY_COLORS,
} from '../helpers';
import type { Incident } from '../helpers';

function makeIncident(overrides: Partial<Incident> = {}): Incident {
  return {
    id: 'inc-1',
    title: 'Test Incident',
    service: 'api-gateway',
    severity: 'warning',
    status: 'open',
    message: 'Something went wrong',
    timestamp: '2025-01-15T10:00:00Z',
    ...overrides,
  };
}

describe('filterByStatus', () => {
  const incidents = [
    makeIncident({ id: '1', status: 'open' }),
    makeIncident({ id: '2', status: 'acknowledged' }),
    makeIncident({ id: '3', status: 'resolved' }),
    makeIncident({ id: '4', status: 'open' }),
  ];

  it('returns all incidents for "all" filter', () => {
    expect(filterByStatus(incidents, 'all')).toHaveLength(4);
  });

  it('filters open incidents', () => {
    const result = filterByStatus(incidents, 'open');
    expect(result).toHaveLength(2);
    expect(result.every((i) => i.status === 'open')).toBe(true);
  });

  it('filters acknowledged incidents', () => {
    const result = filterByStatus(incidents, 'acknowledged');
    expect(result).toHaveLength(1);
    expect(result[0].id).toBe('2');
  });

  it('filters resolved incidents', () => {
    const result = filterByStatus(incidents, 'resolved');
    expect(result).toHaveLength(1);
    expect(result[0].id).toBe('3');
  });

  it('returns empty for unknown filter', () => {
    expect(filterByStatus(incidents, 'nonexistent')).toHaveLength(0);
  });
});

describe('sortBySeverity', () => {
  it('sorts critical first, then warning, then info', () => {
    const incidents = [
      makeIncident({ id: '1', severity: 'info' }),
      makeIncident({ id: '2', severity: 'critical' }),
      makeIncident({ id: '3', severity: 'warning' }),
    ];
    const sorted = sortBySeverity(incidents);
    expect(sorted[0].severity).toBe('critical');
    expect(sorted[1].severity).toBe('warning');
    expect(sorted[2].severity).toBe('info');
  });

  it('preserves order for same severity', () => {
    const incidents = [
      makeIncident({ id: 'a', severity: 'warning' }),
      makeIncident({ id: 'b', severity: 'warning' }),
    ];
    const sorted = sortBySeverity(incidents);
    expect(sorted[0].id).toBe('a');
    expect(sorted[1].id).toBe('b');
  });

  it('does not mutate the original array', () => {
    const incidents = [
      makeIncident({ id: '1', severity: 'info' }),
      makeIncident({ id: '2', severity: 'critical' }),
    ];
    const sorted = sortBySeverity(incidents);
    expect(incidents[0].severity).toBe('info');
    expect(sorted[0].severity).toBe('critical');
  });
});

describe('groupIncidents', () => {
  it('returns empty array for no incidents', () => {
    expect(groupIncidents([], 15 * 60 * 1000)).toEqual([]);
  });

  it('groups incidents within the time window', () => {
    const incidents = [
      makeIncident({ id: '1', timestamp: '2025-01-15T10:00:00Z' }),
      makeIncident({ id: '2', timestamp: '2025-01-15T10:05:00Z' }),
      makeIncident({ id: '3', timestamp: '2025-01-15T10:10:00Z' }),
    ];
    // 15-minute window should group all three
    const groups = groupIncidents(incidents, 15 * 60 * 1000);
    expect(groups).toHaveLength(1);
    expect(groups[0].incidents).toHaveLength(3);
  });

  it('separates incidents outside the time window', () => {
    const incidents = [
      makeIncident({ id: '1', timestamp: '2025-01-15T10:00:00Z' }),
      makeIncident({ id: '2', timestamp: '2025-01-15T12:00:00Z' }),
    ];
    const groups = groupIncidents(incidents, 15 * 60 * 1000);
    expect(groups).toHaveLength(2);
    expect(groups[0].incidents).toHaveLength(1);
    expect(groups[1].incidents).toHaveLength(1);
  });

  it('computes average time for group', () => {
    const incidents = [
      makeIncident({ id: '1', timestamp: '2025-01-15T10:00:00Z' }),
      makeIncident({ id: '2', timestamp: '2025-01-15T10:10:00Z' }),
    ];
    const groups = groupIncidents(incidents, 15 * 60 * 1000);
    const ts1 = new Date('2025-01-15T10:00:00Z').getTime();
    const ts2 = new Date('2025-01-15T10:10:00Z').getTime();
    expect(groups[0].time).toBe((ts1 + ts2) / 2);
  });

  it('handles single incident', () => {
    const incidents = [makeIncident({ id: '1' })];
    const groups = groupIncidents(incidents, 15 * 60 * 1000);
    expect(groups).toHaveLength(1);
    expect(groups[0].incidents).toHaveLength(1);
  });
});

describe('severityIcon', () => {
  it('returns !! for critical', () => {
    expect(severityIcon('critical')).toBe('!!');
  });

  it('returns ! for warning', () => {
    expect(severityIcon('warning')).toBe('!');
  });

  it('returns i for info', () => {
    expect(severityIcon('info')).toBe('i');
  });

  it('returns i for unknown severity', () => {
    expect(severityIcon('unknown')).toBe('i');
  });
});

describe('SEVERITY_COLORS', () => {
  it('maps critical to red', () => {
    expect(SEVERITY_COLORS.critical).toBe('#ff4e5e');
  });

  it('maps info to blue', () => {
    expect(SEVERITY_COLORS.info).toBe('#538eff');
  });

  it('maps warning to yellow', () => {
    expect(SEVERITY_COLORS.warning).toBe('#fad065');
  });
});

describe('formatTime', () => {
  it('formats valid timestamp', () => {
    const result = formatTime('2025-01-15T10:30:00Z');
    // Should return a locale string, not '--:--:--'
    expect(result).not.toBe('--:--:--');
  });

  it('returns fallback for invalid timestamp', () => {
    expect(formatTime('not-a-date')).toBe('--:--:--');
  });
});

describe('relativeTime', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('returns "just now" for very recent timestamps', () => {
    vi.spyOn(Date, 'now').mockReturnValue(new Date('2025-01-15T10:00:30Z').getTime());
    expect(relativeTime('2025-01-15T10:00:00Z')).toBe('just now');
  });

  it('returns minutes ago', () => {
    vi.spyOn(Date, 'now').mockReturnValue(new Date('2025-01-15T10:05:00Z').getTime());
    expect(relativeTime('2025-01-15T10:00:00Z')).toBe('5m ago');
  });

  it('returns hours ago', () => {
    vi.spyOn(Date, 'now').mockReturnValue(new Date('2025-01-15T13:00:00Z').getTime());
    expect(relativeTime('2025-01-15T10:00:00Z')).toBe('3h ago');
  });

  it('returns empty for invalid timestamp', () => {
    expect(relativeTime('invalid')).toBe('');
  });
});
