import { describe, it, expect, beforeEach, vi } from 'vitest';
import {
  loadDashboards,
  saveDashboards,
  loadDashboardPreferences,
  saveDashboardPreferences,
} from '../storage';
import type { DashboardPreferences } from '../storage';
import { PRESET_DASHBOARDS, DASHBOARD_LABELS } from '../presets';
import type { Dashboard } from '../types';

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

describe('loadDashboards', () => {
  it('returns empty array when nothing stored', () => {
    expect(loadDashboards()).toEqual([]);
  });

  it('loads saved dashboards', () => {
    const dashboards: Dashboard[] = [
      {
        id: 'd1', name: 'Test', widgets: [], timeWindow: '15m',
        refreshInterval: 0, createdAt: '2025-01-01', updatedAt: '2025-01-01',
      },
    ];
    storage.set('neureaux:dashboards', JSON.stringify(dashboards));
    const loaded = loadDashboards();
    expect(loaded).toHaveLength(1);
    expect(loaded[0].name).toBe('Test');
  });

  it('returns empty array for corrupted JSON', () => {
    storage.set('neureaux:dashboards', '{{{bad json');
    expect(loadDashboards()).toEqual([]);
  });
});

describe('saveDashboards', () => {
  it('persists dashboards to localStorage', () => {
    const dashboards: Dashboard[] = [
      {
        id: 'd2', name: 'Saved', widgets: [], timeWindow: '1h',
        refreshInterval: 30, createdAt: '2025-01-01', updatedAt: '2025-01-01',
      },
    ];
    saveDashboards(dashboards);
    const raw = storage.get('neureaux:dashboards');
    expect(raw).toBeDefined();
    const parsed = JSON.parse(raw!);
    expect(parsed[0].name).toBe('Saved');
  });
});

describe('loadDashboardPreferences', () => {
  it('returns default preferences when nothing stored', () => {
    const prefs = loadDashboardPreferences();
    expect(prefs).toEqual({ hidden: [], favorites: [] });
  });

  it('loads saved preferences', () => {
    const saved: DashboardPreferences = {
      hidden: ['preset-1'],
      favorites: ['preset-2'],
    };
    storage.set('neureaux:dashboard-prefs', JSON.stringify(saved));
    const prefs = loadDashboardPreferences();
    expect(prefs.hidden).toEqual(['preset-1']);
    expect(prefs.favorites).toEqual(['preset-2']);
  });

  it('returns defaults for corrupted JSON', () => {
    storage.set('neureaux:dashboard-prefs', 'not json');
    const prefs = loadDashboardPreferences();
    expect(prefs).toEqual({ hidden: [], favorites: [] });
  });
});

describe('saveDashboardPreferences', () => {
  it('persists preferences to localStorage', () => {
    saveDashboardPreferences({ hidden: ['a'], favorites: ['b', 'c'] });
    const raw = storage.get('neureaux:dashboard-prefs');
    const parsed = JSON.parse(raw!);
    expect(parsed.hidden).toEqual(['a']);
    expect(parsed.favorites).toEqual(['b', 'c']);
  });
});

describe('PRESET_DASHBOARDS', () => {
  it('has at least 3 presets', () => {
    expect(PRESET_DASHBOARDS.length).toBeGreaterThanOrEqual(3);
  });

  it('each preset has a unique ID', () => {
    const ids = PRESET_DASHBOARDS.map((p) => p.id);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it('each preset has non-empty widgets', () => {
    for (const preset of PRESET_DASHBOARDS) {
      expect(preset.widgets.length).toBeGreaterThan(0);
    }
  });

  it('all preset labels are valid', () => {
    for (const preset of PRESET_DASHBOARDS) {
      expect(DASHBOARD_LABELS).toContain(preset.label);
    }
  });

  it('each preset widget has valid grid positions', () => {
    for (const preset of PRESET_DASHBOARDS) {
      for (const widget of preset.widgets) {
        expect(widget.col).toBeGreaterThanOrEqual(0);
        expect(widget.col).toBeLessThan(12);
        expect(widget.colSpan).toBeGreaterThan(0);
        expect(widget.colSpan).toBeLessThanOrEqual(12);
        expect(widget.row).toBeGreaterThanOrEqual(0);
        expect(widget.rowSpan).toBeGreaterThan(0);
      }
    }
  });
});

describe('preset merging with custom dashboards', () => {
  it('combines preset IDs with custom dashboard IDs without collision', () => {
    const custom: Dashboard[] = [
      {
        id: 'custom-1', name: 'My Dashboard', widgets: [], timeWindow: '15m',
        refreshInterval: 0, createdAt: '2025-01-01', updatedAt: '2025-01-01',
      },
    ];
    const allIds = [...PRESET_DASHBOARDS.map((p) => p.id), ...custom.map((d) => d.id)];
    expect(new Set(allIds).size).toBe(allIds.length);
  });

  it('favorites filter correctly from mixed set', () => {
    const prefs: DashboardPreferences = {
      hidden: [],
      favorites: [PRESET_DASHBOARDS[0].id, 'custom-1'],
    };
    const customIds = ['custom-1', 'custom-2'];
    const presetIds = PRESET_DASHBOARDS.map((p) => p.id);
    const allIds = [...presetIds, ...customIds];
    const favorited = allIds.filter((id) => prefs.favorites.includes(id));
    expect(favorited).toContain(PRESET_DASHBOARDS[0].id);
    expect(favorited).toContain('custom-1');
    expect(favorited).not.toContain('custom-2');
  });
});
