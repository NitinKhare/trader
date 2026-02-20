"use client";

import { StrategyPerformanceItem } from "@/types/api";
import { formatCurrency, formatPercent } from "@/utils/formatting";

interface StrategyTableProps {
  strategies: StrategyPerformanceItem[];
}

export default function StrategyTable({ strategies }: StrategyTableProps) {
  if (!strategies || strategies.length === 0) {
    return (
      <div className="bg-white rounded-2xl shadow-md border border-slate-100 p-6">
        <h3 className="text-lg font-black text-slate-900 mb-4">Strategy Performance</h3>
        <p className="text-slate-500 text-sm">No strategy performance data available</p>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-2xl shadow-md border border-slate-100 p-6">
      <h3 className="text-lg font-black text-slate-900 mb-6">Strategy Performance</h3>

      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b-2 border-slate-200">
              <th className="text-left py-3 px-2 font-bold text-slate-600 uppercase text-xs">
                Strategy
              </th>
              <th className="text-right py-3 px-2 font-bold text-slate-600 uppercase text-xs">
                Trades
              </th>
              <th className="text-right py-3 px-2 font-bold text-slate-600 uppercase text-xs">
                Win Rate
              </th>
              <th className="text-right py-3 px-2 font-bold text-slate-600 uppercase text-xs">
                Total P&L
              </th>
              <th className="text-right py-3 px-2 font-bold text-slate-600 uppercase text-xs">
                Avg P&L
              </th>
              <th className="text-right py-3 px-2 font-bold text-slate-600 uppercase text-xs">
                Sharpe
              </th>
              <th className="text-right py-3 px-2 font-bold text-slate-600 uppercase text-xs">
                Max DD
              </th>
            </tr>
          </thead>
          <tbody>
            {strategies.map((strat) => (
              <tr
                key={strat.strategy_id}
                className="border-b border-slate-50 hover:bg-slate-50 transition-colors"
              >
                <td className="py-3 px-2 font-bold text-slate-900">{strat.strategy_id}</td>
                <td className="py-3 px-2 text-right font-semibold text-slate-700">
                  {strat.total_trades}
                </td>
                <td className="py-3 px-2 text-right">
                  <span
                    className={`font-bold ${
                      strat.win_rate >= 50 ? "text-green-600" : "text-red-600"
                    }`}
                  >
                    {formatPercent(strat.win_rate, 1)}
                  </span>
                </td>
                <td className="py-3 px-2 text-right">
                  <span
                    className={`font-bold ${
                      strat.total_pnl >= 0 ? "text-green-600" : "text-red-600"
                    }`}
                  >
                    {strat.total_pnl >= 0 ? "+" : ""}
                    {formatCurrency(strat.total_pnl)}
                  </span>
                </td>
                <td className="py-3 px-2 text-right">
                  <span
                    className={`font-semibold ${
                      strat.avg_pnl >= 0 ? "text-green-600" : "text-red-600"
                    }`}
                  >
                    {formatCurrency(strat.avg_pnl)}
                  </span>
                </td>
                <td className="py-3 px-2 text-right">
                  <span
                    className={`font-bold ${
                      strat.rolling_sharpe >= 0 ? "text-blue-600" : "text-red-600"
                    }`}
                  >
                    {strat.rolling_sharpe.toFixed(2)}
                  </span>
                </td>
                <td className="py-3 px-2 text-right font-semibold text-red-600">
                  {formatPercent(strat.max_drawdown, 1)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
