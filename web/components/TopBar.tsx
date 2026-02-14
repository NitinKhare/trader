"use client";

import { useMetrics } from "@/hooks/useMetrics";

export default function TopBar() {
  const { status, connected } = useMetrics();

  return (
    <header className="bg-white border-b border-slate-200 shadow-sm sticky top-0 z-10">
      <div className="px-8 py-4 flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-black text-slate-900">Dashboard</h2>
          <p className="text-sm text-slate-500 mt-1">Real-time trading metrics</p>
        </div>

        <div className="flex items-center gap-6">
          {/* Connection Status */}
          <div className="flex items-center gap-2">
            <div
              className={`w-3 h-3 rounded-full ${
                connected ? "bg-emerald-500 animate-pulse" : "bg-rose-500"
              }`}
            ></div>
            <span className={`text-sm font-semibold ${
              connected ? "text-emerald-600" : "text-rose-600"
            }`}>
              {connected ? "Connected" : "Disconnected"}
            </span>
          </div>

          {/* Trading Mode */}
          <div className="flex items-center gap-2">
            <span className="text-sm text-slate-600 font-medium">Mode:</span>
            <span className="px-3 py-1 bg-blue-100 text-blue-700 text-sm font-bold rounded-full">
              {status?.is_running ? "🟢 LIVE" : "🔴 INACTIVE"}
            </span>
          </div>

          {/* Time */}
          <div className="text-right">
            <p className="text-sm font-semibold text-slate-900">
              {new Date().toLocaleTimeString()}
            </p>
            <p className="text-xs text-slate-500">
              {new Date().toLocaleDateString()}
            </p>
          </div>
        </div>
      </div>
    </header>
  );
}
