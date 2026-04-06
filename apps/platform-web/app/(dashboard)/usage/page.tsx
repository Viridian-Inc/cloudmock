// TODO: Replace mock data with real API call for usage stats

const FREE_LIMIT = 1000;

const MOCK_TOTAL_REQUESTS = 1842;

const MOCK_APP_BREAKDOWN = [
  { id: "app_1", name: "my-dev-env", requests: 1242 },
  { id: "app_2", name: "staging-backend", requests: 347 },
  { id: "app_3", name: "prod-mirror", requests: 253 },
];

function calcEstimatedCost(requests: number): string {
  const billable = Math.max(0, requests - FREE_LIMIT);
  const cost = (billable / 10000) * 0.5;
  return `$${cost.toFixed(2)}`;
}

export default function UsagePage() {
  const totalRequests = MOCK_TOTAL_REQUESTS;
  const progressPct = Math.min(100, (totalRequests / FREE_LIMIT) * 100);
  const estimatedCost = calcEstimatedCost(totalRequests);

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-100">Usage</h1>
        <p className="mt-1 text-sm text-gray-400">
          Your request usage this billing period
        </p>
      </div>

      {/* Summary cards */}
      <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-2">
        {/* Requests with progress bar */}
        <div className="rounded-lg border border-gray-800 bg-gray-900 p-6">
          <p className="text-sm font-medium text-gray-400">
            Requests This Month
          </p>
          <p className="mt-2 text-3xl font-bold text-gray-100">
            {totalRequests.toLocaleString()}
          </p>
          <p className="mt-1 text-sm text-gray-500">
            Free limit: {FREE_LIMIT.toLocaleString()} / month
          </p>
          <div className="mt-4 h-2 w-full overflow-hidden rounded-full bg-gray-800">
            <div
              className={`h-full rounded-full transition-all ${
                progressPct >= 100 ? "bg-red-500" : "bg-[#52b788]"
              }`}
              style={{ width: `${progressPct}%` }}
            />
          </div>
          <p className="mt-1.5 text-xs text-gray-500">
            {progressPct.toFixed(0)}% of free tier used
          </p>
        </div>

        {/* Estimated cost */}
        <div className="rounded-lg border border-gray-800 bg-gray-900 p-6">
          <p className="text-sm font-medium text-gray-400">Estimated Cost</p>
          <p className="mt-2 text-3xl font-bold text-gray-100">
            {estimatedCost}
          </p>
          <p className="mt-1 text-sm text-gray-500">
            First 1,000 requests free, then $0.50 per 10,000
          </p>
        </div>
      </div>

      {/* Per-app breakdown */}
      <div className="rounded-lg border border-gray-800 bg-gray-900 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-800">
          <h2 className="text-sm font-medium text-gray-400">
            Per-App Breakdown
          </h2>
        </div>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-800">
              <th className="px-6 py-3 text-left font-medium text-gray-400">
                App
              </th>
              <th className="px-6 py-3 text-right font-medium text-gray-400">
                Requests
              </th>
              <th className="px-6 py-3 text-right font-medium text-gray-400">
                Share
              </th>
              <th className="px-6 py-3 text-left font-medium text-gray-400 w-48">
                Usage
              </th>
            </tr>
          </thead>
          <tbody>
            {MOCK_APP_BREAKDOWN.map((app) => {
              const pct = ((app.requests / totalRequests) * 100).toFixed(1);
              return (
                <tr
                  key={app.id}
                  className="border-b border-gray-800 last:border-0"
                >
                  <td className="px-6 py-3 font-medium text-gray-200">
                    {app.name}
                  </td>
                  <td className="px-6 py-3 text-right text-gray-300">
                    {app.requests.toLocaleString()}
                  </td>
                  <td className="px-6 py-3 text-right text-gray-400">
                    {pct}%
                  </td>
                  <td className="px-6 py-3">
                    <div className="h-1.5 w-full overflow-hidden rounded-full bg-gray-800">
                      <div
                        className="h-full rounded-full bg-[#52b788]/70"
                        style={{ width: `${pct}%` }}
                      />
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
