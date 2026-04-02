import type { Dashboard } from './types';
import { api, getAdminBase } from '../../lib/api';

const STORAGE_KEY = 'neureaux:dashboards';
const PREFS_KEY = 'neureaux:dashboard-prefs';

export interface DashboardPreferences {
  hidden: string[];    // dashboard IDs the user has hidden
  favorites: string[]; // dashboard IDs the user has favorited
}

export function loadDashboards(): Dashboard[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) return JSON.parse(raw);
  } catch {
    // Corrupted data -- fall back to empty
  }
  return [];
}

export function saveDashboards(dashboards: Dashboard[]): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(dashboards));
}

export function loadDashboardPreferences(): DashboardPreferences {
  try {
    const raw = localStorage.getItem(PREFS_KEY);
    if (raw) return JSON.parse(raw);
  } catch {
    // Corrupted data -- fall back to defaults
  }
  return { hidden: [], favorites: [] };
}

export function saveDashboardPreferences(prefs: DashboardPreferences): void {
  localStorage.setItem(PREFS_KEY, JSON.stringify(prefs));
}

// ===== API persistence (backend-first, localStorage fallback) =====

export async function loadDashboardsFromAPI(): Promise<Dashboard[] | null> {
  try {
    const data = await api<Dashboard[]>('/api/dashboards');
    return Array.isArray(data) ? data : null;
  } catch {
    return null; // API not available, use localStorage
  }
}

export async function saveDashboardToAPI(dashboard: Dashboard): Promise<boolean> {
  try {
    const base = getAdminBase();
    const res = await fetch(`${base}/api/dashboards`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(dashboard),
    });
    return res.ok;
  } catch {
    return false;
  }
}

export async function saveDashboardsToAPI(dashboards: Dashboard[]): Promise<boolean> {
  try {
    const base = getAdminBase();
    const res = await fetch(`${base}/api/dashboards`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(dashboards),
    });
    return res.ok;
  } catch {
    return false;
  }
}

// ===== Export / Import =====

export function exportDashboard(dashboard: Dashboard): void {
  const json = JSON.stringify(dashboard, null, 2);
  const blob = new Blob([json], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `${dashboard.name.replace(/\s+/g, '-').toLowerCase()}.json`;
  a.click();
  URL.revokeObjectURL(url);
}

export function importDashboard(file: File): Promise<Dashboard> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      try {
        const data = JSON.parse(reader.result as string);
        if (!data.name || !Array.isArray(data.widgets)) {
          reject(new Error('Invalid dashboard format'));
          return;
        }
        // Assign new ID to avoid conflicts
        data.id = `imported-${Date.now()}`;
        resolve(data as Dashboard);
      } catch (e) {
        reject(e);
      }
    };
    reader.onerror = () => reject(new Error('Failed to read file'));
    reader.readAsText(file);
  });
}
