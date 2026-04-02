import { useState, useEffect } from 'preact/hooks';
import { t } from '../../lib/i18n';
import { cacheGet } from '../../lib/cache';
import { EnvironmentSelector } from '../environment-selector/environment-selector';
import type { Environment } from '../../lib/environments';
import './status-bar.css';

/** Format a timestamp into a human-readable "Xs ago" / "Xm ago" string. */
function formatTimeAgo(ts: number): string {
  const delta = Math.max(0, Math.floor((Date.now() - ts) / 1000));
  if (delta < 5) return 'just now';
  if (delta < 60) return `${delta}s ago`;
  const m = Math.floor(delta / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  return `${h}h ago`;
}

interface StatusBarProps {
  connected?: boolean;
  endpoint?: string;
  region?: string;
  profile?: string;
  iamMode?: string;
  serviceCount?: number;
  lastUpdated?: number | null;
  onEnvironmentChange?: (env: Environment) => void;
}

export function StatusBar({
  connected = true,
  endpoint = 'localhost:4566',
  region = 'us-east-1',
  profile = 'default',
  iamMode = 'permissive',
  serviceCount = 0,
  lastUpdated,
  onEnvironmentChange,
}: StatusBarProps) {
  // Re-render every 5s so the "Xs ago" label stays current
  const [, setTick] = useState(0);
  useEffect(() => {
    if (!lastUpdated) return;
    const id = setInterval(() => setTick((t) => t + 1), 5000);
    return () => clearInterval(id);
  }, [lastUpdated]);

  // Detect offline-with-cache state: disconnected but have cached topology data
  const hasCachedData = !connected && cacheGet('topology:graph') !== null;

  const statusLabel = connected
    ? t('status.connected')
    : hasCachedData
      ? t('status.offline_cached')
      : t('status.disconnected');

  return (
    <div class="status-bar">
      <div class="status-bar-item">
        <span class={`status-bar-dot ${connected ? 'connected' : 'disconnected'}`} />
        {!connected && hasCachedData && (
          <span class="status-bar-offline-badge">{statusLabel}</span>
        )}
        {onEnvironmentChange ? (
          <EnvironmentSelector onEnvironmentChange={onEnvironmentChange} />
        ) : (
          <span class="status-bar-value">Local — {endpoint}</span>
        )}
      </div>

      <div class="status-bar-separator" />

      <div class="status-bar-item">
        <span class="status-bar-label">region</span>
        <span class="status-bar-value">{region}</span>
      </div>

      <div class="status-bar-separator" />

      <div class="status-bar-item">
        <span class="status-bar-label">profile</span>
        <span class="status-bar-value">{profile}</span>
      </div>

      <div class="status-bar-separator" />

      <div class="status-bar-item">
        <span class="status-bar-label">iam</span>
        <span class="status-bar-value">{iamMode}</span>
      </div>

      <div class="status-bar-spacer" />

      {lastUpdated && (
        <>
          <div class="status-bar-item">
            <span class="status-bar-label">updated</span>
            <span class="status-bar-value">{formatTimeAgo(lastUpdated)}</span>
          </div>
          <div class="status-bar-separator" />
        </>
      )}

      <div class="status-bar-item">
        <span class="status-bar-value">{serviceCount} services</span>
      </div>
    </div>
  );
}
