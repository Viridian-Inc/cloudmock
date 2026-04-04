import './platform-usage.css';

const FREE_LIMIT = 1000;
const PRICE_PER_10K = 0.5;

// TODO: Replace with API call to GET /api/platform/usage
const MOCK_TOTAL = 44094;

const MOCK_APP_BREAKDOWN = [
  { id: '1', name: 'staging', request_count: 42891 },
  { id: '2', name: 'ci-tests', request_count: 1203 },
];

// Generate 30 days of mock daily request counts ending today
function mockDailyData(): { day: string; count: number }[] {
  const days: { day: string; count: number }[] = [];
  const now = new Date('2026-04-03');
  for (let i = 29; i >= 0; i--) {
    const d = new Date(now);
    d.setDate(now.getDate() - i);
    const label = `${d.getMonth() + 1}/${d.getDate()}`;
    // Mock: ramp up toward the end of the month
    const base = 800 + Math.sin(i * 0.4) * 400;
    days.push({ day: label, count: Math.max(0, Math.round(base + Math.random() * 500)) });
  }
  return days;
}

const DAILY_DATA = mockDailyData();
const MAX_DAY = Math.max(...DAILY_DATA.map((d) => d.count));

function formatCost(requests: number): string {
  const billable = Math.max(0, requests - FREE_LIMIT);
  const cost = (billable / 10000) * PRICE_PER_10K;
  return cost === 0 ? '$0.00' : `$${cost.toFixed(2)}`;
}

export function PlatformUsageView() {
  const total = MOCK_TOTAL;
  const pct = Math.min(100, (total / FREE_LIMIT) * 100);

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
              {total.toLocaleString()} / {FREE_LIMIT.toLocaleString()} free
            </div>
          </div>
        </div>

        <div class="usage-card">
          <div class="usage-card-label">Estimated Cost</div>
          <div class="usage-card-value usage-cost">{formatCost(total)}</div>
          <div class="usage-cost-note">
            First {FREE_LIMIT.toLocaleString()} req/mo free · ${PRICE_PER_10K.toFixed(2)} per 10k after
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
              {MOCK_APP_BREAKDOWN.map((app) => (
                <tr key={app.id}>
                  <td class="key-name">{app.name}</td>
                  <td>{app.request_count.toLocaleString()}</td>
                  <td>
                    <div class="usage-share-bar">
                      <div
                        class="usage-share-fill"
                        style={{ width: `${Math.round((app.request_count / total) * 100)}%` }}
                      />
                      <span class="usage-share-pct">
                        {Math.round((app.request_count / total) * 100)}%
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
          {DAILY_DATA.map((d) => (
            <div class="usage-bar-wrap" key={d.day} title={`${d.day}: ${d.count.toLocaleString()}`}>
              <div
                class="usage-bar"
                style={{ height: `${Math.round((d.count / MAX_DAY) * 100)}%` }}
              />
              <div class="usage-bar-label">{d.day.split('/')[1]}</div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
