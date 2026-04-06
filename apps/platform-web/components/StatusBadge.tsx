type AppStatus = "running" | "shared" | "dedicated" | "stopped";

interface StatusBadgeProps {
  status: AppStatus;
}

const statusStyles: Record<AppStatus, string> = {
  running: "bg-green-900/40 text-[#52b788] border border-green-800/50",
  shared: "bg-yellow-900/40 text-yellow-400 border border-yellow-800/50",
  dedicated: "bg-blue-900/40 text-blue-400 border border-blue-800/50",
  stopped: "bg-red-900/40 text-red-400 border border-red-800/50",
};

export default function StatusBadge({ status }: StatusBadgeProps) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${statusStyles[status]}`}
    >
      {status}
    </span>
  );
}
