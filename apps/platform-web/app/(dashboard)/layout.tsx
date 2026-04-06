import { auth } from "@clerk/nextjs/server";
import Sidebar from "@/components/Sidebar";
import { getRole } from "@/lib/roles";

export default async function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { orgRole } = await auth();
  const role = getRole(orgRole ?? undefined);

  return (
    <div className="flex h-screen overflow-hidden bg-gray-950">
      <Sidebar role={role} />
      <main className="flex-1 overflow-y-auto p-6">{children}</main>
    </div>
  );
}
