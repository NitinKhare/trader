import { useEffect, useState } from "react";
import { StockCandlesResponse } from "@/types/api";

interface UseStockCandlesState {
  data: StockCandlesResponse | null;
  loading: boolean;
  error: string | null;
}

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

export function useStockCandles(
  symbol: string,
  fromDate?: Date,
  toDate?: Date
): UseStockCandlesState {
  const [state, setState] = useState<UseStockCandlesState>({
    data: null,
    loading: true,
    error: null,
  });

  useEffect(() => {
    if (!symbol) {
      setState({ data: null, loading: false, error: "Symbol is required" });
      return;
    }

    let isMounted = true;
    let timeoutId: NodeJS.Timeout;

    const fetchCandles = async () => {
      try {
        setState((prev) => ({ ...prev, loading: true, error: null }));

        // Build URL with query parameters
        const params = new URLSearchParams();
        params.set("symbol", symbol);

        if (fromDate) {
          params.set("from", fromDate.toISOString().split("T")[0] || "");
        }
        if (toDate) {
          params.set("to", toDate.toISOString().split("T")[0] || "");
        }

        // Set a 5-second timeout for the fetch
        const controller = new AbortController();
        timeoutId = setTimeout(() => controller.abort(), 5000);

        const response = await fetch(
          `${API_BASE_URL}/api/stocks/candles?${params.toString()}`,
          {
            signal: controller.signal,
          }
        );

        clearTimeout(timeoutId);

        if (!response.ok) {
          throw new Error(
            `API error: ${response.status} ${response.statusText}`
          );
        }

        const data = (await response.json()) as StockCandlesResponse;

        if (isMounted) {
          setState({ data, loading: false, error: null });
        }
      } catch (err) {
        if (isMounted) {
          let errorMessage = "Unknown error";

          if (err instanceof Error) {
            if (err.name === "AbortError") {
              errorMessage = "Request timeout - Backend not responding";
            } else {
              errorMessage = err.message;
            }
          }

          setState({ data: null, loading: false, error: errorMessage });
          console.error(`Error fetching candles for ${symbol}:`, err);
        }
      }
    };

    fetchCandles();

    return () => {
      isMounted = false;
      clearTimeout(timeoutId);
    };
  }, [symbol, fromDate, toDate]);

  return state;
}
