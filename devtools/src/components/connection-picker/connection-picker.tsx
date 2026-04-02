import { useState } from 'preact/hooks';
import './connection-picker.css';

type ConnectionMode = 'local' | 'hosted' | 'custom';

const STORAGE_KEY = 'cloudmock-connection-mode';

function getStoredMode(): ConnectionMode {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === 'local' || stored === 'hosted' || stored === 'custom') {
      return stored;
    }
  } catch {
    // localStorage unavailable
  }
  return 'local';
}

function storeMode(mode: ConnectionMode) {
  try {
    localStorage.setItem(STORAGE_KEY, mode);
  } catch {
    // localStorage unavailable
  }
}

interface ConnectionPickerProps {
  onConnect: (adminUrl: string, gatewayUrl: string) => void;
  onClose: () => void;
}

export function ConnectionPicker({ onConnect, onClose }: ConnectionPickerProps) {
  const [showHosted, setShowHosted] = useState(false);
  const [showCustom, setShowCustom] = useState(false);
  const [customUrl, setCustomUrl] = useState('');

  const handleLocalConnect = () => {
    storeMode('local');
    onConnect('http://localhost:4599', 'http://localhost:4566');
  };

  const handleHostedClick = () => {
    setShowHosted(!showHosted);
    setShowCustom(false);
  };

  const handleHostedSignIn = () => {
    storeMode('hosted');
    // After Clerk sign-in completes (future), the org endpoint URL will be
    // stored and used for connection. For now, redirect to sign-in page.
    window.open('https://cloudmock.io/sign-in', '_blank');
  };

  const handleCustomConnect = () => {
    if (!customUrl.trim()) return;
    const base = customUrl.replace(/\/+$/, '');
    // Don't assume port relationship — auto-detect for known port, otherwise use same URL
    const adminUrl = base.includes(':4566') ? base.replace(':4566', ':4599') : base;
    storeMode('custom');
    onConnect(adminUrl, base);
  };

  return (
    <div class="connection-picker-overlay" onClick={onClose}>
      <div class="connection-picker" onClick={(e) => e.stopPropagation()}>
        <div class="connection-picker-title">cloudmock</div>
        <div class="connection-picker-subtitle">
          Connect to a cloudmock instance
        </div>

        <div class="connection-picker-options">
          <button class="connection-option" onClick={handleLocalConnect}>
            <span class="connection-option-icon">💻</span>
            <div class="connection-option-content">
              <div class="connection-option-title">Local Instance</div>
              <div class="connection-option-desc">
                Start cloudmock on this machine
              </div>
            </div>
            <div class="connection-option-tags">
              <span class="connection-tag green">Free</span>
              <span class="connection-tag">No account needed</span>
            </div>
          </button>

          <button class="connection-option" onClick={handleHostedClick}>
            <span class="connection-option-icon">☁️</span>
            <div class="connection-option-content">
              <div class="connection-option-title">cloudmock.io</div>
              <div class="connection-option-desc">
                Connect to a hosted instance
              </div>
            </div>
            <div class="connection-option-tags">
              <span class="connection-tag">Pro / Team</span>
            </div>
          </button>

          {showHosted && (
            <div class="connection-hosted-panel">
              <p class="connection-hosted-desc">
                Sign in with your cloudmock.io account to connect to your
                organization's hosted endpoint.
              </p>
              <button
                class="btn btn-primary connection-hosted-signin"
                onClick={handleHostedSignIn}
              >
                Sign in with Clerk
              </button>
            </div>
          )}

          <button
            class="connection-option"
            onClick={() => {
              setShowCustom(!showCustom);
              setShowHosted(false);
            }}
          >
            <span class="connection-option-icon">🔗</span>
            <div class="connection-option-content">
              <div class="connection-option-title">Custom Endpoint</div>
              <div class="connection-option-desc">
                Connect to any cloudmock instance by URL
              </div>
            </div>
          </button>

          {showCustom && (
            <div class="connection-custom-input">
              <input
                class="input"
                type="text"
                placeholder="http://192.168.1.100:4566"
                value={customUrl}
                onInput={(e) =>
                  setCustomUrl((e.target as HTMLInputElement).value)
                }
                onKeyDown={(e) => {
                  if (e.key === 'Enter') handleCustomConnect();
                }}
              />
              <button
                class="btn btn-primary"
                onClick={handleCustomConnect}
                disabled={!customUrl.trim()}
              >
                Connect
              </button>
            </div>
          )}
        </div>

        <div class="connection-picker-version">v0.1.0</div>
      </div>
    </div>
  );
}
