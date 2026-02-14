export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";
export const WS_BASE_URL =
  process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8081";

export const API_ENDPOINTS = {
  METRICS: "/api/metrics",
  POSITIONS: "/api/positions/open",
  EQUITY_CURVE: "/api/charts/equity",
  STATUS: "/api/status",
  HEALTH: "/health",
};

export const COLORS = {
  PROFIT: "#10b981",
  LOSS: "#ef4444",
  NEUTRAL: "#6b7280",
  PRIMARY: "#3b82f6",
  SECONDARY: "#8b5cf6",
  DARK: "#1f2937",
  LIGHT: "#f3f4f6",
};

export const REFRESH_INTERVALS = {
  WEBSOCKET: 5000, // 5 seconds
  RETRY: 3000, // 3 seconds
  HEALTH_CHECK: 30000, // 30 seconds
};

export const DEFAULT_PAGINATION = {
  PAGE_SIZE: 10,
};
