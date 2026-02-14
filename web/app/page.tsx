"use client";

import Link from "next/link";

export default function Home() {
  return (
    <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-xl font-semibold mb-4">📈 Dashboard</h2>
        <p className="text-gray-600 mb-6">
          View real-time trading metrics, equity curve, and open positions.
        </p>
        <Link
          href="/dashboard"
          className="inline-block bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700"
        >
          Go to Dashboard →
        </Link>
      </div>

      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-xl font-semibold mb-4">📚 Documentation</h2>
        <p className="text-gray-600 mb-6">
          Complete guide with API reference, setup, troubleshooting, and more.
        </p>
        <Link
          href="/docs"
          className="inline-block bg-green-600 text-white px-6 py-2 rounded-lg hover:bg-green-700"
        >
          Read Docs →
        </Link>
      </div>

      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-xl font-semibold mb-4">⚙️ Configuration</h2>
        <p className="text-gray-600 mb-4">
          <strong>API URL:</strong> http://localhost:8081
        </p>
        <p className="text-gray-600 mb-6">
          <strong>WebSocket:</strong> ws://localhost:8081/ws
        </p>
        <p className="text-sm text-gray-500">
          Ensure the backend dashboard is running before viewing the dashboard.
        </p>
      </div>
    </div>
  );
}
