import { useEffect, useState } from "react";
import {
  MetricsResponse,
  PositionsResponse,
  EquityCurveResponse,
  StatusResponse,
  WebSocketMessage,
} from "@/types/api";
import { useAPI } from "./useAPI";
import { useWebSocket } from "./useWebSocket";

interface UsemeticsState {
  metrics: MetricsResponse | null;
  positions: PositionsResponse | null;
  equityCurve: EquityCurveResponse | null;
  status: StatusResponse | null;
  loading: boolean;
  error: string | null;
  connected: boolean;
  lastUpdate: Date | null;
}

export function useMetrics(): UsemeticsState {
  const [state, setState] = useState<UsemeticsState>({
    metrics: null,
    positions: null,
    equityCurve: null,
    status: null,
    loading: true,
    error: null,
    connected: false,
    lastUpdate: null,
  });

  // Fetch initial data from REST API
  const metricsRest = useAPI<MetricsResponse>("/api/metrics");
  const positionsRest = useAPI<PositionsResponse>("/api/positions/open");
  const equityCurveRest = useAPI<EquityCurveResponse>("/api/charts/equity");
  const statusRest = useAPI<StatusResponse>("/api/status");

  // Handle WebSocket messages
  const handleWebSocketMessage = (message: WebSocketMessage) => {
    if (message.type === "metrics" && message.data.metrics) {
      setState((prev) => ({
        ...prev,
        metrics: message.data.metrics,
        lastUpdate: new Date(),
      }));
    }
  };

  const websocket = useWebSocket(handleWebSocketMessage);

  // Set initial state from REST API
  useEffect(() => {
    const allLoaded =
      !metricsRest.loading &&
      !positionsRest.loading &&
      !equityCurveRest.loading &&
      !statusRest.loading;

    const hasError =
      metricsRest.error ||
      positionsRest.error ||
      equityCurveRest.error ||
      statusRest.error;

    setState((prev) => ({
      ...prev,
      metrics: metricsRest.data || prev.metrics,
      positions: positionsRest.data || prev.positions,
      equityCurve: equityCurveRest.data || prev.equityCurve,
      status: statusRest.data || prev.status,
      loading: !allLoaded,
      error: hasError ? "Failed to load some data" : null,
      lastUpdate: allLoaded && !prev.lastUpdate ? new Date() : prev.lastUpdate,
    }));
  }, [
    metricsRest.data,
    metricsRest.loading,
    positionsRest.data,
    positionsRest.loading,
    equityCurveRest.data,
    equityCurveRest.loading,
    statusRest.data,
    statusRest.loading,
  ]);

  // Update connection status
  useEffect(() => {
    setState((prev) => ({
      ...prev,
      connected: websocket.connected,
    }));
  }, [websocket.connected]);

  return state;
}
