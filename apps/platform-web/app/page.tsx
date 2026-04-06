import { auth } from "@clerk/nextjs/server";
import { redirect } from "next/navigation";
import Link from "next/link";

export default async function HomePage() {
  const { userId } = await auth();

  if (userId) {
    redirect("/apps");
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-gray-950 text-gray-100">
      <div className="text-center space-y-6 max-w-md px-4">
        <h1 className="text-4xl font-bold" style={{ color: "#52b788" }}>
          CloudMock
        </h1>
        <p className="text-gray-400 text-lg">
          Managed mock cloud instances for your development workflow.
        </p>
        <div className="flex flex-col sm:flex-row gap-4 justify-center pt-4">
          <Link
            href="/sign-in"
            className="inline-flex items-center justify-center rounded-md bg-gray-800 px-6 py-3 text-sm font-medium text-gray-100 hover:bg-gray-700 transition-colors"
          >
            Sign In
          </Link>
          <Link
            href="/sign-up"
            className="inline-flex items-center justify-center rounded-md px-6 py-3 text-sm font-medium text-gray-900 transition-colors"
            style={{ backgroundColor: "#52b788" }}
          >
            Sign Up
          </Link>
        </div>
      </div>
    </div>
  );
}
