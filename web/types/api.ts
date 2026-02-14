/**
 * TypeScript interfaces matching backend response types from cmd/dashboard/response.go
 */

export interface MetricsResponse {
  total_pnl: number;
  total_pnl_percent: number;
  win_rate: number;
  profit_factor: number;
  drawdown: number;
  drawdown_percent: number;
  sharpe_ratio: number;
  total_trades: number;
  winning_trades: number;
  losing_trades: number;
  avg_pnl: number;
  gross_profit: number;
  gross_loss: number;
  avg_hold_days: number;
  initial_capital: number;
  final_capital: number;
  timestamp: string;
}

export interface PositionResponse {
  id: number;
  symbol: string;
  quantity: number;
  entry_price: number;
  entry_time: string;
  stop_loss: number;
  target: number;
  strategy_id: string;
  unrealized_pnl: number;
  unrealized_pnl_percent: number;
}

export interface PositionsResponse {
  positions: PositionResponse[];
  total_capital_used: number;
  available_capital: number;
  capital_utilization_percent: number;
  open_position_count: number;
  timestamp: string;
}

export interface EquityCurvePoint {
  date: string;
  equity: number;
  drawdown: number;
  drawdown_percent: number;
}

export interface EquityCurveResponse {
  points: EquityCurvePoint[];
  start_equity: number;
  final_equity: number;
  max_drawdown: number;
  max_drawdown_percent: number;
  total_return: number;
  total_return_percent: number;
  timestamp: string;
}

export interface StatusResponse {
  is_running: boolean;
  open_positions: number;
  available_capital: number;
  total_capital: number;
  daily_pnl: number;
  message: string;
  timestamp: string;
}

export interface WebSocketMessage {
  type: string;
  data: any;
  timestamp: string;
}

export interface DashboardData {
  metrics: MetricsResponse | null;
  positions: PositionsResponse | null;
  equityCurve: EquityCurveResponse | null;
  status: StatusResponse | null;
  loading: boolean;
  error: string | null;
  connected: boolean;
}
