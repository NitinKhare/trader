'use client';

import React from 'react';
import {
  ComposedChart,
  Line,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import { BacktestEquityCurvePoint } from '@/types/api';

interface EquityCurveChartProps {
  data: BacktestEquityCurvePoint[];
}

export function EquityCurveChart({ data }: EquityCurveChartProps) {
  if (!data || data.length === 0) {
    return <p className="text-center text-slate-500 py-8">No equity curve data available</p>;
  }

  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      const point = payload[0].payload;
      return (
        <div className="bg-white p-3 border border-slate-200 rounded shadow-lg">
          <p className="text-sm font-semibold text-slate-900">{point.date}</p>
          <p className="text-sm text-blue-600">
            Equity: ₹{parseFloat(point.equity).toLocaleString('en-IN', { maximumFractionDigits: 2 })}
          </p>
          <p className="text-sm text-red-600">
            Drawdown: ₹{parseFloat(point.drawdown).toLocaleString('en-IN', { maximumFractionDigits: 2 })}
          </p>
        </div>
      );
    }
    return null;
  };

  return (
    <ResponsiveContainer width="100%" height={400}>
      <ComposedChart data={data} margin={{ top: 20, right: 30, left: 0, bottom: 60 }}>
        <defs>
          <linearGradient id="equityGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.8} />
            <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
          </linearGradient>
          <linearGradient id="drawdownGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#ef4444" stopOpacity={0.6} />
            <stop offset="95%" stopColor="#ef4444" stopOpacity={0} />
          </linearGradient>
        </defs>

        <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
        <XAxis
          dataKey="date"
          stroke="#64748b"
          style={{ fontSize: '12px' }}
          angle={-45}
          textAnchor="end"
          height={80}
        />
        <YAxis stroke="#64748b" style={{ fontSize: '12px' }} />
        <Tooltip content={<CustomTooltip />} />
        <Legend wrapperStyle={{ fontSize: '12px', paddingTop: '20px' }} />

        {/* Drawdown Area */}
        <Area
          type="monotone"
          dataKey="drawdown"
          fill="url(#drawdownGradient)"
          stroke="#ef4444"
          strokeWidth={2}
          name="Drawdown"
          yAxisId="left"
        />

        {/* Equity Line */}
        <Line
          type="monotone"
          dataKey="equity"
          stroke="#3b82f6"
          strokeWidth={2.5}
          dot={false}
          name="Equity"
          yAxisId="left"
        />
      </ComposedChart>
    </ResponsiveContainer>
  );
}
