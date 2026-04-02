import { describe, it, expect } from 'vitest';
import {
  vitalColor,
  scoreColor,
  TIME_WINDOW_OPTIONS,
  formatTimeWindow,
  RUM_TABS,
  tabLabel,
  classifyVital,
} from '../helpers';

describe('vitalColor', () => {
  it('returns green for good rating', () => {
    expect(vitalColor('good')).toBe('#22c55e');
  });

  it('returns yellow for needs-improvement rating', () => {
    expect(vitalColor('needs-improvement')).toBe('#fbbf24');
  });

  it('returns red for poor rating', () => {
    expect(vitalColor('poor')).toBe('#ef4444');
  });

  it('returns red for unknown rating', () => {
    expect(vitalColor('unknown')).toBe('#ef4444');
  });
});

describe('scoreColor', () => {
  it('returns green for score >= 75', () => {
    expect(scoreColor(75)).toBe('#22c55e');
    expect(scoreColor(100)).toBe('#22c55e');
    expect(scoreColor(90)).toBe('#22c55e');
  });

  it('returns yellow for score >= 50 and < 75', () => {
    expect(scoreColor(50)).toBe('#fbbf24');
    expect(scoreColor(60)).toBe('#fbbf24');
    expect(scoreColor(74)).toBe('#fbbf24');
  });

  it('returns red for score < 50', () => {
    expect(scoreColor(49)).toBe('#ef4444');
    expect(scoreColor(0)).toBe('#ef4444');
    expect(scoreColor(25)).toBe('#ef4444');
  });

  it('handles boundary values exactly', () => {
    expect(scoreColor(75)).toBe('#22c55e');
    expect(scoreColor(74.9)).toBe('#fbbf24');
    expect(scoreColor(50)).toBe('#fbbf24');
    expect(scoreColor(49.9)).toBe('#ef4444');
  });
});

describe('TIME_WINDOW_OPTIONS', () => {
  it('has 4 options', () => {
    expect(TIME_WINDOW_OPTIONS).toHaveLength(4);
  });

  it('starts with 15 minutes', () => {
    expect(TIME_WINDOW_OPTIONS[0].minutes).toBe(15);
    expect(TIME_WINDOW_OPTIONS[0].label).toBe('15m');
  });

  it('ends with 2 hours', () => {
    const last = TIME_WINDOW_OPTIONS[TIME_WINDOW_OPTIONS.length - 1];
    expect(last.minutes).toBe(120);
    expect(last.label).toBe('2h');
  });

  it('is in ascending order', () => {
    for (let i = 1; i < TIME_WINDOW_OPTIONS.length; i++) {
      expect(TIME_WINDOW_OPTIONS[i].minutes).toBeGreaterThan(TIME_WINDOW_OPTIONS[i - 1].minutes);
    }
  });
});

describe('formatTimeWindow', () => {
  it('formats minutes < 60 as "Xm"', () => {
    expect(formatTimeWindow(15)).toBe('15m');
    expect(formatTimeWindow(30)).toBe('30m');
  });

  it('formats minutes >= 60 as "Xh"', () => {
    expect(formatTimeWindow(60)).toBe('1h');
    expect(formatTimeWindow(120)).toBe('2h');
  });
});

describe('RUM_TABS', () => {
  it('contains 4 tabs', () => {
    expect(RUM_TABS).toHaveLength(4);
  });

  it('includes vitals, pages, errors, sessions', () => {
    expect(RUM_TABS).toContain('vitals');
    expect(RUM_TABS).toContain('pages');
    expect(RUM_TABS).toContain('errors');
    expect(RUM_TABS).toContain('sessions');
  });
});

describe('tabLabel', () => {
  it('returns "Web Vitals" for vitals tab', () => {
    expect(tabLabel('vitals')).toBe('Web Vitals');
  });

  it('returns "Pages" for pages tab', () => {
    expect(tabLabel('pages')).toBe('Pages');
  });

  it('returns "Errors" for errors tab', () => {
    expect(tabLabel('errors')).toBe('Errors');
  });

  it('returns "Sessions" for sessions tab', () => {
    expect(tabLabel('sessions')).toBe('Sessions');
  });
});

describe('classifyVital', () => {
  it('classifies LCP as good when p75 <= 2500ms', () => {
    expect(classifyVital('lcp', 2000)).toBe('good');
    expect(classifyVital('lcp', 2500)).toBe('good');
  });

  it('classifies LCP as needs-improvement between 2500-4000ms', () => {
    expect(classifyVital('lcp', 3000)).toBe('needs-improvement');
    expect(classifyVital('lcp', 4000)).toBe('needs-improvement');
  });

  it('classifies LCP as poor above 4000ms', () => {
    expect(classifyVital('lcp', 5000)).toBe('poor');
  });

  it('classifies FID as good when p75 <= 100ms', () => {
    expect(classifyVital('fid', 50)).toBe('good');
    expect(classifyVital('fid', 100)).toBe('good');
  });

  it('classifies FID as poor above 300ms', () => {
    expect(classifyVital('fid', 400)).toBe('poor');
  });

  it('classifies CLS as good when p75 <= 0.1', () => {
    expect(classifyVital('cls', 0.05)).toBe('good');
    expect(classifyVital('cls', 0.1)).toBe('good');
  });

  it('classifies CLS as needs-improvement between 0.1 and 0.25', () => {
    expect(classifyVital('cls', 0.15)).toBe('needs-improvement');
  });

  it('classifies CLS as poor above 0.25', () => {
    expect(classifyVital('cls', 0.3)).toBe('poor');
  });

  it('classifies TTFB using correct thresholds', () => {
    expect(classifyVital('ttfb', 500)).toBe('good');
    expect(classifyVital('ttfb', 1000)).toBe('needs-improvement');
    expect(classifyVital('ttfb', 2000)).toBe('poor');
  });

  it('classifies FCP using correct thresholds', () => {
    expect(classifyVital('fcp', 1000)).toBe('good');
    expect(classifyVital('fcp', 2000)).toBe('needs-improvement');
    expect(classifyVital('fcp', 4000)).toBe('poor');
  });
});
