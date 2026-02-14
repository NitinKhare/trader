"use client";

import { useState, useMemo } from "react";
import Link from "next/link";
import { useStockCandles } from "@/hooks/useStockCandles";
import { PriceChart } from "../components/PriceChart";
import { DateRangeSelector, DateRangePreset } from "../components/DateRangeSelector";
import { formatCurrency, formatPercent } from "@/utils/formatting";

interface StockDetailPageProps {
  params: {
    symbol: string;
  };
}

export default function StockDetailPage({ params }: StockDetailPageProps) {
  const symbol = params.symbol.toUpperCase();
  const [dateRange, setDateRange] = useState<{
    from: Date | undefined;
    to: Date | undefined;
  }>({
    from: new Date(new Date().setFullYear(new Date().getFullYear() - 1)),
    to: new Date(),
  });

  const { data, loading, error } = useStockCandles(
    symbol,
    dateRange.from,
    dateRange.to
  );

  // Calculate stats from candles
  const stats = useMemo(() => {
    if (!data || !data.candles || data.candles.length === 0) {
      return {
        highest: 0,
        lowest: 0,
        avgVolume: 0,
        change: 0,
        changePercent: 0,
      };
    }

    const candles = data.candles as any[];
    let highest = candles[0]?.high || 0;
    let lowest = candles[0]?.low || 0;
    let totalVolume = 0;

    candles.forEach((candle) => {
      if (candle.high > highest) highest = candle.high;
      if (candle.low < lowest) lowest = candle.low;
      totalVolume += candle.volume;
    });

    const firstClose = candles[0].close;
    const lastClose = candles[candles.length - 1].close;
    const change = lastClose - firstClose;
    const changePercent = (change / firstClose) * 100;
    const avgVolume = totalVolume / candles.length;

    return {
      highest,
      lowest,
      avgVolume,
      change,
      changePercent,
    };
  }, [data]);

  const latestPrice =
    data && data.candles && data.candles.length > 0
      ? (data.candles[data.candles.length - 1] as any)?.close || 0
      : 0;

  // Loading state
  if (loading) {
    return (
      <div className="space-y-6">
        <Link
          href="/stocks"
          className="inline-flex items-center gap-2 text-blue-600 hover:text-blue-700 font-bold"
        >
          ← Back to Stocks
        </Link>

        <div className="bg-blue-50 border border-blue-200 rounded-2xl p-6">
          <p className="text-blue-800 font-semibold">Loading {symbol} data...</p>
          <p className="text-blue-700 text-sm mt-2">
            Fetching price history from the database
          </p>
        </div>

        <div className="animate-pulse space-y-6">
          <div className="h-96 bg-slate-200 rounded-2xl"></div>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            {[1, 2, 3, 4].map((i) => (
              <div key={i} className="h-24 bg-slate-200 rounded-2xl"></div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div className="space-y-6">
        <Link
          href="/stocks"
          className="inline-flex items-center gap-2 text-blue-600 hover:text-blue-700 font-bold"
        >
          ← Back to Stocks
        </Link>

        <div className="bg-red-50 border-2 border-red-200 rounded-2xl p-6">
          <h2 className="text-red-800 font-black text-lg mb-2">
            ⚠️ Failed to Load Data
          </h2>
          <p className="text-red-700 mb-2">Could not fetch price history for {symbol}</p>
          <p className="text-red-600 text-sm">Error: {error}</p>
        </div>
      </div>
    );
  }

  // No data state
  if (!data || !data.candles || data.candles.length === 0) {
    return (
      <div className="space-y-6">
        <Link
          href="/stocks"
          className="inline-flex items-center gap-2 text-blue-600 hover:text-blue-700 font-bold"
        >
          ← Back to Stocks
        </Link>

        <div className="bg-amber-50 border border-amber-200 rounded-2xl p-6">
          <h2 className="text-amber-800 font-black text-lg mb-2">📊 No Data Available</h2>
          <p className="text-amber-700">
            No price history found for {symbol} in the requested date range.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {/* Back Button */}
      <Link
        href="/stocks"
        className="inline-flex items-center gap-2 text-blue-600 hover:text-blue-700 font-bold hover:underline"
      >
        ← Back to Stocks
      </Link>

      {/* Header */}
      <div className="bg-gradient-to-r from-slate-900 to-slate-800 rounded-2xl shadow-xl p-8 text-white">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-4xl font-black mb-2">📈 {symbol}</h1>
            <div className="flex items-center gap-4 text-xl">
              <span className="font-black">{formatCurrency(latestPrice)}</span>
              <span
                className={`font-bold ${
                  stats.change >= 0 ? "text-green-300" : "text-red-300"
                }`}
              >
                {stats.change >= 0 ? "+" : ""}
                {formatCurrency(stats.change)} (
                {formatPercent(stats.changePercent)})
              </span>
            </div>
          </div>
          <div className="text-right">
            <div className="text-3xl font-black">{data.candles.length}</div>
            <div className="text-sm text-slate-300">Data Points</div>
          </div>
        </div>
      </div>

      {/* Date Range Selector */}
      <DateRangeSelector
        onRangeChange={(from, to) => {
          setDateRange({ from, to });
        }}
      />

      {/* Price Chart */}
      <PriceChart candles={data.candles} symbol={symbol} />

      {/* Stats Grid */}
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
        {/* Highest Price */}
        <div className="bg-white rounded-2xl shadow-md border border-slate-100 p-6">
          <p className="text-sm font-semibold text-slate-600 mb-2">Highest Price</p>
          <p className="text-3xl font-black text-green-600">
            {formatCurrency(stats.highest)}
          </p>
          <p className="text-xs text-slate-500 mt-2">
            +{formatPercent(
              ((stats.highest - latestPrice) / latestPrice) * 100
            )}{" "}
            from current
          </p>
        </div>

        {/* Lowest Price */}
        <div className="bg-white rounded-2xl shadow-md border border-slate-100 p-6">
          <p className="text-sm font-semibold text-slate-600 mb-2">Lowest Price</p>
          <p className="text-3xl font-black text-red-600">
            {formatCurrency(stats.lowest)}
          </p>
          <p className="text-xs text-slate-500 mt-2">
            {formatPercent(
              ((latestPrice - stats.lowest) / stats.lowest) * 100
            )}{" "}
            above lowest
          </p>
        </div>

        {/* Average Volume */}
        <div className="bg-white rounded-2xl shadow-md border border-slate-100 p-6">
          <p className="text-sm font-semibold text-slate-600 mb-2">Avg Volume</p>
          <p className="text-3xl font-black text-blue-600">
            {(stats.avgVolume / 1000000).toFixed(2)}M
          </p>
          <p className="text-xs text-slate-500 mt-2">Average daily volume</p>
        </div>

        {/* Price Range */}
        <div className="bg-white rounded-2xl shadow-md border border-slate-100 p-6">
          <p className="text-sm font-semibold text-slate-600 mb-2">Price Range</p>
          <p className="text-3xl font-black text-purple-600">
            {formatCurrency(stats.highest - stats.lowest)}
          </p>
          <p className="text-xs text-slate-500 mt-2">
            {formatPercent(
              ((stats.highest - stats.lowest) / stats.lowest) * 100
            )}{" "}
            volatility
          </p>
        </div>
      </div>
    </div>
  );
}
