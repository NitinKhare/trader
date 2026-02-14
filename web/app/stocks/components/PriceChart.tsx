"use client";

import {
  ComposedChart,
  Line,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";
import { Candle } from "@/types/api";
import { formatCurrency, formatNumber } from "@/utils/formatting";

interface PriceChartProps {
  candles: Candle[];
  symbol: string;
}

export function PriceChart({ candles, symbol }: PriceChartProps) {
  if (!candles || candles.length === 0) {
    return (
      <div className="w-full h-96 flex items-center justify-center bg-slate-50 rounded-2xl border border-slate-200">
        <p className="text-slate-600 font-semibold">No price data available</p>
      </div>
    );
  }

  // Transform candle data for Recharts
  const chartData = candles.map((candle) => ({
    date: new Date(candle.date).toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
    }),
    open: candle.open,
    high: candle.high,
    low: candle.low,
    close: candle.close,
    volume: candle.volume / 1000000, // Convert to millions
    fullDate: candle.date,
  }));

  // Custom tooltip
  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      return (
        <div className="bg-white p-4 rounded-lg shadow-lg border border-slate-200 z-50">
          <p className="font-bold text-slate-900">{data.date}</p>
          <p className="text-sm text-slate-600">
            Open: <span className="font-semibold">{formatCurrency(data.open)}</span>
          </p>
          <p className="text-sm text-slate-600">
            High: <span className="font-semibold text-green-600">{formatCurrency(data.high)}</span>
          </p>
          <p className="text-sm text-slate-600">
            Low: <span className="font-semibold text-red-600">{formatCurrency(data.low)}</span>
          </p>
          <p className="text-sm text-slate-600">
            Close: <span className="font-semibold">{formatCurrency(data.close)}</span>
          </p>
          <p className="text-sm text-slate-600">
            Volume: <span className="font-semibold">{formatNumber(data.volume, 2)}M</span>
          </p>
        </div>
      );
    }
    return null;
  };

  return (
    <div className="w-full bg-white rounded-2xl shadow-lg border border-slate-100 p-6">
      <div className="mb-6">
        <h3 className="text-2xl font-black text-slate-900 mb-1">
          📊 Price History - {symbol}
        </h3>
        <p className="text-sm text-slate-600">
          {candles.length} days of OHLCV data
        </p>
      </div>

      <ResponsiveContainer width="100%" height={400}>
        <ComposedChart
          data={chartData}
          margin={{ top: 20, right: 30, left: 0, bottom: 60 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
          <XAxis
            dataKey="date"
            angle={-45}
            textAnchor="end"
            height={80}
            tick={{ fontSize: 12 }}
          />
          <YAxis
            yAxisId="left"
            label={{
              value: "Price (₹)",
              angle: -90,
              position: "insideLeft",
            }}
            tick={{ fontSize: 12 }}
          />
          <YAxis
            yAxisId="right"
            orientation="right"
            label={{
              value: "Volume (M)",
              angle: 90,
              position: "insideRight",
            }}
            tick={{ fontSize: 12 }}
          />

          <Tooltip content={<CustomTooltip />} />
          <Legend
            wrapperStyle={{ paddingTop: "20px" }}
            iconType="line"
          />

          {/* Volume bars */}
          <Bar
            yAxisId="right"
            dataKey="volume"
            fill="#a78bfa"
            opacity={0.3}
            name="Volume (M)"
            isAnimationActive={false}
          />

          {/* Price lines */}
          <Line
            yAxisId="left"
            type="monotone"
            dataKey="open"
            stroke="#3b82f6"
            dot={false}
            name="Open"
            strokeWidth={2}
            isAnimationActive={false}
          />
          <Line
            yAxisId="left"
            type="monotone"
            dataKey="high"
            stroke="#10b981"
            dot={false}
            name="High"
            strokeWidth={2}
            isAnimationActive={false}
          />
          <Line
            yAxisId="left"
            type="monotone"
            dataKey="low"
            stroke="#ef4444"
            dot={false}
            name="Low"
            strokeWidth={2}
            isAnimationActive={false}
          />
          <Line
            yAxisId="left"
            type="monotone"
            dataKey="close"
            stroke="#8b5cf6"
            dot={false}
            name="Close"
            strokeWidth={2.5}
            isAnimationActive={false}
          />
        </ComposedChart>
      </ResponsiveContainer>
    </div>
  );
}
