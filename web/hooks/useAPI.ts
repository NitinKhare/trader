import { useEffect, useState } from "react";

interface UseAPIState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
}

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

export function useAPI<T>(endpoint: string): UseAPIState<T> {
  const [state, setState] = useState<UseAPIState<T>>({
    data: null,
    loading: true,
    error: null,
  });

  useEffect(() => {
    let isMounted = true;
    let timeoutId: NodeJS.Timeout;

    const fetchData = async () => {
      try {
        setState((prev) => ({ ...prev, loading: true, error: null }));

        // Set a 5-second timeout for the fetch
        const controller = new AbortController();
        timeoutId = setTimeout(() => controller.abort(), 5000);

        const response = await fetch(`${API_BASE_URL}${endpoint}`, {
          signal: controller.signal,
        });

        clearTimeout(timeoutId);

        if (!response.ok) {
          throw new Error(
            `API error: ${response.status} ${response.statusText}`
          );
        }

        const data = (await response.json()) as T;

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
          console.error(`Error fetching ${endpoint}:`, err);
        }
      }
    };

    fetchData();

    return () => {
      isMounted = false;
      clearTimeout(timeoutId);
    };
  }, [endpoint]);

  return state;
}
