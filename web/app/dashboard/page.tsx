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
  const { metrics, positions, equityCurve, status, loading, connected } =
    useMetrics();

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="animate-pulse space-y-4">
          <div className="h-32 bg-gray-200 rounded-lg"></div>
          <div className="h-64 bg-gray-200 rounded-lg"></div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Status Indicator */}
      <div className="bg-white rounded-lg shadow p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div>
              <p className="text-sm font-medium text-gray-600">
                Connection Status
              </p>
              <div className="flex items-center gap-2 mt-1">
                <div
                  className={`w-3 h-3 rounded-full ${
                    connected ? "bg-green-500" : "bg-red-500"
                  }`}
                ></div>
                <span className="text-sm font-semibold">
                  {connected ? "Connected" : "Disconnected"}
                </span>
              </div>
            </div>
          </div>
          <div className="text-right">
            <p className="text-sm font-medium text-gray-600">
              Trading Mode
            </p>
            <p className="text-lg font-bold text-blue-600">
              {status?.is_running ? "🟢 ACTIVE" : "🔴 INACTIVE"}
            </p>
          </div>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {/* Total P&L */}
        <div className="metric-card">
          <p className="metric-label">Total P&L</p>
          <p
            className={`metric-value ${
              (metrics?.total_pnl || 0) >= 0
                ? "text-green-600"
                : "text-red-600"
            }`}
          >
            {formatCurrency(metrics?.total_pnl || 0)}
          </p>
          <p className="text-sm text-gray-600 mt-2">
            {formatPercent(metrics?.total_pnl_percent || 0)}
          </p>
        </div>

        {/* Win Rate */}
        <div className="metric-card">
          <p className="metric-label">Win Rate</p>
          <p className="metric-value text-blue-600">
            {formatPercent(metrics?.win_rate || 0)}
          </p>
          <p className="text-sm text-gray-600 mt-2">
            {metrics?.winning_trades}/{metrics?.total_trades} trades
          </p>
        </div>

        {/* Profit Factor */}
        <div className="metric-card">
          <p className="metric-label">Profit Factor</p>
          <p className="metric-value text-purple-600">
            {(metrics?.profit_factor || 0).toFixed(2)}x
          </p>
          <p className="text-sm text-gray-600 mt-2">
            Gross Profit / Loss ratio
          </p>
        </div>

        {/* Sharpe Ratio */}
        <div className="metric-card">
          <p className="metric-label">Sharpe Ratio</p>
          <p className="metric-value text-indigo-600">
            {(metrics?.sharpe_ratio || 0).toFixed(2)}
          </p>
          <p className="text-sm text-gray-600 mt-2">Risk-adjusted returns</p>
        </div>

        {/* Max Drawdown */}
        <div className="metric-card">
          <p className="metric-label">Max Drawdown</p>
          <p className="metric-value text-orange-600">
            {formatPercent(metrics?.drawdown_percent || 0)}
          </p>
          <p className="text-sm text-gray-600 mt-2">
            {formatCurrency(metrics?.drawdown || 0)}
          </p>
        </div>

        {/* Available Capital */}
        <div className="metric-card">
          <p className="metric-label">Available Capital</p>
          <p className="metric-value text-teal-600">
            {formatCurrency(status?.available_capital || 0)}
          </p>
          <p className="text-sm text-gray-600 mt-2">
            {status?.total_capital && status.available_capital
              ? formatPercent(
                  (status.available_capital / status.total_capital) * 100
                )
              : "0%"}{" "}
            free
          </p>
        </div>
      </div>

      {/* Equity Curve Chart */}
      {equityCurve && equityCurve.points.length > 0 && (
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold mb-4">📈 Equity Curve</h2>
          <ResponsiveContainer width="100%" height={300}>
            <ComposedChart data={equityCurve.points}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="date"
                tickFormatter={(date) => new Date(date).toLocaleDateString()}
              />
              <YAxis />
              <Tooltip
                formatter={(value) => formatCurrency(value as number)}
                labelFormatter={(label) => formatDate(label as string)}
              />
              <Legend />
              <Line
                type="monotone"
                dataKey="equity"
                stroke="#3b82f6"
                name="Equity"
                dot={false}
              />
              <Area
                type="monotone"
                dataKey="drawdown"
                fill="#ef4444"
                stroke="#ef4444"
                name="Drawdown"
                opacity={0.3}
              />
            </ComposedChart>
          </ResponsiveContainer>
          <div className="grid gap-4 md:grid-cols-3 mt-6 pt-6 border-t">
            <div>
              <p className="text-sm text-gray-600">Final Equity</p>
              <p className="text-xl font-bold text-blue-600">
                {formatCurrency(equityCurve.final_equity)}
              </p>
            </div>
            <div>
              <p className="text-sm text-gray-600">Total Return</p>
              <p className="text-xl font-bold text-green-600">
                {formatPercent(equityCurve.total_return_percent)}
              </p>
            </div>
            <div>
              <p className="text-sm text-gray-600">Max Drawdown</p>
              <p className="text-xl font-bold text-red-600">
                {formatPercent(equityCurve.max_drawdown_percent)}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Open Positions */}
      {positions && positions.positions.length > 0 && (
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold mb-4">
            📍 Open Positions ({positions.open_position_count})
          </h2>
          <div className="overflow-x-auto">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Symbol</th>
                  <th>Qty</th>
                  <th>Entry Price</th>
                  <th>Stop Loss</th>
                  <th>Target</th>
                  <th>Unrealized P&L</th>
                  <th>Strategy</th>
                </tr>
              </thead>
              <tbody>
                {positions.positions.map((pos) => (
                  <tr key={pos.id}>
                    <td className="font-semibold">{pos.symbol}</td>
                    <td>{pos.quantity}</td>
                    <td>{formatCurrency(pos.entry_price)}</td>
                    <td>{formatCurrency(pos.stop_loss)}</td>
                    <td>{formatCurrency(pos.target)}</td>
                    <td
                      className={
                        pos.unrealized_pnl >= 0
                          ? "text-green-600 font-semibold"
                          : "text-red-600 font-semibold"
                      }
                    >
                      {formatCurrency(pos.unrealized_pnl)}
                      <span className="text-xs ml-1">
                        ({formatPercent(pos.unrealized_pnl_percent)})
                      </span>
                    </td>
                    <td className="text-sm">{pos.strategy_id}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="grid gap-4 md:grid-cols-2 mt-6 pt-6 border-t">
            <div>
              <p className="text-sm text-gray-600">Capital Deployed</p>
              <p className="text-xl font-bold">
                {formatCurrency(positions.total_capital_used)}
              </p>
            </div>
            <div>
              <p className="text-sm text-gray-600">Utilization</p>
              <p className="text-xl font-bold">
                {formatPercent(positions.capital_utilization_percent)}
              </p>
            </div>
          </div>
        </div>
      )}

      {!positions ||
        (positions.positions.length === 0 && (
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-6 text-center">
            <p className="text-blue-800">No open positions</p>
          </div>
        ))}
    </div>
  );
}
