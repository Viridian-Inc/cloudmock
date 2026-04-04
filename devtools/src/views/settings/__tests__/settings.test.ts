import { describe, it, expect, beforeEach, vi } from 'vitest';
import {
  getSavedTheme,
  saveTheme,
  getSavedFontSize,
  saveFontSize,
  isValidTab,
  getTabLabel,
  isLocalMode,
  resetDashboardPreferences,
  SETTINGS_TABS,
  DEFAULT_FONT_SIZE,
  MIN_FONT_SIZE,
  MAX_FONT_SIZE,
} from '../helpers';

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

describe('getSavedTheme / saveTheme', () => {
  it('returns dark as default when nothing stored', () => {
    expect(getSavedTheme()).toBe('dark');
  });

  it('returns dark for invalid stored value', () => {
    storage.set('cloudmock-theme', 'neon');
    expect(getSavedTheme()).toBe('dark');
  });

  it('returns light when light is stored', () => {
    storage.set('cloudmock-theme', 'light');
    expect(getSavedTheme()).toBe('light');
  });

  it('persists theme via saveTheme', () => {
    saveTheme('light');
    expect(storage.get('cloudmock-theme')).toBe('light');
    expect(getSavedTheme()).toBe('light');
  });

  it('roundtrips dark theme', () => {
    saveTheme('dark');
    expect(getSavedTheme()).toBe('dark');
  });
});

describe('getSavedFontSize / saveFontSize', () => {
  it('returns default (13) when nothing stored', () => {
    expect(getSavedFontSize()).toBe(DEFAULT_FONT_SIZE);
  });

  it('loads stored font size', () => {
    storage.set('cloudmock-font-size', '15');
    expect(getSavedFontSize()).toBe(15);
  });

  it('returns default for non-numeric stored value', () => {
    storage.set('cloudmock-font-size', 'abc');
    expect(getSavedFontSize()).toBe(DEFAULT_FONT_SIZE);
  });

  it('clamps below minimum to MIN_FONT_SIZE', () => {
    storage.set('cloudmock-font-size', '5');
    expect(getSavedFontSize()).toBe(MIN_FONT_SIZE);
  });

  it('clamps above maximum to MAX_FONT_SIZE', () => {
    storage.set('cloudmock-font-size', '30');
    expect(getSavedFontSize()).toBe(MAX_FONT_SIZE);
  });

  it('persists font size via saveFontSize', () => {
    saveFontSize(14);
    expect(storage.get('cloudmock-font-size')).toBe('14');
  });
});

describe('SETTINGS_TABS', () => {
  it('has 8 tabs', () => {
    expect(SETTINGS_TABS).toHaveLength(8);
  });

  it('includes all expected tab IDs', () => {
    const ids = SETTINGS_TABS.map((t) => t.id);
    expect(ids).toContain('connections');
    expect(ids).toContain('routing');
    expect(ids).toContain('domains');
    expect(ids).toContain('webhooks');
    expect(ids).toContain('config');
    expect(ids).toContain('appearance');
    expect(ids).toContain('audit');
    expect(ids).toContain('account');
  });
});

describe('isValidTab', () => {
  it('returns true for valid tab IDs', () => {
    expect(isValidTab('connections')).toBe(true);
    expect(isValidTab('appearance')).toBe(true);
    expect(isValidTab('account')).toBe(true);
  });

  it('returns false for invalid tab IDs', () => {
    expect(isValidTab('nonexistent')).toBe(false);
    expect(isValidTab('')).toBe(false);
  });
});

describe('getTabLabel', () => {
  it('returns correct labels for each tab', () => {
    expect(getTabLabel('connections')).toBe('Connections');
    expect(getTabLabel('routing')).toBe('Routing');
    expect(getTabLabel('appearance')).toBe('Appearance');
    expect(getTabLabel('account')).toBe('Account');
  });
});

describe('isLocalMode', () => {
  it('returns true for null URL', () => {
    expect(isLocalMode(null)).toBe(true);
  });

  it('returns true for undefined URL', () => {
    expect(isLocalMode(undefined)).toBe(true);
  });

  it('returns true for empty string', () => {
    expect(isLocalMode('')).toBe(true);
  });

  it('returns true for localhost URL', () => {
    expect(isLocalMode('http://localhost:4599')).toBe(true);
  });

  it('returns false for production URL', () => {
    expect(isLocalMode('https://api.cloudmock.app')).toBe(false);
  });
});

describe('resetDashboardPreferences', () => {
  it('returns empty hidden and favorites arrays', () => {
    const result = resetDashboardPreferences();
    expect(result).toEqual({ hidden: [], favorites: [] });
  });
});
