"use client";

import { use, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";

// TODO: Load app settings from API and allow updates/deletion

const TABS = [
  { label: "Overview", href: "" },
  { label: "Keys", href: "/keys" },
  { label: "Settings", href: "/settings" },
];

export default function AppSettingsPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const pathname = usePathname();
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleteInput, setDeleteInput] = useState("");

  return (
    <div>
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-100">App: {id}</h1>
      </div>

      {/* Tab navigation */}
      <div className="mb-6 border-b border-gray-800">
        <nav className="flex gap-6">
          {TABS.map((tab) => {
            const href = `/apps/${id}${tab.href}`;
            const isActive =
              tab.href === "/settings"
                ? pathname === href
                : tab.href === ""
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

      <div className="max-w-lg space-y-6">
        {/* Danger zone */}
        <div className="rounded-lg border border-red-900/50 bg-gray-900 p-6">
          <h2 className="mb-1 text-sm font-semibold text-red-400">
            Danger Zone
          </h2>
          <p className="mb-4 text-sm text-gray-400">
            Permanently delete this app and all associated data, including API
            keys and request logs.
          </p>
          <button
            onClick={() => setShowDeleteConfirm(true)}
            className="rounded-md border border-red-700 px-4 py-2 text-sm font-medium text-red-400 hover:bg-red-900/30"
          >
            Delete App
          </button>
        </div>
      </div>

      {/* Delete confirmation dialog */}
      {showDeleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
          <div className="w-full max-w-sm rounded-xl border border-gray-800 bg-gray-900 p-6 shadow-2xl">
            <h2 className="text-lg font-bold text-gray-100">Delete App?</h2>
            <p className="mt-2 text-sm text-gray-400">
              This action is irreversible. Type{" "}
              <code className="font-mono text-red-400">{id}</code> to confirm.
            </p>
            <input
              type="text"
              value={deleteInput}
              onChange={(e) => setDeleteInput(e.target.value)}
              placeholder={id}
              className="mt-3 w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-gray-100 placeholder-gray-600 focus:border-red-500 focus:outline-none"
            />
            <div className="mt-4 flex justify-end gap-3">
              <button
                onClick={() => {
                  setShowDeleteConfirm(false);
                  setDeleteInput("");
                }}
                className="rounded-md px-4 py-2 text-sm font-medium text-gray-400 hover:text-gray-200"
              >
                Cancel
              </button>
              <button
                disabled={deleteInput !== id}
                className="rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-40"
                onClick={() => {
                  // TODO: call deleteApp(token, id) then redirect to /apps
                  setShowDeleteConfirm(false);
                }}
              >
                Delete App
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
