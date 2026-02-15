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

// Stock-related types
export interface Candle {
  date: string;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
}

export interface StockSummary {
  symbol: string;
  latest_date: string;
  latest_close: number;
  high_price: number;
  low_price: number;
  percent_change: number;
  average_volume: number;
  winning_trades_count: number;
  total_pnl: number;
}

export interface StocksListResponse {
  stocks: StockSummary[];
  timestamp: string;
}

export interface StockCandlesResponse {
  symbol: string;
  candles: Candle[];
  from_date: string;
  to_date: string;
  timestamp: string;
}

// Backtest-related types
export interface StrategyParameter {
  name: string;
  type: "float" | "int" | "bool" | "string";
  display_name: string;
  default: number | string | boolean;
  min?: number | string;
  max?: number | string;
  step?: number;
  description?: string;
}

export interface StrategyInfo {
  id: string;
  name: string;
  description: string;
  parameters: StrategyParameter[];
}

export interface StrategiesResponse {
  strategies: StrategyInfo[];
  timestamp: string;
}

export interface BacktestRunRequest {
  name: string;
  strategy_id: string;
  stocks?: string[] | null;
  parameters: Record<string, any>;
  date_from: string;
  date_to: string;
}

export interface BacktestRun {
  id: string;
  name: string;
  strategy_id: string;
  stocks?: string[] | null;
  parameters: Record<string, any>;
  date_from: string;
  date_to: string;
  status: "PENDING" | "RUNNING" | "COMPLETED" | "FAILED";
  progress_percent: number;
  error_message?: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
}

export interface BacktestResults {
  id: string;
  backtest_run_id: string;
  total_trades: number;
  winning_trades: number;
  losing_trades: number;
  win_rate: number;
  total_pnl: number;
  pnl_percent: number;
  profit_factor: number;
  sharpe_ratio: number;
  max_drawdown: number;
  max_drawdown_date: string;
  avg_hold_days: number;
  best_trade_pnl: number;
  worst_trade_pnl: number;
  created_at: string;
}

export interface BacktestTrade {
  id: string;
  backtest_run_id: string;
  symbol: string;
  entry_date: string;
  exit_date: string;
  entry_price: number;
  exit_price: number;
  quantity: number;
  pnl: number;
  pnl_percent: number;
  exit_reason: string;
  created_at: string;
}

export interface BacktestEquityCurvePoint {
  date: string;
  equity: number;
  drawdown: number;
}

export interface BacktestRunResponse {
  backtest_run_id: string;
  status: string;
  message: string;
  timestamp: string;
}

export interface BacktestDetailResponse {
  backtest_run: BacktestRun;
  results: BacktestResults;
  trades: BacktestTrade[];
  equity_curve: BacktestEquityCurvePoint[];
  timestamp: string;
}

export interface BacktestListResponse {
  runs: BacktestRun[];
  total_count: number;
  limit: number;
  offset: number;
  timestamp: string;
}

export interface BacktestComparisonMetric {
  backtest_run_id: string;
  name: string;
  strategy_id: string;
  total_trades: number;
  win_rate: number;
  total_pnl: number;
  sharpe_ratio: number;
  max_drawdown: number;
  profit_factor: number;
  created_at: string;
}

export interface BacktestComparisonResponse {
  comparison: BacktestComparisonMetric[];
  best_by: Record<string, string>;
  timestamp: string;
}
