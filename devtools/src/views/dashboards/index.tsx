import { useState, useCallback, useMemo, useEffect, useRef } from 'preact/hooks';
import type { Dashboard, Widget, TimeWindowOption, QueryResult } from './types';
import { useMetricQuery } from './use-metric-query';
import { DashboardGrid } from './grid';
import { WidgetEditor } from './widget-editor';
import {
  loadDashboards,
  saveDashboards,
  loadDashboardPreferences,
  saveDashboardPreferences,
  loadDashboardsFromAPI,
  saveDashboardsToAPI,
  exportDashboard,
  importDashboard,
} from './storage';
import type { DashboardPreferences } from './storage';
import { PRESET_DASHBOARDS, DASHBOARD_LABELS } from './presets';
import type { DashboardLabel } from './presets';
import { TimeRangeSelector } from '../../components/time-range-selector/time-range-selector';
import './dashboards.css';

const TIME_WINDOWS: TimeWindowOption[] = [
  { label: '15m', value: '15m', seconds: 900 },
  { label: '1h', value: '1h', seconds: 3600 },
  { label: '6h', value: '6h', seconds: 21600 },
  { label: '24h', value: '24h', seconds: 86400 },
  { label: '7d', value: '7d', seconds: 604800 },
];

const REFRESH_OPTIONS = [
  { label: 'Off', value: 0 },
  { label: '10s', value: 10 },
  { label: '30s', value: 30 },
  { label: '1m', value: 60 },
  { label: '5m', value: 300 },
];

