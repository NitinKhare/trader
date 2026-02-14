"use client";

import { useMetrics } from "@/hooks/useMetrics";
import { formatCurrency, formatPercent, formatDate } from "@/utils/formatting";
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  ComposedChart,
} from "recharts";

export default function DashboardPage() {
  const { metrics, positions, equityCurve, status, loading, connected, error } =
    useMetrics();

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="bg-blue-50 border border-blue-200 rounded-2xl p-6">
          <p className="text-blue-800 font-semibold">Loading dashboard...</p>
          <p className="text-blue-700 text-sm mt-2">
            Make sure the backend is running: <code className="bg-blue-100 px-2 py-1 rounded text-xs font-mono">./dashboard --port 8081</code>
          </p>
        </div>
        <div className="animate-pulse space-y-4">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {[1, 2, 3, 4, 5, 6].map((i) => (
              <div key={i} className="h-28 bg-slate-200 rounded-2xl"></div>
            ))}
          </div>
          <div className="h-80 bg-slate-200 rounded-2xl"></div>
        </div>
      </div>
    );
  }

  // Show error message if backend is not responding
  if (error) {
    return (
      <div className="space-y-6">
        <div className="bg-red-50 border-2 border-red-200 rounded-2xl p-6">
          <h2 className="text-red-800 font-black text-lg mb-2">⚠️ Backend Not Running</h2>
          <p className="text-red-700 mb-4">
            The dashboard backend is not responding. Please start it first:
          </p>
          <div className="bg-red-100 rounded-lg p-4 font-mono text-sm text-red-900 mb-4 overflow-x-auto space-y-1">
            <div>$ cd /Users/nitinkhare/Downloads/algoTradingAgent</div>
            <div>$ ./dashboard --port 8081</div>
          </div>
          <p className="text-red-700 text-sm">
            Error: {error}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {/* Top Status Bar */}
      <div className="bg-gradient-to-r from-slate-900 to-slate-800 rounded-2xl shadow-xl p-6 text-white">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-6">
            <div>
              <p className="text-slate-400 text-sm font-medium uppercase tracking-wide">
                Connection Status
              </p>
              <div className="flex items-center gap-3 mt-2">
                <div
                  className={`w-4 h-4 rounded-full animate-pulse ${
                    connected ? "bg-emerald-400" : "bg-rose-400"
                  }`}
                ></div>
                <span className="text-lg font-bold">
                  {connected ? "Connected" : "Disconnected"}
                </span>
              </div>
            </div>
            <div className="w-px h-12 bg-slate-700"></div>
            <div>
              <p className="text-slate-400 text-sm font-medium uppercase tracking-wide">
                Trading Mode
              </p>
              <p className="text-lg font-bold mt-2">
                {status?.is_running ? "🟢 ACTIVE" : "🔴 INACTIVE"}
              </p>
            </div>
          </div>
          <div className="text-right">
            <p className="text-slate-400 text-sm font-medium uppercase tracking-wide">
              Last Update
            </p>
            <p className="text-lg font-bold mt-2">
              {metrics?.timestamp ? new Date(metrics.timestamp).toLocaleTimeString() : "—"}
            </p>
          </div>
        </div>
      </div>

      {/* Key Metrics Grid */}
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        {/* Total P&L - Primary Metric */}
        <div className="lg:col-span-1 bg-gradient-to-br from-white to-slate-50 rounded-2xl shadow-lg p-6 border border-slate-100 hover:shadow-xl transition-shadow">
          <div className="flex items-start justify-between">
            <div>
              <p className="text-slate-600 text-sm font-semibold uppercase tracking-wide">
                Total P&L
              </p>
              <p
                className={`text-4xl font-black mt-3 ${
                  (metrics?.total_pnl || 0) >= 0
                    ? "text-emerald-600"
                    : "text-rose-600"
                }`}
              >
                {formatCurrency(metrics?.total_pnl || 0)}
              </p>
              <p className={`text-sm font-semibold mt-2 ${
                (metrics?.total_pnl_percent || 0) >= 0
                  ? "text-emerald-600"
                  : "text-rose-600"
              }`}>
                {metrics && metrics.total_pnl_percent >= 0 ? "+" : ""}
                {formatPercent(metrics?.total_pnl_percent || 0)}
              </p>
            </div>
            <div className="text-4xl">
              {(metrics?.total_pnl || 0) >= 0 ? "📈" : "📉"}
            </div>
          </div>
        </div>

        {/* Win Rate */}
        <div className="bg-gradient-to-br from-white to-slate-50 rounded-2xl shadow-lg p-6 border border-slate-100 hover:shadow-xl transition-shadow">
          <div className="flex items-start justify-between">
            <div>
              <p className="text-slate-600 text-sm font-semibold uppercase tracking-wide">
                Win Rate
              </p>
              <p className="text-4xl font-black mt-3 text-blue-600">
                {formatPercent(metrics?.win_rate || 0)}
              </p>
              <p className="text-sm text-slate-600 font-medium mt-2">
                {metrics?.winning_trades || 0}/{metrics?.total_trades || 0} trades
              </p>
            </div>
            <div className="text-4xl">🎯</div>
          </div>
        </div>

        {/* Profit Factor */}
        <div className="bg-gradient-to-br from-white to-slate-50 rounded-2xl shadow-lg p-6 border border-slate-100 hover:shadow-xl transition-shadow">
          <div className="flex items-start justify-between">
            <div>
              <p className="text-slate-600 text-sm font-semibold uppercase tracking-wide">
                Profit Factor
              </p>
              <p className="text-4xl font-black mt-3 text-purple-600">
                {(metrics?.profit_factor || 0).toFixed(2)}x
              </p>
              <p className="text-sm text-slate-600 font-medium mt-2">
                Profit/Loss ratio
              </p>
            </div>
            <div className="text-4xl">⚖️</div>
          </div>
        </div>

        {/* Sharpe Ratio */}
        <div className="bg-gradient-to-br from-white to-slate-50 rounded-2xl shadow-lg p-6 border border-slate-100 hover:shadow-xl transition-shadow">
          <div className="flex items-start justify-between">
            <div>
              <p className="text-slate-600 text-sm font-semibold uppercase tracking-wide">
                Sharpe Ratio
              </p>
              <p className="text-4xl font-black mt-3 text-indigo-600">
                {(metrics?.sharpe_ratio || 0).toFixed(2)}
              </p>
              <p className="text-sm text-slate-600 font-medium mt-2">
                Risk-adjusted returns
              </p>
            </div>
            <div className="text-4xl">📊</div>
          </div>
        </div>

        {/* Max Drawdown */}
        <div className="bg-gradient-to-br from-white to-slate-50 rounded-2xl shadow-lg p-6 border border-slate-100 hover:shadow-xl transition-shadow">
          <div className="flex items-start justify-between">
            <div>
              <p className="text-slate-600 text-sm font-semibold uppercase tracking-wide">
                Max Drawdown
              </p>
              <p className="text-4xl font-black mt-3 text-orange-600">
                {formatPercent(metrics?.drawdown_percent || 0)}
              </p>
              <p className="text-sm text-slate-600 font-medium mt-2">
                {formatCurrency(metrics?.drawdown || 0)}
              </p>
            </div>
            <div className="text-4xl">⬇️</div>
          </div>
        </div>

        {/* Available Capital */}
        <div className="bg-gradient-to-br from-white to-slate-50 rounded-2xl shadow-lg p-6 border border-slate-100 hover:shadow-xl transition-shadow">
          <div className="flex items-start justify-between">
            <div>
              <p className="text-slate-600 text-sm font-semibold uppercase tracking-wide">
                Available Capital
              </p>
              <p className="text-4xl font-black mt-3 text-teal-600">
                {formatCurrency(status?.available_capital || 0)}
              </p>
              <p className="text-sm text-slate-600 font-medium mt-2">
                {status?.total_capital && status.available_capital
                  ? formatPercent(
                      (status.available_capital / status.total_capital) * 100
                    )
                  : "0%"}{" "}
                free
              </p>
            </div>
            <div className="text-4xl">💰</div>
          </div>
        </div>
      </div>

      {/* Equity Curve Chart */}
      {equityCurve && equityCurve.points.length > 0 && (
        <div className="bg-gradient-to-br from-white to-slate-50 rounded-2xl shadow-lg p-8 border border-slate-100">
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-2xl font-black text-slate-900">
              📈 Equity Growth
            </h2>
            <div className="flex gap-4 text-sm">
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-blue-500"></div>
                <span className="text-slate-600 font-medium">Equity</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-rose-400"></div>
                <span className="text-slate-600 font-medium">Drawdown</span>
              </div>
            </div>
          </div>

          <ResponsiveContainer width="100%" height={350}>
            <ComposedChart data={equityCurve.points}>
              <defs>
                <linearGradient id="colorEquity" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.8}/>
                  <stop offset="95%" stopColor="#3b82f6" stopOpacity={0.1}/>
                </linearGradient>
                <linearGradient id="colorDrawdown" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#ef4444" stopOpacity={0.4}/>
                  <stop offset="95%" stopColor="#ef4444" stopOpacity={0.05}/>
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
              <XAxis
                dataKey="date"
                tickFormatter={(date) => new Date(date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}
                stroke="#94a3b8"
                style={{ fontSize: '12px' }}
              />
              <YAxis
                stroke="#94a3b8"
                style={{ fontSize: '12px' }}
                tickFormatter={(value) => `$${(value / 1000).toFixed(0)}k`}
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: '#1e293b',
                  border: '1px solid #475569',
                  borderRadius: '12px',
                  padding: '12px'
                }}
                formatter={(value) => formatCurrency(value as number)}
                labelFormatter={(label) => formatDate(label as string)}
                labelStyle={{ color: '#f1f5f9', fontWeight: 'bold' }}
              />
              <Area
                type="monotone"
                dataKey="equity"
                stroke="#3b82f6"
                strokeWidth={2}
                fill="url(#colorEquity)"
                name="Equity"
              />
              <Area
                type="monotone"
                dataKey="drawdown"
                stroke="#ef4444"
                fill="url(#colorDrawdown)"
                name="Drawdown"
              />
            </ComposedChart>
          </ResponsiveContainer>

          <div className="grid gap-4 md:grid-cols-3 mt-8 pt-8 border-t border-slate-200">
            <div className="p-4 bg-blue-50 rounded-xl border border-blue-200">
              <p className="text-slate-600 text-sm font-semibold uppercase tracking-wide">
                Final Equity
              </p>
              <p className="text-3xl font-black text-blue-600 mt-2">
                {formatCurrency(equityCurve.final_equity)}
              </p>
            </div>
            <div className="p-4 bg-emerald-50 rounded-xl border border-emerald-200">
              <p className="text-slate-600 text-sm font-semibold uppercase tracking-wide">
                Total Return
              </p>
              <p className="text-3xl font-black text-emerald-600 mt-2">
                {formatPercent(equityCurve.total_return_percent)}
              </p>
            </div>
            <div className="p-4 bg-rose-50 rounded-xl border border-rose-200">
              <p className="text-slate-600 text-sm font-semibold uppercase tracking-wide">
                Max Drawdown
              </p>
              <p className="text-3xl font-black text-rose-600 mt-2">
                {formatPercent(equityCurve.max_drawdown_percent)}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Open Positions */}
      {positions && positions.positions.length > 0 && (
        <div className="bg-gradient-to-br from-white to-slate-50 rounded-2xl shadow-lg p-8 border border-slate-100">
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-2xl font-black text-slate-900">
              📍 Open Positions
              <span className="inline-block ml-3 px-3 py-1 bg-blue-100 text-blue-700 rounded-full text-lg font-bold">
                {positions.open_position_count}
              </span>
            </h2>
          </div>

          <div className="overflow-x-auto rounded-xl border border-slate-200">
            <table className="w-full">
              <thead>
                <tr className="bg-gradient-to-r from-slate-900 to-slate-800 text-white">
                  <th className="px-6 py-4 text-left text-sm font-bold uppercase tracking-wide">Symbol</th>
                  <th className="px-6 py-4 text-left text-sm font-bold uppercase tracking-wide">Qty</th>
                  <th className="px-6 py-4 text-left text-sm font-bold uppercase tracking-wide">Entry</th>
                  <th className="px-6 py-4 text-left text-sm font-bold uppercase tracking-wide">Stop Loss</th>
                  <th className="px-6 py-4 text-left text-sm font-bold uppercase tracking-wide">Target</th>
                  <th className="px-6 py-4 text-left text-sm font-bold uppercase tracking-wide">P&L</th>
                  <th className="px-6 py-4 text-left text-sm font-bold uppercase tracking-wide">Strategy</th>
                </tr>
              </thead>
              <tbody>
                {positions.positions.map((pos, idx) => (
                  <tr
                    key={pos.id}
                    className={`border-t border-slate-200 hover:bg-blue-50 transition-colors ${
                      idx % 2 === 0 ? "bg-white" : "bg-slate-50"
                    }`}
                  >
                    <td className="px-6 py-4 font-black text-slate-900">{pos.symbol}</td>
                    <td className="px-6 py-4 font-semibold text-slate-700">{pos.quantity}</td>
                    <td className="px-6 py-4 font-medium text-slate-700">{formatCurrency(pos.entry_price)}</td>
                    <td className="px-6 py-4 font-medium text-slate-700">{formatCurrency(pos.stop_loss)}</td>
                    <td className="px-6 py-4 font-medium text-slate-700">{formatCurrency(pos.target)}</td>
                    <td className="px-6 py-4">
                      <div className={`font-black text-lg ${
                        pos.unrealized_pnl >= 0
                          ? "text-emerald-600"
                          : "text-rose-600"
                      }`}>
                        {formatCurrency(pos.unrealized_pnl)}
                      </div>
                      <div className={`text-xs font-bold ${
                        pos.unrealized_pnl >= 0
                          ? "text-emerald-600"
                          : "text-rose-600"
                      }`}>
                        {pos.unrealized_pnl >= 0 ? "+" : ""}
                        {formatPercent(pos.unrealized_pnl_percent)}
                      </div>
                    </td>
                    <td className="px-6 py-4 text-sm font-medium text-slate-600 bg-slate-100 rounded">{pos.strategy_id}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="grid gap-4 md:grid-cols-2 mt-8 pt-8 border-t border-slate-200">
            <div className="p-6 bg-gradient-to-br from-blue-50 to-blue-100 rounded-xl border border-blue-200">
              <p className="text-slate-600 text-sm font-semibold uppercase tracking-wide">
                Capital Deployed
              </p>
              <p className="text-3xl font-black text-blue-600 mt-3">
                {formatCurrency(positions.total_capital_used)}
              </p>
            </div>
            <div className="p-6 bg-gradient-to-br from-purple-50 to-purple-100 rounded-xl border border-purple-200">
              <p className="text-slate-600 text-sm font-semibold uppercase tracking-wide">
                Capital Utilization
              </p>
              <p className="text-3xl font-black text-purple-600 mt-3">
                {formatPercent(positions.capital_utilization_percent)}
              </p>
            </div>
          </div>
        </div>
      )}

      {!positions ||
        (positions.positions.length === 0 && (
          <div className="bg-gradient-to-br from-blue-50 to-indigo-50 border-2 border-blue-200 rounded-2xl p-12 text-center">
            <p className="text-blue-800 text-lg font-semibold">
              ✨ No open positions
            </p>
            <p className="text-blue-600 text-sm mt-2">
              Trading strategies are evaluating the market...
            </p>
          </div>
        ))}
    </div>
  );
}
