"use client";

import { OrganizationProfile } from "@clerk/nextjs";

export default function TeamPage() {
  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-100">Team</h1>
        <p className="mt-1 text-sm text-gray-400">
          Manage members, roles, and invitations
        </p>
      </div>

      <OrganizationProfile
        appearance={{
          elements: {
            rootBox: "w-full",
            card: "bg-gray-900 border border-gray-800 shadow-none rounded-lg",
            navbar: "bg-gray-900",
            navbarButton: "text-gray-400",
            navbarButtonActive: "text-[#52b788]",
            headerTitle: "text-gray-100",
            headerSubtitle: "text-gray-400",
            formFieldLabel: "text-gray-300",
            formFieldInput:
              "bg-gray-800 border-gray-700 text-gray-100 focus:border-[#52b788]",
            formButtonPrimary:
              "bg-[#52b788] text-gray-950 hover:bg-[#3d9a6f]",
            tableHead: "text-gray-400",
            tableData: "text-gray-300",
          },
        }}
      />
    </div>
  );
}
