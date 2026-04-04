import { useState, useEffect } from 'preact/hooks';
import { api } from '../../lib/api';
import './platform-usage.css';

interface AppUsage {
  name: string;
  requests: number;
}

interface DayUsage {
  day: string;
  count: number;
}

interface UsageData {
  total: number;
  free_limit: number;
  cost: number;
  apps: AppUsage[];
  daily: DayUsage[];
}

function formatCost(cost: number): string {
  return cost === 0 ? '$0.00' : `$${cost.toFixed(2)}`;
}

export function PlatformUsageView() {
  const [data, setData] = useState<UsageData | null>(null);

  useEffect(() => {
    api<UsageData>('/api/platform/usage').then(setData).catch(console.error);
  }, []);

  if (!data) {
    return (
      <div class="platform-view">
        <div class="platform-header">
          <div class="platform-header-left">
            <h2 class="platform-title">Usage</h2>
            <p class="platform-subtitle">Request usage for this billing period</p>
          </div>
        </div>
        <div class="platform-loading">Loading...</div>
      </div>
    );
  }

  const { total, free_limit, cost, apps, daily } = data;
  const pct = Math.min(100, (total / free_limit) * 100);
  const maxDay = daily.length > 0 ? Math.max(...daily.map((d) => d.count)) : 1;

  return (
    <div class="platform-view">
      <div class="platform-header">
        <div class="platform-header-left">
          <h2 class="platform-title">Usage</h2>
          <p class="platform-subtitle">Request usage for this billing period</p>
        </div>
      </div>

      {/* Summary cards */}
      <div class="usage-summary-row">
        <div class="usage-card">
          <div class="usage-card-label">Total Requests</div>
          <div class="usage-card-value">{total.toLocaleString()}</div>
          <div class="usage-progress-wrap">
            <div class="usage-progress-track">
              <div
                class="usage-progress-bar"
                style={{ width: `${pct}%`, background: pct >= 100 ? 'var(--error)' : 'var(--brand-green)' }}
              />
            </div>
            <div class="usage-progress-label">
              {total.toLocaleString()} / {free_limit.toLocaleString()} free
            </div>
          </div>
        </div>

        <div class="usage-card">
          <div class="usage-card-label">Estimated Cost</div>
          <div class="usage-card-value usage-cost">{formatCost(cost)}</div>
          <div class="usage-cost-note">
            First {free_limit.toLocaleString()} req/mo free · $0.05 per 10k after
          </div>
        </div>
      </div>

      {/* Per-app breakdown */}
      <div class="usage-section">
        <h3 class="usage-section-title">Per-App Breakdown</h3>
        <div class="platform-table-wrap">
          <table class="platform-table">
            <thead>
              <tr>
                <th>App</th>
                <th>Requests</th>
                <th>Share</th>
              </tr>
            </thead>
            <tbody>
              {apps.map((app) => (
                <tr key={app.name}>
                  <td class="key-name">{app.name}</td>
                  <td>{app.requests.toLocaleString()}</td>
                  <td>
                    <div class="usage-share-bar">
                      <div
                        class="usage-share-fill"
                        style={{ width: `${total > 0 ? Math.round((app.requests / total) * 100) : 0}%` }}
                      />
                      <span class="usage-share-pct">
                        {total > 0 ? Math.round((app.requests / total) * 100) : 0}%
                      </span>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Daily bar chart */}
      <div class="usage-section">
        <h3 class="usage-section-title">Daily Requests (Last 30 Days)</h3>
        <div class="usage-chart">
          {daily.map((d) => (
            <div class="usage-bar-wrap" key={d.day} title={`${d.day}: ${d.count.toLocaleString()}`}>
              <div
                class="usage-bar"
                style={{ height: `${maxDay > 0 ? Math.round((d.count / maxDay) * 100) : 0}%` }}
              />
              <div class="usage-bar-label">{d.day.split('/')[1]}</div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
