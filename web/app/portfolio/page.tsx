"use client";

import { usePortfolioData } from "@/hooks/usePortfolioData";
import { useRegimeData } from "@/hooks/useRegimeData";
import { formatCurrency, formatPercent, formatRatio } from "@/utils/formatting";
import AllocationChart from "./components/AllocationChart";
import RegimeIndicator from "./components/RegimeIndicator";
import StrategyTable from "./components/CorrelationMatrix";

export default function PortfolioPage() {
  const { allocation, performance, extended, loading, error } = usePortfolioData();
  const { regime, loading: regimeLoading, error: regimeError } = useRegimeData();

  // Loading state
  if (loading || regimeLoading) {
    return (
      <div className="space-y-6">
        <div className="bg-blue-50 border border-blue-200 rounded-2xl p-6">
          <p className="text-blue-800 font-semibold">Loading portfolio data...</p>
          <p className="text-blue-700 text-sm mt-2">
            Fetching allocation, performance, and regime data
          </p>
        </div>
        <div className="animate-pulse space-y-4">
          <div className="h-48 bg-slate-200 rounded-2xl"></div>
          <div className="grid gap-6 md:grid-cols-2">
            <div className="h-64 bg-slate-200 rounded-2xl"></div>
            <div className="h-64 bg-slate-200 rounded-2xl"></div>
          </div>
        </div>
      </div>
    );
  }

  // Error state
  if (error || regimeError) {
    return (
      <div className="space-y-6">
        <div className="bg-red-50 border-2 border-red-200 rounded-2xl p-6">
          <h2 className="text-red-800 font-black text-lg mb-2">Failed to Load Portfolio</h2>
          <p className="text-red-700 mb-2">Could not fetch portfolio data from backend</p>
          <p className="text-red-600 text-sm">Error: {error || regimeError}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="bg-gradient-to-r from-slate-900 to-slate-800 rounded-2xl shadow-xl p-8 text-white">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-3xl font-black mb-2">Portfolio</h1>
            <p className="text-slate-300 text-lg">
              Strategy allocation, performance, and market regime
            </p>
          </div>
          {regime && (
            <div className="text-right">
              <div className="text-sm text-slate-400 mb-1">Market Regime</div>
              <div className="text-2xl font-black">
                {regime.detected_regime.replace("_", " ")}
              </div>
            </div>
          )}
        </div>

        {/* Extended metrics summary */}
        {extended && (
          <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
            <div className="bg-white/10 rounded-xl p-3">
              <p className="text-xs text-slate-400 font-bold uppercase">CAGR</p>
              <p className="text-lg font-black">{formatPercent(extended.cagr * 100, 1)}</p>
            </div>
            <div className="bg-white/10 rounded-xl p-3">
              <p className="text-xs text-slate-400 font-bold uppercase">Sharpe</p>
              <p className="text-lg font-black">{formatRatio(extended.sharpe_ratio)}</p>
            </div>
            <div className="bg-white/10 rounded-xl p-3">
              <p className="text-xs text-slate-400 font-bold uppercase">Sortino</p>
              <p className="text-lg font-black">{formatRatio(extended.sortino_ratio)}</p>
            </div>
            <div className="bg-white/10 rounded-xl p-3">
              <p className="text-xs text-slate-400 font-bold uppercase">Calmar</p>
              <p className="text-lg font-black">{formatRatio(extended.calmar_ratio)}</p>
            </div>
            <div className="bg-white/10 rounded-xl p-3">
              <p className="text-xs text-slate-400 font-bold uppercase">Win Rate</p>
              <p className="text-lg font-black">{formatPercent(extended.win_rate, 1)}</p>
            </div>
            <div className="bg-white/10 rounded-xl p-3">
              <p className="text-xs text-slate-400 font-bold uppercase">Total P&L</p>
              <p className="text-lg font-black">{formatCurrency(extended.total_pnl)}</p>
            </div>
          </div>
        )}
      </div>

      {/* Extended metrics detail cards */}
      {extended && (
        <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-5">
          <div className="bg-white rounded-2xl shadow-md border border-slate-100 p-5">
            <p className="text-xs font-bold text-slate-500 uppercase mb-1">Profit Factor</p>
            <p className="text-2xl font-black text-slate-900">
              {formatRatio(extended.profit_factor)}
            </p>
          </div>
          <div className="bg-white rounded-2xl shadow-md border border-slate-100 p-5">
            <p className="text-xs font-bold text-slate-500 uppercase mb-1">Expectancy</p>
            <p className="text-2xl font-black text-slate-900">
              {formatCurrency(extended.expectancy)}
            </p>
          </div>
          <div className="bg-white rounded-2xl shadow-md border border-slate-100 p-5">
            <p className="text-xs font-bold text-slate-500 uppercase mb-1">Payoff Ratio</p>
            <p className="text-2xl font-black text-slate-900">
              {formatRatio(extended.payoff_ratio)}
            </p>
          </div>
          <div className="bg-white rounded-2xl shadow-md border border-slate-100 p-5">
            <p className="text-xs font-bold text-slate-500 uppercase mb-1">Recovery Factor</p>
            <p className="text-2xl font-black text-slate-900">
              {formatRatio(extended.recovery_factor)}
            </p>
          </div>
          <div className="bg-white rounded-2xl shadow-md border border-slate-100 p-5">
            <p className="text-xs font-bold text-slate-500 uppercase mb-1">Max Drawdown</p>
            <p className="text-2xl font-black text-red-600">
              {formatPercent(extended.max_drawdown, 1)}
            </p>
          </div>
        </div>
      )}

      {/* Regime + Allocation side by side */}
      <div className="grid gap-6 lg:grid-cols-2">
        {regime && <RegimeIndicator regime={regime} />}
        {allocation && (
          <AllocationChart allocations={allocation.allocations} mode={allocation.mode} />
        )}
      </div>

      {/* Strategy Performance Table */}
      {performance && <StrategyTable strategies={performance.strategies} />}

      {/* Streak info */}
      {extended && (
        <div className="grid gap-6 md:grid-cols-2">
          <div className="bg-white rounded-2xl shadow-md border border-slate-100 p-6">
            <h3 className="text-lg font-black text-slate-900 mb-4">Win/Loss Streaks</h3>
            <div className="flex items-center gap-8">
              <div>
                <p className="text-xs font-bold text-slate-500 uppercase mb-1">
                  Max Consecutive Wins
                </p>
                <p className="text-3xl font-black text-green-600">
                  {extended.max_consecutive_wins}
                </p>
              </div>
              <div>
                <p className="text-xs font-bold text-slate-500 uppercase mb-1">
                  Max Consecutive Losses
                </p>
                <p className="text-3xl font-black text-red-600">
                  {extended.max_consecutive_losses}
                </p>
              </div>
            </div>
          </div>
          <div className="bg-white rounded-2xl shadow-md border border-slate-100 p-6">
            <h3 className="text-lg font-black text-slate-900 mb-4">Risk-Adjusted Returns</h3>
            <div className="flex items-center gap-8">
              <div>
                <p className="text-xs font-bold text-slate-500 uppercase mb-1">
                  Avg R-Multiple
                </p>
                <p className="text-3xl font-black text-blue-600">
                  {formatRatio(extended.avg_r_multiple)}
                </p>
              </div>
              <div>
                <p className="text-xs font-bold text-slate-500 uppercase mb-1">Total Trades</p>
                <p className="text-3xl font-black text-slate-900">{extended.total_trades}</p>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
