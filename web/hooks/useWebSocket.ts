import { useEffect, useState, useRef, useCallback } from "react";
import { WebSocketMessage } from "@/types/api";

interface UseWebSocketState {
  connected: boolean;
  lastMessage: WebSocketMessage | null;
  error: string | null;
}

const WS_BASE_URL = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8081";

export function useWebSocket(
  onMessage?: (message: WebSocketMessage) => void
): UseWebSocketState {
  const [state, setState] = useState<UseWebSocketState>({
    connected: false,
    lastMessage: null,
    error: null,
  });

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const maxReconnectAttempts = 5;
  const reconnectDelayMs = 3000;

  const connect = useCallback(() => {
    if (typeof window === "undefined") return;

    try {
      const ws = new WebSocket(`${WS_BASE_URL}/ws`);

      ws.onopen = () => {
        console.log("[WebSocket] Connected");
        setState((prev) => ({
          ...prev,
          connected: true,
          error: null,
        }));
        reconnectAttemptsRef.current = 0;
      };

      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data) as WebSocketMessage;
          setState((prev) => ({
            ...prev,
            lastMessage: message,
          }));
          onMessage?.(message);
        } catch (err) {
          console.error("[WebSocket] Failed to parse message:", err);
        }
      };

      ws.onerror = (error) => {
        console.error("[WebSocket] Error:", error);
        setState((prev) => ({
          ...prev,
          error: "WebSocket error occurred",
        }));
      };

      ws.onclose = () => {
        console.log("[WebSocket] Disconnected");
        setState((prev) => ({
          ...prev,
          connected: false,
        }));

        // Attempt to reconnect
        if (reconnectAttemptsRef.current < maxReconnectAttempts) {
          reconnectAttemptsRef.current += 1;
          const delay = reconnectDelayMs * reconnectAttemptsRef.current;
          console.log(
            `[WebSocket] Reconnecting in ${delay}ms (attempt ${reconnectAttemptsRef.current})`
          );

          reconnectTimeoutRef.current = setTimeout(() => {
            connect();
          }, delay);
        } else {
          setState((prev) => ({
            ...prev,
            error: "Failed to reconnect after multiple attempts",
          }));
        }
      };

      wsRef.current = ws;
    } catch (err) {
      console.error("[WebSocket] Connection failed:", err);
      setState((prev) => ({
        ...prev,
        error: "Failed to establish WebSocket connection",
      }));
    }
  }, [onMessage]);

  useEffect(() => {
    connect();

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [connect]);

  return state;
}
