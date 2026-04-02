import { render } from 'preact';
import { App } from './app';
import { applyTheme, getSavedTheme } from './views/settings/appearance';

// Clear stale caches from previous versions to prevent ghost data.
// The build ID changes on each Vite build, so mismatches trigger a purge.
const BUILD_ID = '__BUILD_' + Date.now().toString(36) + '__';
const PREV_BUILD = localStorage.getItem('neureaux:build-id');
if (PREV_BUILD !== BUILD_ID) {
  // Purge topology and layout caches — keeps theme/appearance preferences
  for (const key of Object.keys(localStorage)) {
    if (key.startsWith('neureaux:cache:') || key === 'neureaux-devtools:layouts') {
      localStorage.removeItem(key);
    }
  }
  localStorage.setItem('neureaux:build-id', BUILD_ID);
}

// Apply saved theme before first render to avoid flash
applyTheme(getSavedTheme());

render(<App />, document.getElementById('app')!);
