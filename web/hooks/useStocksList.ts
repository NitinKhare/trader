import { useEffect, useState } from "react";
import { StocksListResponse } from "@/types/api";

interface UseStocksListState {
  stocks: any[] | null;
  loading: boolean;
  error: string | null;
}

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

export function useStocksList(): UseStocksListState {
  const [state, setState] = useState<UseStocksListState>({
    stocks: null,
    loading: true,
    error: null,
  });

  useEffect(() => {
    let isMounted = true;
    let timeoutId: NodeJS.Timeout;

    const fetchStocks = async () => {
      try {
        setState((prev) => ({ ...prev, loading: true, error: null }));

        // Set a 5-second timeout for the fetch
        const controller = new AbortController();
        timeoutId = setTimeout(() => controller.abort(), 5000);

        const response = await fetch(`${API_BASE_URL}/api/stocks/list`, {
          signal: controller.signal,
        });

        clearTimeout(timeoutId);

        if (!response.ok) {
          throw new Error(
            `API error: ${response.status} ${response.statusText}`
          );
        }

        const data = (await response.json()) as StocksListResponse;

        if (isMounted) {
          setState({ stocks: data.stocks, loading: false, error: null });
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

          setState({ stocks: null, loading: false, error: errorMessage });
          console.error("Error fetching stocks:", err);
        }
      }
    };

    fetchStocks();

    return () => {
      isMounted = false;
      clearTimeout(timeoutId);
    };
  }, []);

  return state;
}
