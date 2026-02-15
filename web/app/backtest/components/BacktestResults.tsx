'use client';

import React from 'react';
import { BacktestDetailResponse } from '@/types/api';
import { formatCurrency, formatPercent, getPnLColor } from '@/utils/formatting';
import { EquityCurveChart } from './EquityCurveChart';

interface BacktestResultsProps {
  data: BacktestDetailResponse;
}

export function BacktestResults({ data }: BacktestResultsProps) {
  const { backtest_run, results, trades, equity_curve } = data;

  // Metrics cards data
  const metricsCards = [
    {
      label: 'Total Trades',
      value: results.total_trades,
      unit: '',
      icon: '📊',
      color: 'blue',
    },
    {
      label: 'Win Rate',
      value: `${(results.win_rate * 100).toFixed(2)}%`,
      unit: '',
      icon: '🎯',
      color: 'emerald',
    },
    {
      label: 'Profit Factor',
      value: results.profit_factor.toFixed(2),
      unit: '',
      icon: '💰',
      color: 'purple',
    },
    {
      label: 'Sharpe Ratio',
      value: results.sharpe_ratio.toFixed(2),
      unit: '',
      icon: '📈',
      color: 'indigo',
    },
    {
      label: 'Max Drawdown',
      value: formatCurrency(Math.abs(results.max_drawdown)),
      unit: '',
      icon: '📉',
      color: 'orange',
    },
    {
      label: 'Total P&L',
      value: formatCurrency(results.total_pnl),
      unit: `(${formatPercent(results.pnl_percent)}%)`,
      icon: '💹',
      color: getPnLColor(results.total_pnl),
    },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="bg-gradient-to-r from-slate-900 to-slate-800 text-white rounded-2xl p-8 shadow-xl">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-3xl font-black mb-2">{backtest_run.name}</h2>
            <p className="text-slate-300">
              {backtest_run.strategy_id} • {backtest_run.date_from} to {backtest_run.date_to}
            </p>
          </div>
          <div className="text-right">
            <div
              className={`text-4xl font-black ${
                results.total_pnl >= 0 ? 'text-emerald-400' : 'text-rose-400'
              }`}
            >
              {formatCurrency(results.total_pnl)}
            </div>
            <p className="text-sm text-slate-300 mt-1">
              {results.total_pnl >= 0 ? '📈' : '📉'} {formatPercent(results.pnl_percent)}
            </p>
          </div>
        </div>
      </div>

      {/* Metrics Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {metricsCards.map((metric, idx) => (
          <div
            key={idx}
            className="bg-white rounded-2xl shadow p-6 border border-slate-100 hover:shadow-lg transition-shadow"
          >
            <div className="flex items-start justify-between mb-4">
              <div>
                <p className="text-xs font-bold uppercase text-slate-500 tracking-widest">
                  {metric.label}
                </p>
              </div>
              <span className="text-2xl">{metric.icon}</span>
            </div>
            <div>
              <p
                className={`text-3xl font-black ${
                  metric.color === 'emerald'
                    ? 'text-emerald-600'
                    : metric.color === 'rose'
                      ? 'text-rose-600'
                      : metric.color === 'blue'
                        ? 'text-blue-600'
                        : metric.color === 'purple'
                          ? 'text-purple-600'
                          : metric.color === 'indigo'
                            ? 'text-indigo-600'
                            : 'text-orange-600'
                }`}
              >
                {metric.value}
              </p>
              {metric.unit && <p className="text-sm text-slate-500 mt-1">{metric.unit}</p>}
            </div>
          </div>
        ))}
      </div>

      {/* Equity Curve Chart */}
      <div className="bg-white rounded-2xl shadow-lg p-8 border border-slate-100">
        <h3 className="text-xl font-black text-slate-900 mb-6">Equity Curve</h3>
        {equity_curve && equity_curve.length > 0 ? (
          <EquityCurveChart data={equity_curve} />
        ) : (
          <p className="text-slate-500 text-center py-8">No equity curve data available</p>
        )}
      </div>

      {/* Trades Table */}
      <div className="bg-white rounded-2xl shadow-lg border border-slate-100 overflow-hidden">
        <div className="p-8 border-b border-slate-200">
          <h3 className="text-xl font-black text-slate-900">Trades ({trades.length})</h3>
        </div>
        {trades && trades.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-slate-900 text-white">
                <tr>
                  <th className="px-6 py-3 text-left text-sm font-bold">Symbol</th>
                  <th className="px-6 py-3 text-left text-sm font-bold">Entry Date</th>
                  <th className="px-6 py-3 text-left text-sm font-bold">Exit Date</th>
                  <th className="px-6 py-3 text-right text-sm font-bold">Entry Price</th>
                  <th className="px-6 py-3 text-right text-sm font-bold">Exit Price</th>
                  <th className="px-6 py-3 text-right text-sm font-bold">Quantity</th>
                  <th className="px-6 py-3 text-right text-sm font-bold">P&L</th>
                  <th className="px-6 py-3 text-right text-sm font-bold">Return %</th>
                </tr>
              </thead>
              <tbody>
                {trades.map((trade, idx) => (
                  <tr
                    key={idx}
                    className={`${
                      idx % 2 === 0 ? 'bg-white' : 'bg-slate-50'
                    } hover:bg-blue-50 transition-colors`}
                  >
                    <td className="px-6 py-3 font-semibold text-slate-900">{trade.symbol}</td>
                    <td className="px-6 py-3 text-slate-600 text-sm">{trade.entry_date}</td>
                    <td className="px-6 py-3 text-slate-600 text-sm">{trade.exit_date}</td>
                    <td className="px-6 py-3 text-right text-slate-600">
                      ₹{trade.entry_price.toFixed(2)}
                    </td>
                    <td className="px-6 py-3 text-right text-slate-600">
                      ₹{trade.exit_price.toFixed(2)}
                    </td>
                    <td className="px-6 py-3 text-right text-slate-600">{trade.quantity}</td>
                    <td
                      className={`px-6 py-3 text-right font-semibold ${
                        trade.pnl >= 0 ? 'text-emerald-600' : 'text-rose-600'
                      }`}
                    >
                      {formatCurrency(trade.pnl)}
                    </td>
                    <td
                      className={`px-6 py-3 text-right font-semibold ${
                        trade.pnl_percent >= 0 ? 'text-emerald-600' : 'text-rose-600'
                      }`}
                    >
                      {formatPercent(trade.pnl_percent)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="p-8 text-center text-slate-500">No trades in this backtest</div>
        )}
      </div>

      {/* Summary Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-emerald-50 rounded-xl p-6 border border-emerald-200">
          <p className="text-sm font-bold text-emerald-700 mb-2">Best Trade</p>
          <p className="text-2xl font-black text-emerald-900">
            {formatCurrency(results.best_trade_pnl)}
          </p>
        </div>
        <div className="bg-rose-50 rounded-xl p-6 border border-rose-200">
          <p className="text-sm font-bold text-rose-700 mb-2">Worst Trade</p>
          <p className="text-2xl font-black text-rose-900">
            {formatCurrency(results.worst_trade_pnl)}
          </p>
        </div>
        <div className="bg-blue-50 rounded-xl p-6 border border-blue-200">
          <p className="text-sm font-bold text-blue-700 mb-2">Avg Hold Days</p>
          <p className="text-2xl font-black text-blue-900">{results.avg_hold_days} days</p>
        </div>
      </div>
    </div>
  );
}
