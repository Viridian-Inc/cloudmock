import { render } from 'preact';
import { App } from './app';
import { applyTheme, getSavedTheme } from './views/settings/appearance';

// Apply saved theme before first render to avoid flash
applyTheme(getSavedTheme());

render(<App />, document.getElementById('app')!);
