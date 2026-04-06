"use client";

import { use } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import StatusBadge from "@/components/StatusBadge";
import CopyButton from "@/components/CopyButton";

// TODO: Replace mock data with real API call using:
// import { getApp } from "@/lib/api";
// const token = await auth().getToken();
// const app = await getApp(token, params.id);

interface MockApp {
  id: string;
  name: string;
  status: "running" | "shared" | "dedicated" | "stopped";
  endpoint: string;
  infraType: string;
  createdAt: string;
  updatedAt: string;
}

const MOCK_APPS: Record<string, MockApp> = {
  app_1: {
    id: "app_1",
    name: "my-dev-env",
    status: "running",
    endpoint: "https://app1abc123.cloudmock.app",
    infraType: "shared",
    createdAt: "2026-03-01T10:00:00Z",
    updatedAt: "2026-04-01T08:30:00Z",
  },
  app_2: {
    id: "app_2",
    name: "staging-backend",
    status: "shared",
    endpoint: "https://app2def456.cloudmock.app",
    infraType: "shared",
    createdAt: "2026-03-15T14:00:00Z",
    updatedAt: "2026-03-30T11:00:00Z",
  },
  app_3: {
    id: "app_3",
    name: "prod-mirror",
    status: "dedicated",
    endpoint: "https://app3ghi789.cloudmock.app",
    infraType: "dedicated",
    createdAt: "2026-02-10T09:00:00Z",
    updatedAt: "2026-04-02T16:00:00Z",
  },
};

const TABS = [
  { label: "Overview", href: "" },
  { label: "Keys", href: "/keys" },
  { label: "Settings", href: "/settings" },
];

export default function AppDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const pathname = usePathname();
  const app = MOCK_APPS[id] ?? {
    id,
    name: id,
    status: "stopped" as const,
    endpoint: `https://${id}.cloudmock.app`,
    infraType: "shared",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  };

  const quickStart = `export AWS_ENDPOINT_URL=${app.endpoint}`;

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-gray-100">{app.name}</h1>
            <StatusBadge status={app.status} />
          </div>
          <div className="mt-2 flex items-center gap-2">
            <span className="font-mono text-sm text-gray-400">
              {app.endpoint}
            </span>
            <CopyButton text={app.endpoint} />
          </div>
        </div>
      </div>

      {/* Quick start */}
      <div className="mb-6 rounded-lg border border-gray-800 bg-gray-900 p-4">
        <p className="mb-2 text-xs font-medium uppercase tracking-wider text-gray-500">
          Quick Start
        </p>
        <div className="flex items-center gap-3">
          <code className="flex-1 font-mono text-sm text-[#52b788]">
            {quickStart}
          </code>
          <CopyButton text={quickStart} />
        </div>
      </div>

      {/* Tab navigation */}
      <div className="mb-6 border-b border-gray-800">
        <nav className="flex gap-6">
          {TABS.map((tab) => {
            const href = `/apps/${id}${tab.href}`;
            const isActive =
              tab.href === ""
                ? pathname === `/apps/${id}`
                : pathname.startsWith(href);
            return (
              <Link
                key={tab.label}
                href={href}
                className={`pb-3 text-sm font-medium transition-colors ${
                  isActive
                    ? "border-b-2 border-[#52b788] text-[#52b788]"
                    : "text-gray-400 hover:text-gray-200"
                }`}
              >
                {tab.label}
              </Link>
            );
          })}
        </nav>
      </div>

      {/* Overview content */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="rounded-lg border border-gray-800 bg-gray-900 p-5">
          <p className="text-xs font-medium uppercase tracking-wider text-gray-500">
            Endpoint
          </p>
          <p className="mt-2 font-mono text-sm text-gray-200">
            {app.endpoint}
          </p>
        </div>
        <div className="rounded-lg border border-gray-800 bg-gray-900 p-5">
          <p className="text-xs font-medium uppercase tracking-wider text-gray-500">
            Infrastructure Type
          </p>
          <p className="mt-2 text-sm capitalize text-gray-200">
            {app.infraType}
          </p>
        </div>
        <div className="rounded-lg border border-gray-800 bg-gray-900 p-5">
          <p className="text-xs font-medium uppercase tracking-wider text-gray-500">
            Created
          </p>
          <p className="mt-2 text-sm text-gray-200">
            {new Date(app.createdAt).toLocaleString()}
          </p>
        </div>
        <div className="rounded-lg border border-gray-800 bg-gray-900 p-5">
          <p className="text-xs font-medium uppercase tracking-wider text-gray-500">
            Last Updated
          </p>
          <p className="mt-2 text-sm text-gray-200">
            {new Date(app.updatedAt).toLocaleString()}
          </p>
        </div>
      </div>
    </div>
  );
}
