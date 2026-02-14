"use client";

import { useState, useMemo } from "react";
import Link from "next/link";
import { useStocksList } from "@/hooks/useStocksList";
import { StockSummary } from "@/types/api";
import { formatCurrency, formatPercent } from "@/utils/formatting";

export default function StocksPage() {
  const { stocks, loading, error } = useStocksList();
  const [searchTerm, setSearchTerm] = useState("");
  const [sortBy, setSortBy] = useState<"symbol" | "price" | "change">("symbol");

  // Filter and sort stocks
  const filteredAndSortedStocks = useMemo(() => {
    if (!stocks) return [];

    let filtered = stocks.filter((stock: StockSummary) =>
      stock.symbol.toLowerCase().includes(searchTerm.toLowerCase())
    );

    // Sort
    filtered.sort((a: StockSummary, b: StockSummary) => {
      switch (sortBy) {
        case "symbol":
          return a.symbol.localeCompare(b.symbol);
        case "price":
          return b.latest_close - a.latest_close;
        case "change":
          return Math.abs(b.percent_change) - Math.abs(a.percent_change);
        default:
          return 0;
      }
    });

    return filtered;
  }, [stocks, searchTerm, sortBy]);

  // Loading state
  if (loading) {
    return (
      <div className="space-y-6">
        <div className="bg-blue-50 border border-blue-200 rounded-2xl p-6">
          <p className="text-blue-800 font-semibold">Loading stocks...</p>
          <p className="text-blue-700 text-sm mt-2">
            Fetching all stocks from the database
          </p>
        </div>
        <div className="animate-pulse space-y-4">
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {[1, 2, 3, 4, 5, 6].map((i) => (
              <div key={i} className="h-40 bg-slate-200 rounded-2xl"></div>
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
        <div className="bg-red-50 border-2 border-red-200 rounded-2xl p-6">
          <h2 className="text-red-800 font-black text-lg mb-2">
            ⚠️ Failed to Load Stocks
          </h2>
          <p className="text-red-700 mb-2">Could not fetch stocks list from backend</p>
          <p className="text-red-600 text-sm">Error: {error}</p>
        </div>
      </div>
    );
  }

  // No stocks state
  if (!stocks || stocks.length === 0) {
    return (
      <div className="space-y-6">
        <div className="bg-amber-50 border border-amber-200 rounded-2xl p-6">
          <h2 className="text-amber-800 font-black text-lg mb-2">📊 No Stocks Found</h2>
          <p className="text-amber-700">
            No stocks have been added to the system yet. Complete some trades to see stocks
            appear here.
          </p>
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
            <h1 className="text-3xl font-black mb-2">📈 Stocks</h1>
            <p className="text-slate-300 text-lg">
              Explore price history and performance for {stocks.length} stocks
            </p>
          </div>
          <div className="text-right">
            <div className="text-4xl font-black">{stocks.length}</div>
            <div className="text-sm text-slate-300">Total Stocks</div>
          </div>
        </div>
      </div>

      {/* Search and Filter */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <input
          type="text"
          placeholder="🔍 Search by stock symbol..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="flex-1 px-6 py-3 bg-white rounded-xl border border-slate-200 shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 font-semibold"
        />

        <div className="flex gap-2">
          <label className="text-sm font-semibold text-slate-600">Sort by:</label>
          <select
            value={sortBy}
            onChange={(e) => setSortBy(e.target.value as typeof sortBy)}
            className="px-4 py-2 bg-white rounded-lg border border-slate-200 shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 font-semibold"
          >
            <option value="symbol">Symbol</option>
            <option value="price">Price</option>
            <option value="change">% Change</option>
          </select>
        </div>
      </div>

      {/* Stocks Grid */}
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        {filteredAndSortedStocks.map((stock: StockSummary) => (
          <Link
            key={stock.symbol}
            href={`/stocks/${stock.symbol}`}
            className="group"
          >
            <div className="bg-white rounded-2xl shadow-md border border-slate-100 p-6 hover:shadow-xl hover:border-blue-300 transition-all duration-300 cursor-pointer h-full flex flex-col justify-between">
              {/* Symbol and Price */}
              <div className="mb-4">
                <div className="flex items-center justify-between mb-2">
                  <h3 className="text-2xl font-black text-slate-900 group-hover:text-blue-600 transition-colors">
                    📊 {stock.symbol}
                  </h3>
                  <span className="text-xs font-bold text-slate-500 bg-slate-100 px-2 py-1 rounded">
                    {new Date(stock.latest_date).toLocaleDateString("en-US", {
                      month: "short",
                      day: "numeric",
                    })}
                  </span>
                </div>
                <div className="text-3xl font-black text-slate-900">
                  {formatCurrency(stock.latest_close)}
                </div>
              </div>

              {/* Stats */}
              <div className="space-y-3 border-t border-slate-100 pt-4">
                {/* Price Range */}
                <div className="flex items-center justify-between">
                  <span className="text-sm font-semibold text-slate-600">Price Range</span>
                  <span className="text-sm font-bold text-slate-900">
                    {formatCurrency(stock.low_price)} - {formatCurrency(stock.high_price)}
                  </span>
                </div>

                {/* % Change */}
                <div className="flex items-center justify-between">
                  <span className="text-sm font-semibold text-slate-600">Change</span>
                  <span
                    className={`text-sm font-bold ${
                      stock.percent_change >= 0
                        ? "text-green-600"
                        : "text-red-600"
                    }`}
                  >
                    {stock.percent_change >= 0 ? "+" : ""}
                    {formatPercent(stock.percent_change)}
                  </span>
                </div>

                {/* Average Volume */}
                <div className="flex items-center justify-between">
                  <span className="text-sm font-semibold text-slate-600">Avg Volume</span>
                  <span className="text-sm font-bold text-slate-900">
                    {(stock.average_volume / 1000000).toFixed(1)}M
                  </span>
                </div>

                {/* Winning Trades */}
                <div className="flex items-center justify-between">
                  <span className="text-sm font-semibold text-slate-600">
                    Win Trades
                  </span>
                  <span className="text-sm font-bold text-green-600">
                    {stock.winning_trades_count}
                  </span>
                </div>

                {/* Total P&L */}
                <div className="flex items-center justify-between">
                  <span className="text-sm font-semibold text-slate-600">Total P&L</span>
                  <span
                    className={`text-sm font-bold ${
                      stock.total_pnl >= 0 ? "text-green-600" : "text-red-600"
                    }`}
                  >
                    {stock.total_pnl >= 0 ? "+" : ""}
                    {formatCurrency(stock.total_pnl)}
                  </span>
                </div>
              </div>

              {/* View Button */}
              <div className="mt-4 pt-4 border-t border-slate-100">
                <button className="w-full py-2 bg-blue-50 text-blue-600 rounded-lg font-bold hover:bg-blue-100 transition-colors group-hover:bg-blue-600 group-hover:text-white">
                  View History →
                </button>
              </div>
            </div>
          </Link>
        ))}
      </div>

      {/* No Results */}
      {filteredAndSortedStocks.length === 0 && stocks.length > 0 && (
        <div className="text-center py-12">
          <p className="text-2xl font-black text-slate-900 mb-2">
            🔍 No stocks found
          </p>
          <p className="text-slate-600">
            Try adjusting your search term
          </p>
        </div>
      )}
    </div>
  );
}
