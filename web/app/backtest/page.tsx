'use client';

import React, { useState, useEffect } from 'react';
import { BacktestForm } from './components/BacktestForm';
import { BacktestResults } from './components/BacktestResults';
import { BacktestRunRequest, BacktestDetailResponse, StrategyInfo } from '@/types/api';
import { useBacktest } from '@/hooks/useBacktest';
import { useStocksList } from '@/hooks/useStocksList';

export default function BacktestPage() {
  const [results, setResults] = useState<BacktestDetailResponse | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const { getStrategies, runBacktest } = useBacktest();
  const { stocks: stockList } = useStocksList();

  const [strategies, setStrategies] = useState<StrategyInfo[]>([]);
  const [strategiesLoading, setStrategiesLoading] = useState(true);
  const [strategiesError, setStrategiesError] = useState<string | null>(null);

  // Load strategies on mount
  useEffect(() => {
    const loadStrategies = async () => {
      try {
        setStrategiesLoading(true);
        setStrategiesError(null);

        const response = await fetch('http://localhost:8081/api/backtest/strategies');
        if (!response.ok) {
          throw new Error('Failed to load strategies');
        }

        const data = await response.json();
        setStrategies(data.strategies || []);
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : 'Failed to load strategies';
        setStrategiesError(errorMsg);
        console.error('Error loading strategies:', err);
      } finally {
        setStrategiesLoading(false);
      }
    };

    loadStrategies();
  }, []);

  const handleBacktestSubmit = async (config: BacktestRunRequest) => {
    setIsLoading(true);
    try {
      // Submit backtest
      const runResponse = await runBacktest(config);

      if (runResponse && runResponse.backtest_run_id) {
        // For synchronous execution, fetch results after completion
        // In a real scenario, you'd poll or wait for completion
        // For now, show a success message
        alert(`Backtest submitted: ${runResponse.message}`);
      }
    } catch (err) {
      console.error('Backtest submission error:', err);
      alert(err instanceof Error ? err.message : 'Failed to run backtest');
    } finally {
      setIsLoading(false);
    }
  };

  if (strategiesLoading) {
    return (
      <main className="flex-1 overflow-auto">
        <div className="p-8">
          <div className="space-y-6">
            <div className="h-8 bg-slate-200 rounded animate-pulse"></div>
            <div className="h-96 bg-slate-200 rounded animate-pulse"></div>
          </div>
        </div>
      </main>
    );
  }

  if (strategiesError) {
    return (
      <main className="flex-1 overflow-auto">
        <div className="p-8">
          <div className="p-4 bg-rose-50 border border-rose-200 rounded-lg text-rose-700">
            <p className="font-semibold">Error loading strategies</p>
            <p className="text-sm">{strategiesError}</p>
            <p className="text-xs mt-2">
              Make sure the backend server is running on port 8081
            </p>
          </div>
        </div>
      </main>
    );
  }

  return (
    <main className="flex-1 overflow-auto">
      <div className="p-8">
        <div className="mb-8">
          <h1 className="text-4xl font-black text-slate-900 mb-2">📊 Backtest</h1>
          <p className="text-slate-600">
            Configure and run strategy backtests on historical data
          </p>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Form Section */}
          <div className="lg:col-span-1">
            <BacktestForm
              strategies={strategies}
              allStocks={stockList || []}
              onSubmit={handleBacktestSubmit}
              loading={isLoading}
            />
          </div>

          {/* Results Section */}
          <div className="lg:col-span-2">
            {results ? (
              <BacktestResults data={results} />
            ) : (
              <div className="bg-white rounded-2xl shadow-lg p-8 border border-slate-100 text-center">
                <div className="text-5xl mb-4">📈</div>
                <h3 className="text-xl font-black text-slate-900 mb-2">No Results Yet</h3>
                <p className="text-slate-600 mb-4">
                  Configure a backtest on the left and click "Run Backtest" to see results here
                </p>
                <div className="inline-block bg-blue-50 rounded-lg p-4 border border-blue-200">
                  <p className="text-sm text-blue-900">
                    💡 <strong>Tip:</strong> Start with default parameters and adjust as needed
                  </p>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </main>
  );
}
