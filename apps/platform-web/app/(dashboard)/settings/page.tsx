"use client";

import { useState } from "react";
import { useOrganization } from "@clerk/nextjs";

// TODO: Load and save retention settings via API
// Admin-only: this page should only be accessible to admins
// The layout or middleware should enforce this check

const DEFAULT_RETENTION = {
  auditLog: 365,
  requestLog: 90,
  stateSnapshots: 30,
};

export default function SettingsPage() {
  const { organization } = useOrganization();

  const [retention, setRetention] = useState(DEFAULT_RETENTION);
  const [isSaving, setIsSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSaving(true);

    // TODO: Replace with real API call:
    // const token = await getToken();
    // await fetch("/api/settings/retention", {
    //   method: "PUT",
    //   headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
    //   body: JSON.stringify(retention),
    // });

    await new Promise((r) => setTimeout(r, 600));
    setIsSaving(false);
    setSaved(true);
    setTimeout(() => setSaved(false), 3000);
  };

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-100">Settings</h1>
        <p className="mt-1 text-sm text-gray-400">
          Organization configuration
        </p>
      </div>

      <div className="max-w-lg space-y-6">
        {/* Org name (read-only) */}
        <div className="rounded-lg border border-gray-800 bg-gray-900 p-6">
          <h2 className="mb-4 text-sm font-semibold text-gray-300">
            Organization
          </h2>
          <div>
            <label className="block text-xs font-medium uppercase tracking-wider text-gray-500">
              Name
            </label>
            <p className="mt-1.5 text-sm text-gray-200">
              {organization?.name ?? "Loading..."}
            </p>
          </div>
        </div>

        {/* Data retention */}
        <form
          onSubmit={handleSave}
          className="rounded-lg border border-gray-800 bg-gray-900 p-6"
        >
          <h2 className="mb-1 text-sm font-semibold text-gray-300">
            Data Retention
          </h2>
          <p className="mb-5 text-xs text-gray-500">
            Configure how long data is retained before automatic deletion.
          </p>

          <div className="space-y-4">
            <RetentionField
              label="Audit Log Retention"
              id="audit-log"
              value={retention.auditLog}
              onChange={(v) => setRetention((r) => ({ ...r, auditLog: v }))}
            />
            <RetentionField
              label="Request Log Retention"
              id="request-log"
              value={retention.requestLog}
              onChange={(v) => setRetention((r) => ({ ...r, requestLog: v }))}
            />
            <RetentionField
              label="State Snapshots Retention"
              id="state-snapshots"
              value={retention.stateSnapshots}
              onChange={(v) =>
                setRetention((r) => ({ ...r, stateSnapshots: v }))
              }
            />
          </div>

          <div className="mt-6 flex items-center gap-4">
            <button
              type="submit"
              disabled={isSaving}
              className="rounded-md bg-[#52b788] px-4 py-2 text-sm font-medium text-gray-950 hover:bg-[#3d9a6f] disabled:opacity-50"
            >
              {isSaving ? "Saving..." : "Save Settings"}
            </button>
            {saved && (
              <span className="text-sm text-[#52b788]">
                Settings saved successfully
              </span>
            )}
          </div>
        </form>
      </div>
    </div>
  );
}

function RetentionField({
  label,
  id,
  value,
  onChange,
}: {
  label: string;
  id: string;
  value: number;
  onChange: (v: number) => void;
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <label htmlFor={id} className="text-sm text-gray-300">
        {label}
      </label>
      <div className="flex items-center gap-2">
        <input
          id={id}
          type="number"
          min={1}
          max={3650}
          value={value}
          onChange={(e) => onChange(Number(e.target.value))}
          className="w-24 rounded-md border border-gray-700 bg-gray-800 px-3 py-1.5 text-right text-sm text-gray-100 focus:border-[#52b788] focus:outline-none focus:ring-1 focus:ring-[#52b788]"
        />
        <span className="text-sm text-gray-500">days</span>
      </div>
    </div>
  );
}
