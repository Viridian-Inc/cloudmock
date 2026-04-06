"use client";

import { useState } from "react";
import Link from "next/link";
import StatusBadge from "@/components/StatusBadge";

// TODO: Replace mock data with real API call using:
// import { listApps } from "@/lib/api";
// const token = await auth().getToken();
// const apps = await listApps(token);

interface MockApp {
  id: string;
  name: string;
  status: "running" | "shared" | "dedicated" | "stopped";
  endpoint: string;
  requestCount: number;
  createdAt: string;
}

const MOCK_APPS: MockApp[] = [
  {
    id: "app_1",
    name: "my-dev-env",
    status: "running",
    endpoint: "https://app1abc123.cloudmock.app",
    requestCount: 1842,
    createdAt: "2026-03-01",
  },
  {
    id: "app_2",
    name: "staging-backend",
    status: "shared",
    endpoint: "https://app2def456.cloudmock.app",
    requestCount: 347,
    createdAt: "2026-03-15",
  },
  {
    id: "app_3",
    name: "prod-mirror",
    status: "dedicated",
    endpoint: "https://app3ghi789.cloudmock.app",
    requestCount: 9201,
    createdAt: "2026-02-10",
  },
];

export default function AppsPage() {
  const [showDialog, setShowDialog] = useState(false);
  const [appName, setAppName] = useState("");
  const [infraType, setInfraType] = useState<"shared" | "dedicated">("shared");
  const [isCreating, setIsCreating] = useState(false);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsCreating(true);

    // TODO: Replace with real API call:
    // const token = await getToken(); // from useAuth()
    // await createApp(token, { name: appName, description: infraType });
    // router.refresh();

    await new Promise((r) => setTimeout(r, 600)); // simulate network
    setIsCreating(false);
    setShowDialog(false);
    setAppName("");
    setInfraType("shared");
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-100">Apps</h1>
          <p className="mt-1 text-sm text-gray-400">
            Manage your CloudMock environments
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {MOCK_APPS.map((app) => (
          <Link
            key={app.id}
            href={`/apps/${app.id}`}
            className="group block rounded-lg border border-gray-800 bg-gray-900 p-5 transition-colors hover:border-gray-700 hover:bg-gray-800"
          >
            <div className="flex items-start justify-between">
              <h2 className="font-semibold text-gray-100 group-hover:text-white">
                {app.name}
              </h2>
              <StatusBadge status={app.status} />
            </div>
            <p className="mt-3 truncate font-mono text-xs text-gray-500">
              {app.endpoint}
            </p>
            <div className="mt-4 flex items-center justify-between text-xs text-gray-500">
              <span>{app.requestCount.toLocaleString()} requests</span>
              <span>Created {app.createdAt}</span>
            </div>
          </Link>
        ))}

        {/* New App card */}
        <button
          onClick={() => setShowDialog(true)}
          className="flex min-h-[140px] items-center justify-center rounded-lg border-2 border-dashed border-gray-700 bg-transparent p-5 text-gray-500 transition-colors hover:border-gray-500 hover:text-gray-300"
        >
          <div className="text-center">
            <span className="block text-2xl font-light">+</span>
            <span className="mt-1 block text-sm font-medium">New App</span>
          </div>
        </button>
      </div>

      {/* Create App Dialog */}
      {showDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
          <div className="w-full max-w-md rounded-xl border border-gray-800 bg-gray-900 p-6 shadow-2xl">
            <h2 className="text-lg font-bold text-gray-100">Create New App</h2>
            <p className="mt-1 text-sm text-gray-400">
              Set up a new CloudMock environment
            </p>

            <form onSubmit={handleCreate} className="mt-5 space-y-5">
              <div>
                <label
                  htmlFor="app-name"
                  className="block text-sm font-medium text-gray-300"
                >
                  App name
                </label>
                <input
                  id="app-name"
                  type="text"
                  required
                  value={appName}
                  onChange={(e) => setAppName(e.target.value)}
                  placeholder="my-dev-env"
                  className="mt-1.5 w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-gray-100 placeholder-gray-600 focus:border-[#52b788] focus:outline-none focus:ring-1 focus:ring-[#52b788]"
                />
              </div>

              <div>
                <p className="block text-sm font-medium text-gray-300">
                  Infrastructure type
                </p>
                <div className="mt-2 space-y-2">
                  {(["shared", "dedicated"] as const).map((type) => (
                    <label
                      key={type}
                      className="flex cursor-pointer items-start gap-3 rounded-md border border-gray-700 p-3 transition-colors hover:border-gray-600"
                    >
                      <input
                        type="radio"
                        name="infra_type"
                        value={type}
                        checked={infraType === type}
                        onChange={() => setInfraType(type)}
                        className="mt-0.5 accent-[#52b788]"
                      />
                      <div>
                        <span className="block text-sm font-medium capitalize text-gray-200">
                          {type}
                        </span>
                        <span className="text-xs text-gray-500">
                          {type === "shared"
                            ? "Shared infrastructure, lower cost"
                            : "Dedicated resources, higher isolation"}
                        </span>
                      </div>
                    </label>
                  ))}
                </div>
              </div>

              <div className="flex justify-end gap-3 pt-1">
                <button
                  type="button"
                  onClick={() => setShowDialog(false)}
                  className="rounded-md px-4 py-2 text-sm font-medium text-gray-400 hover:text-gray-200"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isCreating}
                  className="rounded-md bg-[#52b788] px-4 py-2 text-sm font-medium text-gray-950 hover:bg-[#3d9a6f] disabled:opacity-50"
                >
                  {isCreating ? "Creating..." : "Create App"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
