import { useState, useEffect } from 'preact/hooks';
import { getTipsEnabled, setTipsEnabled, resetDismissedTips, onTipsChange } from '../../lib/tips';

const STORAGE_KEY = 'cloudmock-font-size';
const THEME_STORAGE_KEY = 'cloudmock-theme';
const DEFAULT_SIZE = 13;
const MIN_SIZE = 11;
const MAX_SIZE = 16;

type Theme = 'dark' | 'light';

export function applyTheme(theme: Theme) {
  if (theme === 'light') {
    document.documentElement.setAttribute('data-theme', 'light');
  } else {
    document.documentElement.removeAttribute('data-theme');
  }
}

export function getSavedTheme(): Theme {
  const stored = localStorage.getItem(THEME_STORAGE_KEY);
  return stored === 'light' ? 'light' : 'dark';
}

export function Appearance() {
  const [fontSize, setFontSize] = useState(() => {
    const stored = localStorage.getItem(STORAGE_KEY);
    return stored ? parseInt(stored, 10) : DEFAULT_SIZE;
  });

  const [theme, setTheme] = useState<Theme>(getSavedTheme);
  const [tipsEnabled, setTipsEnabledState] = useState(getTipsEnabled);

  // Re-sync if another tab or the TipBanner dismisses/changes tip state.
  useEffect(() => onTipsChange(() => setTipsEnabledState(getTipsEnabled())), []);

  useEffect(() => {
    document.documentElement.style.fontSize = `${fontSize}px`;
    localStorage.setItem(STORAGE_KEY, String(fontSize));
  }, [fontSize]);

  useEffect(() => {
    applyTheme(theme);
    localStorage.setItem(THEME_STORAGE_KEY, theme);
  }, [theme]);

  return (
    <div class="settings-section">
      <h3 class="settings-section-title">Appearance</h3>

      <div class="settings-field">
        <label class="settings-label">Theme</label>
        <div class="appearance-theme-toggle">
          <button
            class={`appearance-theme-option ${theme === 'dark' ? 'appearance-theme-option-active' : ''}`}
            onClick={() => setTheme('dark')}
          >
            Dark
          </button>
          <button
            class={`appearance-theme-option ${theme === 'light' ? 'appearance-theme-option-active' : ''}`}
            onClick={() => setTheme('light')}
          >
            Light
          </button>
        </div>
        <p class="settings-section-desc">
          Choose between dark and light color themes.
        </p>
      </div>

      <div class="settings-field">
        <label class="settings-label">
          Font Size: {fontSize}px
        </label>
        <div class="appearance-slider-row">
          <span class="appearance-slider-label">{MIN_SIZE}px</span>
          <input
            type="range"
            class="appearance-slider"
            min={MIN_SIZE}
            max={MAX_SIZE}
            step={1}
            value={fontSize}
            onInput={(e) =>
              setFontSize(parseInt((e.target as HTMLInputElement).value, 10))
            }
          />
          <span class="appearance-slider-label">{MAX_SIZE}px</span>
        </div>
        <p class="settings-section-desc">
          Adjusts the base font size for the application. Default is {DEFAULT_SIZE}px.
        </p>
      </div>

      <div class="settings-field">
        <label class="settings-label">UI Tips</label>
        <div class="appearance-theme-toggle">
          <button
            class={`appearance-theme-option ${tipsEnabled ? 'appearance-theme-option-active' : ''}`}
            onClick={() => {
              setTipsEnabled(true);
              setTipsEnabledState(true);
            }}
          >
            Show
          </button>
          <button
            class={`appearance-theme-option ${!tipsEnabled ? 'appearance-theme-option-active' : ''}`}
            onClick={() => {
              setTipsEnabled(false);
              setTipsEnabledState(false);
            }}
          >
            Hide
          </button>
          <button
            class="appearance-theme-option"
            onClick={() => resetDismissedTips()}
            title="Show previously dismissed tips again"
          >
            Reset dismissed
          </button>
        </div>
        <p class="settings-section-desc">
          Small dismissible hints in the header that teach non-obvious shortcuts.
        </p>
      </div>
    </div>
  );
}
