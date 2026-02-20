import {
  StrategyAllocationResponse,
  StrategyPerformanceResponse,
  ExtendedMetricsResponse,
} from "@/types/api";
import { useAPI } from "./useAPI";

interface UsePortfolioDataState {
  allocation: StrategyAllocationResponse | null;
  performance: StrategyPerformanceResponse | null;
  extended: ExtendedMetricsResponse | null;
  loading: boolean;
  error: string | null;
}

export function usePortfolioData(): UsePortfolioDataState {
  const alloc = useAPI<StrategyAllocationResponse>("/api/strategies/allocation");
  const perf = useAPI<StrategyPerformanceResponse>("/api/strategies/performance");
  const ext = useAPI<ExtendedMetricsResponse>("/api/metrics/extended");

  const loading = alloc.loading || perf.loading || ext.loading;
  const error = alloc.error || perf.error || ext.error || null;

  return {
    allocation: alloc.data,
    performance: perf.data,
    extended: ext.data,
    loading,
    error,
  };
}
