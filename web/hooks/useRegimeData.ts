import { RegimeResponse } from "@/types/api";
import { useAPI } from "./useAPI";

interface UseRegimeDataState {
  regime: RegimeResponse | null;
  loading: boolean;
  error: string | null;
}

export function useRegimeData(): UseRegimeDataState {
  const { data, loading, error } = useAPI<RegimeResponse>("/api/regime/current");

  return {
    regime: data,
    loading,
    error,
  };
}
