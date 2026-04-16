// Lightweight UI tips: small dismissible hints that teach non-obvious
// shortcuts. Each tip has a stable id; once a tip is dismissed (or tips are
// disabled globally), it stops showing. State lives in localStorage so
// hints don't resurface across page reloads.

const GLOBAL_KEY = 'cloudmock:tips:enabled';
const DISMISSED_KEY = 'cloudmock:tips:dismissed';
const CHANGE_EVENT = 'cloudmock:tips:change';

export interface Tip {
  id: string;
  text: string;
}

export const TIPS: Tip[] = [
  {
    id: 'peek-pane-alt-click',
    text: 'Alt/Option-click a sidebar item to open it as a peek pane alongside the current view. Middle-click works too. Plain click navigates.',
  },
];

export function getTipsEnabled(): boolean {
  // Default on. Only disabled if the user has explicitly turned them off.
  return localStorage.getItem(GLOBAL_KEY) !== 'false';
}

export function setTipsEnabled(enabled: boolean): void {
  localStorage.setItem(GLOBAL_KEY, enabled ? 'true' : 'false');
  document.dispatchEvent(new CustomEvent(CHANGE_EVENT));
}

function readDismissed(): Set<string> {
  try {
    const raw = localStorage.getItem(DISMISSED_KEY);
    if (!raw) return new Set();
    const arr = JSON.parse(raw);
    return new Set(Array.isArray(arr) ? arr : []);
  } catch {
    return new Set();
  }
}

export function isTipDismissed(id: string): boolean {
  return readDismissed().has(id);
}

export function dismissTip(id: string): void {
  const set = readDismissed();
  set.add(id);
  localStorage.setItem(DISMISSED_KEY, JSON.stringify([...set]));
  document.dispatchEvent(new CustomEvent(CHANGE_EVENT));
}

export function resetDismissedTips(): void {
  localStorage.removeItem(DISMISSED_KEY);
  document.dispatchEvent(new CustomEvent(CHANGE_EVENT));
}

/** Subscribe to tip-state changes (enable/disable, dismissals). */
export function onTipsChange(handler: () => void): () => void {
  document.addEventListener(CHANGE_EVENT, handler);
  return () => document.removeEventListener(CHANGE_EVENT, handler);
}

/** Get the next tip to show, or null when nothing should be shown. */
export function nextTip(): Tip | null {
  if (!getTipsEnabled()) return null;
  const dismissed = readDismissed();
  return TIPS.find((t) => !dismissed.has(t.id)) ?? null;
}
