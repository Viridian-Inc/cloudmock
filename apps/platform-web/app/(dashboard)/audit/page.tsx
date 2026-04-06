"use client";

import { useState } from "react";
import { redirect } from "next/navigation";

// TODO: Replace mock data with real API call for audit logs
// Admin-only: check role server-side or use middleware

const ACTION_TYPES = [
  "all",
  "app.create",
  "app.delete",
  "key.create",
  "key.revoke",
  "member.invite",
  "member.remove",
  "settings.update",
];

interface AuditEntry {
  id: number;
  timestamp: string;
  actor: string;
  action: string;
  resource: string;
  ip: string;
}

const MOCK_AUDIT: AuditEntry[] = [
  {
    id: 1,
    timestamp: "2026-04-03T09:12:00Z",
    actor: "alice@example.com",
    action: "app.create",
    resource: "app:my-dev-env",
    ip: "203.0.113.10",
  },
  {
    id: 2,
    timestamp: "2026-04-03T08:55:00Z",
    actor: "bob@example.com",
    action: "key.revoke",
    resource: "key:cm_live_abc",
    ip: "203.0.113.22",
  },
  {
    id: 3,
    timestamp: "2026-04-02T17:30:00Z",
    actor: "alice@example.com",
    action: "member.invite",
    resource: "user:carol@example.com",
    ip: "203.0.113.10",
  },
  {
    id: 4,
    timestamp: "2026-04-02T14:10:00Z",
    actor: "carol@example.com",
    action: "app.delete",
    resource: "app:old-env",
    ip: "198.51.100.5",
  },
  {
    id: 5,
    timestamp: "2026-04-02T11:00:00Z",
    actor: "bob@example.com",
    action: "key.create",
    resource: "key:cm_live_xyz",
    ip: "203.0.113.22",
  },
  {
    id: 6,
    timestamp: "2026-04-01T16:45:00Z",
    actor: "alice@example.com",
    action: "settings.update",
    resource: "org:settings",
    ip: "203.0.113.10",
  },
  {
    id: 7,
    timestamp: "2026-04-01T10:20:00Z",
    actor: "carol@example.com",
    action: "app.create",
    resource: "app:staging-backend",
    ip: "198.51.100.5",
  },
  {
    id: 8,
    timestamp: "2026-03-31T15:00:00Z",
    actor: "bob@example.com",
    action: "member.remove",
    resource: "user:dave@example.com",
    ip: "203.0.113.22",
  },
];

const PAGE_SIZE = 5;

export default function AuditPage() {
  const [actionFilter, setActionFilter] = useState("all");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");
  const [page, setPage] = useState(0);

  const filtered = MOCK_AUDIT.filter((entry) => {
    if (actionFilter !== "all" && entry.action !== actionFilter) return false;
    if (dateFrom && entry.timestamp < dateFrom) return false;
    if (dateTo && entry.timestamp > dateTo + "T23:59:59Z") return false;
    return true;
  });

  const totalPages = Math.ceil(filtered.length / PAGE_SIZE);
  const paginated = filtered.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);

  const handleExportCSV = () => {
    const header = "timestamp,actor,action,resource,ip\n";
    const rows = filtered
      .map(
        (e) =>
          `"${e.timestamp}","${e.actor}","${e.action}","${e.resource}","${e.ip}"`
      )
      .join("\n");
    const blob = new Blob([header + rows], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "audit-log.csv";
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-100">Audit Log</h1>
          <p className="mt-1 text-sm text-gray-400">
            All organization activity events
          </p>
        </div>
        <button
          onClick={handleExportCSV}
          className="rounded-md border border-gray-700 px-4 py-2 text-sm font-medium text-gray-300 hover:border-gray-500 hover:text-gray-100"
        >
          Export CSV
        </button>
      </div>

      {/* Filters */}
      <div className="mb-4 flex flex-wrap gap-3">
        <select
          value={actionFilter}
          onChange={(e) => {
            setActionFilter(e.target.value);
            setPage(0);
          }}
          className="rounded-md border border-gray-700 bg-gray-900 px-3 py-2 text-sm text-gray-300 focus:border-[#52b788] focus:outline-none"
        >
          {ACTION_TYPES.map((a) => (
            <option key={a} value={a}>
              {a === "all" ? "All actions" : a}
            </option>
          ))}
        </select>
        <input
          type="date"
          value={dateFrom}
          onChange={(e) => {
            setDateFrom(e.target.value);
            setPage(0);
          }}
          className="rounded-md border border-gray-700 bg-gray-900 px-3 py-2 text-sm text-gray-300 focus:border-[#52b788] focus:outline-none"
          placeholder="From"
        />
        <input
          type="date"
          value={dateTo}
          onChange={(e) => {
            setDateTo(e.target.value);
            setPage(0);
          }}
          className="rounded-md border border-gray-700 bg-gray-900 px-3 py-2 text-sm text-gray-300 focus:border-[#52b788] focus:outline-none"
          placeholder="To"
        />
      </div>

      {/* Table */}
      <div className="rounded-lg border border-gray-800 bg-gray-900 overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-800">
              <th className="px-4 py-3 text-left font-medium text-gray-400">
                Timestamp
              </th>
              <th className="px-4 py-3 text-left font-medium text-gray-400">
                Actor
              </th>
              <th className="px-4 py-3 text-left font-medium text-gray-400">
                Action
              </th>
              <th className="px-4 py-3 text-left font-medium text-gray-400">
                Resource
              </th>
              <th className="px-4 py-3 text-left font-medium text-gray-400">
                IP Address
              </th>
            </tr>
          </thead>
          <tbody>
            {paginated.length === 0 ? (
              <tr>
                <td
                  colSpan={5}
                  className="px-4 py-8 text-center text-gray-500"
                >
                  No audit entries match the current filters.
                </td>
              </tr>
            ) : (
              paginated.map((entry) => (
                <tr
                  key={entry.id}
                  className="border-b border-gray-800 last:border-0"
                >
                  <td className="px-4 py-3 font-mono text-xs text-gray-400">
                    {new Date(entry.timestamp).toLocaleString()}
                  </td>
                  <td className="px-4 py-3 text-gray-200">{entry.actor}</td>
                  <td className="px-4 py-3">
                    <span className="rounded bg-gray-800 px-2 py-0.5 font-mono text-xs text-gray-300">
                      {entry.action}
                    </span>
                  </td>
                  <td className="px-4 py-3 font-mono text-xs text-gray-400">
                    {entry.resource}
                  </td>
                  <td className="px-4 py-3 font-mono text-xs text-gray-500">
                    {entry.ip}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      <div className="mt-4 flex items-center justify-between text-sm text-gray-400">
        <span>
          Showing {paginated.length} of {filtered.length} entries
        </span>
        <div className="flex gap-2">
          <button
            onClick={() => setPage((p) => Math.max(0, p - 1))}
            disabled={page === 0}
            className="rounded-md border border-gray-700 px-3 py-1.5 text-xs font-medium hover:border-gray-500 hover:text-gray-200 disabled:opacity-40"
          >
            Previous
          </button>
          <button
            onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
            disabled={page >= totalPages - 1}
            className="rounded-md border border-gray-700 px-3 py-1.5 text-xs font-medium hover:border-gray-500 hover:text-gray-200 disabled:opacity-40"
          >
            Next
          </button>
        </div>
      </div>
    </div>
  );
}