function newId(): string {
  return `db-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

/** Convert preset dashboards to Dashboard objects */
function presetToDashboard(preset: (typeof PRESET_DASHBOARDS)[number]): Dashboard {
  return {
    id: preset.id,
    name: preset.name,
    description: preset.description,
    widgets: preset.widgets,
    timeWindow: '1h',
    refreshInterval: 30,
    createdAt: '2025-01-01T00:00:00.000Z',
    updatedAt: '2025-01-01T00:00:00.000Z',
  };
}

/** Check whether a dashboard ID belongs to a preset */
function isPresetId(id: string): boolean {
  return PRESET_DASHBOARDS.some((p) => p.id === id);
}

/** Get the label for a dashboard (presets have labels, custom dashboards do not) */
function getDashboardLabel(id: string): DashboardLabel | null {
  const preset = PRESET_DASHBOARDS.find((p) => p.id === id);
  return preset?.label ?? null;
}

const LABEL_COLORS: Record<DashboardLabel, string> = {
  infrastructure: 'var(--brand-teal, #4AE5F8)',
  application: '#a78bfa',
  performance: '#fbbf24',
  security: '#f472b6',
};

/** Hook that manages query results for all widgets in a dashboard */
function useDashboardQueries(
  dashboard: Dashboard,
): Map<string, QueryResult | null> {
  const results = new Map<string, QueryResult | null>();

  // Each widget uses the metric query hook
  for (const widget of dashboard.widgets) {
    // We can't conditionally call hooks, but the hook handles disabled state.
    // Because the number of widgets is dynamic, we use a single combined approach:
    // fetch all via a single effect. However, for simplicity and to match the
    // hook-per-widget pattern, we collect results after render.
    // In practice each widget fetches its own data via the grid rendering.
  }

  return results;
}

/** Wrapper component that fetches data for one widget */
function WidgetDataProvider({
  widget,
  timeWindow,
  refreshInterval,
  children,
  onData,
}: {
  widget: Widget;
  timeWindow: string;
  refreshInterval: number;
  children: preact.ComponentChildren;
  onData: (widgetId: string, data: QueryResult | null) => void;
}) {
  const { data } = useMetricQuery({
    query: widget.query,
    timeWindow,
    refreshInterval,
  });

  // Push data up so the grid can consume it
  if (data) onData(widget.id, data);

  return <>{children}</>;
}

export function DashboardsView() {
  // Dashboard management state
  const [mode, setMode] = useState<'list' | 'dashboard'>('list');
  const [dashboards, setDashboards] = useState<Dashboard[]>(() => {
    const saved = loadDashboards();
    // Filter out any saved dashboards that clash with preset IDs --
    // presets are always injected fresh from the source code.
    return saved.filter((d) => !isPresetId(d.id));
  });
  const [activeDashboardId, setActiveDashboardId] = useState<string>(
    PRESET_DASHBOARDS[0].id,
  );

  // Preferences for hide/favorite
  const [prefs, setPrefs] = useState<DashboardPreferences>(loadDashboardPreferences);
  const [labelFilter, setLabelFilter] = useState<DashboardLabel | null>(null);
  const [showHidden, setShowHidden] = useState(false);
  const importInputRef = useRef<HTMLInputElement>(null);

  // On mount: try loading dashboards from API, fall back to localStorage
  useEffect(() => {
    let cancelled = false;
    loadDashboardsFromAPI().then((apiDashboards) => {
      if (cancelled) return;
      if (apiDashboards && apiDashboards.length > 0) {
        // Filter out presets (same logic as localStorage load)
        const custom = apiDashboards.filter((d) => !isPresetId(d.id));
        setDashboards(custom);
        // Cache to localStorage for offline use
        saveDashboards(custom);
      }
    });
    return () => { cancelled = true; };
  }, []);

  // Persist dashboards (user-created only) to both localStorage and API
  useEffect(() => {
    saveDashboards(dashboards);
    saveDashboardsToAPI(dashboards);
  }, [dashboards]);

  // Persist preferences
  useEffect(() => {
    saveDashboardPreferences(prefs);
  }, [prefs]);

  // Merged list: presets first, then user dashboards
  const allDashboards: Dashboard[] = useMemo(() => {
    const presetDashboards = PRESET_DASHBOARDS.map(presetToDashboard);
    return [...presetDashboards, ...dashboards];
  }, [dashboards]);
  const [editingWidgetId, setEditingWidgetId] = useState<string | null>(null);
  const [showNewWidget, setShowNewWidget] = useState(false);
  const [queryResults, setQueryResults] = useState<Map<string, QueryResult | null>>(new Map());

  const activeDashboard = allDashboards.find((d) => d.id === activeDashboardId) ?? null;

  // Time window & refresh
  const [timeWindow, setTimeWindow] = useState(activeDashboard?.timeWindow ?? '1h');
  const [refreshInterval, setRefreshInterval] = useState(activeDashboard?.refreshInterval ?? 30);

  // Dashboard-level time range selector state
  const dashboardDataRange = useMemo(() => {
    const twOption = TIME_WINDOWS.find((tw) => tw.value === timeWindow);
    const seconds = twOption?.seconds ?? 3600;
    const now = Date.now();
    return { start: now - seconds * 1000, end: now };
  }, [timeWindow]);
  const [dashboardSelectedRange, setDashboardSelectedRange] = useState(dashboardDataRange);
  // Keep the selected range in sync when the time window preset changes
  useEffect(() => {
    setDashboardSelectedRange(dashboardDataRange);
  }, [dashboardDataRange]);

  const handleDataUpdate = useCallback((widgetId: string, data: QueryResult | null) => {
    setQueryResults((prev) => {
      const next = new Map(prev);
      next.set(widgetId, data);
      return next;
    });
  }, []);

  const handleWidgetSave = useCallback(
    (widget: Widget) => {
      setDashboards((prev) =>
        prev.map((d) => {
          if (d.id !== activeDashboardId) return d;
          const existing = d.widgets.findIndex((w) => w.id === widget.id);
          const widgets =
            existing >= 0
              ? d.widgets.map((w) => (w.id === widget.id ? widget : w))
              : [...d.widgets, widget];
          return { ...d, widgets, updatedAt: new Date().toISOString() };
        }),
      );
      setEditingWidgetId(null);
      setShowNewWidget(false);
    },
    [activeDashboardId],
  );

  const handleWidgetDelete = useCallback(
    (widgetId: string) => {
      setDashboards((prev) =>
        prev.map((d) => {
          if (d.id !== activeDashboardId) return d;
          return {
            ...d,
            widgets: d.widgets.filter((w) => w.id !== widgetId),
            updatedAt: new Date().toISOString(),
          };
        }),
      );
      setEditingWidgetId(null);
    },
    [activeDashboardId],
  );

  const handleWidgetResize = useCallback(
    (widgetId: string, colSpan: number, rowSpan: number) => {
      setDashboards((prev) =>
        prev.map((d) => {
          if (d.id !== activeDashboardId) return d;
          return {
            ...d,
            widgets: d.widgets.map((w) =>
              w.id === widgetId ? { ...w, colSpan, rowSpan } : w,
            ),
          };
        }),
      );
    },
    [activeDashboardId],
  );

  const handleCreateDashboard = () => {
    const id = newId();
    const dashboard: Dashboard = {
      id,
      name: 'New Dashboard',
      widgets: [],
      timeWindow: '1h',
      refreshInterval: 30,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    setDashboards((prev) => [...prev, dashboard]);
    setActiveDashboardId(id);
    setMode('dashboard');
  };

  const editingWidget = editingWidgetId
    ? activeDashboard?.widgets.find((w) => w.id === editingWidgetId)
    : undefined;

  // Helper: toggle a value in a string array
  const toggleInArray = (arr: string[], id: string) =>
    arr.includes(id) ? arr.filter((x) => x !== id) : [...arr, id];

  const toggleFavorite = (id: string) => {
    setPrefs((p) => ({ ...p, favorites: toggleInArray(p.favorites, id) }));
  };

  const toggleHidden = (id: string) => {
    setPrefs((p) => ({ ...p, hidden: toggleInArray(p.hidden, id) }));
  };

  const handleDeleteDashboard = (id: string) => {
    setDashboards((prev) => prev.filter((d) => d.id !== id));
    // Also clean up preferences
    setPrefs((p) => ({
      hidden: p.hidden.filter((x) => x !== id),
      favorites: p.favorites.filter((x) => x !== id),
    }));
  };

  const handleImportFile = async (e: Event) => {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    try {
      const dashboard = await importDashboard(file);
      // Ensure timestamps
      if (!dashboard.createdAt) dashboard.createdAt = new Date().toISOString();
      if (!dashboard.updatedAt) dashboard.updatedAt = new Date().toISOString();
      if (!dashboard.timeWindow) dashboard.timeWindow = '1h';
      if (dashboard.refreshInterval == null) dashboard.refreshInterval = 30;
      setDashboards((prev) => [...prev, dashboard]);
    } catch (err) {
      console.error('Failed to import dashboard:', err);
    }
    // Reset the input so re-importing the same file works
    input.value = '';
  };

  // Build visible list: apply label filter, favorites-first sort, hidden filtering
  const visibleDashboards = useMemo(() => {
    let list = allDashboards;

    // Apply label filter (only affects preset dashboards; custom dashboards always show)
    if (labelFilter) {
      list = list.filter((d) => {
        const label = getDashboardLabel(d.id);
        return label === labelFilter || !isPresetId(d.id);
      });
    }

    // Separate hidden
    const hidden = list.filter((d) => prefs.hidden.includes(d.id));
    const visible = list.filter((d) => !prefs.hidden.includes(d.id));

    // Sort: favorites first, then alphabetical
    const sortFn = (a: Dashboard, b: Dashboard) => {
      const aFav = prefs.favorites.includes(a.id) ? 0 : 1;
      const bFav = prefs.favorites.includes(b.id) ? 0 : 1;
      if (aFav !== bFav) return aFav - bFav;
      return a.name.localeCompare(b.name);
    };

    return {
      visible: visible.sort(sortFn),
      hidden: hidden.sort(sortFn),
    };
  }, [allDashboards, labelFilter, prefs]);

  // List mode
  if (mode === 'list') {
    return (
      <div class="dashboards-view">
        <div class="dashboards-list-header">
          <h2 class="dashboards-title">Dashboards</h2>
          <div style={{ display: 'flex', gap: '6px' }}>
            <button
              class="btn"
              onClick={() => importInputRef.current?.click()}
            >
              Import
            </button>
            <input
              ref={importInputRef}
              type="file"
              accept=".json"
              style={{ display: 'none' }}
              onChange={handleImportFile}
            />
            <button class="btn btn-primary" onClick={handleCreateDashboard}>
              + New Dashboard
            </button>
          </div>
        </div>

        {/* Label filter bar */}
        <div class="dashboards-label-filters">
          <button
            class={`dashboards-label-btn ${labelFilter === null ? 'active' : ''}`}
            onClick={() => setLabelFilter(null)}
          >
            All
          </button>
          {DASHBOARD_LABELS.map((label) => (
            <button
              key={label}
              class={`dashboards-label-btn ${labelFilter === label ? 'active' : ''}`}
              style={
                labelFilter === label
                  ? { borderColor: LABEL_COLORS[label], color: LABEL_COLORS[label] }
                  : undefined
              }
              onClick={() => setLabelFilter(labelFilter === label ? null : label)}
            >
              {label.charAt(0).toUpperCase() + label.slice(1)}
            </button>
          ))}
        </div>

        <div class="dashboards-list">
          {visibleDashboards.visible.map((d) => {
            const label = getDashboardLabel(d.id);
            const isFav = prefs.favorites.includes(d.id);
            const isPreset = isPresetId(d.id);

            return (
              <div key={d.id} class="dashboards-list-item">
                <div
                  class="dashboards-list-item-content"
                  onClick={() => {
                    setActiveDashboardId(d.id);
                    setMode('dashboard');
                  }}
                >
                  <div class="dashboards-list-item-name-row">
                    <span class="dashboards-list-item-name">{d.name}</span>
                    {label && (
                      <span
                        class="dashboards-label-badge"
                        style={{ color: LABEL_COLORS[label], borderColor: LABEL_COLORS[label] }}
                      >
                        {label}
                      </span>
                    )}
                    {isPreset && (
                      <span class="dashboards-preset-badge">preset</span>
                    )}
                  </div>
                  <div class="dashboards-list-item-meta">
                    {d.widgets.length} widget{d.widgets.length !== 1 ? 's' : ''}
                    {' \u00B7 '}
                    {d.description || 'No description'}
                  </div>
                </div>
                <div class="dashboards-list-item-actions">
                  <button
                    class="dashboards-action-btn"
                    onClick={(e) => { e.stopPropagation(); exportDashboard(d); }}
                    title="Export as JSON"
                  >
                    Export
                  </button>
                  <button
                    class={`dashboards-action-btn ${isFav ? 'active' : ''}`}
                    onClick={(e) => { e.stopPropagation(); toggleFavorite(d.id); }}
                    title={isFav ? 'Unfavorite' : 'Favorite'}
                  >
                    {isFav ? '\u2605' : '\u2606'}
                  </button>
                  {isPreset ? (
                    <button
                      class="dashboards-action-btn"
                      onClick={(e) => { e.stopPropagation(); toggleHidden(d.id); }}
                      title="Hide"
                    >
                      {'\u{1F441}'}
                    </button>
                  ) : (
                    <button
                      class="dashboards-action-btn dashboards-delete-btn"
                      onClick={(e) => { e.stopPropagation(); handleDeleteDashboard(d.id); }}
                      title="Delete"
                    >
                      \u2715
                    </button>
                  )}
                </div>
              </div>
            );
          })}
          {visibleDashboards.visible.length === 0 && (
            <div class="dashboards-list-empty">
              No dashboards match the current filter.
            </div>
          )}
        </div>

        {/* Show hidden toggle */}
        {visibleDashboards.hidden.length > 0 && (
          <div class="dashboards-hidden-section">
            <button
              class="dashboards-show-hidden-btn"
              onClick={() => setShowHidden(!showHidden)}
            >
              {showHidden ? 'Hide' : 'Show'} {visibleDashboards.hidden.length} hidden dashboard{visibleDashboards.hidden.length !== 1 ? 's' : ''}
            </button>
            {showHidden && (
              <div class="dashboards-list dashboards-hidden-list">
                {visibleDashboards.hidden.map((d) => {
                  const label = getDashboardLabel(d.id);
                  return (
                    <div key={d.id} class="dashboards-list-item dashboards-list-item-hidden">
                      <div
                        class="dashboards-list-item-content"
                        onClick={() => {
                          setActiveDashboardId(d.id);
                          setMode('dashboard');
                        }}
                      >
                        <div class="dashboards-list-item-name-row">
                          <span class="dashboards-list-item-name">{d.name}</span>
                          {label && (
                            <span
                              class="dashboards-label-badge"
                              style={{ color: LABEL_COLORS[label], borderColor: LABEL_COLORS[label] }}
                            >
                              {label}
                            </span>
                          )}
                        </div>
                        <div class="dashboards-list-item-meta">
                          {d.description || 'No description'}
                        </div>
                      </div>
                      <div class="dashboards-list-item-actions">
                        <button
                          class="dashboards-action-btn"
                          onClick={(e) => { e.stopPropagation(); toggleHidden(d.id); }}
                          title="Unhide"
                        >
                          Show
                        </button>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}
      </div>
    );
  }

  // Dashboard mode
  return (
    <div class="dashboards-view">
      {/* Header toolbar */}
      <div class="dashboards-toolbar">
        <div class="dashboards-toolbar-left">
          <button
            class="dashboards-back-btn"
            onClick={() => setMode('list')}
            title="Back to list"
          >
            &larr;
          </button>
          <h2 class="dashboards-title">
            {activeDashboard?.name ?? 'Dashboard'}
          </h2>
        </div>

        <div class="dashboards-toolbar-right">
          {/* Time window selector */}
          <div class="dashboards-time-selector">
            {TIME_WINDOWS.map((tw) => (
              <button
                key={tw.value}
                class={`dashboards-time-btn ${timeWindow === tw.value ? 'active' : ''}`}
                onClick={() => setTimeWindow(tw.value)}
              >
                {tw.label}
              </button>
            ))}
          </div>

          {/* Refresh interval */}
          <select
            class="dashboards-refresh-select"
            value={refreshInterval}
            onChange={(e) =>
              setRefreshInterval(parseInt((e.target as HTMLSelectElement).value))
            }
          >
            {REFRESH_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>

          <button
            class="btn btn-primary"
            onClick={() => setShowNewWidget(true)}
          >
            + Add Widget
          </button>
        </div>
      </div>

      {/* Widget data providers (invisible -- just for fetching) */}
      {activeDashboard?.widgets.map((w) => (
        <WidgetDataProvider
          key={w.id}
          widget={w}
          timeWindow={timeWindow}
          refreshInterval={refreshInterval}
          onData={handleDataUpdate}
        >
          {null}
        </WidgetDataProvider>
      ))}

      {/* Grid */}
      {activeDashboard && (
        <DashboardGrid
          widgets={activeDashboard.widgets}
          queryResults={queryResults}
          onWidgetEdit={setEditingWidgetId}
          onWidgetResize={handleWidgetResize}
        />
      )}

      {/* Dashboard-level time range selector */}
      <TimeRangeSelector
        dataRange={{
          start: dashboardDataRange.start - (dashboardDataRange.end - dashboardDataRange.start),
          end: Math.max(dashboardDataRange.end, Date.now()),
        }}
        selectedRange={dashboardSelectedRange}
        onRangeChange={setDashboardSelectedRange}
        live={refreshInterval > 0}
        height={40}
      />

      {/* Widget editor overlay */}
      {(showNewWidget || editingWidgetId) && (
        <div class="widget-editor-overlay">
          <WidgetEditor
            widget={editingWidget}
            onSave={handleWidgetSave}
            onCancel={() => {
              setShowNewWidget(false);
              setEditingWidgetId(null);
            }}
            onDelete={editingWidgetId ? handleWidgetDelete : undefined}
          />
        </div>
      )}
    </div>
  );
}
