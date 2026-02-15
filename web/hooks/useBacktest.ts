'use client';

import { useState, useCallback } from 'react';
import {
  BacktestRunRequest,
  BacktestRunResponse,
  BacktestDetailResponse,
  BacktestListResponse,
  StrategiesResponse,
  BacktestComparisonResponse,
} from '@/types/api';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081';

interface UseBacktestReturn {
  strategies: any[] | null;
  backtestRuns: any[] | null;
  backtestDetails: BacktestDetailResponse | null;
  comparison: BacktestComparisonResponse | null;
  loading: boolean;
  error: string | null;
  runBacktest: (config: BacktestRunRequest) => Promise<BacktestRunResponse | null>;
  getStrategies: () => Promise<void>;
  getBacktestRuns: (limit?: number, offset?: number) => Promise<void>;
  getBacktestResults: (runId: string) => Promise<void>;
  compareBacktests: (runIds: string[]) => Promise<void>;
}

export function useBacktest(): UseBacktestReturn {
  const [strategies, setStrategies] = useState<any[] | null>(null);
  const [backtestRuns, setBacktestRuns] = useState<any[] | null>(null);
  const [backtestDetails, setBacktestDetails] = useState<BacktestDetailResponse | null>(null);
  const [comparison, setComparison] = useState<BacktestComparisonResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const getStrategies = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(`${API_BASE_URL}/api/backtest/strategies`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch strategies: ${response.statusText}`);
      }

      const data: StrategiesResponse = await response.json();
      setStrategies(data.strategies);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to fetch strategies';
      setError(errorMsg);
      console.error('getStrategies error:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  const runBacktest = useCallback(async (config: BacktestRunRequest): Promise<BacktestRunResponse | null> => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(`${API_BASE_URL}/api/backtest/run`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(config),
      });

      if (!response.ok) {
        throw new Error(`Failed to run backtest: ${response.statusText}`);
      }

      const data: BacktestRunResponse = await response.json();
      return data;
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to run backtest';
      setError(errorMsg);
      console.error('runBacktest error:', err);
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  const getBacktestRuns = useCallback(async (limit: number = 20, offset: number = 0) => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(
        `${API_BASE_URL}/api/backtest/runs?limit=${limit}&offset=${offset}`,
        {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
          },
        }
      );

      if (!response.ok) {
        throw new Error(`Failed to fetch backtest runs: ${response.statusText}`);
      }

      const data: BacktestListResponse = await response.json();
      setBacktestRuns(data.runs);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to fetch backtest runs';
      setError(errorMsg);
      console.error('getBacktestRuns error:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  const getBacktestResults = useCallback(async (runId: string) => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(`${API_BASE_URL}/api/backtest/results/${runId}`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch backtest results: ${response.statusText}`);
      }

      const data: BacktestDetailResponse = await response.json();
      setBacktestDetails(data);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to fetch backtest results';
      setError(errorMsg);
      console.error('getBacktestResults error:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  const compareBacktests = useCallback(async (runIds: string[]) => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(`${API_BASE_URL}/api/backtest/compare`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ backtest_run_ids: runIds }),
      });

      if (!response.ok) {
        throw new Error(`Failed to compare backtests: ${response.statusText}`);
      }

      const data: BacktestComparisonResponse = await response.json();
      setComparison(data);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to compare backtests';
      setError(errorMsg);
      console.error('compareBacktests error:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    strategies,
    backtestRuns,
    backtestDetails,
    comparison,
    loading,
    error,
    runBacktest,
    getStrategies,
    getBacktestRuns,
    getBacktestResults,
    compareBacktests,
  };
}
