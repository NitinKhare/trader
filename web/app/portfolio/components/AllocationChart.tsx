"use client";

import { StrategyAllocationItem } from "@/types/api";
import { formatCurrency, formatPercent } from "@/utils/formatting";

const COLORS = [
  "bg-blue-500",
  "bg-emerald-500",
  "bg-amber-500",
  "bg-purple-500",
  "bg-rose-500",
  "bg-cyan-500",
  "bg-orange-500",
  "bg-teal-500",
];

const TEXT_COLORS = [
  "text-blue-600",
  "text-emerald-600",
  "text-amber-600",
  "text-purple-600",
  "text-rose-600",
  "text-cyan-600",
  "text-orange-600",
  "text-teal-600",
];

interface AllocationChartProps {
  allocations: StrategyAllocationItem[];
  mode: string;
}

export default function AllocationChart({ allocations, mode }: AllocationChartProps) {
  if (!allocations || allocations.length === 0) {
    return (
      <div className="bg-white rounded-2xl shadow-md border border-slate-100 p-6">
        <h3 className="text-lg font-black text-slate-900 mb-4">Capital Allocation</h3>
        <p className="text-slate-500 text-sm">No allocation data available</p>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-2xl shadow-md border border-slate-100 p-6">
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-lg font-black text-slate-900">Capital Allocation</h3>
        <span className="text-xs font-bold px-3 py-1 bg-blue-50 text-blue-700 rounded-full uppercase">
          {mode}
        </span>
      </div>

      {/* Stacked bar */}
      <div className="flex h-8 rounded-lg overflow-hidden mb-6">
        {allocations.map((alloc, i) => (
          <div
            key={alloc.strategy_id}
            className={`${COLORS[i % COLORS.length]} transition-all duration-500`}
            style={{ width: `${alloc.capital_pct}%` }}
            title={`${alloc.strategy_id}: ${alloc.capital_pct.toFixed(1)}%`}
          />
        ))}
      </div>

      {/* Legend table */}
      <div className="space-y-3">
        {allocations.map((alloc, i) => (
          <div
            key={alloc.strategy_id}
            className="flex items-center justify-between py-2 border-b border-slate-50 last:border-0"
          >
            <div className="flex items-center gap-3">
              <div className={`w-3 h-3 rounded-full ${COLORS[i % COLORS.length]}`} />
              <span className={`text-sm font-bold ${TEXT_COLORS[i % TEXT_COLORS.length]}`}>
                {alloc.strategy_id}
              </span>
            </div>
            <div className="flex items-center gap-6 text-sm">
              <span className="font-bold text-slate-900">
                {formatPercent(alloc.capital_pct, 1)}
              </span>
              <span className="text-slate-600 w-28 text-right">
                {formatCurrency(alloc.capital)}
              </span>
              <span className="text-slate-500 w-16 text-right">
                Sharpe {alloc.rolling_sharpe.toFixed(2)}
              </span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
