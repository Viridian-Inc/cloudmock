"use client";

import { use, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import StatusBadge from "@/components/StatusBadge";
import CopyButton from "@/components/CopyButton";

// TODO: Replace mock data with real API calls using:
// import { listKeys, createKey, revokeKey } from "@/lib/api";
// const token = await getToken(); // from useAuth()
// const keys = await listKeys(token, appId);

interface MockKey {
  id: string;
  prefix: string;
  name: string;
  role: string;
  lastUsed: string | null;
  createdAt: string;
}

const MOCK_KEYS: MockKey[] = [
  {
    id: "key_1",
    prefix: "cm_live_abc",
    name: "CI/CD Pipeline",
    role: "developer",
    lastUsed: "2026-04-02T14:30:00Z",
    createdAt: "2026-03-01T10:00:00Z",
  },
  {
    id: "key_2",
    prefix: "cm_live_def",
    name: "Local Dev",
    role: "admin",
    lastUsed: null,
    createdAt: "2026-03-15T09:00:00Z",
  },
];

const TABS = [
  { label: "Overview", href: "" },
  { label: "Keys", href: "/keys" },
  { label: "Settings", href: "/settings" },
];

export default function AppKeysPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const pathname = usePathname();
  const [keys, setKeys] = useState<MockKey[]>(MOCK_KEYS);

  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [newKeyName, setNewKeyName] = useState("");
  const [newKeyRole, setNewKeyRole] = useState("developer");
  const [isCreating, setIsCreating] = useState(false);
  const [createdKey, setCreatedKey] = useState<string | null>(null);

  const [revokeId, setRevokeId] = useState<string | null>(null);
  const [isRevoking, setIsRevoking] = useState(false);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsCreating(true);

    // TODO: Replace with real API call:
    // const token = await getToken();
    // const result = await createKey(token, id, { name: newKeyName });
    // setCreatedKey(result.key);

    await new Promise((r) => setTimeout(r, 600));
    const fakeKey = `cm_live_${Math.random().toString(36).slice(2, 10)}XXXXXXXXXXXXXXXXXXXX`;
    setCreatedKey(fakeKey);
    setKeys((prev) => [
      ...prev,
      {
        id: `key_${Date.now()}`,
        prefix: fakeKey.slice(0, 11),
        name: newKeyName,
        role: newKeyRole,
        lastUsed: null,
        createdAt: new Date().toISOString(),
      },
    ]);
    setIsCreating(false);
    setNewKeyName("");
    setNewKeyRole("developer");
  };

  const handleRevoke = async () => {
    if (!revokeId) return;
    setIsRevoking(true);

    // TODO: Replace with real API call:
    // const token = await getToken();
    // await revokeKey(token, id, revokeId);

    await new Promise((r) => setTimeout(r, 400));
    setKeys((prev) => prev.filter((k) => k.id !== revokeId));
    setIsRevoking(false);
    setRevokeId(null);
  };

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
              tab.href === "/keys"
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

      {/* Keys section */}
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-100">API Keys</h2>
        <button
          onClick={() => setShowCreateDialog(true)}
          className="rounded-md bg-[#52b788] px-4 py-2 text-sm font-medium text-gray-950 hover:bg-[#3d9a6f]"
        >
          Create Key
        </button>
      </div>

      {/* Keys table */}
      <div className="rounded-lg border border-gray-800 bg-gray-900 overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-800">
              <th className="px-4 py-3 text-left font-medium text-gray-400">Prefix</th>
              <th className="px-4 py-3 text-left font-medium text-gray-400">Name</th>
              <th className="px-4 py-3 text-left font-medium text-gray-400">Role</th>
              <th className="px-4 py-3 text-left font-medium text-gray-400">Last Used</th>
              <th className="px-4 py-3 text-left font-medium text-gray-400">Created</th>
              <th className="px-4 py-3 text-right font-medium text-gray-400">Actions</th>
            </tr>
          </thead>
          <tbody>
            {keys.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-gray-500">
                  No API keys yet. Create one to get started.
                </td>
              </tr>
            ) : (
              keys.map((key) => (
                <tr
                  key={key.id}
                  className="border-b border-gray-800 last:border-0"
                >
                  <td className="px-4 py-3">
                    <code className="font-mono text-xs text-gray-300">
                      {key.prefix}...
                    </code>
                  </td>
                  <td className="px-4 py-3 text-gray-200">{key.name}</td>
                  <td className="px-4 py-3">
                    <StatusBadge
                      status={
                        key.role === "admin"
                          ? "running"
                          : key.role === "developer"
                          ? "dedicated"
                          : "shared"
                      }
                    />
                    <span className="ml-2 text-xs text-gray-400">
                      {key.role}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-gray-400">
                    {key.lastUsed
                      ? new Date(key.lastUsed).toLocaleDateString()
                      : "Never"}
                  </td>
                  <td className="px-4 py-3 text-gray-400">
                    {new Date(key.createdAt).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <button
                      onClick={() => setRevokeId(key.id)}
                      className="text-xs font-medium text-red-400 hover:text-red-300"
                    >
                      Revoke
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Create Key Dialog */}
      {showCreateDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
          <div className="w-full max-w-md rounded-xl border border-gray-800 bg-gray-900 p-6 shadow-2xl">
            {createdKey ? (
              <>
                <h2 className="text-lg font-bold text-gray-100">Key Created</h2>
                <div className="mt-4 rounded-lg border border-yellow-700/50 bg-yellow-900/20 p-4">
                  <p className="mb-2 text-xs font-medium text-yellow-400">
                    This key won&apos;t be shown again. Copy it now.
                  </p>
                  <div className="flex items-center gap-2">
                    <code className="flex-1 break-all font-mono text-xs text-[#52b788]">
                      {createdKey}
                    </code>
                    <CopyButton text={createdKey} />
                  </div>
                </div>
                <div className="mt-4 flex justify-end">
                  <button
                    onClick={() => {
                      setShowCreateDialog(false);
                      setCreatedKey(null);
                    }}
                    className="rounded-md bg-[#52b788] px-4 py-2 text-sm font-medium text-gray-950 hover:bg-[#3d9a6f]"
                  >
                    Done
                  </button>
                </div>
              </>
            ) : (
              <>
                <h2 className="text-lg font-bold text-gray-100">
                  Create API Key
                </h2>
                <form onSubmit={handleCreate} className="mt-4 space-y-4">
                  <div>
                    <label
                      htmlFor="key-name"
                      className="block text-sm font-medium text-gray-300"
                    >
                      Key name
                    </label>
                    <input
                      id="key-name"
                      type="text"
                      required
                      value={newKeyName}
                      onChange={(e) => setNewKeyName(e.target.value)}
                      placeholder="e.g. CI/CD Pipeline"
                      className="mt-1.5 w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-gray-100 placeholder-gray-600 focus:border-[#52b788] focus:outline-none focus:ring-1 focus:ring-[#52b788]"
                    />
                  </div>
                  <div>
                    <label
                      htmlFor="key-role"
                      className="block text-sm font-medium text-gray-300"
                    >
                      Role
                    </label>
                    <select
                      id="key-role"
                      value={newKeyRole}
                      onChange={(e) => setNewKeyRole(e.target.value)}
                      className="mt-1.5 w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-gray-100 focus:border-[#52b788] focus:outline-none focus:ring-1 focus:ring-[#52b788]"
                    >
                      <option value="admin">Admin</option>
                      <option value="developer">Developer</option>
                      <option value="viewer">Viewer</option>
                    </select>
                  </div>
                  <div className="flex justify-end gap-3 pt-1">
                    <button
                      type="button"
                      onClick={() => {
                        setShowCreateDialog(false);
                        setCreatedKey(null);
                      }}
                      className="rounded-md px-4 py-2 text-sm font-medium text-gray-400 hover:text-gray-200"
                    >
                      Cancel
                    </button>
                    <button
                      type="submit"
                      disabled={isCreating}
                      className="rounded-md bg-[#52b788] px-4 py-2 text-sm font-medium text-gray-950 hover:bg-[#3d9a6f] disabled:opacity-50"
                    >
                      {isCreating ? "Creating..." : "Create Key"}
                    </button>
                  </div>
                </form>
              </>
            )}
          </div>
        </div>
      )}

      {/* Revoke Confirmation Dialog */}
      {revokeId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
          <div className="w-full max-w-sm rounded-xl border border-gray-800 bg-gray-900 p-6 shadow-2xl">
            <h2 className="text-lg font-bold text-gray-100">Revoke Key?</h2>
            <p className="mt-2 text-sm text-gray-400">
              This action cannot be undone. Any services using this key will
              lose access immediately.
            </p>
            <div className="mt-5 flex justify-end gap-3">
              <button
                onClick={() => setRevokeId(null)}
                className="rounded-md px-4 py-2 text-sm font-medium text-gray-400 hover:text-gray-200"
              >
                Cancel
              </button>
              <button
                onClick={handleRevoke}
                disabled={isRevoking}
                className="rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
              >
                {isRevoking ? "Revoking..." : "Revoke Key"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
