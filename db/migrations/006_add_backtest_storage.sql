-- Migration 006: Add Backtest Storage Tables
-- Adds tables to persist backtest runs, results, and individual trades

-- Create backtest_runs table to track each backtest execution
CREATE TABLE IF NOT EXISTS backtest_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    strategy_id VARCHAR(50) NOT NULL,
    stocks TEXT[], -- array of symbols, null for all stocks
    parameters JSONB, -- dynamic parameters per strategy
    date_from DATE NOT NULL,
    date_to DATE NOT NULL,
    status VARCHAR(20) DEFAULT 'PENDING', -- PENDING, RUNNING, COMPLETED, FAILED
    progress_percent INT DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for backtest_runs
CREATE INDEX IF NOT EXISTS idx_backtest_runs_created_at ON backtest_runs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_backtest_runs_strategy ON backtest_runs(strategy_id);
CREATE INDEX IF NOT EXISTS idx_backtest_runs_status ON backtest_runs(status);

-- Create backtest_results table for summary metrics per backtest
CREATE TABLE IF NOT EXISTS backtest_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    backtest_run_id UUID NOT NULL REFERENCES backtest_runs(id) ON DELETE CASCADE,
    total_trades INT DEFAULT 0,
    winning_trades INT DEFAULT 0,
    losing_trades INT DEFAULT 0,
    win_rate DECIMAL(5,2) DEFAULT 0,
    total_pnl DECIMAL(15,2) DEFAULT 0,
    pnl_percent DECIMAL(10,2) DEFAULT 0,
    profit_factor DECIMAL(10,2) DEFAULT 0,
    sharpe_ratio DECIMAL(10,2) DEFAULT 0,
    max_drawdown DECIMAL(10,2) DEFAULT 0,
    max_drawdown_date DATE,
    avg_hold_days INT DEFAULT 0,
    best_trade_pnl DECIMAL(15,2),
    worst_trade_pnl DECIMAL(15,2),
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(backtest_run_id)
);

-- Create backtest_trades table for individual simulated trades
CREATE TABLE IF NOT EXISTS backtest_trades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    backtest_run_id UUID NOT NULL REFERENCES backtest_runs(id) ON DELETE CASCADE,
    symbol VARCHAR(20) NOT NULL,
    entry_date DATE NOT NULL,
    exit_date DATE NOT NULL,
    entry_price DECIMAL(10,2),
    exit_price DECIMAL(10,2),
    quantity INT,
    pnl DECIMAL(15,2),
    pnl_percent DECIMAL(10,2),
    exit_reason VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for backtest_trades
CREATE INDEX IF NOT EXISTS idx_backtest_trades_run ON backtest_trades(backtest_run_id);
CREATE INDEX IF NOT EXISTS idx_backtest_trades_symbol ON backtest_trades(symbol);

-- Create view for backtest summary statistics
CREATE OR REPLACE VIEW backtest_runs_summary AS
SELECT
    br.id,
    br.name,
    br.strategy_id,
    br.status,
    br.created_at,
    COALESCE(bres.total_trades, 0) as total_trades,
    COALESCE(bres.winning_trades, 0) as winning_trades,
    COALESCE(bres.win_rate, 0) as win_rate,
    COALESCE(bres.total_pnl, 0) as total_pnl,
    COALESCE(bres.pnl_percent, 0) as pnl_percent,
    COALESCE(bres.sharpe_ratio, 0) as sharpe_ratio,
    COALESCE(bres.max_drawdown, 0) as max_drawdown,
    br.date_from,
    br.date_to
FROM backtest_runs br
LEFT JOIN backtest_results bres ON br.id = bres.backtest_run_id;
