import { auth } from "@clerk/nextjs/server";
import { redirect } from "next/navigation";
import { getRole } from "@/lib/roles";
import StatCard from "@/components/StatCard";

// TODO: Replace mock data with real API call for admin stats

const MOCK_STATS = {
  monthlyRequests: 24_381,
  estimatedCost: "$0.72",
  activeApps: 3,
  teamMembers: 5,
};

const MOCK_USAGE_BARS = Array.from({ length: 30 }, (_, i) => {
  const base = 400 + Math.floor(Math.sin(i * 0.5) * 300 + Math.random() * 200);
  return { day: i + 1, count: base };
});
const maxBar = Math.max(...MOCK_USAGE_BARS.map((b) => b.count));

const MOCK_AUDIT = [
  {
    id: 1,
    actor: "alice@example.com",
    action: "app.create",
    time: "2026-04-03T09:12:00Z",
  },
  {
    id: 2,
    actor: "bob@example.com",
    action: "key.revoke",
    time: "2026-04-03T08:55:00Z",
  },
  {
    id: 3,
    actor: "alice@example.com",
    action: "member.invite",
    time: "2026-04-02T17:30:00Z",
  },
  {
    id: 4,
    actor: "carol@example.com",
    action: "app.delete",
    time: "2026-04-02T14:10:00Z",
  },
  {
    id: 5,
    actor: "bob@example.com",
    action: "key.create",
    time: "2026-04-02T11:00:00Z",
  },
];

export default async function OverviewPage() {
  const { orgRole } = await auth();
  const role = getRole(orgRole ?? undefined);

  if (role !== "admin") {
    redirect("/apps");
  }

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-100">Overview</h1>
        <p className="mt-1 text-sm text-gray-400">
          Organization-wide usage and activity
        </p>
      </div>

      {/* Stat cards */}
      <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label="Monthly Requests"
          value={MOCK_STATS.monthlyRequests.toLocaleString()}
          sublabel="This billing period"
        />
        <StatCard
          label="Estimated Cost"
          value={MOCK_STATS.estimatedCost}
          sublabel="Based on usage"
        />
        <StatCard
          label="Active Apps"
          value={MOCK_STATS.activeApps}
          sublabel="Running environments"
        />
        <StatCard
          label="Team Members"
          value={MOCK_STATS.teamMembers}
          sublabel="Across all roles"
        />
      </div>

      {/* Usage chart */}
      <div className="mb-8 rounded-lg border border-gray-800 bg-gray-900 p-6">
        <h2 className="mb-4 text-sm font-medium text-gray-400">
          Daily Requests — Last 30 Days
        </h2>
        <div className="flex h-32 items-end gap-1">
          {MOCK_USAGE_BARS.map((bar) => (
            <div
              key={bar.day}
              className="flex-1 rounded-t bg-[#52b788]/60 hover:bg-[#52b788] transition-colors"
              style={{ height: `${(bar.count / maxBar) * 100}%` }}
              title={`Day ${bar.day}: ${bar.count.toLocaleString()} requests`}
            />
          ))}
        </div>
        <div className="mt-2 flex justify-between text-xs text-gray-600">
          <span>1</span>
          <span>15</span>
          <span>30</span>
        </div>
      </div>

      {/* Recent audit entries */}
      <div className="rounded-lg border border-gray-800 bg-gray-900 p-6">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-sm font-medium text-gray-400">Recent Activity</h2>
          <a
            href="/audit"
            className="text-xs font-medium text-[#52b788] hover:underline"
          >
            View all
          </a>
        </div>
        <div className="space-y-3">
          {MOCK_AUDIT.map((entry) => (
            <div
              key={entry.id}
              className="flex items-center justify-between text-sm"
            >
              <div className="flex items-center gap-3">
                <span className="h-2 w-2 rounded-full bg-[#52b788]" />
                <span className="font-medium text-gray-300">{entry.actor}</span>
                <span className="rounded bg-gray-800 px-2 py-0.5 font-mono text-xs text-gray-400">
                  {entry.action}
                </span>
              </div>
              <span className="text-xs text-gray-500">
                {new Date(entry.time).toLocaleString()}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
