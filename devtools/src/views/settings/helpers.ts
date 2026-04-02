/**
 * Pure functions extracted from Settings views for testing.
 * Theme persistence, tab management, and account preferences.
 */

export type Theme = 'dark' | 'light';

export type SettingsTab = 'connections' | 'routing' | 'domains' | 'webhooks' | 'config' | 'appearance' | 'audit' | 'account';

export const SETTINGS_TABS: { id: SettingsTab; label: string }[] = [
  { id: 'connections', label: 'Connections' },
  { id: 'routing', label: 'Routing' },
  { id: 'domains', label: 'Domains' },
  { id: 'webhooks', label: 'Webhooks' },
  { id: 'config', label: 'Config' },
  { id: 'appearance', label: 'Appearance' },
  { id: 'audit', label: 'Audit' },
  { id: 'account', label: 'Account' },
];

const FONT_SIZE_KEY = 'cloudmock-font-size';
const THEME_KEY = 'cloudmock-theme';

export const DEFAULT_FONT_SIZE = 13;
export const MIN_FONT_SIZE = 11;
export const MAX_FONT_SIZE = 16;

/**
 * Get the saved theme from localStorage.
 * Returns 'dark' as default if nothing stored or value is invalid.
 */
export function getSavedTheme(): Theme {
  const stored = localStorage.getItem(THEME_KEY);
  return stored === 'light' ? 'light' : 'dark';
}

/**
 * Save theme to localStorage.
 */
export function saveTheme(theme: Theme): void {
  localStorage.setItem(THEME_KEY, theme);
}

/**
 * Get the saved font size from localStorage.
 * Returns DEFAULT_FONT_SIZE if nothing stored or value is invalid.
 */
export function getSavedFontSize(): number {
  const stored = localStorage.getItem(FONT_SIZE_KEY);
  if (!stored) return DEFAULT_FONT_SIZE;
  const parsed = parseInt(stored, 10);
  if (isNaN(parsed)) return DEFAULT_FONT_SIZE;
  return Math.max(MIN_FONT_SIZE, Math.min(MAX_FONT_SIZE, parsed));
}

/**
 * Save font size to localStorage.
 */
export function saveFontSize(size: number): void {
  localStorage.setItem(FONT_SIZE_KEY, String(size));
}

/**
 * Validate a tab ID is a valid settings tab.
 */
export function isValidTab(tab: string): tab is SettingsTab {
  return SETTINGS_TABS.some((t) => t.id === tab);
}

/**
 * Get the label for a settings tab.
 */
export function getTabLabel(tab: SettingsTab): string {
  return SETTINGS_TABS.find((t) => t.id === tab)?.label ?? tab;
}

/**
 * Determine connection mode from admin URL.
 */
export function isLocalMode(adminUrl: string | null | undefined): boolean {
  return !adminUrl || adminUrl.includes('localhost');
}

/**
 * Reset dashboard preferences to defaults.
 */
export function resetDashboardPreferences(): { hidden: string[]; favorites: string[] } {
  return { hidden: [], favorites: [] };
}
