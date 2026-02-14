"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

export default function Navigation() {
  const pathname = usePathname();

  return (
    <nav className="flex gap-4">
      <Link
        href="/"
        className={`px-4 py-2 rounded-lg font-medium transition-colors ${
          pathname === "/"
            ? "bg-blue-100 text-blue-700"
            : "text-gray-600 hover:bg-gray-100"
        }`}
      >
        🏠 Home
      </Link>
      <Link
        href="/dashboard"
        className={`px-4 py-2 rounded-lg font-medium transition-colors ${
          pathname === "/dashboard"
            ? "bg-blue-100 text-blue-700"
            : "text-gray-600 hover:bg-gray-100"
        }`}
      >
        📈 Dashboard
      </Link>
      <Link
        href="/docs"
        className={`px-4 py-2 rounded-lg font-medium transition-colors ${
          pathname === "/docs"
            ? "bg-blue-100 text-blue-700"
            : "text-gray-600 hover:bg-gray-100"
        }`}
      >
        📚 Docs
      </Link>
    </nav>
  );
}
