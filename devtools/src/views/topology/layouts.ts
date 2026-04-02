export interface SavedLayout {
  name: string;
  createdAt: string;
  pinnedPositions: Record<string, { x: number; y: number }>;
  pan: { x: number; y: number };
  scale: number;
  collapsed: boolean;
  isDefault: boolean;
}

const STORAGE_KEY = 'neureaux-devtools:layouts';

export function loadLayouts(): SavedLayout[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed : [];
  } catch (e) {
    console.warn('[Layouts] Failed to parse saved layouts:', e);
    return [];
  }
}

function persistLayouts(layouts: SavedLayout[]): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(layouts));
}

export function saveLayout(layout: SavedLayout): void {
  const layouts = loadLayouts();
  const idx = layouts.findIndex((l) => l.name === layout.name);
  if (idx >= 0) {
    layouts[idx] = layout;
  } else {
    layouts.push(layout);
  }
  persistLayouts(layouts);
}

export function deleteLayout(name: string): void {
  const layouts = loadLayouts().filter((l) => l.name !== name);
  persistLayouts(layouts);
}

export function setDefaultLayout(name: string): void {
  const layouts = loadLayouts().map((l) => ({
    ...l,
    isDefault: l.name === name,
  }));
  persistLayouts(layouts);
}

export function getDefaultLayout(): SavedLayout | null {
  const layouts = loadLayouts();
  return layouts.find((l) => l.isDefault) ?? null;
}

export function exportLayout(layout: SavedLayout): string {
  return JSON.stringify(layout, null, 2);
}

export function importLayout(json: string): SavedLayout | null {
  try {
    const parsed = JSON.parse(json);
    if (
      typeof parsed === 'object' &&
      parsed !== null &&
      typeof parsed.name === 'string' &&
      typeof parsed.pinnedPositions === 'object' &&
      typeof parsed.pan === 'object' &&
      typeof parsed.scale === 'number'
    ) {
      return {
        name: parsed.name,
        createdAt: parsed.createdAt || new Date().toISOString(),
        pinnedPositions: parsed.pinnedPositions,
        pan: parsed.pan,
        scale: parsed.scale,
        collapsed: parsed.collapsed ?? false,
        isDefault: parsed.isDefault ?? false,
      };
    }
  } catch (e) { console.warn('[Layouts] Failed to parse imported layout:', e); }
  return null;
}
