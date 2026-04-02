import { useEffect } from 'preact/hooks';

export type ShortcutAction =
  | 'activity'
  | 'topology'
  | 'services'
  | 'traces'
  | 'metrics'
  | 'slos'
  | 'incidents'
  | 'profiler'
  | 'chaos'
  | 'settings'
  | 'search'
  | 'live'
  | 'deselect';

const KEY_MAP: Record<string, ShortcutAction> = {
  '1': 'activity',
  '2': 'topology',
  '3': 'services',
  '4': 'traces',
  '5': 'metrics',
  '6': 'slos',
  '7': 'incidents',
  '8': 'profiler',
  '9': 'chaos',
  '0': 'settings',
};

const LETTER_MAP: Record<string, ShortcutAction> = {
  k: 'search',
  l: 'live',
};

/**
 * Global keyboard shortcut handler.
 *
 * Cmd+1-9/0 switches views, Cmd+K focuses search, Cmd+L snaps to live mode,
 * Escape deselects the current node/event.
 */
export function useKeyboardShortcuts(onAction: (action: ShortcutAction) => void): void {
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      // Use metaKey on Mac, ctrlKey elsewhere
      const mod = e.metaKey || e.ctrlKey;

      if (e.key === 'Escape') {
        onAction('deselect');
        return;
      }

      if (!mod) return;

      const numberAction = KEY_MAP[e.key];
      if (numberAction) {
        e.preventDefault();
        onAction(numberAction);
        return;
      }

      const letterAction = LETTER_MAP[e.key.toLowerCase()];
      if (letterAction) {
        e.preventDefault();
        onAction(letterAction);
        return;
      }
    }

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onAction]);
}
