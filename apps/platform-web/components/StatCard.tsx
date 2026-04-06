interface StatCardProps {
  label: string;
  value: string | number;
  sublabel?: string;
}

export default function StatCard({ label, value, sublabel }: StatCardProps) {
  return (
    <div className="rounded-lg border border-gray-800 bg-gray-900 p-6">
      <p className="text-sm font-medium text-gray-400">{label}</p>
      <p className="mt-2 text-3xl font-bold text-gray-100">{value}</p>
      {sublabel && <p className="mt-1 text-sm text-gray-500">{sublabel}</p>}
    </div>
  );
}
