import { useState } from 'preact/hooks';
import { useAuth } from '../../lib/auth';
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
  const { auth, setToken } = useAuth();
  const [showHosted, setShowHosted] = useState(false);
  const [showCustom, setShowCustom] = useState(false);
  const [customUrl, setCustomUrl] = useState('');
  const [orgSlug, setOrgSlug] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [hostedError, setHostedError] = useState('');
  const [connecting, setConnecting] = useState(false);

  const handleLocalConnect = () => {
    storeMode('local');
    onConnect('http://localhost:4599', 'http://localhost:4566');
  };

  const handleHostedClick = () => {
    setShowHosted(!showHosted);
    setShowCustom(false);
    setHostedError('');
  };

  const handleHostedConnect = async () => {
    const slug = orgSlug.trim().toLowerCase().replace(/[^a-z0-9-]/g, '');
    if (!slug) {
      setHostedError('Enter your organization slug');
      return;
    }
    if (!apiKey.trim()) {
      setHostedError('Enter your API key');
      return;
    }

    setConnecting(true);
    setHostedError('');

    const adminUrl = `https://${slug}.cloudmock.app`;
    const gatewayUrl = `https://${slug}.cloudmock.app`;

    try {
      // Verify the endpoint is reachable with the API key
      const res = await fetch(`${adminUrl}/api/health`, {
        headers: { 'Authorization': `Bearer ${apiKey.trim()}` },
      });
      if (!res.ok && res.status === 401) {
        setHostedError('Invalid API key');
        setConnecting(false);
        return;
      }
      // Connection works — store token and connect
      setToken(apiKey.trim(), { org_slug: slug });
      storeMode('hosted');
      onConnect(adminUrl, gatewayUrl);
    } catch {
      setHostedError(`Cannot reach ${adminUrl} — is the instance running?`);
    } finally {
      setConnecting(false);
    }
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
              <div class="connection-option-title">CloudMock Cloud</div>
              <div class="connection-option-desc">
                Connect to a hosted instance
              </div>
            </div>
            <div class="connection-option-tags">
              <span class="connection-tag">Pay per use</span>
              <span class="connection-tag">1K free/mo</span>
            </div>
          </button>

          {showHosted && (
            <div class="connection-hosted-panel">
              <p class="connection-hosted-desc">
                Enter your organization slug and API key.
                Get these from{' '}
                <a href="https://app.cloudmock.app" target="_blank" rel="noopener">
                  app.cloudmock.app
                </a>
              </p>
              <input
                class="input"
                type="text"
                placeholder="org slug (e.g. acme)"
                value={orgSlug}
                onInput={(e) => setOrgSlug((e.target as HTMLInputElement).value)}
              />
              <input
                class="input"
                type="password"
                placeholder="API key (cmk_...)"
                value={apiKey}
                onInput={(e) => setApiKey((e.target as HTMLInputElement).value)}
                onKeyDown={(e) => { if (e.key === 'Enter') handleHostedConnect(); }}
              />
              {hostedError && (
                <p class="connection-hosted-error">{hostedError}</p>
              )}
              <button
                class="btn btn-primary connection-hosted-signin"
                onClick={handleHostedConnect}
                disabled={connecting}
              >
                {connecting ? 'Connecting...' : 'Connect'}
              </button>
              <p class="connection-hosted-signup">
                Don't have an account?{' '}
                <a href="https://app.cloudmock.app/sign-up" target="_blank" rel="noopener">
                  Sign up free
                </a>
              </p>
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

        <div class="connection-picker-version">v1.0.0</div>
      </div>
    </div>
  );
}
