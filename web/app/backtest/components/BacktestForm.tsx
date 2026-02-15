'use client';

import React, { useState, useEffect } from 'react';
import { BacktestRunRequest, StrategyInfo } from '@/types/api';
import { useBacktest } from '@/hooks/useBacktest';

interface BacktestFormProps {
  strategies: StrategyInfo[];
  allStocks: string[];
  onSubmit: (config: BacktestRunRequest) => Promise<void>;
  loading?: boolean;
}

export function BacktestForm({ strategies, allStocks, onSubmit, loading = false }: BacktestFormProps) {
  const [name, setName] = useState('');
  const [selectedStrategy, setSelectedStrategy] = useState<string>('');
  const [selectedStocks, setSelectedStocks] = useState<string[]>([]);
  const [useAllStocks, setUseAllStocks] = useState(true);
  const [dateFrom, setDateFrom] = useState('');
  const [dateTo, setDateTo] = useState('');
  const [parameters, setParameters] = useState<Record<string, any>>({});
  const [formError, setFormError] = useState<string | null>(null);

  // Get current strategy info
  const currentStrategy = strategies.find(s => s.id === selectedStrategy);

  // Initialize parameters when strategy changes
  useEffect(() => {
    if (currentStrategy) {
      const newParams: Record<string, any> = {};
      currentStrategy.parameters.forEach(param => {
        newParams[param.name] = param.default;
      });
      setParameters(newParams);
    }
  }, [currentStrategy]);

  // Set default date range (last month)
  useEffect(() => {
    const today = new Date();
    const lastMonth = new Date(today.getFullYear(), today.getMonth() - 1, today.getDate());

    const fromStr = lastMonth.toISOString().split('T')[0];
    const toStr = today.toISOString().split('T')[0];

    if (fromStr && toStr) {
      setDateFrom(fromStr);
      setDateTo(toStr);
    }
  }, []);

  const handleStrategyChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setSelectedStrategy(e.target.value);
  };

  const handleStockChange = (stock: string) => {
    setSelectedStocks(prev => {
      if (prev.includes(stock)) {
        return prev.filter(s => s !== stock);
      } else {
        return [...prev, stock];
      }
    });
  };

  const handleParameterChange = (paramName: string, value: any) => {
    setParameters(prev => ({
      ...prev,
      [paramName]: value,
    }));
  };

  const handlePresetDateRange = (days: number) => {
    const today = new Date();
    const from = new Date(today.getTime() - days * 24 * 60 * 60 * 1000);

    const fromStr = from.toISOString().split('T')[0];
    const toStr = today.toISOString().split('T')[0];

    if (fromStr && toStr) {
      setDateFrom(fromStr);
      setDateTo(toStr);
    }
  };

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setFormError(null);

    // Validation
    if (!selectedStrategy) {
      setFormError('Please select a strategy');
      return;
    }
    if (!dateFrom || !dateTo) {
      setFormError('Please select a date range');
      return;
    }
    if (!useAllStocks && selectedStocks.length === 0) {
      setFormError('Please select at least one stock');
      return;
    }

    const backtest: BacktestRunRequest = {
      name: name || `${selectedStrategy} - ${new Date().toLocaleDateString()}`,
      strategy_id: selectedStrategy,
      stocks: useAllStocks ? null : selectedStocks,
      parameters,
      date_from: dateFrom,
      date_to: dateTo,
    };

    try {
      await onSubmit(backtest);
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to run backtest');
    }
  };

  return (
    <div className="bg-white rounded-2xl shadow-lg p-8 border border-slate-100">
      <h2 className="text-2xl font-black text-slate-900 mb-6">Configure Backtest</h2>

      {formError && (
        <div className="mb-4 p-4 bg-rose-50 border border-rose-200 rounded-lg text-rose-700 text-sm font-medium">
          {formError}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Backtest Name */}
        <div>
          <label className="block text-sm font-semibold text-slate-700 mb-2">
            Backtest Name (Optional)
          </label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g., TrendFollow Sep-Oct 2025"
            className="w-full px-4 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            disabled={loading}
          />
        </div>

        {/* Strategy Selection */}
        <div>
          <label className="block text-sm font-semibold text-slate-700 mb-2">
            Strategy *
          </label>
          <select
            value={selectedStrategy}
            onChange={handleStrategyChange}
            className="w-full px-4 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            disabled={loading}
          >
            <option value="">Select a strategy</option>
            {strategies.map(strategy => (
              <option key={strategy.id} value={strategy.id}>
                {strategy.name}
              </option>
            ))}
          </select>
          {currentStrategy && (
            <p className="text-xs text-slate-500 mt-1">{currentStrategy.description}</p>
          )}
        </div>

        {/* Dynamic Parameters */}
        {currentStrategy && currentStrategy.parameters.length > 0 && (
          <div className="p-4 bg-slate-50 rounded-lg border border-slate-200">
            <h3 className="text-sm font-bold text-slate-900 mb-4">Strategy Parameters</h3>
            <div className="space-y-4">
              {currentStrategy.parameters.map(param => (
                <div key={param.name}>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    {param.display_name}
                  </label>
                  {param.type === 'float' || param.type === 'int' ? (
                    <div className="flex items-center gap-3">
                      <input
                        type="range"
                        min={param.min || 0}
                        max={param.max || 100}
                        step={param.step || 0.1}
                        value={parameters[param.name] || param.default}
                        onChange={(e) =>
                          handleParameterChange(param.name, parseFloat(e.target.value))
                        }
                        className="flex-1"
                        disabled={loading}
                      />
                      <input
                        type="number"
                        min={param.min}
                        max={param.max}
                        step={param.step || 0.1}
                        value={parameters[param.name] || param.default}
                        onChange={(e) =>
                          handleParameterChange(param.name, parseFloat(e.target.value))
                        }
                        className="w-20 px-2 py-1 border border-slate-300 rounded text-sm text-center"
                        disabled={loading}
                      />
                    </div>
                  ) : (
                    <input
                      type={param.type === 'bool' ? 'checkbox' : 'text'}
                      value={parameters[param.name] || param.default}
                      onChange={(e) =>
                        handleParameterChange(
                          param.name,
                          param.type === 'bool' ? e.target.checked : e.target.value
                        )
                      }
                      className="w-full px-2 py-1 border border-slate-300 rounded"
                      disabled={loading}
                    />
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Date Range */}
        <div>
          <label className="block text-sm font-semibold text-slate-700 mb-2">
            Date Range *
          </label>
          <div className="flex gap-2 mb-3 flex-wrap">
            <button
              type="button"
              onClick={() => handlePresetDateRange(7)}
              className="px-3 py-1 text-xs rounded bg-blue-100 text-blue-700 hover:bg-blue-200 disabled:opacity-50"
              disabled={loading}
            >
              1W
            </button>
            <button
              type="button"
              onClick={() => handlePresetDateRange(30)}
              className="px-3 py-1 text-xs rounded bg-blue-100 text-blue-700 hover:bg-blue-200 disabled:opacity-50"
              disabled={loading}
            >
              1M
            </button>
            <button
              type="button"
              onClick={() => handlePresetDateRange(90)}
              className="px-3 py-1 text-xs rounded bg-blue-100 text-blue-700 hover:bg-blue-200 disabled:opacity-50"
              disabled={loading}
            >
              3M
            </button>
            <button
              type="button"
              onClick={() => handlePresetDateRange(180)}
              className="px-3 py-1 text-xs rounded bg-blue-100 text-blue-700 hover:bg-blue-200 disabled:opacity-50"
              disabled={loading}
            >
              6M
            </button>
            <button
              type="button"
              onClick={() => handlePresetDateRange(365)}
              className="px-3 py-1 text-xs rounded bg-blue-100 text-blue-700 hover:bg-blue-200 disabled:opacity-50"
              disabled={loading}
            >
              1Y
            </button>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-xs font-medium text-slate-600 mb-1 block">From</label>
              <input
                type="date"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm"
                disabled={loading}
              />
            </div>
            <div>
              <label className="text-xs font-medium text-slate-600 mb-1 block">To</label>
              <input
                type="date"
                value={dateTo}
                onChange={(e) => setDateTo(e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm"
                disabled={loading}
              />
            </div>
          </div>
        </div>

        {/* Stock Selection */}
        <div>
          <label className="flex items-center gap-2 mb-3">
            <input
              type="checkbox"
              checked={useAllStocks}
              onChange={(e) => setUseAllStocks(e.target.checked)}
              className="w-4 h-4 rounded"
              disabled={loading}
            />
            <span className="text-sm font-semibold text-slate-700">Use All Stocks</span>
          </label>

          {!useAllStocks && (
            <div className="p-4 bg-slate-50 rounded-lg border border-slate-200 max-h-40 overflow-y-auto">
              <div className="grid grid-cols-3 gap-2">
                {allStocks.map(stock => (
                  <label key={stock} className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={selectedStocks.includes(stock)}
                      onChange={() => handleStockChange(stock)}
                      className="w-4 h-4 rounded"
                      disabled={loading}
                    />
                    <span className="text-sm text-slate-700">{stock}</span>
                  </label>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Submit Button */}
        <button
          type="submit"
          disabled={loading}
          className="w-full py-3 bg-gradient-to-r from-blue-600 to-blue-700 text-white font-bold rounded-lg hover:from-blue-700 hover:to-blue-800 disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-200"
        >
          {loading ? '⏳ Running Backtest...' : '▶️ Run Backtest'}
        </button>
      </form>
    </div>
  );
}
